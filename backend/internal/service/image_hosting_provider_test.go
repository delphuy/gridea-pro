package service

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"regexp"
	"strings"
	"testing"

	"gridea-pro/backend/internal/domain"
)

// ── providerName：历史配置兼容 ────────────────────────────────────────

func TestProviderNameEmptyFallsBackToSEE(t *testing.T) {
	if got := providerName(&domain.ImageHostingSetting{APIKey: "k"}); got != domain.ImageHostingProviderSEE {
		t.Fatalf("空 provider 应按 see 处理，得到 %q", got)
	}
	if got := providerName(&domain.ImageHostingSetting{Provider: "gitee"}); got != domain.ImageHostingProviderGitee {
		t.Fatalf("gitee 归一化失败: %q", got)
	}
}

// 旧配置文件（只有 enabled/apiKey 两个字段）必须能反序列化。
func TestImageHostingSettingBackCompatJSON(t *testing.T) {
	var s domain.ImageHostingSetting
	if err := json.Unmarshal([]byte(`{"enabled":true,"apiKey":"old-key"}`), &s); err != nil {
		t.Fatalf("旧配置反序列化失败: %v", err)
	}
	if !s.Enabled || s.APIKey != "old-key" || s.Provider != "" || s.Gitee != nil || s.OSS != nil {
		t.Fatalf("旧配置字段不符合预期: %+v", s)
	}

	// 新配置 round-trip：嵌套字段不丢，omitempty 生效
	full := domain.ImageHostingSetting{
		Enabled:  true,
		Provider: "oss",
		OSS:      &domain.ImageHostingOSSSetting{AccessKeyID: "ak", AccessKeySecret: "sk", Bucket: "b", Endpoint: "oss-cn-shanghai.aliyuncs.com"},
	}
	buf, err := json.Marshal(&full)
	if err != nil {
		t.Fatalf("序列化失败: %v", err)
	}
	if strings.Contains(string(buf), "gitee") {
		t.Fatalf("空 gitee 配置不应序列化: %s", buf)
	}
	var back domain.ImageHostingSetting
	if err := json.Unmarshal(buf, &back); err != nil {
		t.Fatalf("round-trip 失败: %v", err)
	}
	if back.OSS == nil || back.OSS.Bucket != "b" || back.OSS.AccessKeySecret != "sk" {
		t.Fatalf("OSS 嵌套配置丢失: %+v", back.OSS)
	}
}

// ── 存储名与前缀拼接 ─────────────────────────────────────────────────

func TestGenerateStoredName(t *testing.T) {
	name := generateStoredName(`C:\博客图片\岁月 痕迹.PNG`)
	if m := regexp.MustCompile(`^\d{13}_.*\.PNG$`).FindString(name); m == "" {
		t.Fatalf("存储名应为 <13位毫秒时间戳>_<净化名>: %q", name)
	}
	if strings.ContainsAny(name, " \\/:*?\"<>|") || strings.Contains(name, "岁") {
		t.Fatalf("存储名含未净化字符: %q", name)
	}
	if generateStoredName("///") == "" {
		t.Fatalf("极端输入应回退为默认名")
	}
}

func TestJoinPathPrefix(t *testing.T) {
	cases := []struct{ prefix, name, want string }{
		{"", "a.png", "a.png"},
		{"images", "a.png", "images/a.png"},
		{"images/", "a.png", "images/a.png"},
		{"/img/", "a.png", "img/a.png"},
	}
	for _, c := range cases {
		if got := joinPathPrefix(c.prefix, c.name); got != c.want {
			t.Fatalf("joinPathPrefix(%q,%q) = %q, want %q", c.prefix, c.name, got, c.want)
		}
	}
}

// ── Gitee ────────────────────────────────────────────────────────────

func TestGiteePublicURL(t *testing.T) {
	cfg := &domain.ImageHostingGiteeSetting{Owner: "hui", Repo: "img-bed", Branch: "master"}
	got := giteePublicURL(cfg, "images/岁月.png")
	want := "https://gitee.com/hui/img-bed/raw/master/images/%E5%B2%81%E6%9C%88.png"
	if got != want {
		t.Fatalf("默认 raw 地址错误:\n got %q\nwant %q", got, want)
	}
	cfg.PublicBaseURL = "https://cdn.example.com/base/"
	if got, want := giteePublicURL(cfg, "a b.png"), "https://cdn.example.com/base/a%20b.png"; got != want {
		t.Fatalf("自定义基地址错误: %q, want %q", got, want)
	}
}

func TestGiteeBranchDefault(t *testing.T) {
	if got := giteeBranch(&domain.ImageHostingGiteeSetting{}); got != "main" {
		t.Fatalf("默认分支应为 main, got %q", got)
	}
}

// ── OSS ──────────────────────────────────────────────────────────────

func TestOSSHost(t *testing.T) {
	cfg := &domain.ImageHostingOSSSetting{Bucket: "huigg-img", Endpoint: "oss-cn-shanghai.aliyuncs.com"}
	if got, want := ossHost(cfg), "huigg-img.oss-cn-shanghai.aliyuncs.com"; got != want {
		t.Fatalf("默认主机名错误: %q", got)
	}
	cfg.Endpoint = "https://oss-cn-hangzhou.aliyuncs.com/"
	if got, want := ossHost(cfg), "huigg-img.oss-cn-hangzhou.aliyuncs.com"; got != want {
		t.Fatalf("带协议的 endpoint 未归一化: %q", got)
	}
	cfg.CustomDomain = "img.huigg.xyz/"
	if got, want := ossHost(cfg), "img.huigg.xyz"; got != want {
		t.Fatalf("自定义域名未归一化: %q", got)
	}
	cfg.CustomDomain = ""
	cfg.Endpoint = "oss-cn-shanghai"
	if got, want := ossHost(cfg), "huigg-img.oss-cn-shanghai.aliyuncs.com"; got != want {
		t.Fatalf("纯区域名 endpoint 未自动补全 .aliyuncs.com: %q", got)
	}
}

