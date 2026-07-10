<script setup lang="ts">
import { computed, ref, watch } from 'vue';

import BaseDialog from './BaseDialog.vue';

const props = defineProps<{ open: boolean; existing: string[] }>();
const emit = defineEmits<{ close: []; create: [path: string] }>();

const path = ref('');

// Reset the field each time the dialog opens.
watch(
  () => props.open,
  (open) => {
    if (open) {
      path.value = '';
    }
  },
);

// The path the backend will actually see: slashes denote directories, so trim
// the whole thing but keep interior separators, and drop empty components the
// way create_file_at does.
const cleaned = computed(() => path.value.trim().replace(/^\/+|\/+$/g, ''));

const error = computed(() => {
  if (!cleaned.value) {
    return null; // nothing typed yet; not an error, just can't submit
  }
  if (props.existing.includes(cleaned.value)) {
    return 'A file with that path already exists.';
  }
  return null;
});

function submit() {
  if (!cleaned.value || error.value) {
    return;
  }
  emit('create', cleaned.value);
  emit('close');
}
</script>

<template>
  <BaseDialog :open="open" @close="emit('close')">
    <form class="form" @submit.prevent="submit">
      <h2 class="h4">New file</h2>

      <label class="field">
        <span class="label-sm">Path</span>
        <input
          v-model="path"
          autofocus
          required
          minlength="1"
          maxlength="256"
          placeholder="chapter.typ"
        />
        <span class="hint text-xs">Use / to place it in a folder, e.g. chapters/intro.typ</span>
      </label>

      <p v-if="error" class="error text-sm">{{ error }}</p>

      <div class="actions">
        <button type="button" class="label-md btn btn-secondary" @click="emit('close')">Cancel</button>
        <button type="submit" class="label-md btn btn-primary" :disabled="!cleaned || !!error">
          Create
        </button>
      </div>
    </form>
  </BaseDialog>
</template>

<style scoped>
.form {
  display: contents;
}

.field {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
}

.field span {
  color: var(--color-text-secondary);
}

.field input {
  padding: var(--space-2) var(--space-3);
  color: var(--color-text);
  background: var(--color-bg-page);
  border: var(--border-thin) solid var(--color-border);
  border-radius: var(--radius-md);
}

.field input:focus {
  outline: none;
  border-color: var(--color-primary);
}

.hint {
  color: var(--color-text-tertiary);
}

.error {
  color: var(--color-error);
}

.actions {
  display: flex;
  justify-content: flex-end;
  gap: var(--space-3);
}
</style>
