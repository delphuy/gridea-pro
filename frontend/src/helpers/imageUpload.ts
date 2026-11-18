/**
 * 本地图片统一上传入口（图床优先）
 *
 * 图床启用时：编辑器里所有本地图片（弹窗选文件、工具栏插入、粘贴、拖拽、图片组件上传）
 * 优先直传图床并返回外链；上传失败时提示并回退本地附件，写作不中断。
 * 图床未启用时：与原行为一致，存本地附件。
 */
import { toast } from '@/helpers/toast'
import {
  GetSetting,
  UploadImagesFromFrontend as uploadToImageHosting,
  UploadImageBytesFromFrontend as uploadBytesToImageHosting,
} from '@/wailsjs/go/facade/ImageHostingFacade'
import {
  UploadImagesFromFrontend as uploadToPost,
  SaveImageBytesFromFrontend as saveBytesToPost,
} from '@/wailsjs/go/facade/PostFacade'
import { domain } from '@/wailsjs/go/models'

/** 图床是否启用（配置读取失败按未启用处理，走本地附件兜底） */
export async function imageHostingEnabled(): Promise<boolean> {
  try {
    const setting = await GetSetting()
    return !!setting?.enabled
  } catch {
    return false
  }
}

/** 从文件路径提取文件名（兼容 Windows / Unix 分隔符） */
function baseName(path: string): string {
  const p = path.split('/').pop() || path
  return p.split('\\').pop() || 'image'
}

/** 图床上传失败时的统一提示（随后回退本地附件） */
function warnImageHostingFallback(err: unknown) {
  console.error('[imageUpload] 图床上传失败，回退本地附件:', err)
  toast.warning('图床上传失败，已改存本地附件')
}

/**
 * 上传一张本地图片（按路径）。
 * 返回可直接插入编辑器的链接（图床外链或本地附件路径）。
 */
export async function uploadLocalImage(filePath: string): Promise<string> {
  if (await imageHostingEnabled()) {
    try {
      const urls = await uploadToImageHosting([new domain.UploadedFile({ name: baseName(filePath), path: filePath })])
      if (urls?.length) return urls[0]
      throw new Error('图床未返回链接')
    } catch (err) {
      warnImageHostingFallback(err)
    }
  }
  const paths = await uploadToPost([new domain.UploadedFile({ name: baseName(filePath), path: filePath })])
  return paths?.[0] ?? ''
}

/**
 * 上传一张本地图片（按字节，粘贴 / 拖拽 / 图片组件上传）。
 * name 通常形如 "screenshot.png"；为空时兜底 "image.png"。
 */
export async function uploadLocalImageBytes(name: string, dataBase64: string): Promise<string> {
  const safeName = name || 'image.png'
  if (await imageHostingEnabled()) {
    try {
      return await uploadBytesToImageHosting(safeName, dataBase64)
    } catch (err) {
      warnImageHostingFallback(err)
    }
  }
  return saveBytesToPost(safeName, dataBase64)
}
