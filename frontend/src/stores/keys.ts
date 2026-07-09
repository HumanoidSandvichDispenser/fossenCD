import { ref } from 'vue';
import { defineStore } from 'pinia';

import { listKeys, addKey, removeKey, renameKey } from '@/client/sdk.gen';
import type { KeyView } from '@/client/types.gen';
import { useClientStore } from './client';

/**
 * Pull a human-readable message out of a huma ErrorModel response.
 */
function errorMessage(error: unknown, fallback: string): string {
  if (error && typeof error === 'object') {
    const e = error as { detail?: string; title?: string };
    return e.detail ?? e.title ?? fallback;
  }
  return fallback;
}

/**
 * The caller's account-scoped okayeg peer keys: the public keys their browser
 * and native peers upload so they may sync every project they belong to. A key
 * authorizes its owner project-wide, so registration and revocation live here on
 * the account rather than per project.
 */
export const useKeysStore = defineStore('keys', () => {
  const clientStore = useClientStore();
  const client = clientStore.client;

  const keys = ref<KeyView[]>([]);
  const loaded = ref(false);

  /** Fetch the caller's registered keys. */
  async function list(): Promise<KeyView[]> {
    const { data, error } = await listKeys({ client });
    if (error || !data) {
      throw new Error(errorMessage(error, 'Could not load keys'));
    }
    keys.value = data;
    loaded.value = true;
    return keys.value;
  }

  /**
   * Register an endpoint (this browser's public key) on the account. Idempotent:
   * the backend ignores a repeat, so it is safe to call on every connect.
   */
  async function ensure(endpointId: string, label = ''): Promise<void> {
    const { error } = await addKey({ client, body: { endpointId, label } });
    if (error) {
      throw new Error(errorMessage(error, 'Could not register this device'));
    }
  }

  /** Revoke a key, closing its live sessions across every project at once. */
  async function remove(endpointId: string): Promise<void> {
    const { error } = await removeKey({ client, path: { endpointId } });
    if (error) {
      throw new Error(errorMessage(error, 'Could not revoke key'));
    }
    keys.value = keys.value.filter((k) => k.endpoint_id !== endpointId);
  }

  /** Rename a key. The endpoint id stays fixed. */
  async function rename(endpointId: string, newLabel: string): Promise<void> {
    const { error } = await renameKey({
      client,
      path: { endpointId },
      body: { newLabel },
    });
    if (error) {
      throw new Error(errorMessage(error, 'Could not rename key'));
    }
    keys.value = keys.value.map((k) =>
      k.endpoint_id === endpointId ? { ...k, label: newLabel } : k,
    );
  }

  return { keys, loaded, list, ensure, remove, rename };
});
