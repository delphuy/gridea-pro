package service

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"gridea-pro/backend/internal/domain"
)

const seeAPIBase = "https://s.ee/api/v1"

type ImageHostingService struct {
	repo domain.ImageHostingRepository
}

func NewImageHostingService(repo domain.ImageHostingRepository) *ImageHostingService {
	return &ImageHostingService{repo: repo}
}

func (s *ImageHostingService) GetSetting() (*domain.ImageHostingSetting, error) {
	return s.repo.GetSetting()
}

func (s *ImageHostingService) SaveSetting(setting *domain.ImageHostingSetting) error {
	return s.repo.SaveSetting(setting)
}

// providerName 归一化服务商标识：空值按 "see" 处理（历史配置没有该字段）。
func providerName(setting *domain.ImageHostingSetting) string {
	if setting.Provider == "" {
		return domain.ImageHostingProviderSEE
	}
	return setting.Provider
}

func (s *ImageHostingService) Upload(filePath string) (*domain.ImageHostingFile, error) {
	setting, err := s.repo.GetSetting()
	if err != nil {
		return nil, err
	}
	switch providerName(setting) {
	case domain.ImageHostingProviderGitee:
		return s.giteeUpload(setting, filePath)
	case domain.ImageHostingProviderOSS:
		return s.ossUpload(setting, filePath)
	default:
		return s.seeUpload(setting, filePath)
	}
}

func (s *ImageHostingService) List(page int) (*domain.ImageHostingListResponse, error) {
	setting, err := s.repo.GetSetting()
	if err != nil {
		return nil, err
	}
	var (
		resp    *domain.ImageHostingListResponse
		listErr error
	)
	switch providerName(setting) {
	case domain.ImageHostingProviderGitee:
		resp, listErr = s.giteeList(setting, page)
	case domain.ImageHostingProviderOSS:
		resp, listErr = s.ossList(setting, page)
	default:
		resp, listErr = s.seeList(setting, page)
	}
	if listErr != nil {
		return nil, listErr
	}
	// 统一按上传时间从新到旧排序：刚传的图排在第一屏，一眼可见。
	sortImageHostingFilesNewestFirst(resp.Data)
	return resp, nil
}

// sortImageHostingFilesNewestFirst 就地按上传时间倒序排列（稳定排序）。
// 个别服务商（gitee）列表拿不到上传时间：先从对象名前缀的毫秒时间戳回填；
// 仍拿不到（CreatedAt=0）的条目沉底并保持原相对顺序。
func sortImageHostingFilesNewestFirst(files []domain.ImageHostingFile) {
	for i := range files {
		if files[i].CreatedAt == 0 {
			files[i].CreatedAt = int(createdMilliFromName(files[i].Path) / 1000)
		}
	}
	sort.SliceStable(files, func(a, b int) bool {
		return files[a].CreatedAt > files[b].CreatedAt
	})
}

// createdMilliFromName 从「<毫秒时间戳>_<文件名>」对象名（generateStoredName 的命名约定）
// 解析上传时间；解析失败或明显不合理（早于 2000-01-01 / 不早于 2100-01-01）返回 0。
func createdMilliFromName(key string) int64 {
	base := filepath.Base(key)
	i := strings.IndexByte(base, '_')
	if i <= 0 {
		return 0
	}
	ms, err := strconv.ParseInt(base[:i], 10, 64)
	if err != nil {
		return 0
	}
	if sec := ms / 1000; sec < 946684800 || sec >= 4102444800 {
		return 0
	}
	return ms
}

func (s *ImageHostingService) Delete(hash string) error {
	setting, err := s.repo.GetSetting()
	if err != nil {
		return err
	}
	switch providerName(setting) {
	case domain.ImageHostingProviderGitee:
		return s.giteeDelete(setting, hash)
	case domain.ImageHostingProviderOSS:
		return s.ossDelete(setting, hash)
	default:
		return s.seeDelete(setting, hash)
	}
}

