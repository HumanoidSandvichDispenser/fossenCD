/**
 * The browser peer's okayeg identity seed, persisted in IndexedDB.
 *
 * okayeg authorizes a peer by its ed public key (an EndpointId), so this browser
 * needs a stable secret of its own. We keep the raw 32-byte seed here and hand
 * it to okayeg-wasm's `OkayegClient.withSeed`, which derives the keypair. Only
 * the public half is ever uploaded to the server (see the keys store); the seed
 * never leaves the browser. Clearing site data mints a new identity on the next
 * load, leaving the old key as a stale entry the user prunes in settings.
 */

const DB_NAME = 'okayeg';
const STORE = 'identity';
const SEED_KEY = 'seed';

function open(): Promise<IDBDatabase> {
  return new Promise((resolve, reject) => {
    const req = indexedDB.open(DB_NAME, 1);
    req.onupgradeneeded = () => req.result.createObjectStore(STORE);
    req.onsuccess = () => resolve(req.result);
    req.onerror = () => reject(req.error);
  });
}

function tx<T>(mode: IDBTransactionMode, run: (s: IDBObjectStore) => IDBRequest<T>): Promise<T> {
  return open().then(
    (db) =>
      new Promise<T>((resolve, reject) => {
        const req = run(db.transaction(STORE, mode).objectStore(STORE));
        req.onsuccess = () => resolve(req.result);
        req.onerror = () => reject(req.error);
      }),
  );
}

async function loadSeed(): Promise<Uint8Array | null> {
  const raw = await tx<unknown>('readonly', (s) => s.get(SEED_KEY));
  return raw instanceof Uint8Array ? raw : null;
}

/**
 * Return the browser's identity seed, minting and persisting a fresh 32-byte one
 * on first use. `fresh` is true when a new seed was just created, so the caller
 * knows to upload its public half.
 */
export async function loadOrCreateSeed(): Promise<{ seed: Uint8Array; fresh: boolean }> {
  const existing = await loadSeed();
  if (existing) {
    return { seed: existing, fresh: false };
  }
  const seed = crypto.getRandomValues(new Uint8Array(32));
  await tx('readwrite', (s) => s.put(seed, SEED_KEY));
  return { seed, fresh: true };
}
