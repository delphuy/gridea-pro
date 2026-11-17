package facade

import (
	"context"
	"sync/atomic"

	"gridea-pro/backend/internal/service"
)

// MediaRefFacade 图片引用治理（孤儿扫描 / 回收站）。
// internal 用原子读写保存：切站（UpdateAppDir）会热替换它，
// 而 Wails 可能并发调用本 facade 的方法，原子读写避免"读到替换一半的状态"的 data race。
type MediaRefFacade struct {
	internal atomic.Pointer[service.MediaRefService]
}

func NewMediaRefFacade(s *service.MediaRefService) *MediaRefFacade {
	f := &MediaRefFacade{}
	f.internal.Store(s)
	return f
}

func (f *MediaRefFacade) svc() *service.MediaRefService { return f.internal.Load() }

func (f *MediaRefFacade) ctx() context.Context {
	if WailsContext == nil {
		return context.TODO()
	}
	return WailsContext
}

// ScanOrphanImages 干跑扫描未被引用的本地媒体文件（不移动任何文件）
func (f *MediaRefFacade) ScanOrphanImages() (service.OrphanScanResult, error) {
	return f.svc().ScanOrphanImages(f.ctx())
}

// MoveOrphanImagesToTrash 把当前扫描到的孤儿文件移入 .trash-<时间戳> 回收站
func (f *MediaRefFacade) MoveOrphanImagesToTrash() (service.TrashMoveResult, error) {
	return f.svc().MoveOrphanImagesToTrash(f.ctx())
}

// ListMediaTrash 列出回收站目录
func (f *MediaRefFacade) ListMediaTrash() (service.TrashListResult, error) {
	return f.svc().ListMediaTrash(f.ctx())
}

// EmptyMediaTrash 清空回收站，返回删除的目录数
func (f *MediaRefFacade) EmptyMediaTrash() (int, error) {
	return f.svc().EmptyMediaTrash(f.ctx())
}
