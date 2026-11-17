<template>
  <div class="pb-20 max-w-4xl mx-auto pt-4">
    <div class="space-y-6">
      <!-- 启用 CDN -->
      <div class="grid grid-cols-[180px_1fr] items-center gap-4">
        <label class="text-sm font-medium text-right text-muted-foreground">{{ t('settings.cdn.enable') }}</label>
        <div class="flex items-center gap-3">
          <Switch :checked="form.enabled" @update:checked="(v: boolean) => form.enabled = v" size="sm" />
          <span class="text-xs text-muted-foreground">{{ t('settings.cdn.enableDesc') }}</span>
        </div>
      </div>

      <template v-if="form.enabled">
        <!-- 提示信息 -->
        <div class="grid grid-cols-[180px_1fr] items-start gap-4">
          <div></div>
          <div class="rounded-md border border-primary/20 bg-primary/5 p-3 text-xs text-primary/80 max-w-sm">
            {{ t('settings.cdn.notice') }}
          </div>
        </div>

        <!-- CDN 服务商 -->
        <div class="grid grid-cols-[180px_1fr] items-center gap-4">
          <label class="text-sm font-medium text-right text-muted-foreground">{{ t('settings.cdn.provider') }}</label>
          <div class="w-full max-w-sm">
            <Select :model-value="form.provider" @update:model-value="(v) => form.provider = v as string">
              <SelectTrigger>
                <SelectValue :placeholder="t('settings.cdn.provider')" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="jsdelivr">jsDelivr</SelectItem>
                <SelectItem value="custom">{{ t('settings.cdn.custom') }}</SelectItem>
              </SelectContent>
            </Select>
          </div>
        </div>

        <!-- jsDelivr 配置 -->
        <template v-if="form.provider === 'jsdelivr'">
          <div class="grid grid-cols-[180px_1fr] items-center gap-4">
            <label class="text-sm font-medium text-right text-muted-foreground">{{ t('settings.cdn.githubUser') }}</label>
            <div class="max-w-sm">
              <Input v-model="form.githubUser" placeholder="username" />
            </div>
          </div>
          <div class="grid grid-cols-[180px_1fr] items-center gap-4">
            <label class="text-sm font-medium text-right text-muted-foreground">{{ t('settings.cdn.githubRepo') }}</label>
            <div class="max-w-sm">
              <Input v-model="form.githubRepo" placeholder="repo-name" />
            </div>
          </div>
          <div class="grid grid-cols-[180px_1fr] items-center gap-4">
            <label class="text-sm font-medium text-right text-muted-foreground">{{ t('settings.cdn.githubBranch') }}</label>
            <div class="max-w-sm">
              <Input v-model="form.githubBranch" placeholder="main" />
            </div>
          </div>

          <!-- GitHub Token -->
          <div class="grid grid-cols-[180px_1fr] items-start gap-4">
            <label class="text-sm font-medium text-right text-muted-foreground pt-2">{{ t('settings.cdn.githubToken') }}</label>
            <div class="max-w-sm">
              <div class="relative">
                <Input
                  v-model="form.githubToken"
                  :type="showToken ? 'text' : 'password'"
                  placeholder="ghp_xxxxxxxxxxxx"
                />
                <button
                  type="button"
                  class="absolute right-2 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground cursor-pointer"
                  @click="showToken = !showToken"
                >
                  <svg v-if="showToken" xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M2 12s3-7 10-7 10 7 10 7-3 7-10 7-10-7-10-7Z"/><circle cx="12" cy="12" r="3"/></svg>
                  <svg v-else xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M9.88 9.88a3 3 0 1 0 4.24 4.24"/><path d="M10.73 5.08A10.43 10.43 0 0 1 12 5c7 0 10 7 10 7a13.16 13.16 0 0 1-1.67 2.68"/><path d="M6.61 6.61A13.526 13.526 0 0 0 2 12s3 7 10 7a9.74 9.74 0 0 0 5.39-1.61"/><line x1="2" x2="22" y1="2" y2="22"/></svg>
                </button>
              </div>
              <div class="text-xs text-muted-foreground mt-1.5">{{ t('settings.cdn.githubTokenDesc') }}</div>
            </div>
          </div>

          <div class="grid grid-cols-[180px_1fr] items-center gap-4">
            <div></div>
            <div class="text-xs text-muted-foreground">
              {{ t('settings.cdn.jsdelivrTip') }}<br>
              <code class="text-primary/80">cdn.jsdelivr.net/gh/{{ form.githubUser || 'user' }}/{{ form.githubRepo || 'repo' }}@{{ form.githubBranch || 'main' }}/</code>
            </div>
          </div>
        </template>

        <!-- 自定义 CDN -->
        <template v-if="form.provider === 'custom'">
          <div class="grid grid-cols-[180px_1fr] items-start gap-4">
            <label class="text-sm font-medium text-right text-muted-foreground pt-2">{{ t('settings.cdn.baseUrl') }}</label>
            <div class="max-w-sm">
              <Input v-model="form.baseUrl" placeholder="https://cdn.example.com" />
              <div class="text-xs text-muted-foreground mt-1.5">{{ t('settings.cdn.baseUrlDesc') }}</div>
            </div>
          </div>
        </template>

        <!-- 保存路径 -->
        <div class="grid grid-cols-[180px_1fr] items-start gap-4">
          <label class="text-sm font-medium text-right text-muted-foreground pt-2">{{ t('settings.cdn.savePath') }}</label>
          <div class="max-w-sm space-y-2">
            <Select :model-value="savePathOption" @update:model-value="onSavePathOptionChange">
              <SelectTrigger>
                <SelectValue :placeholder="t('settings.cdn.savePath')" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="{year}/{month}/{filename}{.suffix}">{year}/{month}/{filename}{.suffix}</SelectItem>
                <SelectItem value="{year}/{month}/{random}{.suffix}">{year}/{month}/{random}{.suffix}</SelectItem>
                <SelectItem value="{year}/{month}/{day}/{filename}{.suffix}">{year}/{month}/{day}/{filename}{.suffix}</SelectItem>
                <SelectItem value="{year}/{month}/{day}/{random}{.suffix}">{year}/{month}/{day}/{random}{.suffix}</SelectItem>
                <SelectItem value="{year}/{month}/{filename}-{random}{.suffix}">{year}/{month}/{filename}-{random}{.suffix}</SelectItem>
                <SelectItem value="custom">{{ t('settings.cdn.savePathCustom') }}</SelectItem>
              </SelectContent>
            </Select>
            <template v-if="savePathOption === 'custom'">
              <Input
                v-model="form.savePath"
                placeholder="{year}/{month}/{filename}{.suffix}"
              />
              <button
                type="button"
                class="text-xs text-primary/70 hover:text-primary cursor-pointer"
                @click="showVarsHelp = !showVarsHelp"
              >
                {{ t('settings.cdn.configHelp') }}
              </button>
              <div v-if="showVarsHelp" class="text-xs text-muted-foreground">
                <p class="mb-1.5">{{ t('settings.cdn.savePathVars') }}</p>
              </div>
              <ul v-if="showVarsHelp" class="text-xs text-muted-foreground space-y-1 pl-4 list-disc">
                <li><code>{year}</code> — {{ t('settings.cdn.varYear') }}</li>
                <li><code>{month}</code> — {{ t('settings.cdn.varMonth') }}</li>
                <li><code>{day}</code> — {{ t('settings.cdn.varDay') }}</li>
                <li><code>{filename}</code> — {{ t('settings.cdn.varFilename') }}</li>
                <li><code>{random}</code> — {{ t('settings.cdn.varRandom') }}</li>
                <li><code>{.suffix}</code> — {{ t('settings.cdn.varDotSuffix') }}</li>
                <li><code>{suffix}</code> — {{ t('settings.cdn.varSuffix') }}</li>
              </ul>
            </template>
          </div>
        </div>
      </template>

      <!-- 图片引用治理（与 CDN 启用状态无关） -->
      <div class="pt-4 mt-2 border-t border-border/60">
        <div class="grid grid-cols-[180px_1fr] items-start gap-4">
          <div></div>
          <div class="max-w-xl space-y-3">
            <div class="flex items-center justify-between gap-3">
              <div>
                <div class="text-sm font-medium">{{ t('settings.cdn.gov.title') }}</div>
                <div class="text-xs text-muted-foreground mt-0.5">{{ t('settings.cdn.gov.desc') }}</div>
              </div>
              <Button
                variant="outline"
                class="h-8 shrink-0 text-xs justify-center rounded-full border border-primary/20 text-primary/80 hover:bg-primary/5 hover:text-primary cursor-pointer"
                :disabled="scanLoading"
                @click="doScan">
                {{ scanLoading ? t('settings.cdn.gov.scanning') : t('settings.cdn.gov.scanBtn') }}
              </Button>
            </div>

            <!-- 扫描结果 -->
            <div v-if="scanResult" class="rounded-md border p-3 text-xs space-y-2">
              <div class="text-muted-foreground">
                {{ t('settings.cdn.gov.summary', { total: scanResult.total, count: scanResult.orphans.length }) }}
              </div>
              <div v-if="scanResult.orphans.length === 0" class="text-emerald-600 dark:text-emerald-400">
                {{ t('settings.cdn.gov.clean') }}
              </div>
              <template v-else>
                <ul class="max-h-44 overflow-y-auto space-y-1">
                  <li v-for="o in scanResult.orphans" :key="o.relPath" class="flex items-center justify-between gap-3">
                    <span class="truncate font-mono text-[11px] text-foreground/80">{{ o.relPath }}</span>
                    <span class="shrink-0 text-muted-foreground">{{ formatSize(o.sizeByte) }}</span>
                  </li>
                </ul>
                <div class="flex items-center gap-2 pt-1">
                  <Button
                    variant="outline"
                    class="h-7 text-xs justify-center rounded-full border border-destructive/30 text-destructive/90 hover:bg-destructive/10 cursor-pointer"
                    :disabled="moveLoading"
                    @click="doMove">
                    {{ moveLoading ? t('settings.cdn.gov.moving') : t('settings.cdn.gov.moveBtn') }}
                  </Button>
                </div>
              </template>
            </div>

            <!-- 回收站 -->
            <div class="flex items-center justify-between gap-3 rounded-md border border-border/60 bg-muted/30 px-3 py-2">
              <div class="text-xs">
                <span class="font-medium">{{ t('settings.cdn.gov.trashTitle') }}</span>
                <span class="text-muted-foreground ml-2">
                  {{ trash && trash.fileCount > 0 ? t('settings.cdn.gov.trashSummary', { count: trash.fileCount, size: formatSize(trash.sizeByte) }) : t('settings.cdn.gov.trashEmpty') }}
                </span>
              </div>
              <Button
                v-if="trash && trash.fileCount > 0"
                variant="outline"
                class="h-7 shrink-0 text-xs justify-center rounded-full cursor-pointer"
                :class="emptyConfirm
                  ? 'border-destructive/50 bg-destructive/10 text-destructive'
                  : 'border-border text-muted-foreground hover:text-foreground'"
                :disabled="emptyLoading"
                @click="doEmptyTrash">
                {{ emptyLoading ? t('settings.cdn.gov.emptying') : (emptyConfirm ? t('settings.cdn.gov.emptyConfirm') : t('settings.cdn.gov.emptyBtn')) }}
              </Button>
            </div>

            <div class="text-xs text-muted-foreground/80 leading-relaxed">
              {{ t('settings.cdn.gov.note') }}
            </div>
          </div>
        </div>
      </div>
    </div>

    <footer-box>
      <div class="flex justify-end items-center gap-3 w-full">
        <Button
          v-if="form.enabled && form.provider === 'jsdelivr' && form.githubToken"
          variant="outline"
          class="h-8 text-xs justify-center rounded-full border border-primary/20 text-primary/80 hover:bg-primary/5 hover:text-primary cursor-pointer"
          :disabled="testLoading"
          @click="testUpload">
          {{ testLoading ? t('settings.cdn.testing') : t('settings.cdn.testUpload') }}
        </Button>
        <Button
          variant="default"
          :disabled="loadFailed"
          class="w-18 h-8 text-xs justify-center rounded-full bg-primary text-background hover:bg-primary/90 cursor-pointer"
          @click="submit">
          {{ t('common.save') }}
        </Button>
      </div>
    </footer-box>
  </div>
