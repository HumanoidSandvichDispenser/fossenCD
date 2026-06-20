import { defineStore } from 'pinia';
import { reactive, ref } from 'vue';
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
  const ready = ref(false);
  const nodeInfo = ref<NodeInfo | null>(null);
  const peers = ref<string[]>([]);
  const files = ref<string[]>([]);
  // the file shown in the editor (the one the user is editing)
  const currentFile = ref<string | null>(null);
  // the file rendered in the preview pane (the build target) — independent of
  // which file is being edited, like typst.app's preview selector
  const previewFile = ref<string | null>(null);
  // live text of every opened file, kept current for the whole project so the
  // compiler can resolve cross-file imports/includes. Reactive so the preview
  // recompiles when any file changes; `revision` is a cheap change signal that
  // avoids deep-watching the map.
  const texts = reactive(new Map<string, string>());
  // `revision` bumps on any text change; `localRevision` only on edits made
  // here (our own typing). The preview uses the difference to tell whether a
  // change came from us — compile-when-idle — or from a peer — debounced.
  const revision = ref(0);
  const localRevision = ref(0);
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

  function setText(file: string, text: string, local = false) {
    texts.set(file, text);
    revision.value++;
    if (local) {
      localRevision.value++;
    }
  }

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
      // the peer emits files in an unstable order that reshuffles on edits;
      // sort so the sidebar stays put
      files.value = [...f].sort();
      openAll();
      if (!previewFile.value) {
        previewFile.value = defaultPreviewFile(files.value);
      }
    });
    c.onFileContent((file: string, text: string) => {
      setText(file, text);
      for (const listener of contentListeners) {
        listener(file, text);
      }
    });
    c.onEdit((file: string, delta: WireDelta) => {
      setText(file, applyWireDelta(texts.get(file) ?? '', delta));
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
    files.value = [];
    currentFile.value = null;
    previewFile.value = null;
    texts.clear();
    revision.value++;
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

  function applyEdit(file: string, delta: WireDelta) {
    // keep our own copy of the file's text in step with the edit we're sending,
    // so the compiler sees local typing immediately
    setText(file, applyWireDelta(texts.get(file) ?? '', delta), true);
    client?.applyEdit(file, delta);
  }

  function setCursor(file: string, ranges: Range[]) {
    client?.setCursor(file, ranges);
  }

  return {
    ready,
    nodeInfo,
    peers,
    files,
    currentFile,
    previewFile,
    texts,
    revision,
    localRevision,
    logs,
    lastDisconnect,
    start,
    reset,
    connect,
    connectByAddress,
    setName,
    selectFile,
    setPreviewFile,
    applyEdit,
    setCursor,
    onContent: (listener: ContentListener) => subscribe(contentListeners, listener),
    onEdit: (listener: EditListener) => subscribe(editListeners, listener),
    onEphemeral: (listener: EphemeralListener) => subscribe(ephemeralListeners, listener),
    onEphemeralRemoved: (listener: RemovedListener) => subscribe(removedListeners, listener),
  };
});
