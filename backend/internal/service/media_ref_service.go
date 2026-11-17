package service

import (
	"context"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// 图片引用治理（用户需求 4.1 / 4.2）：
//
//	4.1 部署 CDN 时不再盲扫推送全部本地图片，只推送被文章 / 配置引用的文件
//	    （纯外链图片的站点不再向 CDN 仓库推无用图片）。
//	4.2 提供"孤儿扫描 → 移入回收站 → 清空回收站"三段式治理；移入回收站后，
//	    本地源文件消失，output 副本随渲染 manifest 清理自动移除，
//	    CDN 远端副本随部署 manifest 孤儿清理（#96）自动删除。
//
// 引用语料 = 所有可能书写本地图片路径的文本源：
//
//	posts/**/*.md     —— 文章正文、front matter（featureImage 等）
//	config/**/*.json  —— 站点设置、菜单图标、链接图标、memos.json 的 images 数组
//	themes/**/*.json  —— 主题自定义配置（头像 / 背景图等）
//	config.json       —— 兼容设置文件在站点根目录的历史布局

// mediaRefDirs 与 cdn_upload_service.UploadMediaForDeploy / asset_manager.CopySiteAssets
// 扫描的媒体目录保持一致。
var mediaRefDirs = []string{"post-images", "images", "media"}

// trashPrefix 回收站目录前缀（位于站点根目录，.trash-<时间戳>）。
const trashPrefix = ".trash-"

// mediaRefFileMaxBytes 引用语料单文件读取上限：防止误读到异常大文件拖垮扫描。
const mediaRefFileMaxBytes = 8 << 20

// MediaFile 描述站点里的一个本地媒体文件。
type MediaFile struct {
	RelPath  string `json:"relPath"`  // 相对站点根目录的正斜杠路径（如 post-images/xxx.png）
	SizeByte int64  `json:"sizeByte"` // 文件大小（字节）
}

// OrphanScanResult 孤儿扫描结果（干跑预览，不移动任何文件）。
type OrphanScanResult struct {
	Total   int         `json:"total"`   // 本地媒体文件总数
	Orphans []MediaFile `json:"orphans"` // 未被引用的文件列表（按路径排序）
}

// TrashMoveResult 移入回收站的结果。
type TrashMoveResult struct {
	TrashDir string     `json:"trashDir"` // 回收站目录名（.trash-<时间戳>）
	Moved    []MediaFile `json:"moved"`   // 实际移入的文件列表
}

// TrashInfo 一个回收站目录的信息。
type TrashInfo struct {
	Dir       string `json:"dir"`       // 目录名（.trash-<时间戳>）
	FileCount int    `json:"fileCount"` // 文件数
	SizeByte  int64  `json:"sizeByte"`  // 总大小（字节）
	CreatedAt string `json:"createdAt"` // 创建时间
}

// TrashListResult 回收站汇总。
type TrashListResult struct {
	Dirs      []TrashInfo `json:"dirs"`
	FileCount int         `json:"fileCount"`
	SizeByte  int64       `json:"sizeByte"`
}

// MediaRefService 站点级服务：引用扫描、孤儿回收站。
// appDir 随切站由 facade 层热替换（见 facade.AppServices.UpdateAppDir）。
type MediaRefService struct {
	appDir string
}

func NewMediaRefService(appDir string) *MediaRefService {
	return &MediaRefService{appDir: appDir}
}

// BuildMediaRefCorpus 构建引用语料。汇成一个大字符串供匹配，百级文章站点下足够快。
// 单文件读取失败只跳过（尽力而为）；完全读不到任何文件才算失败。
func BuildMediaRefCorpus(appDir string) (string, error) {
	var sb strings.Builder

	appendFile := func(path string) {
		info, err := os.Stat(path)
		if err != nil || !info.Mode().IsRegular() || info.Size() > mediaRefFileMaxBytes {
			return
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return
		}
		sb.Write(data)
		sb.WriteByte('\n')
	}

	addWalk := func(root string, keepExt string) {
		// 目录不存在是合法初始状态（新站没有 posts/config 都正常），直接跳过
		if _, err := os.Stat(root); err != nil {
			return
		}
		_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil // 单个条目读不了就跳过，保持尽力而为
			}
			if d.IsDir() {
				// 跳过输出目录与回收站，避免 output 里的副本被当成"引用语料"
				if d.Name() == "output" || strings.HasPrefix(d.Name(), trashPrefix) {
					return fs.SkipDir
				}
				return nil
			}
			if keepExt != "" && !strings.EqualFold(filepath.Ext(path), keepExt) {
				return nil
			}
			appendFile(path)
			return nil
		})
	}

	addWalk(filepath.Join(appDir, "posts"), ".md")
	addWalk(filepath.Join(appDir, "config"), ".json")
	addWalk(filepath.Join(appDir, "themes"), ".json")
	appendFile(filepath.Join(appDir, "config.json"))

	// 站点连 posts/config/themes 都不存在：视为异常目录，交由调用方决定
	// （上传过滤会退回"全量上传"旧行为；孤儿扫描则报错，绝不误删）。
	if sb.Len() == 0 {
		if _, err := os.Stat(filepath.Join(appDir, "posts")); err != nil {
			return "", fmt.Errorf("站点目录异常（posts 目录不存在）: %w", err)
		}
	}
	return sb.String(), nil
}

