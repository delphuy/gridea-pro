package facade

import "testing"

// TestCompareSemver 覆盖频道更新链路上的全部关键比较场景。
// 频道版版本号约定：官方版本号 + .huigg.N（点分隔追加），见 compareSemver 注释。
func TestCompareSemver(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		// 官方旧正式版 < 频道版：不得误报「有更新」导致降级
		{"1.2.2", "1.3.0-beta.2.huigg.5", -1},
		{"1.2.9", "1.3.0-beta.2.huigg.5", -1},
		// 同版本（含 v 前缀）：不触发更新
		{"v1.3.0-beta.2.huigg.5", "1.3.0-beta.2.huigg.5", 0},
		// 官方 1.3.1 合并进频道后：更高，触发更新
		{"1.3.1-huigg.1", "1.3.0-beta.2.huigg.5", 1},
		// 官方 1.3.0 正式版合并进频道后：正式版 > beta 基线，触发更新
		{"1.3.0", "1.3.0-beta.2.huigg.5", 1},
		{"1.3.0-huigg.1", "1.3.0-beta.2.huigg.5", 1},
		// 官方 beta 递进（beta.2 → beta.3）：触发更新
		{"1.3.0-beta.3.huigg.1", "1.3.0-beta.2.huigg.5", 1},
		// 频道自修订号递增：触发更新
		{"1.3.0-beta.2.huigg.6", "1.3.0-beta.2.huigg.5", 1},
		{"1.3.0-beta.2.huigg.5", "1.3.0-beta.2.huigg.6", -1},
		{"1.3.0-beta.2.huigg.10", "1.3.0-beta.2.huigg.9", 1}, // 数字按数值：10 > 9
		// build metadata 不参与比较
		{"1.3.0-beta.2.huigg.5+build.1", "1.3.0-beta.2.huigg.5", 0},
		// 无法解析为 semver：一律视为相等，宁可不提示也不误判
		{"", "1.3.0", 0},
		{"latest", "1.3.0", 0},
		{"1.3", "1.3.0", 0},
		{"x.y.z", "1.3.0", 0},
	}
	for _, c := range cases {
		if got := compareSemver(c.a, c.b); got != c.want {
			t.Errorf("compareSemver(%q, %q) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}
