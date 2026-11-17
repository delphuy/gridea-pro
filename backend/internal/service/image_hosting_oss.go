package service

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gridea-pro/backend/internal/domain"
)

// ── 阿里云 OSS 图床（REST V1 签名直连，不依赖 SDK）────────────────────
// 设计要点：
//   - 虚拟主机访问风格：请求发到 https://{bucket}.{endpoint}/{key}，
//     签名资源名按 /{bucket}/{key} 计算。
//   - ImageHostingFile.Hash 携带对象 key 作为删除句柄（UI 网格 key 与 DeleteImage 均用 hash）。
//   - 上传带 x-oss-object-acl: public-read，与博客图床的公开访问需求一致。

func (s *ImageHostingService) ossRequire(setting *domain.ImageHostingSetting) (*domain.ImageHostingOSSSetting, error) {
	if setting.OSS == nil {
		return nil, fmt.Errorf("阿里云 OSS 图床配置为空")
	}
	cfg := setting.OSS
	if cfg.AccessKeyID == "" || cfg.AccessKeySecret == "" || cfg.Bucket == "" || cfg.Endpoint == "" {
		return nil, fmt.Errorf("阿里云 OSS 配置不完整（需要 AccessKeyId / AccessKeySecret / Bucket / Endpoint）")
	}
	return cfg, nil
}

// ossHost 返回请求主机名：自定义域名（可带或不带 https://）优先，默认 {bucket}.{endpoint}。
func ossHost(cfg *domain.ImageHostingOSSSetting) string {
	if d := strings.TrimSpace(cfg.CustomDomain); d != "" {
		d = strings.TrimPrefix(strings.TrimPrefix(d, "https://"), "http://")
		return strings.TrimRight(d, "/")
	}
	endpoint := strings.TrimSpace(cfg.Endpoint)
	endpoint = strings.TrimPrefix(strings.TrimPrefix(endpoint, "https://"), "http://")
	return cfg.Bucket + "." + strings.TrimRight(endpoint, "/")
}

// ossBaseURL 拼出对象/列表请求的基础地址（固定走 https）。
func ossBaseURL(cfg *domain.ImageHostingOSSSetting) string {
	return "https://" + ossHost(cfg)
}

// ossURL 计算某个对象 key 的公网访问地址。
func ossURL(cfg *domain.ImageHostingOSSSetting, key string) string {
	return ossBaseURL(cfg) + "/" + escapePathSegments(key)
}

// ossSign 依 OSS V1 规范生成 Authorization 头。
//
//	StringToSign = VERB + "\n" + Content-MD5 + "\n" + Content-Type + "\n" + Date + "\n"
//	               + CanonicalizedOSSHeaders + CanonicalizedResource
func ossSign(method string, cfg *domain.ImageHostingOSSSetting, contentType, date, ossHeaders, resource string) string {
	stringToSign := method + "\n\n" + contentType + "\n" + date + "\n" + ossHeaders + resource
	mac := hmac.New(sha1.New, []byte(cfg.AccessKeySecret))
	mac.Write([]byte(stringToSign))
	return "OSS " + cfg.AccessKeyID + ":" + base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

// ossDo 发起一次 OSS 请求；非 2xx 时解析 <Error><Code>/<Message> 报错。
func ossDo(req *http.Request) (*http.Response, error) {
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求 OSS 失败: %w", err)
	}
	if resp.StatusCode >= 300 {
		defer resp.Body.Close()
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		var e struct {
			Code    string `xml:"Code"`
			Message string `xml:"Message"`
		}
		if xml.Unmarshal(body, &e) == nil && e.Code != "" {
			return resp, fmt.Errorf("OSS 返回 %d: %s（%s）", resp.StatusCode, e.Message, e.Code)
		}
		return resp, fmt.Errorf("OSS 返回状态码 %d", resp.StatusCode)
	}
	return resp, nil
}

// detectImageContentType 优先按扩展名给图片类型（svg 等嗅探不出的格式），
// 否则用内容嗅探，兜底 application/octet-stream。
func detectImageContentType(data []byte, name string) string {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".svg":
		return "image/svg+xml"
	case ".bmp":
		return "image/bmp"
	case ".ico":
		return "image/x-icon"
	case ".avif":
		return "image/avif"
	}
	if ct := http.DetectContentType(data); ct != "application/octet-stream" && !strings.HasPrefix(ct, "text/") {
		return ct
	}
	return "application/octet-stream"
}

