<script setup lang="ts">
import { PhPlus } from '@phosphor-icons/vue';

import FileListItem from './FileListItem.vue';

defineProps<{
  files: string[];
  currentFile: string | null;
  previewFile: string | null;
}>();

const emit = defineEmits<{
  select: [file: string];
  preview: [file: string];
  add: [];
}>();
</script>

<template>
  <aside class="sidebar">
    <div class="sidebar-head label-xs">
      <span>Files</span>
      <button type="button" class="add-file" aria-label="New file" @click="emit('add')">
        <PhPlus :size="14" />
      </button>
    </div>
    <ul v-if="files.length" class="files">
      <FileListItem
        v-for="file in files"
        :key="file"
        :file="file"
        :active="file === currentFile"
        :is-preview="file === previewFile"
        @select="emit('select', $event)"
        @preview="emit('preview', $event)"
      />
    </ul>
    <p v-else class="sidebar-empty text-xs">No files yet.</p>
  </aside>
</template>

<style scoped>
.sidebar {
  display: flex;
  flex-direction: column;
  min-height: 0;
  padding: var(--space-3) var(--space-2);
  overflow: auto;
}

.sidebar-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-2);
  padding: var(--space-1) var(--space-2) var(--space-3);
  color: var(--color-text-tertiary);
}

.add-file {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 2px;
  color: var(--color-text-tertiary);
  background: none;
  border: none;
  border-radius: var(--radius-sm);
  cursor: pointer;
}

.add-file:hover {
  color: var(--color-text);
  background: var(--color-surface-hover);
}

.files {
  list-style: none;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.sidebar-empty {
  padding: var(--space-2) var(--space-3);
  color: var(--color-text-tertiary);
}
</style>
