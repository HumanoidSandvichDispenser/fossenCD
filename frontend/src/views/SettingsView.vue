<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';

import { useEmailsStore } from '@/stores/emails';
import { useKeysStore } from '@/stores/keys';
import { useOkayegStore } from '@/stores/okayeg';

const emails = useEmailsStore();
const keys = useKeysStore();
const okayeg = useOkayegStore();

const keyError = ref<string | null>(null);
const revoking = ref<string | null>(null);
// the key currently being renamed, or null
const renaming = ref<string | null>(null);
const renameLabel = ref('');
// this browser's own endpoint id, so we can mark it and warn before self-revoke
const thisDevice = ref<string | null>(null);
// true while registering this browser's own key
const registering = ref(false);

// this browser's key, once derived, still absent from the registered list
const thisDeviceUnregistered = computed(
  () =>
    thisDevice.value !== null &&
    keys.loaded &&
    !keys.keys.some((k) => k.endpoint_id === thisDevice.value),
);

// Register this browser's key without opening a project. `getId` picks the id to
// register: the current identity, or a freshly minted one.
async function ensureThisDevice(getId: () => Promise<string>, failure: string) {
  keyError.value = null;
  registering.value = true;
  try {
    const id = await getId();
    thisDevice.value = id;
    await keys.ensure(id, 'Browser');
    await keys.list();
  } catch (e) {
    keyError.value = e instanceof Error ? e.message : failure;
  } finally {
    registering.value = false;
  }
}

// Derive this browser's key and register it, without having to open a project.
function registerThisDevice() {
  return ensureThisDevice(
    async () => thisDevice.value ?? (await okayeg.deviceId()),
    'Could not register this browser',
  );
}

// Mint a fresh identity for this browser and register it. The escape hatch when
// the current key is already claimed by another account.
function regenerateThisDevice() {
  if (
    !window.confirm(
      'Generate a new key for this browser? Sync from here uses the new key; ' +
        'the old one stays registered until you revoke it.',
    )
  ) {
    return;
  }
  return ensureThisDevice(okayeg.regenerateIdentity, 'Could not generate a new key');
}

// the add-a-native-key form: paste the id printed by `eg id`
const newId = ref('');
const newLabel = ref('');
const adding = ref(false);
const addError = ref<string | null>(null);
// neutral notice for a benign non-add, such as a key already on the account
const addNote = ref<string | null>(null);

const newIdValid = computed(() => /^[0-9a-f]{64}$/.test(newId.value.trim()));

async function addKey() {
  const endpointId = newId.value.trim();
  if (!newIdValid.value) {
    addError.value = 'That does not look like an endpoint id (64 lowercase hex characters).';
    return;
  }

  addError.value = null;
  addNote.value = null;

  if (keys.keys.some((k) => k.endpoint_id === endpointId)) {
    addNote.value = 'This key is already on your account.';
    return;
  }

  adding.value = true;
  try {
    await keys.ensure(endpointId, newLabel.value.trim());
    await keys.list();
    newId.value = '';
    newLabel.value = '';
  } catch (e) {
    addError.value = e instanceof Error ? e.message : 'Could not add key';
  } finally {
    adding.value = false;
  }
}

const sorted = computed(() =>
  [...keys.keys].sort((a, b) => {
    // surface this device first, then newest
    if (a.endpoint_id === thisDevice.value) {
      return -1;
    }

    if (b.endpoint_id === thisDevice.value) {
      return 1;
    }

    return b.created_at.localeCompare(a.created_at);
  }),
);

// the emails section: add an address that verifies your claimed identity
const newEmail = ref('');
const addingEmail = ref(false);
const emailError = ref<string | null>(null);
const removingEmail = ref<string | null>(null);

async function addEmail() {
  emailError.value = null;
  addingEmail.value = true;

  try {
    await emails.add(newEmail.value.trim());
    newEmail.value = '';
  } catch (e) {
    emailError.value = e instanceof Error ? e.message : 'Could not add email';
  } finally {
    addingEmail.value = false;
  }
}

