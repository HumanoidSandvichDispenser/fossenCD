<script setup lang="ts">
import { PhSealCheck } from '@phosphor-icons/vue';

import BaseChip from '@/components/BaseChip.vue';

defineProps<{
  // whether the claimed email resolved to a verified account address
  verified: boolean;
  // render a worded chip instead of a bare icon; an icon-only badge is hidden
  // entirely when unverified, since an absent seal already reads as "no claim"
  label?: boolean;
}>();
</script>

<template>
  <BaseChip v-if="label" :variant="verified ? 'success' : 'neutral'">
    {{ verified ? 'Verified' : 'Unverified' }}
  </BaseChip>
  <PhSealCheck
    v-else-if="verified"
    class="seal"
    size="16"
    weight="fill"
    aria-label="Verified"
  />
</template>

<style scoped>
.seal {
  color: var(--color-primary);
}
</style>
