package utils

import "testing"

// TestNormalizeLinkDest 直接校验补全函数的判断口径（2026-11-18 辉哥哥拍板）：
// 只要没写协议头又不是站内路径/页内锚点，一律补 https://，
// 不去识别点分结构和域名后缀。
func TestNormalizeLinkDest(t *testing.T) {
	cases := []struct{ name, in, want string }{
		{"裸域名补全", "baidu.com", "https://baidu.com"},
		{"www 开头", "www.baidu.com", "https://www.baidu.com"},
		{"前导空白剥掉", " www.baidu.com", "https://www.baidu.com"},
		{"http 已有不动", "http://baidu.com", ""},
		{"https 已有不动", "https://baidu.com", ""},
		{"大写协议头也认", "HTTP://baidu.com", ""},
		{"站内绝对路径", "/post/a/", ""},
		{"协议相对链接", "//example.com/x", ""},
		{"页内锚点", "#section", ""},
		{"mailto 不动", "mailto:a@b.com", ""},
		{"tel 不动", "tel:13800000000", ""},
		{"data 图床链接不动", "data:image/png;base64,xxx", ""},
		{"空目标不动", "", ""},
	}
	for _, c := range cases {
		if got := normalizeLinkDest(c.in); got != c.want {
			t.Errorf("[%s] 得到 %q，期望 %q", c.name, got, c.want)
		}
	}
}

// TestLinkGuardPatchesHref 端到端：md 语法链接漏写协议头，渲染产物里 href 直接是
// 补全后的绝对地址；原有各形态不误伤；危险协议仍被清空。
func TestLinkGuardPatchesHref(t *testing.T) {
	cases := []struct{ name, md, want string }{
		{"裸域名补全", "[baidu 搜索](www.baidu.com)", `<a href="https://www.baidu.com">baidu 搜索</a>`},
		{"无 www 也补", "[x](baidu.com)", `<a href="https://baidu.com">x</a>`},
		{"已有 https 不动", "[x](https://baidu.com)", `<a href="https://baidu.com">x</a>`},
		{"站内路径不动", "[x](/post/a/)", `<a href="/post/a/">x</a>`},
		{"页内锚点不动", "[x](#sec)", `<a href="#sec">x</a>`},
		{"mailto 不动", "[发邮件](mailto:a@b.com)", `<a href="mailto:a@b.com">发邮件</a>`},
		{"危险协议仍清空", "[x](javascript:alert(1))", `<a href="">x</a>`},
	}
	for _, c := range cases {
		if got := body(c.md); got != c.want {
			t.Errorf("[%s] 得到 %q，期望 %q", c.name, got, c.want)
		}
	}
}
