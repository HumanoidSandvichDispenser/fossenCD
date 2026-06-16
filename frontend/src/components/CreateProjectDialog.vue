<script setup lang="ts">
import { ref, watch } from 'vue';

import { useProjectsStore } from '@/stores/projects';
import type { ProjectView } from '@/client/types.gen';

const props = defineProps<{ open: boolean }>();
const emit = defineEmits<{ close: []; created: [project: ProjectView] }>();

const projects = useProjectsStore();

const dialog = ref<HTMLDialogElement | null>(null);
const name = ref('');
const error = ref<string | null>(null);
const loading = ref(false);

// Drive the native dialog from the `open` prop and reset state on each open.
watch(
  () => props.open,
  (open) => {
    if (open) {
      name.value = '';
      error.value = null;
      dialog.value?.showModal();
    } else {
      dialog.value?.close();
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
  <dialog ref="dialog" class="dialog" @close="emit('close')">
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
        <button type="button" class="label-md secondary" @click="emit('close')">Cancel</button>
        <button type="submit" class="label-md primary" :disabled="loading">
          {{ loading ? 'Creating…' : 'Create' }}
        </button>
      </div>
    </form>
  </dialog>
</template>

<style scoped>
.dialog {
  position: fixed;
  inset: 0;
  margin: auto;
  width: 100%;
  max-width: var(--width-form);
  height: fit-content;
  padding: 0;
  border: var(--border-thin) solid var(--color-border);
  border-radius: var(--radius-xl);
  box-shadow: var(--shadow-lg);
  background: var(--color-bg-card);
}

.dialog::backdrop {
  background: rgba(15, 23, 42, 0.35);
}

.form {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
  padding: var(--space-6);
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

.actions button {
  padding: var(--space-2) var(--space-4);
  border-radius: var(--radius-md);
  cursor: pointer;
}

.primary {
  color: var(--color-text-inverse);
  background: var(--color-primary);
  border: none;
}

.primary:hover:not(:disabled) {
  background: var(--color-accent-700);
}

.primary:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.secondary {
  color: var(--color-text);
  background: var(--color-surface);
  border: var(--border-thin) solid var(--color-border);
}

.secondary:hover {
  background: var(--color-surface-hover);
}
</style>
