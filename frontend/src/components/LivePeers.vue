<script setup lang="ts">
import { onBeforeUnmount, ref, watch } from 'vue';

import { projectPeers } from '@/client/sdk.gen';
import type { PeerView } from '@/client/types.gen';
import { useClientStore } from '@/stores/client';
import { PhSealCheck, PhSealQuestion } from '@phosphor-icons/vue';

const props = defineProps<{
  projectId: string;
  // this browser's endpoint id, filtered out of the list
  selfId: string | null;
}>();

const clientStore = useClientStore();
const client = clientStore.client;

const peers = ref<PeerView[]>([]);

// TODO: polling is a temporary solution for now until we have a proper system
// for ephemeral presence and updates (Loro supports a separate ephemeral channel
// that doesn't hold history making this nearly trivial to implement) this also
// unlocks cursors for us. maybe even a chat too?
// also that basically eliminates the need for a hello packet. oh well.
const POLL_MS = 10_000;
let timer: ReturnType<typeof setInterval> | null = null;

async function poll() {
  const { data, error } = await projectPeers({ client, path: { id: props.projectId } });
  if (error) {
    // presence is decorative; a failed poll keeps the last list and retries
    return;
  }
  peers.value = (data ?? []).filter((p) => p.endpoint_id !== props.selfId);
}

watch(
  () => props.projectId,
  () => {
    peers.value = [];
    if (timer !== null) {
      clearInterval(timer);
    }
    void poll();
    timer = setInterval(poll, POLL_MS);
  },
  { immediate: true },
);

onBeforeUnmount(() => {
  if (timer !== null) {
    clearInterval(timer);
  }
});

function display(peer: PeerView): string {
  return peer.name || peer.email || `${peer.endpoint_id.slice(0, 8)}...`;
}

function title(peer: PeerView): string {
  const claim = [peer.name, peer.email && `<${peer.email}>`].filter(Boolean).join(' ');
  const line = claim ? `${claim} (${peer.endpoint_id})` : peer.endpoint_id;
  if (peer.verified) {
    return `${line}\nVerified as ${peer.username}`;
  }
  return claim ? `${line}\nUnverified claim` : line;
}
</script>

<template>
  <span
    v-for="peer in peers"
    :key="peer.endpoint_id"
    class="peer chip label-sm"
    :title="title(peer)"
  >
    {{ display(peer) }}
    <!--span v-if="peer.verified" class="badge" aria-label="Verified">&check;</span-->
    <PhSealCheck
      v-if="peer.verified"
      class="badge"
      size="16"
      weight="fill"
      aria-label="Verified"
    />
  </span>
</template>

<style scoped>
.peer {
  display: inline-flex;
  align-items: center;
  gap: var(--space-1);
  padding: var(--space-1) var(--space-2);
  color: var(--color-text-secondary);
  background: var(--color-surface);
  border-radius: var(--radius-full);
  white-space: nowrap;
}

.badge {
  color: var(--color-primary);
  font-weight: 700;
}
</style>
