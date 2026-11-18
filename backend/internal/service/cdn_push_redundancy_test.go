package service

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"gridea-pro/backend/internal/domain"
)

// fakeDeploySettingRepo 按构造时的预设返回部署 Setting，模拟 repository 层。
type fakeDeploySettingRepo struct {
	setting domain.Setting
}

func (f fakeDeploySettingRepo) GetSetting(ctx context.Context) (domain.Setting, error) {
	return f.setting, nil
}

func (f fakeDeploySettingRepo) SaveSetting(ctx context.Context, _ domain.Setting) error {
	return nil
}

// ghDeploySetting 构造 github 平台部署配置的 Setting。
func ghDeploySetting(platforms []string, user, repo, branch string) domain.Setting {
	return domain.Setting{
		Platform:         "github",
		EnabledPlatforms: platforms,
		PlatformConfigs: map[string]map[string]any{
			"github": {
				"username":   user,
				"repository": repo,
				"branch":     branch,
			},
		},
	}
}

// TestBuildPushRedundancySkip 回归（2026-11-18 huigg 同步事故）：CDN 仓库与
// github 部署仓库同源且 output 有副本时必须跳过单独上传；任何条件不满足都
// 不得跳过——宁可不跳，不可误跳（误跳 = push 不经过 CDN 仓库的文件被漏传）。
func TestBuildPushRedundancySkip(t *testing.T) {
	// 临时 appDir：output/post-images/a.jpg 存在，b.jpg 不存在
	appDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(appDir, "output", "post-images"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "output", "post-images", "a.jpg"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	cdnSame := domain.CdnSetting{GithubUser: "delphuy", GithubRepo: "delphuy.github.io", GithubBranch: "main"}

	cases := []struct {
		name    string
		setting domain.Setting
		cdn     domain.CdnSetting
		wantNil bool // true = 期望不启用跳过（返回 nil）
	}{
		{"完全一致_启用跳过", ghDeploySetting([]string{"github"}, "delphuy", "delphuy.github.io", "main"), cdnSame, false},
		{"分支缺省_双方都按main", ghDeploySetting([]string{"github"}, "delphuy", "delphuy.github.io", ""), domain.CdnSetting{GithubUser: "delphuy", GithubRepo: "delphuy.github.io"}, false},
		{"用户名大小写不敏感", ghDeploySetting([]string{"github"}, "Delphuy", "delphuy.github.io", "main"), cdnSame, false},
		{"多平台含github_启用跳过", ghDeploySetting([]string{"vercel", "github"}, "delphuy", "delphuy.github.io", "main"), cdnSame, false},
		{"平台列表缺省_按当前平台", ghDeploySetting(nil, "delphuy", "delphuy.github.io", "main"), cdnSame, false},
		{"仓库不同_不跳过", ghDeploySetting([]string{"github"}, "delphuy", "cdn-assets", "main"), cdnSame, true},
		{"用户不同_不跳过", ghDeploySetting([]string{"github"}, "someone", "delphuy.github.io", "main"), cdnSame, true},
		{"分支不同_不跳过", ghDeploySetting([]string{"github"}, "delphuy", "delphuy.github.io", "master"), cdnSame, true},
		{"github未启用_不跳过", ghDeploySetting([]string{"vercel"}, "delphuy", "delphuy.github.io", "main"), cdnSame, true},
		{"部署仓库为空_不跳过", ghDeploySetting([]string{"github"}, "", "", "main"), cdnSame, true},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			svc := &CdnUploadService{settingRepo: fakeDeploySettingRepo{setting: tt.setting}}
			skip := svc.buildPushRedundancySkip(tt.cdn, appDir)
			if tt.wantNil {
				if skip != nil {
					t.Errorf("期望返回 nil（不跳过），got 非 nil")
				}
				return
			}
			if skip == nil {
				t.Fatalf("期望启用跳过，got nil")
			}
			if !skip("post-images/a.jpg") {
				t.Errorf("output 存在副本的文件应命中跳过")
			}
			if skip("post-images/b.jpg") {
				t.Errorf("output 无副本的文件不得跳过（push 不会交付它）")
			}
		})
	}

	// settingRepo 缺失：不跳过，保持原有 CDN 上传行为
	if skip := (&CdnUploadService{}).buildPushRedundancySkip(cdnSame, appDir); skip != nil {
		t.Errorf("settingRepo 为 nil 时应返回 nil")
	}
}