// mediaReferenced 判断 relPath（正斜杠相对路径）是否被语料引用。
// 匹配策略取向保守——宁可多上传 / 少回收，不可漏删：
//  1. 完整相对路径匹配：post-images/1.png
//  2. URL 转义变体匹配：编辑器写出的 src 常是百分号编码（中文文件名场景）
//  3. 文件名兜底匹配：语料里只写了裸文件名（1.png）时也视为引用
func mediaReferenced(corpus, slashRel string) bool {
	if containsToken(corpus, slashRel) {
		return true
	}
	if escaped := escapePathSegments(slashRel); escaped != slashRel && containsToken(corpus, escaped) {
		return true
	}
	if base := filepath.Base(slashRel); base != slashRel && containsToken(corpus, base) {
		return true
	}
	return false
}

// containsToken 边界感知的子串查找：token 前后的字符不能是文件名字符，
// 避免 "1.png" 误命中 "11.png"、"logo.png" 误命中 "mylogo.pngx"。
func containsToken(corpus, token string) bool {
	if token == "" {
		return false
	}
	off := 0
	for {
		i := strings.Index(corpus[off:], token)
		if i < 0 {
			return false
		}
		i += off
		if !isNameByteAt(corpus, i-1) && !isNameByteAt(corpus, i+len(token)) {
			return true
		}
		off = i + 1
	}
}

func isNameByteAt(s string, idx int) bool {
	if idx < 0 || idx >= len(s) {
		return false
	}
	c := s[idx]
	if c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' {
		return true
	}
	switch c {
	case '-', '_', '.', '~', '%':
		return true
	}
	return false
}

// escapePathSegments 对路径逐段做 URL path 转义，再拼回正斜杠路径。
func escapePathSegments(slashRel string) string {
	segs := strings.Split(slashRel, "/")
	for i, s := range segs {
		segs[i] = url.PathEscape(s)
	}
	return strings.Join(segs, "/")
}

// listMediaFiles 枚举三个媒体目录下的所有文件（物理全集）。
func listMediaFiles(appDir string) ([]MediaFile, error) {
	var files []MediaFile
	for _, dir := range mediaRefDirs {
		dirPath := filepath.Join(appDir, dir)
		if _, err := os.Stat(dirPath); err != nil {
			continue
		}
		err := filepath.WalkDir(dirPath, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			info, err := d.Info()
			if err != nil {
				return err
			}
			rel, err := filepath.Rel(appDir, path)
			if err != nil {
				return err
			}
			files = append(files, MediaFile{
				RelPath:  filepath.ToSlash(rel),
				SizeByte: info.Size(),
			})
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("扫描目录 %s 失败: %w", dir, err)
		}
	}
	return files, nil
}

