import { defineStore } from 'pinia';
import { computed, markRaw, ref } from 'vue';
import initOkayeg, {
  OkayegClient,
  type WireDelta,
} from '@sandvichxyz/okayeg-wasm';
import wasmUrl from '@sandvichxyz/okayeg-wasm/wasm-url';

import { applyWireDelta } from '@/teamtype/collab';
import { loadOrCreateSeed, regenerateSeed } from '@/identity';
import { VirtualFs } from '@/vfs';

type ContentListener = (file: string, text: string) => void;

/**
 * A comment as reflected by okayeg-wasm's onComments: anchored to a file span
 * (`range` in Unicode code points) or threaded under a parent comment, with
 * all consumer-owned keys under `fields`.
 */
export interface OkayegComment {
  id: string;
  file: string | null;
  parent: string | null;
  createdAt: number;
  range: [number, number] | null;
  orphaned: boolean;
  fields: Record<string, unknown>;
}

/** The comment field conventions fossenCD writes. */
export interface CommentFields {
  body: string;
  author: string;
  /** The anchored text as it read when the comment was created. */
  quote?: string;
}

function isTypst(file: string): boolean {
  return file.toLowerCase().endsWith('.typ');
}

// Prefer `main.typ` as the default preview target, then any other .typ file.
function defaultPreviewFile(files: string[]): string | null {
  return (
    files.find((f) => f.toLowerCase() === 'main.typ') ?? files.find(isTypst) ?? null
  );
}

