<template>
  <div class="h-full flex flex-col bg-background">
    <!-- Header Tools -->
    <div
      class="flex-shrink-0 flex justify-between items-center px-4 h-12 bg-background sticky top-0 z-50 border-b border-border backdrop-blur-sm bg-opacity-90 select-none"
      style="--wails-draggable: drag">
      <div class="flex-1"></div>
      <div
        class="flex items-center justify-center w-8 h-8 rounded-full hover:bg-primary/10 cursor-pointer transition-colors text-muted-foreground hover:text-foreground"
        :title="t('link.new')" style="--wails-draggable: no-drag" @click="openCreateSheet">
        <PlusIcon class="size-4" />
      </div>
    </div>

    <!-- Content -->
    <div class="flex-1 overflow-y-auto px-4 py-6">
      <!-- 空状态：一条友链都没有时，页面只剩右上角一个 32px 小加号，
           看起来就是一片白。这里给出明确的引导入口。 -->
      <div v-if="linkList.length === 0" class="flex flex-col items-center justify-center py-20 text-muted-foreground">
        <img class="w-32 h-32 mb-4 opacity-50" src="@/assets/images/graphic-empty-box.svg" alt="">
        <div class="text-lg mb-1">{{ t('link.emptyTitle') }}</div>
        <div class="text-xs mb-6">{{ t('link.emptyHint') }}</div>
        <Button
class="h-9 px-6 rounded-full bg-primary text-background text-sm hover:bg-primary/90 cursor-pointer"
          @click="openCreateSheet">
          <PlusIcon class="size-4 mr-1" />
          {{ t('link.emptyAdd') }}
        </Button>
      </div>

      <draggable
v-else
v-model="linkList" handle=".drag-handle" item-key="id" class="grid grid-cols-1 md:grid-cols-3 gap-4"
        @change="handleLinkSort">
        <template #item="{ element: link, index }">
          <LinkCard :link="link" :index="index" @preview="openLink" @edit="openEditSheet" @delete="confirmDelete" />
        </template>
      </draggable>
    </div>

    <!-- Edit/New Drawer -->
    <LinkEditor
v-model:open="visible" :form="form" :can-submit="canSubmit" @save="saveLink" @close="closeSheet"
      @name-change="handleNameChange" @url-change="handleUrlChange" @avatar-change="handleAvatarChange" />

    <DeleteConfirmDialog v-model:open="deleteModalVisible" @confirm="handleDelete" />

  </div>
</template>

<script lang="ts" setup>
import { useI18n } from 'vue-i18n'
import Draggable from 'vuedraggable'
import { PlusIcon } from '@heroicons/vue/24/outline'
import DeleteConfirmDialog from '@/components/ConfirmDialog/DeleteConfirmDialog.vue'
import { Button } from '@/components/ui/button'
import LinkCard from './components/LinkCard.vue'
import LinkEditor from './components/LinkEditor.vue'
import { useLink } from './composables/useLink'

const { t } = useI18n()

const {
  visible,
  form,
  linkList,
  deleteModalVisible,
  canSubmit,
  openCreateSheet,
  openEditSheet,
  closeSheet,
  saveLink,
  confirmDelete,
  handleDelete,
  openLink,
  handleLinkSort,
  handleNameChange,
  handleUrlChange,
  handleAvatarChange,
} = useLink()

</script>
