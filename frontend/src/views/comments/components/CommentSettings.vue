<template>
  <Sheet v-model:open="visible">
    <SheetContent side="right" class="w-[400px] sm:max-w-md p-0 gap-0 flex flex-col">
      <SheetHeader class="px-6 py-6 border-b">
        <SheetTitle>{{ t('settings.comment.title') }}</SheetTitle>
      </SheetHeader>

      <div class="flex-1 overflow-y-auto px-6 py-6">
        <CommentSettingsForm ref="formRef" @saved="onSaved" />
      </div>

      <SheetFooter class="flex-shrink-0 px-6 py-4 border-t gap-3">
        <Button
variant="outline"
          class="w-18 h-8 text-xs justify-center rounded-full border border-primary/20 text-primary/80 hover:bg-primary/5 hover:text-primary cursor-pointer"
          @click="onClose">{{ t('common.cancel') }}</Button>
        <Button
variant="default"
          :disabled="!!formRef?.loadFailed"
          class="w-18 h-8 text-xs justify-center rounded-full bg-primary text-background hover:bg-primary/90 cursor-pointer"
          @click="save">{{ t('common.save') }}</Button>
      </SheetFooter>
    </SheetContent>
  </Sheet>
</template>

<script lang="ts" setup>
import { ref, watch, computed, nextTick } from 'vue'
import { useI18n } from 'vue-i18n'
import { Sheet, SheetContent, SheetHeader, SheetTitle, SheetFooter } from '@/components/ui/sheet'
import { Button } from '@/components/ui/button'
import { useCommentStore } from '@/stores/comment'
import CommentSettingsForm from './CommentSettingsForm.vue'

const props = defineProps<{
  open: boolean
}>()

const { t } = useI18n()

const emit = defineEmits(['update:open'])

const visible = computed({
  get: () => props.open,
  set: (val) => emit('update:open', val)
})

const commentStore = useCommentStore()
const formRef = ref<InstanceType<typeof CommentSettingsForm> | null>(null)

// 打开时先拉后端配置填充表单（表单内部含 loadFailed 保护，防止空表单覆盖密钥）
watch(() => props.open, async (newVal) => {
  if (newVal) {
    await nextTick()
    formRef.value?.reload()
  }
})

const onClose = () => {
  visible.value = false
}

const save = () => {
  formRef.value?.save()
}

const onSaved = () => {
  onClose()
  // Refresh in background so UI doesn't freeze
  commentStore.fetchComments()
}
</script>