async function removeEmail(email: string) {
  if (!window.confirm(`Remove ${email} from your account?`)) {
    return;
  }

  emailError.value = null;
  removingEmail.value = email;

  try {
    await emails.remove(email);
  } catch (e) {
    emailError.value = e instanceof Error ? e.message : 'Could not remove email';
  } finally {
    removingEmail.value = null;
  }
}

onMounted(() => {
  okayeg
    .deviceId()
    .then((id) => {
      thisDevice.value = id;
    })
    // deriving the device id is best-effort; the list still works without it
    .catch(() => {});

  keys.list().catch((e) => {
    keyError.value = e instanceof Error ? e.message : 'Could not load keys';
  });

  emails.list().catch((e) => {
    emailError.value = e instanceof Error ? e.message : 'Could not load emails';
  });
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

  keyError.value = null;
  revoking.value = endpointId;

  try {
    await keys.remove(endpointId);
  } catch (e) {
    keyError.value = e instanceof Error ? e.message : 'Could not revoke key';
  } finally {
    revoking.value = null;
  }
}

function startRename(endpointId: string, label: string) {
  renaming.value = endpointId;
  renameLabel.value = label;
}

function cancelRename() {
  renaming.value = null;
  renameLabel.value = '';
}

async function commitRename(endpointId: string) {
  keyError.value = null;
  try {
    await keys.rename(endpointId, renameLabel.value.trim());
    renaming.value = null;
    renameLabel.value = '';
  } catch (e) {
    keyError.value = e instanceof Error ? e.message : 'Could not rename key';
  }
}
</script>

<template>
  <main class="settings">
    <header>
      <h1 class="h3">Devices &amp; keys</h1>
    </header>

    <p class="intro text-sm">
      Each browser and native peer holds its own key and syncs every project you belong to. Only the
      public half is stored here; revoke a key to cut off a lost or unused device everywhere at
      once.
    </p>

    <div v-if="thisDeviceUnregistered" class="this-device card">
      <div class="info">
        <span class="name label-md">This browser is not registered yet</span>
        <span class="meta text-xs">Register it to sync your projects from here.</span>
      </div>
      <div class="row">
        <button
          class="label-sm btn btn-sm btn-ghost"
          :disabled="registering"
          @click="regenerateThisDevice"
        >
          Generate new key
        </button>
        <button class="label-sm btn btn-sm" :disabled="registering" @click="registerThisDevice">
          {{ registering ? 'Registering...' : 'Register this browser' }}
        </button>
      </div>
    </div>

    <form class="add card" @submit.prevent="addKey">
      <div class="add-fields">
        <input
          v-model="newId"
          class="text-sm field mono"
          placeholder="Endpoint id from `eg id`"
          spellcheck="false"
          autocomplete="off"
        />
        <input
          v-model="newLabel"
          class="text-sm field"
          placeholder="Label (optional)"
          autocomplete="off"
        />
      </div>
      <button type="submit" class="label-sm btn btn-sm" :disabled="adding || !newIdValid">
        {{ adding ? 'Adding...' : 'Add key' }}
      </button>
    </form>
    <p v-if="addError" class="error text-sm">{{ addError }}</p>
    <p v-else-if="addNote" class="state text-sm">{{ addNote }}</p>

    <p v-if="keyError" class="error text-sm">{{ keyError }}</p>

    <p v-if="!keys.loaded" class="state text-sm">Loading...</p>

    <p v-else-if="keys.keys.length === 0" class="state text-sm">
      No keys registered yet. Register this browser above, or open a project.
    </p>

    <ul v-else class="list">
      <li v-for="key in sorted" :key="key.endpoint_id" class="card">
        <div class="info">
          <form
            v-if="renaming === key.endpoint_id"
            class="edit"
            @submit.prevent="commitRename(key.endpoint_id)"
          >
            <input
              v-model="renameLabel"
              class="text-sm field"
              placeholder="New label"
              maxlength="128"
              autocomplete="off"
              autofocus
            />
            <div class="row">
              <button type="submit" class="label-sm btn btn-sm">Save</button>
              <button type="button" class="label-sm btn btn-sm btn-ghost" @click="cancelRename">
                Cancel
              </button>
            </div>
          </form>
          <template v-else>
            <span class="name label-md">
              {{ key.label || 'Unnamed device' }}
              <span v-if="key.endpoint_id === thisDevice" class="chip label-sm">This device</span>
            </span>
            <span class="meta text-xs" :title="key.endpoint_id">{{
              shortId(key.endpoint_id)
            }}</span>
            <span class="meta text-xs">Added {{ formatDate(key.created_at) }}</span>
          </template>
        </div>
        <div v-if="renaming !== key.endpoint_id" class="actions">
          <button
            class="label-sm btn btn-sm btn-ghost"
            @click="startRename(key.endpoint_id, key.label)"
          >
            Rename
          </button>
          <button
            class="label-sm btn btn-sm btn-danger"
            :disabled="revoking === key.endpoint_id"
            @click="revoke(key.endpoint_id)"
          >
            {{ revoking === key.endpoint_id ? 'Revoking...' : 'Revoke' }}
          </button>
        </div>
      </li>
    </ul>

    <header class="section-header">
      <h1 class="h3">Emails</h1>
    </header>

    <p class="intro text-sm">
      Peers claim a name and email when they connect. A claimed email shows as verified to
      collaborators when it is on this list and the peer's key is on your account.
    </p>

    <form class="add card" @submit.prevent="addEmail">
      <div class="add-fields">
        <input
          v-model="newEmail"
          class="text-sm field"
          type="email"
          placeholder="Email address"
          autocomplete="off"
        />
      </div>
      <button type="submit" class="label-sm btn btn-sm" :disabled="addingEmail || !newEmail.trim()">
        {{ addingEmail ? 'Adding...' : 'Add email' }}
      </button>
    </form>
    <p v-if="emailError" class="error text-sm">{{ emailError }}</p>

    <p v-if="!emails.loaded" class="state text-sm">Loading...</p>

    <ul v-else class="list">
      <li v-for="email in emails.emails" :key="email.email" class="card">
        <div class="info">
          <span class="name label-md">
            {{ email.email }}
            <span v-if="email.primary" class="chip label-sm">Primary</span>
            <span v-if="email.verified" class="chip chip-verified label-sm">Verified</span>
            <span v-else class="chip chip-muted label-sm">Unverified</span>
          </span>
          <span class="meta text-xs">Added {{ formatDate(email.created_at) }}</span>
        </div>
        <div v-if="!email.primary" class="actions">
          <button
            class="label-sm btn btn-sm btn-danger"
            :disabled="removingEmail === email.email"
            @click="removeEmail(email.email)"
          >
            {{ removingEmail === email.email ? 'Removing...' : 'Remove' }}
          </button>
        </div>
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

.this-device {
  margin-top: var(--space-6);
  justify-content: space-between;
}

.add {
  margin-top: var(--space-6);
  flex-direction: column;
}

.add .btn {
  align-self: flex-end;
}

.add-fields {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  width: 100%;
  min-width: 0;
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

.actions {
  display: flex;
  align-items: center;
  gap: var(--space-2);
}

.edit {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  width: 100%;
}

.row {
  display: flex;
  gap: var(--space-2);
}

.btn-ghost {
  background: transparent;
  color: var(--color-text-secondary);
}

.section-header {
  margin-top: var(--space-10);
}

.chip-verified {
  background: var(--color-success-light);
  color: var(--color-success);
}

.chip-muted {
  background: var(--color-surface);
  color: var(--color-text-tertiary);
}
</style>
