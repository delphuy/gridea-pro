package service

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
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
	switch providerName(setting) {
	case domain.ImageHostingProviderGitee:
		return s.giteeList(setting, page)
	case domain.ImageHostingProviderOSS:
		return s.ossList(setting, page)
	default:
		return s.seeList(setting, page)
	}
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
