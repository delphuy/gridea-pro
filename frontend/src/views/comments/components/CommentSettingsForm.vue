<template>
  <div class="space-y-6">
    <div class="flex items-center justify-between">
      <label class="text-sm font-medium text-foreground">{{ t('settings.comment.enable') }}</label>
      <Switch
size="sm" class="focus-visible:ring-0 focus-visible:ring-offset-0" :checked="formState.enable"
        @update:checked="val => formState.enable = val" />
    </div>

    <div>
      <Label class="text-sm font-medium text-foreground mb-2 block">{{ t('settings.comment.platform') }}</Label>
      <Select v-model="formState.platform">
        <SelectTrigger>
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          <SelectItem v-for="p in platforms" :key="p" :value="p">{{ p }}</SelectItem>
        </SelectContent>
      </Select>
      <div class="ml-1 mt-2 text-xs text-primary flex items-center gap-1 cursor-pointer" @click="openDocs">
        {{ t('settings.comment.viewDocs', { platform: formState.platform }) }}
        <ArrowTopRightOnSquareIcon class="size-3" />
      </div>
    </div>

    <div v-if="formState.enable" class="space-y-4 pt-4 border-t border-border">
      <template v-if="['Valine', 'Waline'].includes(formState.platform)">
        <!-- Shared config interface for Valine/Waline -->
        <ValineWalineConfig
v-if="formState.platform === 'Valine'"
          v-model:config="formState.platformConfigs['Valine']"
          :server-urls-tip="t('settings.comment.serverUrlsTip')" />
        <ValineWalineConfig v-else v-model:config="formState.platformConfigs['Waline']" />
      </template>

      <template v-else-if="formState.platform === 'Twikoo'">
        <TwikooConfig v-model:config="formState.platformConfigs['Twikoo']" />
      </template>

      <template v-else-if="formState.platform === 'Gitalk'">
        <GitalkConfig v-model:config="formState.platformConfigs['Gitalk']" />
      </template>

      <template v-else-if="formState.platform === 'Disqus'">
        <DisqusConfig v-model:config="formState.platformConfigs['Disqus']" />
      </template>

      <template v-else-if="formState.platform === 'Cusdis'">
        <CusdisConfig v-model:config="formState.platformConfigs['Cusdis']" />
      </template>

      <template v-else-if="formState.platform === 'Giscus'">
        <GiscusConfig v-model:config="formState.platformConfigs['Giscus']" />
      </template>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { ArrowTopRightOnSquareIcon } from '@heroicons/vue/24/outline'
import { Switch } from '@/components/ui/switch'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { useCommentStore } from '@/stores/comment'
import type { CommentSettings } from '@/types/comment'
import { BrowserOpenURL } from '@/wailsjs/runtime'
import { toast } from '@/helpers/toast'

import ValineWalineConfig from './platforms/ValineWalineConfig.vue'
import TwikooConfig from './platforms/TwikooConfig.vue'
import GitalkConfig from './platforms/GitalkConfig.vue'
import DisqusConfig from './platforms/DisqusConfig.vue'
import CusdisConfig from './platforms/CusdisConfig.vue'
import GiscusConfig from './platforms/GiscusConfig.vue'

const { t } = useI18n()
const commentStore = useCommentStore()
const platforms = ['Valine', 'Waline', 'Twikoo', 'Gitalk', 'Cusdis', 'Giscus', 'Disqus']

// 加载失败标志：评论设置读取失败时表单为空，禁止保存以免空值覆盖评论平台密钥。
const loadFailed = ref(false)

const emit = defineEmits(['saved'])

const formState = ref<CommentSettings>({
  enable: false,
  platform: 'Valine',
  platformConfigs: {},
})

// 从后端拉取配置并填充表单；读取失败置 loadFailed 并禁止保存
const reload = async () => {
  try {
    await commentStore.loadSettings()
    loadFailed.value = false
  } catch (e) {
    // 加载失败：禁止保存，否则空表单一存就把评论平台密钥覆盖成空值。
    console.error('Failed to load comment settings', e)
    loadFailed.value = true
    toast.error(t('settings.loadFailedNoSave'))
    return
  }

  const newSettings = JSON.parse(JSON.stringify(commentStore.settings))

  // Ensure platformConfigs exists on the source
  if (!newSettings.platformConfigs) {
    newSettings.platformConfigs = {}
  }

  // Ensure all platform keys exist
  platforms.forEach(p => {
    if (!newSettings.platformConfigs[p]) {
      newSettings.platformConfigs[p] = {}
    }
  })

  formState.value = newSettings
}

const save = async () => {
  if (loadFailed.value) {
    toast.error(t('settings.loadFailedNoSave'))
    return
  }
  // Ensure we are saving the current state
  const settingsToSave = {
    ...formState.value,
    platformConfigs: formState.value.platformConfigs || {}
  }

  await commentStore.saveSettings(settingsToSave)
  toast.success(t('settings.comment.saveSuccess'))
  emit('saved')
}

const openDocs = () => {
  const urls: Record<string, string> = {
    Gitalk: 'https://github.com/gitalk/gitalk',
    Disqus: 'https://disqus.com/',
    Valine: 'https://valine.js.org/',
    Waline: 'https://waline.js.org/',
    Twikoo: 'https://twikoo.js.org/',
    Cusdis: 'https://cusdis.com/',
    Giscus: 'https://giscus.app/',
  }
  const url = urls[formState.value.platform]
  if (url) {
    BrowserOpenURL(url)
  }
}

defineExpose({ reload, save, loadFailed })
</script>
