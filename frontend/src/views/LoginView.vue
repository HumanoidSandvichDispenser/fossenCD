<script setup lang="ts">
import { ref, computed } from 'vue';
import { useRouter, useRoute } from 'vue-router';

import { useAuthStore } from '@/stores/auth';

const auth = useAuthStore();
const router = useRouter();
const route = useRoute();

const mode = ref<'login' | 'register'>('login');
const isRegister = computed(() => mode.value === 'register');

const username = ref('');
const email = ref('');
const password = ref('');
const error = ref<string | null>(null);
const loading = ref(false);

function toggleMode() {
  mode.value = isRegister.value ? 'login' : 'register';
  error.value = null;
}

async function submit() {
  error.value = null;
  loading.value = true;
  try {
    if (isRegister.value) {
      await auth.signUp(username.value, email.value, password.value);
    } else {
      await auth.logIn(username.value, password.value);
    }
    const redirect = typeof route.query.redirect === 'string' ? route.query.redirect : '/projects';
    await router.push(redirect);
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Something went wrong';
  } finally {
    loading.value = false;
  }
}
</script>

<template>
  <main class="login">
    <form class="card" @submit.prevent="submit">
      <h1 class="serif-display">fossenCD</h1>
      <p class="subtitle text-sm">
        {{ isRegister ? 'Create an account to start collaborating.' : 'Sign in to your account.' }}
      </p>

      <label class="field">
        <span class="label-sm">Username</span>
        <input v-model="username" autocomplete="username" required minlength="3" maxlength="32" />
      </label>

      <label v-if="isRegister" class="field">
        <span class="label-sm">Email</span>
        <input v-model="email" type="email" autocomplete="email" required />
      </label>

      <label class="field">
        <span class="label-sm">Password</span>
        <input
          v-model="password"
          type="password"
          :autocomplete="isRegister ? 'new-password' : 'current-password'"
          required
          minlength="8"
          maxlength="128"
        />
      </label>

      <p v-if="error" class="error text-sm">{{ error }}</p>

      <button class="submit label-md" type="submit" :disabled="loading">
        {{ loading ? 'Please wait…' : isRegister ? 'Create account' : 'Sign in' }}
      </button>

      <p class="toggle text-sm">
        {{ isRegister ? 'Already have an account?' : "Don't have an account?" }}
        <a href="#" @click.prevent="toggleMode">
          {{ isRegister ? 'Sign in' : 'Create one' }}
        </a>
      </p>
    </form>
  </main>
</template>

<style scoped>
.login {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 100vh;
  padding: var(--space-4);
  background: var(--color-bg-page);
}

.card {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
  width: 100%;
  max-width: var(--width-form);
  padding: var(--space-8);
  background: var(--color-bg-card);
  border: var(--border-thin) solid var(--color-border);
  border-radius: var(--radius-xl);
  box-shadow: var(--shadow-md);
  animation: slide-up var(--duration-slow) var(--ease-out);
}

.subtitle {
  margin-top: calc(-1 * var(--space-2));
  color: var(--color-text-secondary);
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
  transition: border-color var(--duration-fast) var(--ease-out);
}

.field input:focus {
  outline: none;
  border-color: var(--color-primary);
}

.error {
  color: var(--color-error);
}

.submit {
  padding: var(--space-3);
  color: var(--color-text-inverse);
  background: var(--color-primary);
  border: none;
  border-radius: var(--radius-md);
  cursor: pointer;
  transition: background var(--duration-fast) var(--ease-out);
}

.submit:hover:not(:disabled) {
  background: var(--color-accent-700);
}

.submit:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.toggle {
  text-align: center;
  color: var(--color-text-secondary);
}
</style>
