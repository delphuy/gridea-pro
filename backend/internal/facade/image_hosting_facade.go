package facade

import (
	"encoding/base64"
	"fmt"
	"sync/atomic"

	"gridea-pro/backend/internal/domain"
	"gridea-pro/backend/internal/service"
)

type ImageHostingFacade struct {
	// internal 用原子指针保存：切站（UpdateAppDir）会热替换底层 service，
	// 而 Wails 绑定的是本 facade 对象本身（boot 时绑定一次），只能就地更新指针——
	// 若像旧实现那样替换 AppServices.ImageHosting 字段，Wails 暴露的仍是旧对象、仍指旧站点目录。
	internal atomic.Pointer[service.ImageHostingService]
}

func NewImageHostingFacade(s *service.ImageHostingService) *ImageHostingFacade {
	f := &ImageHostingFacade{}
	f.internal.Store(s)
	return f
}

func (f *ImageHostingFacade) svc() *service.ImageHostingService { return f.internal.Load() }

func (f *ImageHostingFacade) GetSetting() (*domain.ImageHostingSetting, error) {
	return f.svc().GetSetting()
}

func (f *ImageHostingFacade) SaveSetting(setting *domain.ImageHostingSetting) error {
	return f.svc().SaveSetting(setting)
}

func (f *ImageHostingFacade) UploadImage(filePath string) (*domain.ImageHostingFile, error) {
	return f.svc().Upload(filePath)
}

func (f *ImageHostingFacade) ListImages(page int) (*domain.ImageHostingListResponse, error) {
	return f.svc().List(page)
}

func (f *ImageHostingFacade) DeleteImage(hash string) error {
	return f.svc().Delete(hash)
}

func (f *ImageHostingFacade) UploadImagesFromFrontend(files []domain.UploadedFile) ([]string, error) {
	return f.svc().UploadFromFrontend(files)
}

// UploadImageBytesFromFrontend 把粘贴/拖拽的图片字节（base64）直传图床，返回外链 URL。
func (f *ImageHostingFacade) UploadImageBytesFromFrontend(name string, dataBase64 string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(dataBase64)
	if err != nil {
		return "", fmt.Errorf("图片数据解码失败: %w", err)
	}
	return f.svc().UploadImageBytes(name, data)
}

// TestUpload 上传并删除一张 1x1 测试图，验证当前图床配置是否可用，返回测试图 URL。
func (f *ImageHostingFacade) TestUpload() (string, error) {
	return f.svc().TestUpload()
}
