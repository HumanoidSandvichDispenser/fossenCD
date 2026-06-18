<script setup lang="ts">
import { ref, watch } from 'vue';

import { useProjectsStore } from '@/stores/projects';
import type { ProjectView } from '@/client/types.gen';
import BaseDialog from './BaseDialog.vue';

const props = defineProps<{ open: boolean }>();
const emit = defineEmits<{ close: []; created: [project: ProjectView] }>();

const projects = useProjectsStore();

const name = ref('');
const error = ref<string | null>(null);
const loading = ref(false);

// Reset state each time the dialog opens.
watch(
  () => props.open,
  (open) => {
    if (open) {
      name.value = '';
      error.value = null;
    }
  },
);

async function submit() {
  const trimmed = name.value.trim();
  if (!trimmed) {
    return;
  }
  error.value = null;
  loading.value = true;
  try {
    const project = await projects.create(trimmed);
    emit('created', project);
    emit('close');
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Could not create project';
  } finally {
    loading.value = false;
  }
}
</script>

<template>
  <BaseDialog :open="open" @close="emit('close')">
    <form class="form" @submit.prevent="submit">
      <h2 class="h4">New project</h2>

      <label class="field">
        <span class="label-sm">Name</span>
        <input
          v-model="name"
          autofocus
          required
          minlength="1"
          maxlength="128"
          placeholder="My document"
        />
      </label>

      <p v-if="error" class="error text-sm">{{ error }}</p>

      <div class="actions">
        <button type="button" class="label-md btn btn-secondary" @click="emit('close')">Cancel</button>
        <button type="submit" class="label-md btn btn-primary" :disabled="loading">
          {{ loading ? 'Creating…' : 'Create' }}
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

.error {
  color: var(--color-error);
}

.actions {
  display: flex;
  justify-content: flex-end;
  gap: var(--space-3);
}
</style>
