package service

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"gridea-pro/backend/internal/domain"
)

// ── Gitee 图床（仓库 Contents API v5）─────────────────────────────────
// 设计要点：
//   - 上传即向图床仓库提交一个 base64 文件；单文件硬上限 1MB（Gitee Contents API 限制）。
//   - ImageHostingFile.Hash 携带远端路径作为删除句柄（UI 网格 key 与 DeleteImage 均用 hash）。
//   - 删除需要先 GET 拿 blob sha，再 DELETE；远端已不存在视为删除成功。

const (
	giteeAPIBase     = "https://gitee.com/api/v5"
	giteeMaxFileByte = 1 << 20 // 1MB
)

// imageHostingExts 是图床文件列表里允许展示的扩展名（过滤 README 等杂项文件，gitee/oss 共用）。
var imageHostingExts = map[string]bool{
	".png": true, ".jpg": true, ".jpeg": true, ".gif": true,
	".webp": true, ".svg": true, ".bmp": true, ".ico": true, ".avif": true,
}

type giteeContentEntry struct {
	Name string `json:"name"`
	Path string `json:"path"`
	SHA  string `json:"sha"`
	Size int64  `json:"size"`
	Type string `json:"type"` // "file" | "dir"
}

func (s *ImageHostingService) giteeRequire(setting *domain.ImageHostingSetting) (*domain.ImageHostingGiteeSetting, error) {
	if setting.Gitee == nil {
		return nil, fmt.Errorf("Gitee 图床配置为空")
	}
	cfg := setting.Gitee
	if cfg.AccessToken == "" || cfg.Owner == "" || cfg.Repo == "" {
		return nil, fmt.Errorf("Gitee 图床配置不完整（需要私人令牌 / 用户名 / 仓库名）")
	}
	return cfg, nil
}

func giteeBranch(cfg *domain.ImageHostingGiteeSetting) string {
	if strings.TrimSpace(cfg.Branch) == "" {
		return "main"
	}
	return cfg.Branch
}

// giteePublicURL 计算图片的公网访问地址：自定义基地址优先，默认走 Gitee raw。
func giteePublicURL(cfg *domain.ImageHostingGiteeSetting, remotePath string) string {
	if base := strings.TrimRight(strings.TrimSpace(cfg.PublicBaseURL), "/"); base != "" {
		return base + "/" + escapePathSegments(remotePath)
	}
	return fmt.Sprintf("https://gitee.com/%s/%s/raw/%s/%s",
		url.PathEscape(cfg.Owner), url.PathEscape(cfg.Repo), url.PathEscape(giteeBranch(cfg)), escapePathSegments(remotePath))
}

// giteeRequest 发一次 JSON 请求，返回状态码与响应体；非 2xx 时解析 {"message": ...} 报错。
func giteeRequest(method, url string, body any, out any) (int, error) {
	var reader io.Reader
	if body != nil {
		buf, err := json.Marshal(body)
		if err != nil {
			return 0, err
		}
		reader = bytes.NewReader(buf)
	}
	req, err := http.NewRequest(method, url, reader)
	if err != nil {
		return 0, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("请求 Gitee 失败: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, err
	}
	if resp.StatusCode >= 400 {
		var e struct {
			Message string `json:"message"`
		}
		if json.Unmarshal(respBody, &e) == nil && e.Message != "" {
			return resp.StatusCode, fmt.Errorf("Gitee 返回 %d: %s", resp.StatusCode, e.Message)
		}
		return resp.StatusCode, fmt.Errorf("Gitee 返回状态码 %d", resp.StatusCode)
	}
	if out != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, out); err != nil {
			return resp.StatusCode, fmt.Errorf("解析 Gitee 响应失败: %w", err)
		}
	}
	return resp.StatusCode, nil
}

