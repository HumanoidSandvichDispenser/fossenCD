<script setup lang="ts">
import { computed, ref, watch } from 'vue';

import { verifyPeers } from '@/client/sdk.gen';
import type { PeerVerdict } from '@/client/types.gen';
import { useClientStore } from '@/stores/client';
import { useOkayegStore } from '@/stores/okayeg';
import BaseChip from '@/components/BaseChip.vue';
import VerifiedBadge from '@/components/VerifiedBadge.vue';

const props = defineProps<{
  projectId: string;
  // this browser's endpoint id, filtered out of the list
  selfId: string | null;
}>();

const clientStore = useClientStore();
const client = clientStore.client;
const okayeg = useOkayegStore();

interface Peer {
  id: string;
  name: string | null;
  email: string | null;
}

// the live list comes straight from the presence store: entries appear on
// join and expire on leave, no polling
const peers = computed<Peer[]>(() =>
  Object.entries(okayeg.peers)
    .filter(([id]) => id !== props.selfId)
    .map(([id, entry]) => ({
      id,
      name: typeof entry.name === 'string' ? entry.name : null,
      email: typeof entry.email === 'string' ? entry.email : null,
    })),
);

// presence carries only the claim; whether the claimed email resolves to a
// verified account comes from the backend, re-checked when the claims change
const views = ref(new Map<string, PeerVerdict>());

watch(
  () =>
    peers.value
      .map((p) => `${p.id} ${p.email ?? ''}`)
      .sort()
      .join(','),
  async () => {
    const claims = peers.value.map((p) => ({ endpoint_id: p.id, email: p.email ?? '' }));
    if (claims.length === 0) {
      views.value = new Map();
      return;
    }
    const { data, error } = await verifyPeers({
      client,
      path: { id: props.projectId },
      body: { claims },
    });
    if (error) {
      // verification is decorative; a failed check keeps the last answers
      return;
    }
    views.value = new Map((data ?? []).map((v) => [v.endpoint_id, v]));
  },
  { immediate: true },
);

function display(peer: Peer): string {
  return peer.name || peer.email || `${peer.id.slice(0, 8)}...`;
}

function title(peer: Peer): string {
  const claim = [peer.name, peer.email && `<${peer.email}>`].filter(Boolean).join(' ');
  const line = claim ? `${claim} (${peer.id})` : peer.id;
  const view = views.value.get(peer.id);
  if (view?.verified) {
    return `${line}\nVerified as ${view.username}`;
  }
  return claim ? `${line}\nUnverified claim` : line;
}
</script>

<template>
  <BaseChip v-for="peer in peers" :key="peer.id" class="peer" :title="title(peer)">
    {{ display(peer) }}
    <VerifiedBadge :verified="views.get(peer.id)?.verified ?? false" />
  </BaseChip>
</template>

<style scoped>
.peer {
  white-space: nowrap;
}
</style>