export const useOkayegStore = defineStore('okayeg', () => {
  const vfs = markRaw(new VirtualFs());

  const ready = ref(false);
  const connected = ref(false);
  const nodeId = ref<string | null>(null);
  // the project's files live in the VFS (the single source of truth); the file
  // list is just its registered paths
  const files = computed(() => vfs.list());
  // the file shown in the editor (the one the user is editing)
  const currentFile = ref<string | null>(null);
  // the file rendered in the preview pane (the build target)
  const previewFile = ref<string | null>(null);
  const logs = ref<string[]>([]);
  const lastDisconnect = ref<string | null>(null);
  // all comments in the project, re-reflected (with anchors re-resolved) on
  // every change tick
  const comments = ref<OkayegComment[]>([]);
  // the thread highlighted in both the editor and the comments panel
  const activeCommentId = ref<string | null>(null);

  const contentListeners = new Set<ContentListener>();

  let client: OkayegClient | null = null;

  async function start() {
    if (client) {
      return;
    }
    await initOkayeg({ module_or_path: wasmUrl });
    // The identity seed lives in IndexedDB, so this browser keeps the same peer
    // key across sessions; okayeg-wasm derives the keypair from it. Only the
    // public half (nodeId) is ever uploaded, when we register the key.
    const { seed } = await loadOrCreateSeed();
    const c = OkayegClient.withSeed(seed);
    client = c;

    c.onLog((message: string) => {
      logs.value.push(message);
      if (logs.value.length > 200) {
        logs.value.shift();
      }
    });
    c.onFiles((f: string[]) => {
      // reconcile the VFS registry against the peer's authoritative list:
      // drop files that disappeared, register ones we haven't seen yet. Their
      // content arrives in the same change tick via onFileContent.
      const incoming = new Set(f);
      for (const path of vfs.list()) {
        if (!incoming.has(path)) {
          vfs.delete(path, 'remote');
        }
      }
      for (const path of f) {
        vfs.register(path, 'remote');
      }
      if (!previewFile.value) {
        previewFile.value = defaultPreviewFile(vfs.list());
      }
    });
    c.onFileContent((file: string, text: string) => {
      // The wasm fires this for every file on every change tick, including
      // ticks caused by our own applyEdit. The VFS write is a no-op when the
      // text is unchanged, so echoes don't churn the registry; the editor's
      // onContent listener additionally guards against reseeding when its doc
      // already matches.
      vfs.write(file, text, 'remote');
      for (const listener of contentListeners) {
        listener(file, text);
      }
    });
    c.onComments((all: OkayegComment[]) => {
      comments.value = all;
    });
    c.onDisconnect((reason: string) => {
      connected.value = false;
      lastDisconnect.value = reason;
    });

    nodeId.value = c.nodeId();
    ready.value = true;
  }

  /**
   * Dial the project's host endpoint and begin live sync. Resolves once the
   * bidi stream is open and the sync task is running; rejects with the wasm's
   * error message on failure. On a clean or error session end, onDisconnect
   * fires and clears `connected`.
   */
  async function connect(endpointId: string) {
    if (!client) {
      return;
    }
    lastDisconnect.value = null;
    await client.connect(endpointId);
    connected.value = true;
  }

  /**
   * Tear down the current peer and clear all per-project state so the store can
   * call `start()` again to create a new peer.
   */
  function reset() {
    client?.free();
    client = null;
    ready.value = false;
    connected.value = false;
    nodeId.value = null;
    currentFile.value = null;
    previewFile.value = null;
    vfs.clear();
    lastDisconnect.value = null;
    logs.value = [];
    comments.value = [];
    activeCommentId.value = null;
  }

  function subscribe<T>(set: Set<T>, listener: T): () => void {
    set.add(listener);
    return () => {
      set.delete(listener);
    };
  }

  // Choose which file the editor shows. Purely local UI state.
  function selectFile(file: string) {
    currentFile.value = file;
  }

  // Choose which file the preview pane renders (the build target).
  function setPreviewFile(file: string) {
    previewFile.value = file;
  }

  /**
   * Returns the text of a file, or undefined if it has no content yet. Used by the
   * editor to seed CodeMirror when switching files.
   */
  function read(file: string): string | undefined {
    return vfs.read(file);
  }

  function applyEdit(file: string, delta: WireDelta) {
    // keep the VFS copy in step with the edit we're sending, so the compiler
    // (which watches the VFS) sees local typing immediately. Tagged `local` so
    // the preview recompiles eagerly rather than debounced.
    vfs.write(file, applyWireDelta(vfs.read(file) ?? '', delta), 'local');
    client?.applyEdit(file, delta);
  }

  function createFile(path: string) {
    // reflect locally before the round-trip so the file list updates at once;
    // the wasm's change tick will confirm via onFiles/onFileContent (no-ops
    // against the matching local state)
    vfs.create(path, '', 'local');
    client?.createFile(path);
  }

  function deleteFile(path: string) {
    vfs.delete(path, 'local');
    client?.deleteFile(path);
  }

  /**
   * Anchor a comment to `[start, end)` (Unicode code points) in a file.
   * Returns the new comment's id, or null if the file or range is invalid.
   */
  function addComment(
    file: string,
    start: number,
    end: number,
    fields: CommentFields,
  ): string | null {
    return client?.addComment(file, start, end, fields) ?? null;
  }

  /** Reply under a comment. Returns the reply's id, or null. */
  function replyComment(parent: string, fields: CommentFields): string | null {
    return client?.replyComment(parent, fields) ?? null;
  }

  /** Set one consumer field (a scalar) on a comment. */
  function setComment(id: string, key: string, value: unknown): boolean {
    return client?.setComment(id, key, value) ?? false;
  }

  /** Remove a comment and every reply threaded under it. */
  function removeComment(id: string) {
    if (!client) {
      return;
    }
    for (const reply of comments.value.filter((c) => c.parent === id)) {
      removeComment(reply.id);
    }
    client.removeComment(id);
    if (activeCommentId.value === id) {
      activeCommentId.value = null;
    }
  }

  /** Highlight a thread in both the editor and the comments panel. */
  function setActiveComment(id: string | null) {
    activeCommentId.value = id;
  }

  /**
   * This browser's peer id (hex EndpointId), derived from the persisted seed
   * without touching the live client. Returns the running peer's id when one is
   * up; otherwise spins a throwaway client just to read it and frees it again.
   * Used by settings to mark and guard the current device in the key list.
   */
  async function deviceId(): Promise<string> {
    if (nodeId.value) {
      return nodeId.value;
    }
    await initOkayeg({ module_or_path: wasmUrl });
    const { seed } = await loadOrCreateSeed();
    const c = OkayegClient.withSeed(seed);

    const id = c.nodeId();
    c.free();

    return id;
  }

  /**
   * Mint a fresh identity for this browser: tear down any live peer, replace the
   * persisted seed, and return the new endpoint id. Used to escape a key already
   * claimed by another account.
   */
  async function regenerateIdentity(): Promise<string> {
    reset();
    await regenerateSeed();
    return deviceId();
  }

  return {
    vfs,
    ready,
    connected,
    nodeId,
    files,
    currentFile,
    previewFile,
    logs,
    lastDisconnect,
    comments,
    activeCommentId,
    start,
    reset,
    connect,
    deviceId,
    regenerateIdentity,
    selectFile,
    setPreviewFile,
    read,
    applyEdit,
    createFile,
    deleteFile,
    addComment,
    replyComment,
    setComment,
    removeComment,
    setActiveComment,
    onContent: (listener: ContentListener) => subscribe(contentListeners, listener),
  };
});