// ossSign 的关键风险是 StringToSign 布局（连续换行、x-oss 头、资源名）。
// 用测试内独立拼接的期望串锁死布局，防回归。
func TestOSSSignLayout(t *testing.T) {
	cfg := &domain.ImageHostingOSSSetting{AccessKeyID: "testid", AccessKeySecret: "testsecret"}
	const date = "Wed, 01 Jan 2025 00:00:00 GMT"

	// PUT 对象：Content-MD5 为空但仍要占一个 \n
	putWantToSign := "PUT\n\nimage/png\n" + date + "\nx-oss-object-acl:public-read\n/huigg-img/blog/1_a.png"
	mac := hmac.New(sha1.New, []byte(cfg.AccessKeySecret))
	mac.Write([]byte(putWantToSign))
	want := "OSS testid:" + base64.StdEncoding.EncodeToString(mac.Sum(nil))
	if got := ossSign("PUT", cfg, "image/png", date, "x-oss-object-acl:public-read\n", "/huigg-img/blog/1_a.png"); got != want {
		t.Fatalf("PUT 签名布局错误:\n got %q\nwant %q", got, want)
	}

	// GET 列表：无 Content-Type、无 x-oss 头
	getWantToSign := "GET\n\n\n" + date + "\n/huigg-img/"
	mac = hmac.New(sha1.New, []byte(cfg.AccessKeySecret))
	mac.Write([]byte(getWantToSign))
	want = "OSS testid:" + base64.StdEncoding.EncodeToString(mac.Sum(nil))
	if got := ossSign("GET", cfg, "", date, "", "/huigg-img/"); got != want {
		t.Fatalf("GET 签名布局错误:\n got %q\nwant %q", got, want)
	}
}

func TestDetectImageContentType(t *testing.T) {
	if got := detectImageContentType(nil, "x.svg"); got != "image/svg+xml" {
		t.Fatalf("svg 应按扩展名识别: %q", got)
	}
	pngHeader := []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A, 0, 0, 0, 0}
	if got := detectImageContentType(pngHeader, "noext"); got != "image/png" {
		t.Fatalf("PNG 嗅探失败: %q", got)
	}
	if got := detectImageContentType([]byte("<html>text</html>"), "noext"); got != "application/octet-stream" {
		t.Fatalf("文本内容不应当作图片类型: %q", got)
	}
}

// ── 测试图常量：必须是合法 PNG ────────────────────────────────────────

func TestTestPNG1x1IsValid(t *testing.T) {
	data, err := base64.StdEncoding.DecodeString(testPNG1x1)
	if err != nil {
		t.Fatalf("测试图 base64 非法: %v", err)
	}
	if len(data) < 8 || data[0] != 0x89 || string(data[1:4]) != "PNG" {
		t.Fatalf("测试图不是 PNG: %x", data[:8])
	}
}

// ── 列表排序：刚传的图排最前 ──────────────────────────────────────────

func TestSortImageHostingFilesNewestFirst(t *testing.T) {
	files := []domain.ImageHostingFile{
		{Filename: "1000_old.png", Path: "blog/1000_old.png", CreatedAt: 1000},
		{Filename: "1731920000000_newer.png", Path: "blog/1731920000000_newer.png"}, // 无时间：从名字前缀回填
		{Filename: "b_no_time.png", Path: "blog/b_no_time.png"},                     // 无时间无名前缀：沉底
		{Filename: "500_older.png", Path: "blog/500_older.png", CreatedAt: 500},
		{Filename: "a_no_time.png", Path: "blog/a_no_time.png"}, // 沉底项应保持原相对顺序（b 在 a 前 = 稳定）
		{Filename: "1731910000000_mid.png", Path: "blog/1731910000000_mid.png"},
	}
	sortImageHostingFilesNewestFirst(files)

	want := []string{
		"1731920000000_newer.png", // 名字回填 1731920000 秒，最新
		"1731910000000_mid.png",   // 名字回填 1731910000 秒
		"1000_old.png",            // 服务商给的 1000 秒
		"500_older.png",           // 服务商给的 500 秒
		"b_no_time.png",           // 拿不到时间：沉底但保持原顺序
		"a_no_time.png",
	}
	for i, w := range want {
		if files[i].Filename != w {
			var got []string
			for _, f := range files {
				got = append(got, f.Filename)
			}
			t.Fatalf("排序后第 %d 位应为 %q，实际 %q（整体: %v）", i, w, files[i].Filename, got)
		}
	}
	// 回填的时间应该真正写回条目（gitee 场景下前端也能按时间展示）
	if files[0].CreatedAt != 1731920000 {
		t.Fatalf("名字前缀时间戳未回填到 CreatedAt: %d", files[0].CreatedAt)
	}
}

func TestCreatedMilliFromName(t *testing.T) {
	cases := []struct {
		key  string
		want int64
	}{
		{"1731920000000_a.png", 1731920000000},
		{"blog/1731920000000_a.png", 1731920000000}, // 带目录前缀也取文件名
		{"123_too_early.png", 0},                    // 早于 2000 年，视为非法
		{"99999999999999_far_future.png", 0},        // 晚于 2100 年，视为非法
		{"abc_no_digit.png", 0},
		{"_leading.png", 0},
		{"", 0},
	}
	for _, c := range cases {
		if got := createdMilliFromName(c.key); got != c.want {
			t.Fatalf("createdMilliFromName(%q) = %d, want %d", c.key, got, c.want)
		}
	}
}
