import { defineStore } from 'pinia';
import { computed, markRaw, ref } from 'vue';
import initOkayeg, {
  OkayegClient,
  type WireDelta,
} from '@sandvichxyz/okayeg-wasm';
import wasmUrl from '@sandvichxyz/okayeg-wasm/okayeg_wasm_bg.wasm?url';

import { applyWireDelta } from '@/teamtype/collab';
import { VirtualFs } from '@/vfs';

type ContentListener = (file: string, text: string) => void;

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

  const contentListeners = new Set<ContentListener>();

  let client: OkayegClient | null = null;

  async function start() {
    if (client) {
      return;
    }
    await initOkayeg({ module_or_path: wasmUrl });
    const c = new OkayegClient();
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
    start,
    reset,
    connect,
    selectFile,
    setPreviewFile,
    read,
    applyEdit,
    createFile,
    deleteFile,
    onContent: (listener: ContentListener) => subscribe(contentListeners, listener),
  };
});
