<script setup lang="ts">
withDefaults(
  defineProps<{
    variant?: 'neutral' | 'primary' | 'success' | 'error';
  }>(),
  { variant: 'neutral' },
);
</script>

<template>
  <span class="chip label-sm" :class="`is-${variant}`"><slot /></span>
</template>

<style scoped>
.chip {
  display: inline-flex;
  align-items: center;
  gap: var(--space-1);
  padding: var(--space-1) var(--space-2);
  border-radius: var(--radius-full);
}

.is-neutral {
  color: var(--color-text-secondary);
  background: var(--color-surface);
}

.is-primary {
  color: var(--color-primary);
  background: var(--color-primary-light);
}

.is-success {
  color: var(--color-success);
  background: var(--color-success-light);
}

.is-error {
  color: var(--color-error);
  background: var(--color-error-light);
}
</style>
