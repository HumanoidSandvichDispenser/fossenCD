<script setup lang="ts">
import { computed, ref } from 'vue';
import { useRoute } from 'vue-router';

import { useAuthStore } from '@/stores/auth';
import { useKeysStore } from '@/stores/keys';

const route = useRoute();
const auth = useAuthStore();
const keys = useKeysStore();

// the key handed over by the refusal message's enroll link; the whole flow is
// just confirming this key is yours and adding it to the account you're on
const key = String(route.query.key ?? '').trim();
const label = ref('');
const enrolling = ref(false);
const error = ref<string | null>(null);
const done = ref(false);

const keyValid = computed(() => /^[0-9a-f]{64}$/.test(key));

// already on this account: enrolling again is a harmless no-op, but say so
// rather than pretending we just added it
const alreadyRegistered = computed(
  () => keys.loaded && keys.keys.some((k) => k.endpoint_id === key),
);

function shortId(id: string) {
  return `${id.slice(0, 12)}...${id.slice(-6)}`;
}

async function enroll() {
  error.value = null;
  enrolling.value = true;
  try {
    await keys.ensure(key, label.value.trim());
    await keys.list();
    done.value = true;
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Could not register this key';
  } finally {
    enrolling.value = false;
  }
}

// so we can tell the visitor the key is already theirs before they click
keys.list().catch(() => {
  // best-effort: the confirm still works without the existing list
});
</script>

<template>
  <main class="enroll">
    <header>
      <h1 class="h3">Register a key</h1>
    </header>

    <p v-if="!keyValid" class="error text-sm">
      This enroll link is missing a valid key. Retry the sync and open the link it prints again.
    </p>

    <template v-else-if="done">
      <p class="ok text-sm">
        Key registered. <strong>{{ auth.user?.username }}</strong> can sync from that peer now, so
        retry the connection. You can close this tab.
      </p>
      <p class="text-sm">
        <RouterLink :to="{ name: 'settings' }">Manage keys</RouterLink>
      </p>
    </template>

    <template v-else-if="alreadyRegistered">
      <p class="state text-sm">
        This key is already registered on <strong>{{ auth.user?.username }}</strong>. That peer can
        sync now.
      </p>
      <p class="text-sm">
        <RouterLink :to="{ name: 'settings' }">Manage keys</RouterLink>
      </p>
    </template>

    <form v-else class="card" @submit.prevent="enroll">
      <p class="intro text-sm">
        Adding key <span class="mono" :title="key">{{ shortId(key) }}</span> to
        <strong>{{ auth.user?.username }}</strong
        >. It will be able to sync every project you belong to.
      </p>

      <label class="field-label label-sm">
        Name this key (optional)
        <input
          v-model="label"
          class="text-sm field"
          placeholder="e.g. My laptop"
          autocomplete="off"
        />
      </label>

      <button type="submit" class="label-sm btn btn-sm" :disabled="enrolling">
        {{ enrolling ? 'Registering...' : 'Register key' }}
      </button>

      <p v-if="error" class="error text-sm">{{ error }}</p>
    </form>
  </main>
</template>

<style scoped>
.enroll {
  max-width: var(--width-content);
  margin: 0 auto;
  padding: var(--space-8) var(--space-4);
}

header {
  padding-bottom: var(--space-4);
  border-bottom: var(--border-thin) solid var(--color-border);
}

.card {
  margin-top: var(--space-6);
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
  padding: var(--space-4);
  background: var(--color-bg-card);
  border: var(--border-thin) solid var(--color-border);
  border-radius: var(--radius-lg);
}

.card .btn {
  align-self: flex-end;
}

.intro {
  color: var(--color-text-secondary);
}

.field-label {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  color: var(--color-text-secondary);
}

.field {
  width: 100%;
  padding: var(--space-1) var(--space-2);
  border: var(--border-thin) solid var(--color-border);
  border-radius: var(--radius-sm);
  background: var(--color-bg);
  color: var(--color-text);
}

.mono {
  font-family: var(--font-mono, monospace);
}

.error {
  color: var(--color-error);
}

.ok,
.state {
  margin-top: var(--space-6);
  color: var(--color-text-secondary);
}
</style>
