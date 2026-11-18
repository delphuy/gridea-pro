<template>
  <div class="pb-20 max-w-4xl mx-auto pt-4">
    <div class="space-y-6">
      <div class="grid grid-cols-[180px_1fr] items-center gap-4">
        <label class="text-sm font-medium text-right text-muted-foreground">{{ t('settings.imageHosting.enable') }}</label>
        <div class="flex items-center gap-3">
          <Switch :checked="form.enabled" @update:checked="(v: boolean) => form.enabled = v" size="sm" />
          <span class="text-xs text-muted-foreground">{{ t('settings.imageHosting.enableDesc') }}</span>
        </div>
      </div>

      <template v-if="form.enabled">
        <!-- 图床服务选择 -->
        <div class="grid grid-cols-[180px_1fr] items-center gap-4">
          <label class="text-sm font-medium text-right text-muted-foreground">{{ t('settings.imageHosting.provider') }}</label>
          <div class="flex items-center gap-2">
            <Button
              v-for="p in providers"
              :key="p.value"
              variant="outline"
              class="h-7 text-xs rounded-full px-4 cursor-pointer"
              :class="form.provider === p.value ? 'border-primary/40 bg-primary/5 text-primary' : 'text-muted-foreground'"
              @click="form.provider = p.value"
            >
              {{ p.label }}
            </Button>
          </div>
        </div>

        <!-- S.EE 表单 -->
        <template v-if="form.provider === 'see'">
          <div class="grid grid-cols-[180px_1fr] items-start gap-4">
            <label class="text-sm font-medium text-right text-muted-foreground pt-2">{{ t('settings.imageHosting.apiKey') }}</label>
            <div class="max-w-sm">
              <div class="relative">
                <Input
                  v-model="form.apiKey"
                  :type="showSeeKey ? 'text' : 'password'"
                  :placeholder="t('settings.imageHosting.apiKeyPlaceholder')"
                />
                <button
                  type="button"
                  class="absolute right-2 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground cursor-pointer"
                  @click="showSeeKey = !showSeeKey"
                >
                  <svg v-if="showSeeKey" xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M2 12s3-7 10-7 10 7 10 7-3 7-10 7-10-7-10-7Z"/><circle cx="12" cy="12" r="3"/></svg>
                  <svg v-else xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M9.88 9.88a3 3 0 1 0 4.24 4.24"/><path d="M10.73 5.08A10.43 10.43 0 0 1 12 5c7 0 10 7 10 7a13.16 13.16 0 0 1-1.67 2.68"/><path d="M6.61 6.61A13.526 13.526 0 0 0 2 12s3 7 10 7a9.74 9.74 0 0 0 5.39-1.61"/><line x1="2" x2="22" y1="2" y2="22"/></svg>
                </button>
              </div>
              <div class="mt-1.5 text-xs text-primary flex items-center gap-1 cursor-pointer w-fit"
                @click="BrowserOpenURL('https://s.ee')">
                {{ t('settings.imageHosting.apiKeyDesc') }}
                <ArrowTopRightOnSquareIcon class="size-3" />
              </div>
            </div>
          </div>
        </template>

        <!-- Gitee 表单 -->
        <template v-else-if="form.provider === 'gitee'">
          <div class="grid grid-cols-[180px_1fr] items-start gap-4">
            <label class="text-sm font-medium text-right text-muted-foreground pt-2">{{ t('settings.imageHosting.giteeToken') }}</label>
            <div class="max-w-sm">
              <div class="relative">
                <Input
                  v-model="form.gitee.accessToken"
                  :type="showGiteeToken ? 'text' : 'password'"
                  :placeholder="t('settings.imageHosting.giteeTokenPlaceholder')"
                />
                <button
                  type="button"
                  class="absolute right-2 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground cursor-pointer"
                  @click="showGiteeToken = !showGiteeToken"
                >
                  <svg v-if="showGiteeToken" xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M2 12s3-7 10-7 10 7 10 7-3 7-10 7-10-7-10-7Z"/><circle cx="12" cy="12" r="3"/></svg>
                  <svg v-else xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M9.88 9.88a3 3 0 1 0 4.24 4.24"/><path d="M10.73 5.08A10.43 10.43 0 0 1 12 5c7 0 10 7 10 7a13.16 13.16 0 0 1-1.67 2.68"/><path d="M6.61 6.61A13.526 13.526 0 0 0 2 12s3 7 10 7a9.74 9.74 0 0 0 5.39-1.61"/><line x1="2" x2="22" y1="2" y2="22"/></svg>
                </button>
              </div>
              <div class="mt-1.5 text-xs text-primary flex items-center gap-1 cursor-pointer w-fit"
                @click="BrowserOpenURL('https://gitee.com/profile/personal_access_tokens')">
                {{ t('settings.imageHosting.giteeTokenDesc') }}
                <ArrowTopRightOnSquareIcon class="size-3" />
              </div>
            </div>
          </div>

          <div class="grid grid-cols-2 gap-4 max-w-xl">
            <div class="grid grid-cols-[120px_1fr] items-center gap-3">
              <label class="text-sm font-medium text-right text-muted-foreground">{{ t('settings.imageHosting.giteeOwner') }}</label>
              <Input v-model="form.gitee.owner" :placeholder="t('settings.imageHosting.giteeOwnerPlaceholder')" />
            </div>
            <div class="grid grid-cols-[120px_1fr] items-center gap-3">
              <label class="text-sm font-medium text-right text-muted-foreground">{{ t('settings.imageHosting.giteeRepo') }}</label>
              <Input v-model="form.gitee.repo" :placeholder="t('settings.imageHosting.giteeRepoPlaceholder')" />
            </div>
            <div class="grid grid-cols-[120px_1fr] items-center gap-3">
              <label class="text-sm font-medium text-right text-muted-foreground">{{ t('settings.imageHosting.giteeBranch') }}</label>
              <Input v-model="form.gitee.branch" :placeholder="t('settings.imageHosting.giteeBranchPlaceholder')" />
            </div>
            <div class="grid grid-cols-[120px_1fr] items-center gap-3">
              <label class="text-sm font-medium text-right text-muted-foreground">{{ t('settings.imageHosting.giteePrefix') }}</label>
              <Input v-model="form.gitee.pathPrefix" :placeholder="t('settings.imageHosting.giteePrefixPlaceholder')" />
            </div>
            <div class="col-span-2 grid grid-cols-[120px_1fr] items-center gap-3">
              <label class="text-sm font-medium text-right text-muted-foreground">{{ t('settings.imageHosting.giteePublicBase') }}</label>
              <Input v-model="form.gitee.publicBaseUrl" :placeholder="t('settings.imageHosting.giteePublicBasePlaceholder')" />
            </div>
          </div>

          <div class="max-w-xl text-xs text-amber-600 dark:text-amber-400 bg-amber-500/10 border border-amber-500/20 rounded-lg px-3 py-2">
            ⚠ {{ t('settings.imageHosting.giteeRisk') }}
          </div>
        </template>

        <!-- 阿里云 OSS 表单 -->
        <template v-else>
          <div class="grid grid-cols-2 gap-4 max-w-xl">
            <div class="grid grid-cols-[140px_1fr] items-center gap-3">
              <label class="text-sm font-medium text-right text-muted-foreground">{{ t('settings.imageHosting.ossAk') }}</label>
              <Input v-model="form.oss.accessKeyId" :placeholder="t('settings.imageHosting.ossAkPlaceholder')" />
            </div>
            <div class="grid grid-cols-[140px_1fr] items-center gap-3">
              <label class="text-sm font-medium text-right text-muted-foreground">{{ t('settings.imageHosting.ossSk') }}</label>
              <div class="relative">
                <Input
                  v-model="form.oss.accessKeySecret"
                  :type="showOssSk ? 'text' : 'password'"
                  :placeholder="t('settings.imageHosting.ossSkPlaceholder')"
                />
                <button
                  type="button"
                  class="absolute right-2 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground cursor-pointer"
                  @click="showOssSk = !showOssSk"
                >
                  <svg v-if="showOssSk" xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M2 12s3-7 10-7 10 7 10 7-3 7-10 7-10-7-10-7Z"/><circle cx="12" cy="12" r="3"/></svg>
                  <svg v-else xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M9.88 9.88a3 3 0 1 0 4.24 4.24"/><path d="M10.73 5.08A10.43 10.43 0 0 1 12 5c7 0 10 7 10 7a13.16 13.16 0 0 1-1.67 2.68"/><path d="M6.61 6.61A13.526 13.526 0 0 0 2 12s3 7 10 7a9.74 9.74 0 0 0 5.39-1.61"/><line x1="2" x2="22" y1="2" y2="22"/></svg>
                </button>
              </div>
            </div>
            <div class="grid grid-cols-[140px_1fr] items-center gap-3">
              <label class="text-sm font-medium text-right text-muted-foreground">{{ t('settings.imageHosting.ossBucket') }}</label>
              <Input v-model="form.oss.bucket" :placeholder="t('settings.imageHosting.ossBucketPlaceholder')" />
            </div>
            <div class="grid grid-cols-[140px_1fr] items-center gap-3">
              <label class="text-sm font-medium text-right text-muted-foreground">{{ t('settings.imageHosting.ossEndpoint') }}</label>
              <Input v-model="form.oss.endpoint" :placeholder="t('settings.imageHosting.ossEndpointPlaceholder')" />
            </div>
            <div class="grid grid-cols-[140px_1fr] items-center gap-3">
              <label class="text-sm font-medium text-right text-muted-foreground">{{ t('settings.imageHosting.ossPrefix') }}</label>
              <Input v-model="form.oss.pathPrefix" :placeholder="t('settings.imageHosting.ossPrefixPlaceholder')" />
            </div>
            <div class="grid grid-cols-[140px_1fr] items-center gap-3">
              <label class="text-sm font-medium text-right text-muted-foreground">{{ t('settings.imageHosting.ossDomain') }}</label>
              <Input v-model="form.oss.customDomain" :placeholder="t('settings.imageHosting.ossDomainPlaceholder')" />
            </div>
          </div>
        </template>

        <!-- 测试上传 -->
        <div class="flex items-center gap-3">
          <Button
            variant="outline"
            class="h-7 text-xs rounded-full px-4 border-primary/20 text-primary/80 hover:bg-primary/5 hover:text-primary cursor-pointer"
            :disabled="testing"
            @click="testUpload"
          >
            {{ testing ? t('settings.imageHosting.testing') : t('settings.imageHosting.testBtn') }}
          </Button>
          <span v-if="testing" class="animate-spin size-3.5 border border-primary border-t-transparent rounded-full" />
        </div>

        <div class="border-t border-border pt-6 mt-6">
          <div class="flex items-center justify-between mb-4">
            <h3 class="text-sm font-medium">{{ t('settings.imageHosting.imageBrowser') }}</h3>
            <div class="flex items-center gap-2">
              <Button
                variant="default"
                class="h-7 text-xs rounded-full px-3.5 bg-primary text-background hover:bg-primary/90 cursor-pointer"
                :disabled="uploading || listLoading"
                @click="uploadImages">
                {{ uploading ? t('settings.imageHosting.uploading') : t('settings.imageHosting.uploadBtn') }}
              </Button>
              <Button
                variant="outline"
                class="h-7 text-xs rounded-full px-3.5 border border-primary/20 text-primary/80 hover:bg-primary/5 hover:text-primary cursor-pointer"
                :disabled="listLoading"
                @click="loadImages(1)">
                {{ t('settings.imageHosting.refresh') }}
              </Button>
            </div>
          </div>

          <div v-if="listLoading" class="flex items-center justify-center py-12 text-muted-foreground text-sm">
            {{ t('settings.imageHosting.loading') }}
          </div>

          <template v-else-if="files.length > 0">
            <div class="grid grid-cols-3 sm:grid-cols-4 md:grid-cols-5 gap-3">
              <div
                v-for="file in files"
                :key="file.hash"
                class="group relative aspect-square rounded-lg overflow-hidden border border-border bg-muted cursor-pointer"
                @click="copyUrl(file.url)"
              >
                <img :src="file.url" :alt="file.filename" referrerpolicy="no-referrer" class="w-full h-full object-cover" />
                <div class="absolute inset-0 bg-black/50 opacity-0 group-hover:opacity-100 transition-opacity flex items-center justify-center gap-2">
                  <button
                    class="text-white text-xs bg-destructive/80 hover:bg-destructive rounded px-2 py-1 cursor-pointer"
                    @click.stop="deleteImage(file.hash)"
                  >
                    {{ t('common.delete') }}
                  </button>
                </div>
                <div class="absolute bottom-0 left-0 right-0 bg-gradient-to-t from-black/60 to-transparent px-1.5 pb-1 pt-4">
                  <p class="text-[10px] text-white truncate">{{ file.filename }}</p>
                </div>
              </div>
            </div>

            <div class="flex items-center justify-center gap-2 mt-4">
              <Button
                variant="outline"
                class="h-7 w-7 text-xs rounded-full cursor-pointer"
                :disabled="currentPage <= 1"
                @click="loadImages(currentPage - 1)">
                &lt;
              </Button>
              <span class="text-xs text-muted-foreground">{{ currentPage }}</span>
              <Button
                variant="outline"
                class="h-7 w-7 text-xs rounded-full cursor-pointer"
                :disabled="!hasNextPage"
                @click="loadImages(currentPage + 1)">
                &gt;
              </Button>
            </div>
          </template>

          <div v-else class="flex items-center justify-center py-12 text-muted-foreground text-sm">
            {{ t('settings.imageHosting.noImages') }}
          </div>
        </div>
      </template>
    </div>

    <footer-box>
      <div class="flex justify-end items-center gap-3 w-full">
        <Button
          variant="default"
          class="w-18 h-8 text-xs justify-center rounded-full bg-primary text-background hover:bg-primary/90 cursor-pointer"
          @click="submit">
          {{ t('common.save') }}
        </Button>
      </div>
    </footer-box>
  </div>