// ScanOrphanImages 干跑扫描：返回未被引用的本地媒体文件列表，不移动任何文件。
func (s *MediaRefService) ScanOrphanImages(ctx context.Context) (OrphanScanResult, error) {
	var res OrphanScanResult
	if err := ctx.Err(); err != nil {
		return res, err
	}
	files, err := listMediaFiles(s.appDir)
	if err != nil {
		return res, err
	}
	corpus, err := BuildMediaRefCorpus(s.appDir)
	if err != nil {
		return res, err
	}
	res.Total = len(files)
	for _, f := range files {
		if !mediaReferenced(corpus, f.RelPath) {
			res.Orphans = append(res.Orphans, f)
		}
	}
	sort.Slice(res.Orphans, func(i, j int) bool {
		return res.Orphans[i].RelPath < res.Orphans[j].RelPath
	})
	return res, nil
}

// MoveOrphanImagesToTrash 把当前扫描到的孤儿文件移入 .trash-<时间戳> 回收站目录。
//
// 执行时重新扫描（而不是信任前端预览列表），避免"预览之后用户新写了引用"造成误删。
// 移入回收站后：
//   - 本地源文件消失 → 下次渲染 output 副本随 manifest 清理自动移除
//   - 下次部署时 CDN manifest 差分（#96 孤儿清理）把远端副本一并删除
//   - 反悔：在站点根目录 .trash-<时间戳>/ 里按原目录结构手动把文件移回即可
func (s *MediaRefService) MoveOrphanImagesToTrash(ctx context.Context) (TrashMoveResult, error) {
	var res TrashMoveResult
	if err := ctx.Err(); err != nil {
		return res, err
	}
	scan, err := s.ScanOrphanImages(ctx)
	if err != nil {
		return res, err
	}
	if len(scan.Orphans) == 0 {
		return res, nil
	}
	trashDir := fmt.Sprintf("%s%s", trashPrefix, time.Now().Format("20060102-150405"))
	for _, f := range scan.Orphans {
		src := filepath.Join(s.appDir, filepath.FromSlash(f.RelPath))
		dst := filepath.Join(s.appDir, trashDir, filepath.FromSlash(f.RelPath))
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return res, fmt.Errorf("创建回收站目录失败: %w", err)
		}
		if err := os.Rename(src, dst); err != nil {
			return res, fmt.Errorf("移动 %s 到回收站失败: %w", f.RelPath, err)
		}
		res.Moved = append(res.Moved, f)
	}
	res.TrashDir = trashDir
	return res, nil
}

// ListMediaTrash 列出站点根目录下所有回收站目录。
func (s *MediaRefService) ListMediaTrash(ctx context.Context) (TrashListResult, error) {
	var res TrashListResult
	if err := ctx.Err(); err != nil {
		return res, err
	}
	entries, err := os.ReadDir(s.appDir)
	if err != nil {
		return res, err
	}
	for _, e := range entries {
		if !e.IsDir() || !strings.HasPrefix(e.Name(), trashPrefix) {
			continue
		}
		info := TrashInfo{Dir: e.Name()}
		if fi, err := e.Info(); err == nil {
			info.CreatedAt = fi.ModTime().Format("2006-01-02 15:04:05")
		}
		dirPath := filepath.Join(s.appDir, e.Name())
		_ = filepath.WalkDir(dirPath, func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			info.FileCount++
			if fi, err := d.Info(); err == nil {
				info.SizeByte += fi.Size()
			}
			return nil
		})
		res.Dirs = append(res.Dirs, info)
		res.FileCount += info.FileCount
		res.SizeByte += info.SizeByte
	}
	sort.Slice(res.Dirs, func(i, j int) bool {
		return res.Dirs[i].Dir < res.Dirs[j].Dir
	})
	return res, nil
}

// EmptyMediaTrash 清空全部回收站目录，返回删除的目录数。
func (s *MediaRefService) EmptyMediaTrash(ctx context.Context) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	entries, err := os.ReadDir(s.appDir)
	if err != nil {
		return 0, err
	}
	removed := 0
	for _, e := range entries {
		if !e.IsDir() || !strings.HasPrefix(e.Name(), trashPrefix) {
			continue
		}
		if err := os.RemoveAll(filepath.Join(s.appDir, e.Name())); err != nil {
			return removed, fmt.Errorf("清空回收站目录 %s 失败: %w", e.Name(), err)
		}
		removed++
	}
	return removed, nil
}
