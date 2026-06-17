<script setup lang="ts">
import { ref, watch } from 'vue';

const props = defineProps<{ open: boolean }>();
const emit = defineEmits<{ close: [] }>();

const dialog = ref<HTMLDialogElement | null>(null);

// Drive the native modal from the `open` prop. The `@close` handler covers the
// Esc key and backdrop dismissal so the parent's state stays in sync.
watch(
  () => props.open,
  (open) => {
    if (open) {
      dialog.value?.showModal();
    } else {
      dialog.value?.close();
    }
  },
);
</script>

<template>
  <dialog ref="dialog" class="dialog" @close="emit('close')">
    <div class="body">
      <slot />
    </div>
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
  background: var(--color-overlay);
}

.body {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
  padding: var(--space-6);
}
</style>