</template>

<script lang="ts" setup>
import { reactive, ref, watch, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { toast } from '@/helpers/toast'
import FooterBox from '@/components/FooterBox/index.vue'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import { Switch } from '@/components/ui/switch'
import { GetSetting, SaveSetting, ListImages, DeleteImage, TestUpload, UploadImagesFromFrontend } from '@/wailsjs/go/facade/ImageHostingFacade'
import { OpenImagesDialog } from '@/wailsjs/go/app/App'
import { domain } from '@/wailsjs/go/models'
import { BrowserOpenURL } from '@/wailsjs/runtime'
import { ArrowTopRightOnSquareIcon } from '@heroicons/vue/24/outline'

const { t } = useI18n()

// Wails 抛出的 Go 错误是纯字符串（e.message 取不到），统一解包，避免一律只显示「保存失败」。
const errMsg = (e: any, fallback: string): string => (typeof e === 'string' ? e : e?.message) || fallback

const showSeeKey = ref(false)
const showGiteeToken = ref(false)
const showOssSk = ref(false)
const listLoading = ref(false)
const testing = ref(false)
const uploading = ref(false)

const providers = [
  { value: 'see', label: t('settings.imageHosting.providerSee') },
  { value: 'gitee', label: t('settings.imageHosting.providerGitee') },
  { value: 'oss', label: t('settings.imageHosting.providerOss') },
]

const form = reactive({
  enabled: false,
  provider: 'see',
  apiKey: '',
  gitee: {
    accessToken: '',
    owner: '',
    repo: '',
    branch: '',
    pathPrefix: '',
    publicBaseUrl: '',
  },
  oss: {
    accessKeyId: '',
    accessKeySecret: '',
    bucket: '',
    endpoint: '',
    pathPrefix: '',
    customDomain: '',
  },
})

const files = ref<domain.ImageHostingFile[]>([])
const currentPage = ref(1)
const hasNextPage = ref(true)

const loadImages = async (page: number) => {
  listLoading.value = true
  try {
    const resp = await ListImages(page)
    currentPage.value = page
    if (resp.data && Array.isArray(resp.data) && resp.data.length > 0) {
      files.value = resp.data
      hasNextPage.value = true
    } else {
      files.value = resp.data || []
      hasNextPage.value = false
    }
  } catch (e: any) {
    console.error(e)
    const msg = typeof e === 'string' ? e : e?.message
    if (msg) {
      toast.error(msg)
    }
  } finally {
    listLoading.value = false
  }
}

const copyUrl = async (url: string) => {
  try {
    await navigator.clipboard.writeText(url)
    toast.success(t('settings.imageHosting.copied'))
  } catch {
    toast.error(t('settings.imageHosting.copyFailed'))
  }
}

const deleteImage = async (hash: string) => {
  try {
    await DeleteImage(hash)
    toast.success(t('settings.imageHosting.deleteSuccess'))
    loadImages(currentPage.value)
  } catch (e: any) {
    console.error(e)
    toast.error(errMsg(e, t('settings.imageHosting.deleteFailed')))
  }
}

onMounted(async () => {
  try {
    const setting = await GetSetting()
    if (setting) {
      form.enabled = setting.enabled || false
      // 兼容旧配置：provider 为空按 S.EE 处理
      form.provider = (setting as any).provider || 'see'
      form.apiKey = setting.apiKey || ''
      const gitee = (setting as any).gitee
      if (gitee) {
        form.gitee.accessToken = gitee.accessToken || ''
        form.gitee.owner = gitee.owner || ''
        form.gitee.repo = gitee.repo || ''
        form.gitee.branch = gitee.branch || ''
        form.gitee.pathPrefix = gitee.pathPrefix || ''
        form.gitee.publicBaseUrl = gitee.publicBaseUrl || ''
      }
      const oss = (setting as any).oss
      if (oss) {
        form.oss.accessKeyId = oss.accessKeyId || ''
        form.oss.accessKeySecret = oss.accessKeySecret || ''
        form.oss.bucket = oss.bucket || ''
        form.oss.endpoint = oss.endpoint || ''
        form.oss.pathPrefix = oss.pathPrefix || ''
        form.oss.customDomain = oss.customDomain || ''
      }
    }
  } catch (e) {
    console.error('Failed to load image hosting settings', e)
  }
})

const buildSettingPayload = () => ({
  enabled: form.enabled,
  provider: form.provider,
  apiKey: form.apiKey,
  gitee: { ...form.gitee },
  oss: { ...form.oss },
})

const submit = async () => {
  try {
    const s = new domain.ImageHostingSetting(buildSettingPayload())
    await SaveSetting(s)
    toast.success(t('common.saveSuccess'))
  } catch (e: any) {
    console.error(e)
    toast.error(errMsg(e, t('common.saveFailed')))
  }
}

// 测试上传：先静默保存当前表单，再传一张 1x1 测试图并立即删除。
const testUpload = async () => {
  testing.value = true
  try {
    const s = new domain.ImageHostingSetting(buildSettingPayload())
    await SaveSetting(s)
    await TestUpload()
    toast.success(t('settings.imageHosting.testSuccess'))
  } catch (e: any) {
    console.error(e)
    toast.error(errMsg(e, t('common.saveFailed')))
  } finally {
    testing.value = false
  }
}

// 上传图片：先静默保存当前表单（后端读的是已保存配置），多选文件上传到当前图床，
// 成功后刷新列表并把图片地址复制到剪贴板。
const uploadImages = async () => {
  uploading.value = true
  try {
    const s = new domain.ImageHostingSetting(buildSettingPayload())
    await SaveSetting(s)
    const paths = await OpenImagesDialog()
    if (!paths || paths.length === 0) return
    const items = paths.map((p: string) => {
      const name = p.split(/[\\/]/).pop() || p
      return new domain.UploadedFile({ name, path: p })
    })
    const urls = await UploadImagesFromFrontend(items)
    await loadImages(1)
    const text = (urls || []).join('\n')
    if (text) {
      try {
        await navigator.clipboard.writeText(text)
      } catch {
        // 剪贴板写入失败不视为上传失败
      }
      toast.success(
        urls.length === 1
          ? t('settings.imageHosting.uploadCopied')
          : t('settings.imageHosting.uploadCopiedMulti', { n: urls.length }),
      )
    }
  } catch (e: any) {
    console.error(e)
    toast.error(errMsg(e, t('settings.imageHosting.uploadFailed')))
  } finally {
    uploading.value = false
  }
}

// 切换图床服务商后刷新图片浏览器。
// 后端 List 每次都读「已保存配置」：只改表单不落盘，列出来的还是旧服务商的图。
// 与上传 / 测试按钮同套路：先静默保存当前表单，再拉列表；刷新按钮随之自然正确。
watch(() => form.provider, async () => {
  if (!form.enabled) return
  try {
    const s = new domain.ImageHostingSetting(buildSettingPayload())
    await SaveSetting(s)
  } catch (e: any) {
    console.error(e)
    toast.error(errMsg(e, t('common.saveFailed')))
    return
  }
  loadImages(1)
})
</script>
