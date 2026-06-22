// The compiler's shadow filesystem: the in-memory mirror of the project's files
// that the Typst compiler resolves imports/includes against. Owns the `$typst`
// shadow-FS mutations and a cache that skips redundant re-syncs.

import { $typst } from '@myriaddreamin/typst.ts';
import { configure, vfsPath } from './compiler';
import type { VirtualFs } from '@/vfs';

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

/**
 * Sync every file in `vfs` into the shadow FS. Files registered without content
 * yet are skipped; unchanged files are no-ops (see {@link syncSource}).
 */
export async function syncAll(vfs: VirtualFs): Promise<void> {
  for (const file of vfs.list()) {
    const text = vfs.read(file);
    if (text !== undefined) {
      await syncSource(file, text);
    }
  }
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
