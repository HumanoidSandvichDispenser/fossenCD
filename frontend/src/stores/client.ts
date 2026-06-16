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

  /**
   * Wraps an SDK method into a deduped callable with the same signature: the
   * returned function takes the method's own params and returns its own type
   * (hey-api's `{ data } | { error }` result). By default the dedup key is the
   * method name; pass `key` to fold params in for parameterised endpoints (e.g.
   * `getProject` keyed by id) so different args don't collide.
   *
   * `F` is left unconstrained so hey-api's `RequestResult` (a union of promises)
   * is accepted as-is; the casts bridge it to `call`'s plain `Promise` and back
   * to `ReturnType<F>`, keeping the public signature exact for callers.
   */
  function wrap<F extends (...args: never[]) => unknown>(
    fn: F,
    key: (...args: Parameters<F>) => string = () => fn.name,
  ): (...args: Parameters<F>) => ReturnType<F> {
    return (...args) =>
      call(key(...args), () => fn(...args) as Promise<unknown>) as ReturnType<F>;
  }

  return { client, call, wrap };
});
