<template>
  <div class="pb-20 max-w-4xl mx-auto pt-4">
    <CommentSettingsForm ref="formRef" class="max-w-xl" />

    <footer-box>
      <div class="flex justify-end items-center gap-3 w-full">
        <Button
          variant="default"
          :disabled="!!formRef?.loadFailed"
          class="w-18 h-8 text-xs justify-center rounded-full bg-primary text-background hover:bg-primary/90 cursor-pointer"
          @click="save">
          {{ t('common.save') }}
        </Button>
      </div>
    </footer-box>
  </div>
</template>

<script lang="ts" setup>
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import FooterBox from '@/components/FooterBox/index.vue'
import { Button } from '@/components/ui/button'
import CommentSettingsForm from '@/views/comments/components/CommentSettingsForm.vue'

const { t } = useI18n()
const formRef = ref<InstanceType<typeof CommentSettingsForm> | null>(null)

onMounted(() => {
  formRef.value?.reload()
})

const save = () => {
  formRef.value?.save()
}
</script>
