package service

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// newMediaRefFixture 构造一个最小站点目录，覆盖各类引用形态。
func newMediaRefFixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	// 文章：完整路径引用、裸文件名引用、URL 转义中文文件名引用
	post := `---
title: 测试
featureImage: /post-images/1.png
---

![](post-images/1.png)
![logo](logo.png)
![](/post-images/%E5%B2%81%E6%9C%88.png)
`
	if err := os.MkdirAll(filepath.Join(dir, "posts"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "posts", "hello.md"), []byte(post), 0o644); err != nil {
		t.Fatal(err)
	}

	// memos：images 数组引用（带前导斜杠）
	memos := `{"memos":[{"id":"m1","content":"memo","images":["/post-images/2.png"]}]}`
	if err := os.MkdirAll(filepath.Join(dir, "config"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config", "memos.json"), []byte(memos), 0o644); err != nil {
		t.Fatal(err)
	}

	// 主题配置：引用 images/bg.png
	themeCfg := `{"avatar": "images/bg.png"}`
	if err := os.MkdirAll(filepath.Join(dir, "themes", "fluid"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "themes", "fluid", "config.json"), []byte(themeCfg), 0o644); err != nil {
		t.Fatal(err)
	}

	// 媒体文件全集
	for _, rel := range []string{
		"post-images/1.png",    // 正文引用
		"post-images/2.png",    // memo 引用
		"post-images/3.png",   // 孤儿
		"post-images/11.png",  // 边界：文件名是 1.png 的超串，不能被 1.png 的引用"顺带"救活
		"post-images/岁月.png", // 转义引用
		"images/logo.png",     // 裸文件名引用
		"images/bg.png",       // 主题配置引用
		"media/video.mp4",     // 孤儿（media 目录同样纳入治理）
	} {
		p := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestContainsTokenBoundary(t *testing.T) {
	tests := []struct {
		corpus string
		token  string
		want   bool
	}{
		{"post-images/11.png", "1.png", false},              // 文件名超串不能误命中
		{"xx post-images/1.png xx", "post-images/1.png", true},
		{"mylogo.pngx", "logo.png", false},                   // 前后都是文件名字符
		{"/images/logo.png", "logo.png", true},               // 前是'/'后是结尾
		{"a/logo.png/b", "logo.png", true},                   // 前后都是'/'
		{"", "1.png", false},                                 // 空语料
		{"1.png", "", false},                                 // 空 token
	}
	for _, tt := range tests {
		if got := containsToken(tt.corpus, tt.token); got != tt.want {
			t.Errorf("containsToken(%q, %q) = %v, want %v", tt.corpus, tt.token, got, tt.want)
		}
	}
}

func TestEscapePathSegments(t *testing.T) {
	got := escapePathSegments("post-images/岁月.png")
	if got != "post-images/%E5%B2%81%E6%9C%88.png" {
		t.Errorf("escapePathSegments = %q", got)
	}
	if got := escapePathSegments("images/a.png"); got != "images/a.png" {
		t.Errorf("escapePathSegments(ascii) = %q, want unchanged", got)
	}
}

func TestScanOrphanImages(t *testing.T) {
	dir := newMediaRefFixture(t)
	svc := NewMediaRefService(dir)

	res, err := svc.ScanOrphanImages(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if res.Total != 8 {
		t.Errorf("Total = %d, want 8", res.Total)
	}

	got := make([]string, 0, len(res.Orphans))
	for _, o := range res.Orphans {
		got = append(got, o.RelPath)
	}
	want := []string{"media/video.mp4", "post-images/11.png", "post-images/3.png"} // 排序后
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("Orphans = %v, want %v", got, want)
	}

	// 全部被引用的文件在移动前后都原样存在
	for _, rel := range []string{"post-images/1.png", "post-images/2.png", "post-images/岁月.png", "images/logo.png", "images/bg.png"} {
		if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(rel))); err != nil {
			t.Errorf("被引用文件不应被扫描动过: %s: %v", rel, err)
		}
	}
}

func TestMoveOrphanImagesToTrashFlow(t *testing.T) {
	dir := newMediaRefFixture(t)
	svc := NewMediaRefService(dir)
	ctx := context.Background()

	// 1) 移入回收站
	move, err := svc.MoveOrphanImagesToTrash(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(move.Moved) != 3 {
		t.Fatalf("Moved = %d, want 3", len(move.Moved))
	}
	if !strings.HasPrefix(move.TrashDir, ".trash-") {
		t.Errorf("TrashDir = %q", move.TrashDir)
	}
	// 源位置已消失，回收站内按原目录结构存在
	for _, m := range move.Moved {
		if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(m.RelPath))); !os.IsNotExist(err) {
			t.Errorf("孤儿 %s 应已从源目录移走", m.RelPath)
		}
		dst := filepath.Join(dir, move.TrashDir, filepath.FromSlash(m.RelPath))
		if _, err := os.Stat(dst); err != nil {
			t.Errorf("回收站内应存在 %s: %v", dst, err)
		}
	}

	// 2) 回收站列表
	list, err := svc.ListMediaTrash(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(list.Dirs) != 1 || list.FileCount != 3 || list.SizeByte != 3 {
		t.Errorf("ListMediaTrash = %+v, want 1 dir / 3 files / 3 bytes", list)
	}

	// 3) 再次扫描：孤儿清零（已在回收站，不在媒体目录里）
	res, err := svc.ScanOrphanImages(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Orphans) != 0 {
		t.Errorf("移入回收站后 Orphans = %v, want empty", res.Orphans)
	}

	// 4) 清空回收站
	removed, err := svc.EmptyMediaTrash(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if removed != 1 {
		t.Errorf("EmptyMediaTrash removed = %d, want 1", removed)
	}
	list2, _ := svc.ListMediaTrash(ctx)
	if list2.FileCount != 0 {
		t.Errorf("清空后 FileCount = %d, want 0", list2.FileCount)
	}
}

func TestBuildMediaRefCorpusErrorOnBadDir(t *testing.T) {
	dir := t.TempDir() // 空目录：没有 posts
	if _, err := BuildMediaRefCorpus(dir); err == nil {
		t.Error("空站点目录应返回错误，避免孤儿扫描在异常目录下误判")
	}
}

func TestBuildMediaRefCorpusSkipsOutputAndTrash(t *testing.T) {
	dir := newMediaRefFixture(t)
	// output 里的副本不能算引用语料
	if err := os.MkdirAll(filepath.Join(dir, "posts", "output"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "posts", "output", "a.md"), []byte("![](post-images/orphan-in-output.png)"), 0o644); err != nil {
		t.Fatal(err)
	}
	corpus, err := BuildMediaRefCorpus(dir)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(corpus, "orphan-in-output.png") {
		t.Error("posts/output 下的文件不应进入引用语料")
	}
}
