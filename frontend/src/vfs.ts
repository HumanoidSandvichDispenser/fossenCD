import { reactive } from 'vue';

/**
 * Where a change originated. `local` changes are ours (the user typing, or a
 * file we created here); `remote` changes arrived from a peer. Watchers use
 * this to avoid echoing remote changes back to peers, and the compiler uses it
 * to decide whether to recompile eagerly (local) or debounced (remote).
 */
export type ChangeOrigin = 'local' | 'remote';

/**
 * - `create`: a path that didn't exist now does.
 * - `write`: an existing path's text changed.
 * - `delete`: a path was removed.
 */
export type ChangeKind = 'create' | 'write' | 'delete';

export interface VfsChange {
  path: string;
  kind: ChangeKind;
  origin: ChangeOrigin;
}

export type VfsListener = (change: VfsChange) => void;

/**
 * An in-memory project filesystem that tracks file existence and text content,
 * and notifies watchers of changes.
 *
 * A file's value is its text, or `undefined` for a path that is registered but
 * whose content hasn't streamed in yet (or edge case: the content is binary).
 */
export class VirtualFs {
  // reactive so Vue can derive the file list and content from it directly
  private readonly entries = reactive(new Map<string, string | undefined>());
  private readonly listeners = new Set<VfsListener>();

  /**
   * A file's current text, or `undefined` if registered without content.
   */
  read(path: string): string | undefined {
    return this.entries.get(path);
  }

  /**
   * Whether a path exists in the VFS, even if its content hasn't arrived yet.
   */
  has(path: string): boolean {
    return this.entries.has(path);
  }

  /**
   * Every registered path sorted alphabetically.
   */
  list(): string[] {
    return [...this.entries.keys()].sort();
  }

  /**
   * Note a path's existence without (yet) giving it content. Used when a peer
   * announces a file via the files list before its content has streamed in.
   */
  register(path: string, origin: ChangeOrigin): void {
    if (this.entries.has(path)) {
      return;
    }

    this.entries.set(path, undefined);
    this.emit({ path, kind: 'create', origin });
  }

  /**
   * Set a file's text, registering it first if it's new.
   */
  write(path: string, content: string, origin: ChangeOrigin): void {
    const existed = this.entries.has(path);
    if (existed && this.entries.get(path) === content) {
      return;
    }

    this.entries.set(path, content);
    this.emit({ path, kind: existed ? 'write' : 'create', origin });
  }

  /**
   * Create a brand-new file with content. No-op if the path already exists, so
   * it can't clobber a file that arrived from a peer first.
   */
  create(path: string, content: string, origin: ChangeOrigin): void {
    if (this.entries.has(path)) {
      return;
    }

    this.entries.set(path, content);
    this.emit({ path, kind: 'create', origin });
  }

  delete(path: string, origin: ChangeOrigin): void {
    if (!this.entries.has(path)) {
      return;
    }

    this.entries.delete(path);
    this.emit({ path, kind: 'delete', origin });
  }

  /**
   * Drop every entry, e.g. when switching projects. Deliberately silent — the
   * watchers tear down with the project rather than reacting per file.
   */
  clear(): void {
    this.entries.clear();
  }

  /**
   * Subscribe to changes. Returns an unsubscribe function. Listeners are called
   * synchronously, so they can react to a change before the next one arrives
   * (e.g. to debounce a burst of remote changes into one compile).
   */
  subscribe(listener: VfsListener): () => void {
    this.listeners.add(listener);

    return () => {
      this.listeners.delete(listener);
    };
  }

  private emit(change: VfsChange): void {
    for (const listener of this.listeners) {
      listener(change);
    }
  }
}
