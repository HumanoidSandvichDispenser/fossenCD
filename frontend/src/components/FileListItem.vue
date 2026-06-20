<script setup lang="ts">
import { computed } from 'vue';
import { PhEye, PhEyeClosed } from '@phosphor-icons/vue';

const props = defineProps<{
  file: string;
  // the file currently open in the editor
  active: boolean;
  // the file currently rendered in the preview pane (the build target)
  isPreview: boolean;
}>();

const emit = defineEmits<{
  select: [file: string];
  preview: [file: string];
}>();

// right now we can only preview .typ files
const canPreview = computed(() => props.file.toLowerCase().endsWith('.typ'));
</script>

<template>
  <li class="file-item" :class="{ active }">
    <button type="button" class="name label-sm" :title="file" @click="emit('select', file)">
      {{ file }}
    </button>
    <button
      v-if="canPreview"
      type="button"
      class="eye"
      :class="{ on: isPreview }"
      :title="isPreview ? 'Previewing this file' : 'Preview this file'"
      :aria-pressed="isPreview"
      @click="emit('preview', file)"
    >
      <PhEye v-if="isPreview" :size="16" weight="fill" />
      <PhEyeClosed v-else :size="16" />
    </button>
  </li>
</template>

<style scoped>
.file-item {
  display: flex;
  align-items: center;
  gap: var(--space-1);
  border-radius: var(--radius-md);
  transition: background var(--duration-fast) var(--ease-out);
}

.file-item:hover {
  background: var(--color-surface-hover);
}

.file-item.active {
  background: var(--color-primary-light);
}

.name {
  flex: 1;
  min-width: 0;
  text-align: left;
  padding: var(--space-2) var(--space-3);
  color: var(--color-text-secondary);
  background: none;
  border: none;
  cursor: pointer;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.file-item:hover .name {
  color: var(--color-text);
}

.file-item.active .name {
  color: var(--color-primary);
}

.eye {
  flex: none;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: var(--space-1);
  margin-right: var(--space-1);
  color: var(--color-text-tertiary);
  background: none;
  border: none;
  border-radius: var(--radius-sm);
  cursor: pointer;
  opacity: 0;
  transition: opacity var(--duration-fast) var(--ease-out), color var(--duration-fast) var(--ease-out);
}

.file-item:hover .eye {
  opacity: 1;
}

.eye:hover {
  color: var(--color-text);
}

/* HACK: typst.ts renderer injects global svg { fill: none } for output */
.eye :deep(svg) {
  fill: currentColor;
}

.eye.on {
  opacity: 1;
  color: var(--color-primary);
}
</style>
