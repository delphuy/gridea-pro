package utils

import (
	"bytes"
	"regexp"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

var (
	// 定义两个全局实例，复用以提升性能
	mdSafe   goldmark.Markdown
	mdUnsafe goldmark.Markdown
)

// dangerousURLScheme 判断链接目标是否为可执行/危险协议。
// 前导空白与控制字符先剥掉（浏览器解析 URL 时会忽略它们，"java\tscript:" 一样可执行），
// 再小写比对，堵住大小写/夹空白绕过。与前端 editor/extensions/index.ts 的 isSafeHref 口径一致。
var controlAndSpace = regexp.MustCompile(`[\x00-\x20]+`)

func isDangerousURL(dest string) bool {
	s := strings.ToLower(controlAndSpace.ReplaceAllString(dest, ""))
	switch {
	case strings.HasPrefix(s, "javascript:"),
		strings.HasPrefix(s, "vbscript:"),
		strings.HasPrefix(s, "file:"):
		return true
	case strings.HasPrefix(s, "data:"):
		// data:image/* 用于内联图片是常见且安全的；其余 data:（尤其 data:text/html）拒绝。
		return !strings.HasPrefix(s, "data:image/")
	}
	return false
}

// urlSchemeRe 匹配「已带协议头」的目标（http: https: mailto: tel: data: javascript: …）。
// scheme 语法参照 RFC 3986：字母开头，后接字母/数字/加号/减号/点。
var urlSchemeRe = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9+.-]*:`)

// normalizeLinkDest 给漏写协议头的链接目标补全 https://（如 [x](www.baidu.com)、
// [x](baidu.com)——按相对路径解析会 404）。规则：已带协议头的一律尊重原样
// （危险协议已在 isDangerousURL 拦截），/ 开头（站内绝对路径）与 # 开头（页内锚点）
// 不碰，其余一律补。返回空串表示无需处理。
func normalizeLinkDest(dest string) string {
	s := strings.TrimLeft(dest, " \t\n\r\x00") // 浏览器解析 URL 时忽略前导空白/控制符
	if s == "" {
		return ""
	}
	if urlSchemeRe.MatchString(s) {
		return ""
	}
	if strings.HasPrefix(s, "/") || strings.HasPrefix(s, "#") {
		return ""
	}
	return "https://" + s
}

// linkProtocolGuard 是 goldmark AST Transformer：渲染前把危险协议的链接目标清空，
// 使 [文字](javascript:...) 发布后不再产出可点击的 href；同时给漏写协议头的链接
// 补全 https://，构建产物里 href 直接是正确的绝对地址。
// 纵深防御——前端 SafeLink 已在 Markdown 落盘前拦截危险协议，这里再兜一道，
// 覆盖历史脏数据与非编辑器写入的 .md。
type linkProtocolGuard struct{}

func (linkProtocolGuard) Transform(node *ast.Document, reader text.Reader, _ parser.Context) {
	_ = ast.Walk(node, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		if link, ok := n.(*ast.Link); ok {
			if isDangerousURL(string(link.Destination)) {
				link.Destination = []byte("")
			} else if patched := normalizeLinkDest(string(link.Destination)); patched != "" {
				link.Destination = []byte(patched)
			}
		}
		if al, ok := n.(*ast.AutoLink); ok {
			if isDangerousURL(string(al.URL(reader.Source()))) {
				// AutoLink 的 URL 由源文本派生，无法就地改写为安全值，直接降级为纯文本节点。
				textNode := ast.NewString(al.Label(reader.Source()))
				n.Parent().ReplaceChild(n.Parent(), n, textNode)
			}
		}
		return ast.WalkContinue, nil
	})
}

func linkGuardOption() goldmark.Option {
	return goldmark.WithParserOptions(
		parser.WithASTTransformers(util.Prioritized(linkProtocolGuard{}, 999)),
	)
}

func init() {
	mdSafe = goldmark.New(
		goldmark.WithExtensions(extension.GFM, extension.Typographer, extension.Footnote, KatexExtension(), DialectExtension(), SmartQuoteExtension()),
		goldmark.WithParserOptions(parser.WithAutoHeadingID()),
		linkGuardOption(),
		goldmark.WithRendererOptions(html.WithHardWraps(), html.WithXHTML()),
	)

	mdUnsafe = goldmark.New(
		goldmark.WithExtensions(extension.GFM, extension.Typographer, extension.Footnote, KatexExtension(), DialectExtension(), SmartQuoteExtension()),
		goldmark.WithParserOptions(parser.WithAutoHeadingID()),
		linkGuardOption(),
		goldmark.WithRendererOptions(
			html.WithHardWraps(),
			html.WithXHTML(),
			html.WithUnsafe(),
		),
	)
}

// ToHTML 将 Markdown 文本转换为 HTML
func ToHTML(markdown string) string {
	return convert(mdSafe, markdown)
}

// ToHTMLUnsafe 将 Markdown 文本转换为 HTML（允许原始 HTML）
// 警告: 此函数允许 Markdown 中的原始 HTML，可能存在 XSS 风险
func ToHTMLUnsafe(markdown string) string {
	return convert(mdUnsafe, markdown)
}

// 统一的内部转换逻辑
func convert(engine goldmark.Markdown, markdown string) string {
	if markdown == "" {
		return ""
	}
	var buf bytes.Buffer
	if err := engine.Convert([]byte(markdown), &buf); err != nil {
		// fallback: simple wrapper
		return "<p>" + markdown + "</p>"
	}
	return buf.String()
}
