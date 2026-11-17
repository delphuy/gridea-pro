<template>
  <div class="pb-20 max-w-4xl mx-auto pt-4">
    <div class="space-y-6">
      <!-- 关闭时最小化到系统托盘（应用级设置，切换即时生效） -->
      <div class="grid grid-cols-[180px_1fr] items-start gap-4">
        <label class="text-sm font-medium text-right text-muted-foreground pt-2">{{ t('settings.window.minimizeToTray') }}</label>
        <div class="flex items-center gap-3">
          <Switch :checked="minimizeToTray" size="sm" @update:checked="(v: boolean) => onToggleMinimizeToTray(v)" />
          <span class="text-xs text-muted-foreground">{{ t('settings.window.minimizeToTrayDesc') }}</span>
        </div>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { toast } from '@/helpers/toast'
import { Switch } from '@/components/ui/switch'
import { GetMinimizeToTray, SetMinimizeToTray } from '@/wailsjs/go/app/App'

const { t } = useI18n()

// ─── 窗口设置 ───────────────────────────────────────────
// 「关闭时最小化到托盘」（应用级，默认关闭）
const minimizeToTray = ref(false)

const loadWindowSetting = async () => {
  try {
    minimizeToTray.value = await GetMinimizeToTray()
  } catch (e) {
    console.error('Failed to load minimize-to-tray setting:', e)
  }
}

const onToggleMinimizeToTray = async (val: boolean) => {
  const previous = minimizeToTray.value
  minimizeToTray.value = val
  try {
    await SetMinimizeToTray(val)
    toast.success(t('settings.window.saveSuccess'))
  } catch (e: any) {
    console.error('Save minimize-to-tray setting failed:', e)
    minimizeToTray.value = previous
    toast.error(e?.message || t('settings.window.saveFailed'))
  }
}

onMounted(loadWindowSetting)
</script>