func (s *ImageHostingService) UploadFromFrontend(files []domain.UploadedFile) ([]string, error) {
	var urls []string
	for _, f := range files {
		result, err := s.Upload(f.Path)
		if err != nil {
			return nil, fmt.Errorf("上传 %s 失败: %w", f.Name, err)
		}
		urls = append(urls, result.URL)
	}
	return urls, nil
}

// UploadImageBytes 把图片字节直传图床（粘贴/拖拽等前端拿不到磁盘路径的场景）：
// 写入临时目录（保留原始文件名，供生成「<时间戳>_<文件名>」对象名），
// 走统一 Upload 通道上传，随后清理临时文件，返回外链 URL。
func (s *ImageHostingService) UploadImageBytes(name string, data []byte) (string, error) {
	base := filepath.Base(strings.TrimSpace(strings.ReplaceAll(name, `\`, "/")))
	if base == "" || base == "." || base == "/" || base == `\` {
		base = "image.png"
	}
	tmpDir, err := os.MkdirTemp("", "gridea-img-*")
	if err != nil {
		return "", fmt.Errorf("创建临时目录失败: %w", err)
	}
	defer os.RemoveAll(tmpDir)
	tmpPath := filepath.Join(tmpDir, base)
	if err := os.WriteFile(tmpPath, data, 0o600); err != nil {
		return "", fmt.Errorf("写入临时图片文件失败: %w", err)
	}
	file, err := s.Upload(tmpPath)
	if err != nil {
		return "", err
	}
	return file.URL, nil
}

// testPNG1x1 是一张 1x1 透明 PNG，用于连通性测试。
const testPNG1x1 = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNkYPhfDwAChwGA60e6kgAAAABJRU5ErkJggg=="

// generateStoredName 生成远端对象名：<毫秒时间戳>_<净化后的文件名>。
// 主体与扩展名分开净化：只保留字母/数字/点/下划线/连字符，其余替换为 _，
// 避免特殊字符在 URL 与签名里出问题，同时不丢扩展名。
func generateStoredName(base string) string {
	name := filepath.Base(base)
	stem, ext := name, ""
	if dot := strings.LastIndex(name, "."); dot > 0 {
		stem, ext = name[:dot], name[dot:]
	} else if dot == 0 { // ".png" 这种极端名
		ext, stem = name, ""
	}
	stem = strings.Trim(sanitizeASCIIName(stem), "._-")
	if stem == "" {
		stem = "image"
	}
	if e := strings.Trim(sanitizeASCIIName(ext), "."); e != "" {
		ext = "." + e
	} else {
		ext = ""
	}
	return fmt.Sprintf("%d_%s%s", time.Now().UnixMilli(), stem, ext)
}

// sanitizeASCIIName 把非「字母/数字/./_/-」的字节替换为下划线。
func sanitizeASCIIName(s string) string {
	var b []byte
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9',
			c == '.', c == '_', c == '-':
			b = append(b, c)
		default:
			b = append(b, '_')
		}
	}
	return string(b)
}

// joinPathPrefix 拼接目录前缀与文件名：前缀去首尾斜杠后拼 "prefix/name"，空前缀直接返回 name。
func joinPathPrefix(prefix, name string) string {
	p := strings.Trim(prefix, "/")
	if p == "" {
		return name
	}
	return p + "/" + name
}

// TestUpload 上传一张 1x1 测试图并立即删除，验证图床配置可用。
// 返回上传成功后的 URL（即使删除失败也返回，便于人工排查）。
func (s *ImageHostingService) TestUpload() (string, error) {
	data, err := base64.StdEncoding.DecodeString(testPNG1x1)
	if err != nil {
		return "", fmt.Errorf("构造测试图片失败: %w", err)
	}
	tmp, err := os.CreateTemp("", "gridea-img-test-*.png")
	if err != nil {
		return "", fmt.Errorf("创建临时文件失败: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return "", fmt.Errorf("写入临时文件失败: %w", err)
	}
	tmp.Close()

	file, err := s.Upload(tmpPath)
	if err != nil {
		return "", err
	}
	if err := s.Delete(file.Hash); err != nil {
		return file.URL, fmt.Errorf("上传成功但清理测试文件失败: %w（可稍后在图床里手动删除 %s）", err, file.Path)
	}
	return file.URL, nil
}
