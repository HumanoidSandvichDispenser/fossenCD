import { defineStore } from 'pinia';
import { computed, markRaw, ref } from 'vue';
import init, {
  TeamtypeClient,
  type DisconnectReason,
  type EphemeralMessage,
  type NodeInfo,
  type Range,
  type WireDelta,
} from '@sandvichxyz/teamtype-wasm';
import wasmUrl from '@sandvichxyz/teamtype-wasm/teamtype_wasm_bg.wasm?url';

import { applyWireDelta } from '@/teamtype/collab';
import { VirtualFs } from '@/vfs';

type ContentListener = (file: string, text: string) => void;
type EditListener = (file: string, delta: WireDelta) => void;
type EphemeralListener = (message: EphemeralMessage) => void;
type RemovedListener = (cursorId: string) => void;

function isTypst(file: string): boolean {
  return file.toLowerCase().endsWith('.typ');
}

// Prefer `main.typ` as the default preview target, then any other .typ file.
function defaultPreviewFile(files: string[]): string | null {
  return (
    files.find((f) => f.toLowerCase() === 'main.typ') ?? files.find(isTypst) ?? null
  );
}

export const useTeamtypeStore = defineStore('teamtype', () => {
  const vfs = markRaw(new VirtualFs());

  const ready = ref(false);
  const nodeInfo = ref<NodeInfo | null>(null);
  const peers = ref<string[]>([]);
  // the project's files live in the VFS (the single source of truth); the file
  // list is just its registered paths
  const files = computed(() => vfs.list());
  // the file shown in the editor (the one the user is editing)
  const currentFile = ref<string | null>(null);
  // the file rendered in the preview pane (the build target)
  const previewFile = ref<string | null>(null);
  const logs = ref<string[]>([]);
  const lastDisconnect = ref<DisconnectReason | null>(null);

  const contentListeners = new Set<ContentListener>();
  const editListeners = new Set<EditListener>();
  const ephemeralListeners = new Set<EphemeralListener>();
  const removedListeners = new Set<RemovedListener>();

  // files we've asked the peer to stream edits for; we open every file so the
  // compiler always sees the whole project
  const opened = new Set<string>();

  let client: TeamtypeClient | null = null;

  // Open every project file we haven't opened yet, so edits stream for all of
  // them (not just the focused one) and the compiler stays whole-project.
  function openAll() {
    if (!client) {
      return;
    }
    for (const file of files.value) {
      if (!opened.has(file)) {
        opened.add(file);
        client.openFile(file);
      }
    }
  }

  async function start() {
    if (client) {
      return;
    }
    await init({ module_or_path: wasmUrl });
    const c = new TeamtypeClient();
    client = c;

    c.onLog((message: string) => {
      logs.value.push(message);
      if (logs.value.length > 200) {
        logs.value.shift();
      }
    });
    c.onPeers((p: string[]) => {
      peers.value = p;
      const info = c.nodeInfo();
      if (info) {
        nodeInfo.value = info;
      }
    });
    c.onFiles((f: string[]) => {
      // reconcile the VFS registry against the peer's authoritative list:
      // drop files that disappeared, register ones we haven't seen yet (their
      // content streams in via onFileContent once opened)
      const incoming = new Set(f);
      for (const path of vfs.list()) {
        if (!incoming.has(path)) {
          vfs.delete(path, 'remote');
        }
      }
      for (const path of f) {
        vfs.register(path, 'remote');
      }
      openAll();
      if (!previewFile.value) {
        previewFile.value = defaultPreviewFile(vfs.list());
      }
    });
    c.onFileContent((file: string, text: string) => {
      vfs.write(file, text, 'remote');
      for (const listener of contentListeners) {
        listener(file, text);
      }
    });
    c.onEdit((file: string, delta: WireDelta) => {
      vfs.write(file, applyWireDelta(vfs.read(file) ?? '', delta), 'remote');
      for (const listener of editListeners) {
        listener(file, delta);
      }
    });
    c.onEphemeral((message: EphemeralMessage) => {
      for (const listener of ephemeralListeners) {
        listener(message);
      }
    });
    c.onEphemeralRemoved((cursorId: string) => {
      for (const listener of removedListeners) {
        listener(cursorId);
      }
    });
    c.onDisconnect((reason: DisconnectReason) => {
      lastDisconnect.value = reason;
    });

    nodeInfo.value = c.nodeInfo();
    ready.value = true;
  }

  /**
   * Tear down the current peer and clear all per-project state so the store can
   * call `start()` again to create a new peer.
   */
  function reset() {
    client?.free();
    client = null;
    ready.value = false;
    nodeInfo.value = null;
    peers.value = [];
    currentFile.value = null;
    previewFile.value = null;
    vfs.clear();
    opened.clear();
    lastDisconnect.value = null;
    logs.value = [];
  }

  function subscribe<T>(set: Set<T>, listener: T): () => void {
    set.add(listener);
    return () => {
      set.delete(listener);
    };
  }

  function connect(joinCode: string) {
    client?.connectByJoinCode(joinCode);
  }

  function connectByAddress(address: string) {
    client?.connectByAddress(address);
  }

  function setName(name: string) {
    client?.setName(name);
  }

  // Choose which file the editor shows. Every file is already opened for the
  // compiler, so this is purely local UI state.
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

  function setCursor(file: string, ranges: Range[]) {
    client?.setCursor(file, ranges);
  }

  return {
    vfs,
    ready,
    nodeInfo,
    peers,
    files,
    currentFile,
    previewFile,
    logs,
    lastDisconnect,
    start,
    reset,
    connect,
    connectByAddress,
    setName,
    selectFile,
    setPreviewFile,
    read,
    applyEdit,
    setCursor,
    onContent: (listener: ContentListener) => subscribe(contentListeners, listener),
    onEdit: (listener: EditListener) => subscribe(editListeners, listener),
    onEphemeral: (listener: EphemeralListener) => subscribe(ephemeralListeners, listener),
    onEphemeralRemoved: (listener: RemovedListener) => subscribe(removedListeners, listener),
  };
});
