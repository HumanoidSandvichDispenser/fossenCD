<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';

import { useKeysStore } from '@/stores/keys';
import { useOkayegStore } from '@/stores/okayeg';

const keys = useKeysStore();
const okayeg = useOkayegStore();

const error = ref<string | null>(null);
const revoking = ref<string | null>(null);
// this browser's own endpoint id, so we can mark it and warn before self-revoke
const thisDevice = ref<string | null>(null);

const sorted = computed(() =>
  [...keys.keys].sort((a, b) => {
    // surface this device first, then newest
    if (a.endpoint_id === thisDevice.value) {return -1;}
    if (b.endpoint_id === thisDevice.value) {return 1;}
    return b.created_at.localeCompare(a.created_at);
  }),
);

onMounted(async () => {
  try {
    thisDevice.value = await okayeg.deviceId();
  } catch {
    // deriving the device id is best-effort; the list still works without it
  }
  try {
    await keys.list();
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Could not load keys';
  }
});

function shortId(id: string) {
  return `${id.slice(0, 12)}...${id.slice(-6)}`;
}

function formatDate(iso: string) {
  return new Date(iso).toLocaleDateString(undefined, {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
  });
}

async function revoke(endpointId: string) {
  const isSelf = endpointId === thisDevice.value;
  const warning = isSelf
    ? 'Revoke THIS device? You will lose sync access until you reconnect and register again.'
    : 'Revoke this key? Its live sessions on every project close immediately.';
  if (!window.confirm(warning)) {
    return;
  }
  error.value = null;
  revoking.value = endpointId;
  try {
    await keys.remove(endpointId);
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Could not revoke key';
  } finally {
    revoking.value = null;
  }
}
</script>

<template>
  <main class="settings">
    <header>
      <h1 class="h3">Devices &amp; keys</h1>
    </header>

    <p class="intro text-sm">
      Each browser and native peer holds its own key and syncs every project you
      belong to. Only the public half is stored here; revoke a key to cut off a
      lost or unused device everywhere at once.
    </p>

    <p v-if="error" class="error text-sm">{{ error }}</p>

    <p v-if="!keys.loaded" class="state text-sm">Loading...</p>

    <p v-else-if="keys.keys.length === 0" class="state text-sm">
      No keys registered yet. Open a project to register this browser.
    </p>

    <ul v-else class="list">
      <li v-for="key in sorted" :key="key.endpoint_id" class="card">
        <div class="info">
          <span class="name label-md">
            {{ key.label || 'Unnamed device' }}
            <span v-if="key.endpoint_id === thisDevice" class="chip label-sm">This device</span>
          </span>
          <span class="meta text-xs" :title="key.endpoint_id">{{ shortId(key.endpoint_id) }}</span>
          <span class="meta text-xs">Added {{ formatDate(key.created_at) }}</span>
        </div>
        <button
          class="label-sm btn btn-sm btn-danger"
          :disabled="revoking === key.endpoint_id"
          @click="revoke(key.endpoint_id)"
        >
          {{ revoking === key.endpoint_id ? 'Revoking...' : 'Revoke' }}
        </button>
      </li>
    </ul>
  </main>
</template>

<style scoped>
.settings {
  max-width: var(--width-content);
  margin: 0 auto;
  padding: var(--space-8) var(--space-4);
}

header {
  padding-bottom: var(--space-4);
  border-bottom: var(--border-thin) solid var(--color-border);
}

.intro {
  margin-top: var(--space-4);
  color: var(--color-text-secondary);
}

.error {
  margin-top: var(--space-4);
  color: var(--color-error);
}

.state {
  margin-top: var(--space-8);
  color: var(--color-text-tertiary);
  text-align: center;
}

.list {
  margin-top: var(--space-6);
  list-style: none;
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}

.card {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  padding: var(--space-4);
  background: var(--color-bg-card);
  border: var(--border-thin) solid var(--color-border);
  border-radius: var(--radius-lg);
}

.info {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
  flex: 1;
  min-width: 0;
}

.name {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  color: var(--color-text);
}

.chip {
  padding: var(--space-1) var(--space-2);
  border-radius: var(--radius-full);
  background: var(--color-primary-light);
  color: var(--color-primary);
}

.meta {
  color: var(--color-text-tertiary);
  font-family: var(--font-mono, monospace);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>