func (s *ImageHostingService) giteeUpload(setting *domain.ImageHostingSetting, filePath string) (*domain.ImageHostingFile, error) {
	cfg, err := s.giteeRequire(setting)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("读取文件失败: %w", err)
	}
	if len(data) > giteeMaxFileByte {
		return nil, fmt.Errorf("Gitee 图床单文件不能超过 1MB（当前约 %.1fMB），请压缩后上传", float64(len(data))/(1<<20))
	}

	// 第一次尝试失败（如同名冲突）时，换个名字重试一次。
	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		name := generateStoredName(filePath)
		remotePath := joinPathPrefix(cfg.PathPrefix, name)

		var resp struct {
			Content giteeContentEntry `json:"content"`
		}
		payload := map[string]any{
			"access_token": cfg.AccessToken,
			"content":      base64.StdEncoding.EncodeToString(data),
			"message":      "Upload " + name + " via Gridea Pro",
			"branch":       giteeBranch(cfg),
		}
		apiURL := giteeAPIBase + "/repos/" + url.PathEscape(cfg.Owner) + "/" + url.PathEscape(cfg.Repo) + "/contents/" + escapePathSegments(remotePath)
		status, err := giteeRequest("POST", apiURL, payload, &resp)
		if err == nil {
			entry := resp.Content
			path := entry.Path
			if path == "" {
				path = remotePath
			}
			return &domain.ImageHostingFile{
				Filename:  entry.Name,
				Storename: name,
				URL:       giteePublicURL(cfg, path),
				Hash:      path, // 删除句柄 = 远端路径
				Path:      path,
				Size:      len(data),
			}, nil
		}
		lastErr = err
		// 仅对疑似「文件已存在」类错误重试改名；鉴权/网络类错误直接报出。
		if status != http.StatusBadRequest && status != http.StatusConflict &&
			status != http.StatusUnprocessableEntity && status != http.StatusNotFound {
			break
		}
	}
	return nil, lastErr
}

func (s *ImageHostingService) giteeList(setting *domain.ImageHostingSetting, page int) (*domain.ImageHostingListResponse, error) {
	cfg, err := s.giteeRequire(setting)
	if err != nil {
		return nil, err
	}
	// Gitee 目录列表无分页语义：第 1 页返回全部，之后返回空（前端据此收起翻页按钮）。
	result := &domain.ImageHostingListResponse{Code: 200, Success: true, Data: []domain.ImageHostingFile{}}
	if page > 1 {
		return result, nil
	}

	dir := strings.Trim(cfg.PathPrefix, "/")
	apiURL := giteeAPIBase + "/repos/" + url.PathEscape(cfg.Owner) + "/" + url.PathEscape(cfg.Repo) + "/contents"
	if dir != "" {
		apiURL += "/" + escapePathSegments(dir)
	}
	apiURL += "?access_token=" + url.QueryEscape(cfg.AccessToken) + "&ref=" + url.QueryEscape(giteeBranch(cfg))

	var entries []giteeContentEntry
	status, err := giteeRequest("GET", apiURL, nil, &entries)
	if err != nil {
		// 目录不存在（还没传过图）→ 返回空列表
		if status == http.StatusNotFound {
			return result, nil
		}
		return nil, err
	}

	for _, e := range entries {
		if e.Type != "file" {
			continue
		}
		lower := strings.ToLower(e.Name)
		dot := strings.LastIndex(lower, ".")
		if dot < 0 || !imageHostingExts[lower[dot:]] {
			continue
		}
		result.Data = append(result.Data, domain.ImageHostingFile{
			Filename:  e.Name,
			URL:       giteePublicURL(cfg, e.Path),
			Hash:      e.Path,
			Path:      e.Path,
			Size:      int(e.Size),
		})
	}
	return result, nil
}

func (s *ImageHostingService) giteeDelete(setting *domain.ImageHostingSetting, hash string) error {
	cfg, err := s.giteeRequire(setting)
	if err != nil {
		return err
	}
	remotePath := strings.TrimPrefix(hash, "/")

	// 取 blob sha（删除接口必需）
	metaURL := giteeAPIBase + "/repos/" + url.PathEscape(cfg.Owner) + "/" + url.PathEscape(cfg.Repo) +
		"/contents/" + escapePathSegments(remotePath) +
		"?access_token=" + url.QueryEscape(cfg.AccessToken) + "&ref=" + url.QueryEscape(giteeBranch(cfg))
	var entry giteeContentEntry
	status, err := giteeRequest("GET", metaURL, nil, &entry)
	if err != nil {
		if status == http.StatusNotFound {
			return nil // 远端已不存在，视为删除成功
		}
		return err
	}

	delURL := giteeAPIBase + "/repos/" + url.PathEscape(cfg.Owner) + "/" + url.PathEscape(cfg.Repo) +
		"/contents/" + escapePathSegments(remotePath)
	payload := map[string]any{
		"access_token": cfg.AccessToken,
		"sha":          entry.SHA,
		"message":      "Delete " + remotePath + " via Gridea Pro",
		"branch":       giteeBranch(cfg),
	}
	status, err = giteeRequest("DELETE", delURL, payload, nil)
	if err != nil {
		if status == http.StatusNotFound {
			return nil
		}
		return err
	}
	return nil
}