func (s *ImageHostingService) ossUpload(setting *domain.ImageHostingSetting, filePath string) (*domain.ImageHostingFile, error) {
	cfg, err := s.ossRequire(setting)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("读取文件失败: %w", err)
	}

	name := generateStoredName(filePath)
	key := joinPathPrefix(cfg.PathPrefix, name)
	contentType := detectImageContentType(data, name)
	date := time.Now().UTC().Format(http.TimeFormat)

	req, err := http.NewRequest("PUT", ossURL(cfg, key), bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Date", date)
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("x-oss-object-acl", "public-read")
	req.Header.Set("Authorization", ossSign("PUT", cfg, contentType, date,
		"x-oss-object-acl:public-read\n", "/"+cfg.Bucket+"/"+key))

	resp, err := ossDo(req)
	if err != nil {
		return nil, err
	}
	resp.Body.Close()

	return &domain.ImageHostingFile{
		Filename:  filepath.Base(key),
		Storename: name,
		URL:       ossURL(cfg, key),
		Hash:      key, // 删除句柄 = 对象 key
		Path:      key,
		Size:      len(data),
	}, nil
}

// ossListBucketResult / ossContent 是 GET bucket（列举对象）返回的 XML 结构。
type ossListBucketResult struct {
	XMLName  xml.Name     `xml:"ListBucketResult"`
	Contents []ossContent `xml:"Contents"`
}

type ossContent struct {
	Key          string `xml:"Key"`
	Size         int64  `xml:"Size"`
	LastModified string `xml:"LastModified"`
}

func (s *ImageHostingService) ossList(setting *domain.ImageHostingSetting, page int) (*domain.ImageHostingListResponse, error) {
	cfg, err := s.ossRequire(setting)
	if err != nil {
		return nil, err
	}
	// OSS 列举不做翻页：第 1 页返回最多 1000 个，之后返回空（前端据此收起翻页按钮）。
	result := &domain.ImageHostingListResponse{Code: 200, Success: true, Data: []domain.ImageHostingFile{}}
	if page > 1 {
		return result, nil
	}

	listURL := ossBaseURL(cfg) + "/?max-keys=1000"
	if prefix := strings.Trim(cfg.PathPrefix, "/"); prefix != "" {
		listURL += "&prefix=" + url.QueryEscape(prefix+"/")
	}
	date := time.Now().UTC().Format(http.TimeFormat)

	req, err := http.NewRequest("GET", listURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Date", date)
	req.Header.Set("Authorization", ossSign("GET", cfg, "", date, "", "/"+cfg.Bucket+"/"))

	resp, err := ossDo(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, err
	}
	var parsed ossListBucketResult
	if err := xml.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("解析 OSS 列表响应失败: %w", err)
	}

	for _, c := range parsed.Contents {
		name := filepath.Base(c.Key)
		lower := strings.ToLower(name)
		dot := strings.LastIndex(lower, ".")
		if dot < 0 || !imageHostingExts[lower[dot:]] {
			continue
		}
		var createdAt int64
		if t, err := time.Parse(time.RFC3339, c.LastModified); err == nil {
			createdAt = t.Unix()
		}
		result.Data = append(result.Data, domain.ImageHostingFile{
			Filename:  name,
			URL:       ossURL(cfg, c.Key),
			Hash:      c.Key,
			Path:      c.Key,
			Size:      int(c.Size),
			CreatedAt: int(createdAt),
		})
	}
	return result, nil
}

func (s *ImageHostingService) ossDelete(setting *domain.ImageHostingSetting, hash string) error {
	cfg, err := s.ossRequire(setting)
	if err != nil {
		return err
	}
	key := strings.TrimPrefix(hash, "/")
	if key == "" {
		return fmt.Errorf("删除句柄为空")
	}

	date := time.Now().UTC().Format(http.TimeFormat)
	req, err := http.NewRequest("DELETE", ossURL(cfg, key), nil)
	if err != nil {
		return err
	}
	req.Header.Set("Date", date)
	req.Header.Set("Authorization", ossSign("DELETE", cfg, "", date, "", "/"+cfg.Bucket+"/"+key))

	resp, err := ossDo(req)
	if err != nil {
		// NoSuchKey：远端已不存在，视为删除成功
		if resp != nil && strings.Contains(err.Error(), "NoSuchKey") {
			return nil
		}
		return err
	}
	resp.Body.Close()
	return nil
}
