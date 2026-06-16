import { ref, computed } from 'vue';
import { defineStore } from 'pinia';

import { login, register, logout, me } from '@/client/sdk.gen';
import type { UserView } from '@/client/types.gen';
import { useClientStore } from './client';

/** Pull a human-readable message out of a huma ErrorModel response. */
function errorMessage(error: unknown, fallback: string): string {
  if (error && typeof error === 'object') {
    const e = error as { detail?: string; title?: string };
    return e.detail ?? e.title ?? fallback;
  }
  return fallback;
}

export const useAuthStore = defineStore('auth', () => {
  const clientStore = useClientStore();
  const client = clientStore.client;

  const user = ref<UserView | null>(null);
  /** Whether the initial `me()` hydration has run, so guards don't bounce early. */
  const hasCheckedAuth = ref(false);

  const isLoggedIn = computed(() => user.value !== null);

  /** Restore the session from the HttpOnly cookie. Deduped so concurrent
   *  guards/components share one request; cached after the first check. */
  async function getUser() {
    if (hasCheckedAuth.value) {
      return user.value;
    }
    const { data } = await clientStore.call('auth.me', () => me({ client }));
    user.value = data ?? null;
    hasCheckedAuth.value = true;
    return user.value;
  }

  async function logIn(username: string, password: string) {
    const { data, error } = await login({ client, body: { username, password } });
    if (error) {
      throw new Error(errorMessage(error, 'Invalid username or password'));
    }
    user.value = data ?? null;
    hasCheckedAuth.value = true;
    return user.value;
  }

  async function signUp(username: string, email: string, password: string) {
    const { data, error } = await register({ client, body: { username, email, password } });
    if (error) {
      throw new Error(errorMessage(error, 'Could not create account'));
    }
    user.value = data ?? null;
    hasCheckedAuth.value = true;
    return user.value;
  }

  async function logOut() {
    const { error } = await logout({ client });
    if (error) {
      throw new Error(errorMessage(error, 'Could not log out'));
    }
    user.value = null;
  }

  return { user, hasCheckedAuth, isLoggedIn, getUser, logIn, signUp, logOut };
});
