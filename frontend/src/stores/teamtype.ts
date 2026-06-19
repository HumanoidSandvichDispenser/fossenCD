import { defineStore } from 'pinia';
import { ref } from 'vue';
import init, {
  TeamtypeClient,
  type DisconnectReason,
  type EphemeralMessage,
  type NodeInfo,
  type Range,
  type WireDelta,
} from '@sandvichxyz/teamtype-wasm';
import wasmUrl from '@sandvichxyz/teamtype-wasm/teamtype_wasm_bg.wasm?url';

type ContentListener = (file: string, text: string) => void;
type EditListener = (file: string, delta: WireDelta) => void;
type EphemeralListener = (message: EphemeralMessage) => void;
type RemovedListener = (cursorId: string) => void;

export const useTeamtypeStore = defineStore('teamtype', () => {
  const ready = ref(false);
  const nodeInfo = ref<NodeInfo | null>(null);
  const peers = ref<string[]>([]);
  const files = ref<string[]>([]);
  const currentFile = ref<string | null>(null);
  const logs = ref<string[]>([]);
  const lastDisconnect = ref<DisconnectReason | null>(null);

  const contentListeners = new Set<ContentListener>();
  const editListeners = new Set<EditListener>();
  const ephemeralListeners = new Set<EphemeralListener>();
  const removedListeners = new Set<RemovedListener>();

  let client: TeamtypeClient | null = null;

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
      files.value = f;
    });
    c.onFileContent((file: string, text: string) => {
      for (const listener of contentListeners) {
        listener(file, text);
      }
    });
    c.onEdit((file: string, delta: WireDelta) => {
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

  function selectFile(file: string) {
    currentFile.value = file;
    client?.selectFile(file);
  }

  function applyEdit(file: string, delta: WireDelta) {
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
    logs,
    lastDisconnect,
    start,
    reset,
    connect,
    connectByAddress,
    setName,
    selectFile,
    applyEdit,
    setCursor,
    onContent: (listener: ContentListener) => subscribe(contentListeners, listener),
    onEdit: (listener: EditListener) => subscribe(editListeners, listener),
    onEphemeral: (listener: EphemeralListener) => subscribe(ephemeralListeners, listener),
    onEphemeralRemoved: (listener: RemovedListener) => subscribe(removedListeners, listener),
  };
});
