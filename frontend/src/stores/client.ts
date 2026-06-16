import { defineStore } from 'pinia';

import { client } from '@/client/client.gen';

/**
 * Owns the generated hey-api client so other stores share a single configured
 * instance. `call` dedupes concurrent identical requests by key: a second call
 * with the same key while the first is in flight reuses the same promise.
 */
export const useClientStore = defineStore('client', () => {
  const calls = new Map<string, Promise<unknown>>();

  function call<T>(key: string, apiCall: () => Promise<T>): Promise<T> {
    const existing = calls.get(key) as Promise<T> | undefined;
    if (existing) {
      return existing;
    }
    const promise = apiCall();
    calls.set(key, promise);
    void promise.finally(() => calls.delete(key));
    return promise;
  }

  return { client, call };
});
