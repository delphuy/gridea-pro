package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"gridea-pro/backend/internal/domain"
)

// ── S.EE 图床（https://s.ee）──────────────────────────────────────────

func (s *ImageHostingService) seeUpload(setting *domain.ImageHostingSetting, filePath string) (*domain.ImageHostingFile, error) {
	if setting.APIKey == "" {
		return nil, fmt.Errorf("API 密钥未配置")
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("打开文件失败: %w", err)
	}
	defer file.Close()

	part, err := writer.CreateFormFile("file", filepath.Base(filePath))
	if err != nil {
		return nil, err
	}
	if _, err := io.Copy(part, file); err != nil {
		return nil, err
	}
	writer.Close()

	req, err := http.NewRequest("POST", seeAPIBase+"/file/upload", body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", setting.APIKey)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("上传请求失败: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result domain.ImageHostingUploadResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, err
	}

	if !result.Success && result.Code != 0 && result.Code != 200 {
		return nil, fmt.Errorf("上传失败: %s", result.Message)
	}

	return &result.Data, nil
}

func (s *ImageHostingService) seeList(setting *domain.ImageHostingSetting, page int) (*domain.ImageHostingListResponse, error) {
	if setting.APIKey == "" {
		return nil, fmt.Errorf("API 密钥未配置")
	}

	url := seeAPIBase + "/files"
	if page > 0 {
		url += "?page=" + strconv.Itoa(page)
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", setting.APIKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("获取文件列表失败: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result domain.ImageHostingListResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, err
	}

	if !result.Success && result.Code != 0 && result.Code != 200 {
		return nil, fmt.Errorf("获取失败: %s", result.Message)
	}

	return &result, nil
}

func (s *ImageHostingService) seeDelete(setting *domain.ImageHostingSetting, hash string) error {
	if setting.APIKey == "" {
		return fmt.Errorf("API 密钥未配置")
	}

	req, err := http.NewRequest("GET", seeAPIBase+"/file/delete/"+hash, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", setting.APIKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("删除请求失败: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var result domain.ImageHostingDeleteResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return err
	}

	if !result.Success {
		return fmt.Errorf("删除失败: %s", result.Message)
	}

	return nil
}