</template>

<script lang="ts" setup>
import { reactive, ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { toast } from '@/helpers/toast'
import FooterBox from '@/components/FooterBox/index.vue'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import { Switch } from '@/components/ui/switch'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { GetCdnSetting, SaveCdnSettingFromFrontend } from '@/wailsjs/go/facade/CdnSettingFacade'
import { TestCdnUpload } from '@/wailsjs/go/facade/CdnUploadFacade'
import { ScanOrphanImages, MoveOrphanImagesToTrash, ListMediaTrash, EmptyMediaTrash } from '@/wailsjs/go/facade/MediaRefFacade'
import { domain } from '@/wailsjs/go/models'

const { t } = useI18n()

const showToken = ref(false)
const testLoading = ref(false)
// 加载失败标志：设置读取失败时表单为空，禁止保存以免空值覆盖磁盘配置。
const loadFailed = ref(false)
const showVarsHelp = ref(false)

const presetPaths = [
  '{year}/{month}/{filename}{.suffix}',
  '{year}/{month}/{day}/{filename}{.suffix}',
  '{year}/{month}/{filename}-{random}{.suffix}',
  '{year}/{month}/{random}{.suffix}',
  '{year}/{month}/{day}/{random}{.suffix}',
]

const form = reactive({
  enabled: false,
  provider: 'jsdelivr',
  githubUser: '',
  githubRepo: '',
  githubBranch: 'main',
  baseUrl: '',
  githubToken: '',
  savePath: '{year}/{month}/{filename}{.suffix}',
})

const savePathOption = computed(() => {
  if (presetPaths.includes(form.savePath)) {
    return form.savePath
  }
  return 'custom'
})

const onSavePathOptionChange = (v: string) => {
  if (v === 'custom') {
    // Keep current value for editing
    if (presetPaths.includes(form.savePath)) {
      form.savePath = ''
    }
  } else {
    form.savePath = v
  }
}

onMounted(async () => {
  try {
    const setting = await GetCdnSetting()
    if (setting) {
      form.enabled = setting.enabled || false
      form.provider = setting.provider || 'jsdelivr'
      form.githubUser = setting.githubUser || ''
      form.githubRepo = setting.githubRepo || ''
      form.githubBranch = setting.githubBranch || 'main'
      form.baseUrl = setting.baseUrl || ''
      form.githubToken = setting.githubToken || ''
      form.savePath = setting.savePath || '{year}/{month}/{filename}{.suffix}'
    }
  } catch (e) {
    console.error('Failed to load CDN settings', e)
    loadFailed.value = true
    toast.error(t('settings.loadFailedNoSave'))
  }
})

const submit = async () => {
  if (loadFailed.value) {
    toast.error(t('settings.loadFailedNoSave'))
    return
  }
  try {
    const settingDomain = new domain.CdnSetting(form)
    await SaveCdnSettingFromFrontend(settingDomain)
    toast.success(t('settings.cdn.saveSuccess'))
  } catch (e) {
    console.error(e)
    toast.error(t('settings.cdn.saveFailed'))
  }
}

const testUpload = async () => {
  testLoading.value = true
  try {
    // Save settings first
    const settingDomain = new domain.CdnSetting(form)
    await SaveCdnSettingFromFrontend(settingDomain)

    const cdnUrl = await TestCdnUpload()
    toast.success(t('settings.cdn.testUploadSuccess'))
  } catch (e: any) {
    console.error(e)
    toast.error(e.message || e)
  } finally {
    testLoading.value = false
  }
}

// ── 图片引用治理（孤儿扫描 / 回收站）──
interface MediaFileLite {
  relPath: string
  sizeByte: number
}

const scanLoading = ref(false)
const moveLoading = ref(false)
const emptyLoading = ref(false)
// 清空回收站采用"二次点击确认"：第一次点击进入确认态（4 秒后自动退出），再点一次才执行
const emptyConfirm = ref(false)
const scanResult = ref<{ total: number; orphans: MediaFileLite[] } | null>(null)
const trash = ref<{ fileCount: number; sizeByte: number } | null>(null)
let emptyConfirmTimer: ReturnType<typeof setTimeout> | null = null

const formatSize = (n: number): string => {
  if (!n || n < 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB']
  let i = 0
  let v = n
  while (v >= 1024 && i < units.length - 1) {
    v /= 1024
    i++
  }
  return `${v >= 100 ? Math.round(v) : v.toFixed(1)} ${units[i]}`
}

const doScan = async () => {
  scanLoading.value = true
  try {
    const res = await ScanOrphanImages()
    scanResult.value = { total: res?.total ?? 0, orphans: res?.orphans ?? [] }
  } catch (e: any) {
    console.error(e)
    toast.error(e?.message || t('settings.cdn.gov.failed'))
  } finally {
    scanLoading.value = false
  }
}

const doMove = async () => {
  moveLoading.value = true
  try {
    const res = await MoveOrphanImagesToTrash()
    toast.success(t('settings.cdn.gov.moved', { count: (res?.moved ?? []).length }))
    scanResult.value = null
    await loadTrash()
  } catch (e: any) {
    console.error(e)
    toast.error(e?.message || t('settings.cdn.gov.failed'))
  } finally {
    moveLoading.value = false
  }
}

const loadTrash = async () => {
  try {
    const res = await ListMediaTrash()
    trash.value = { fileCount: res?.fileCount ?? 0, sizeByte: res?.sizeByte ?? 0 }
  } catch (e) {
    console.error('Failed to list media trash', e)
  }
}

const doEmptyTrash = async () => {
  if (!emptyConfirm.value) {
    emptyConfirm.value = true
    if (emptyConfirmTimer) clearTimeout(emptyConfirmTimer)
    emptyConfirmTimer = setTimeout(() => {
      emptyConfirm.value = false
    }, 4000)
    return
  }
  emptyLoading.value = true
  try {
    await EmptyMediaTrash()
    toast.success(t('settings.cdn.gov.emptied'))
    emptyConfirm.value = false
    if (emptyConfirmTimer) {
      clearTimeout(emptyConfirmTimer)
      emptyConfirmTimer = null
    }
    await loadTrash()
  } catch (e: any) {
    console.error(e)
    toast.error(e?.message || t('settings.cdn.gov.failed'))
  } finally {
    emptyLoading.value = false
  }
}

onMounted(loadTrash)
</script>
