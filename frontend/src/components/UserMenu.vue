<script setup lang="ts">
import { ref } from 'vue';
import { useRouter } from 'vue-router';

import { useAuthStore } from '@/stores/auth';

const auth = useAuthStore();
const router = useRouter();

const error = ref<string | null>(null);

async function logOut() {
  error.value = null;
  try {
    await auth.logOut();
    await router.push({ name: 'login' });
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Could not log out';
  }
}
</script>

<template>
  <div class="user-menu">
    <span v-if="auth.user" class="text-sm user">
      {{ auth.user.username }}
    </span>
    <button class="label-sm btn btn-sm btn-secondary" @click="logOut">Log out</button>
    <span v-if="error" class="error text-sm" role="alert">
      {{ error }}
    </span>
  </div>
</template>

<style scoped>
.user-menu {
  display: flex;
  align-items: center;
  gap: var(--space-3);
}

.user {
  color: var(--color-text-secondary);
}

.error {
  color: var(--color-error);
}
</style>
