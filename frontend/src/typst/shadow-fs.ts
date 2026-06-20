// The compiler's shadow filesystem: the in-memory mirror of the project's files
// that the Typst compiler resolves imports/includes against. Owns the `$typst`
// shadow-FS mutations and a cache that skips redundant re-syncs.

import { $typst } from '@myriaddreamin/typst.ts';
import { configure, vfsPath } from './compiler';

/**
 * Last synced content for each file. Used to avoid unnecessary re-syncs.
 */
const synced = new Map<string, string>();

/**
 * Make a file's current text available to the compiler. Re-syncing unchanged
 * content is a no-op.
 */
export async function syncSource(file: string, content: string): Promise<void> {
  configure();
  const path = vfsPath(file);
  if (synced.get(path) === content) {
    return;
  }
  synced.set(path, content);
  await $typst.addSource(path, content);
}

/** Remove a file from the shadow FS. */
export async function dropSource(file: string): Promise<void> {
  configure();
  const path = vfsPath(file);
  if (!synced.has(path)) {
    return;
  }
  synced.delete(path);
  await $typst.unmapShadow(path);
}

/** Clear the entire shadow FS. */
export async function resetSources(): Promise<void> {
  configure();
  synced.clear();
  await $typst.resetShadow();
}
