<script setup lang="ts">
import { useCompileStatus } from '@/composables/useCompileStatus';

const props = defineProps<{
  status: ReturnType<typeof useCompileStatus>;
}>();
</script>

<template>
  <span v-if="props.status.label.value" class="compile-status label-sm">
    {{ props.status.label.value }}
  </span>
</template>

<style scoped>
.compile-status {
  color: var(--color-text-secondary);
  white-space: nowrap;
}
</style>
