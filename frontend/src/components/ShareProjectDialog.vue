<script setup lang="ts">
import { computed, onUnmounted, ref, watch } from 'vue';

import { useProjectsStore } from '@/stores/projects';
import { useAuthStore } from '@/stores/auth';
import type { JoinCodeBody, MemberView, ProjectView } from '@/client/types.gen';
import BaseDialog from './BaseDialog.vue';

const props = defineProps<{ open: boolean; project: ProjectView | null }>();
const emit = defineEmits<{ close: [] }>();

const projects = useProjectsStore();
const auth = useAuthStore();

const members = ref<MemberView[]>([]);
const login = ref('');
const error = ref<string | null>(null);
const loading = ref(false);
const adding = ref(false);
const removing = ref<number | null>(null);

const invite = ref<JoinCodeBody | null>(null);
const minting = ref(false);
const copied = ref(false);
const now = ref(Date.now());
let ticker: ReturnType<typeof setInterval> | null = null;
let copyTimer: ReturnType<typeof setTimeout> | null = null;

// Time left on the active code as m:ss, counting down once per second.
const countdown = computed(() => {
  const left = invite.value
    ? Math.max(0, new Date(invite.value.expires_at).getTime() - now.value)
    : 0;
  const secs = Math.ceil(left / 1000);
  return `${Math.floor(secs / 60)}:${String(secs % 60).padStart(2, '0')}`;
});

function stopTicker() {
  if (ticker) {
    clearInterval(ticker);
    ticker = null;
  }
}

// Tick the countdown and drop the code once it expires server-side.
function startTicker() {
  stopTicker();
  ticker = setInterval(() => {
    now.value = Date.now();
    if (invite.value && new Date(invite.value.expires_at).getTime() <= now.value) {
      invite.value = null;
      stopTicker();
    }
  }, 1000);
}

onUnmounted(stopTicker);

// Whether the current user owns this project, gating the add/remove controls.
const isOwner = computed(
  () => members.value.find((m) => m.user_id === auth.user?.id)?.role === 'owner',
);

// Load the member list each time the dialog opens.
watch(
  () => props.open,
  async (open) => {
    invite.value = null;
    copied.value = false;
    stopTicker();
    if (!open || !props.project) {
      return;
    }
    login.value = '';
    error.value = null;
    members.value = [];
    loading.value = true;
    try {
      members.value = await projects.members(props.project.id);
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Could not load collaborators';
    } finally {
      loading.value = false;
    }
  },
);

async function add() {
  const trimmed = login.value.trim();
  if (!trimmed || !props.project) {
    return;
  }
  error.value = null;
  adding.value = true;
  try {
    members.value = await projects.addMember(props.project.id, trimmed);
    login.value = '';
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Could not add collaborator';
  } finally {
    adding.value = false;
  }
}

async function generate() {
  if (!props.project) {
    return;
  }
  error.value = null;
  minting.value = true;
  try {
    invite.value = await projects.joinCode(props.project.id);
    now.value = Date.now();
    startTicker();
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Could not mint join code';
  } finally {
    minting.value = false;
  }
}

async function copy() {
  if (!invite.value) {
    return;
  }
  await navigator.clipboard.writeText(invite.value.code);
  copied.value = true;
  if (copyTimer) {
    clearTimeout(copyTimer);
  }
  copyTimer = setTimeout(() => (copied.value = false), 1500);
}

async function remove(member: MemberView) {
  if (!props.project) {
    return;
  }
  error.value = null;
  removing.value = member.user_id;
  try {
    await projects.removeMember(props.project.id, member.user_id);
    members.value = members.value.filter((m) => m.user_id !== member.user_id);
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Could not remove collaborator';
  } finally {
    removing.value = null;
  }
}
</script>

<template>
  <BaseDialog class="share-dialog" :open="open" @close="emit('close')">
    <h2 class="h4">Share {{ project?.name }}</h2>

    <p v-if="loading" class="state text-sm">Loading...</p>

    <ul v-else class="members">
      <li v-for="member in members" :key="member.user_id" class="member">
        <span class="who">
          <span class="name label-sm">
            {{ member.username }}
            <span v-if="member.user_id === auth.user?.id" class="you text-xs">you</span>
          </span>
          <span class="email text-xs">{{ member.email }}</span>
        </span>
        <span v-if="member.role === 'owner'" class="role text-xs">owner</span>
        <button
          v-else-if="isOwner"
          class="label-sm btn btn-sm btn-danger"
          :disabled="removing === member.user_id"
          @click="remove(member)"
        >
          {{ removing === member.user_id ? 'Removing...' : 'Remove' }}
        </button>
      </li>
    </ul>

    <form v-if="isOwner" class="add" @submit.prevent="add">
      <input v-model="login" placeholder="username or email" aria-label="username or email" />
      <button type="submit" class="label-sm btn btn-secondary" :disabled="adding">
        {{ adding ? 'Adding…' : 'Add' }}
      </button>
    </form>

    <div class="invite">
      <p class="invite-label label-sm">Edit locally</p>
      <p class="hint text-xs">
        Open this project in your own editor with the teamtype CLI. Share the
        code to let a collaborator join the same way.
      </p>
      <template v-if="invite">
        <div class="code-row">
          <code class="code">{{ invite.code }}</code>
          <button type="button" class="label-sm btn btn-secondary" @click="copy">
            {{ copied ? 'Copied' : 'Copy' }}
          </button>
        </div>
        <p class="hint text-xs">
          Run <code>teamtype join {{ invite.code }}</code> (single use, expires in
          {{ countdown }})
        </p>
      </template>
      <button
        v-else
        type="button"
        class="label-sm btn btn-secondary"
        :disabled="minting"
        @click="generate"
      >
        {{ minting ? 'Generating...' : 'Generate join code' }}
      </button>
    </div>

    <p v-if="error" class="error text-sm">{{ error }}</p>

    <div class="actions">
      <button type="button" class="label-md btn btn-secondary" @click="emit('close')">Done</button>
    </div>
  </BaseDialog>
</template>

<style scoped>
.share-dialog {
  --dialog-width: var(--width-narrow);
}

.state {
  color: var(--color-text-tertiary);
}

.members {
  list-style: none;
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.member {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  padding: var(--space-2) var(--space-3);
  background: var(--color-bg-page);
  border: var(--border-thin) solid var(--color-border);
  border-radius: var(--radius-md);
}

.who {
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

.you {
  padding: 0 var(--space-1);
  color: var(--color-text-tertiary);
  background: var(--color-surface);
  border-radius: var(--radius-sm);
}

.email {
  color: var(--color-text-tertiary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.role {
  color: var(--color-text-tertiary);
  text-transform: uppercase;
  letter-spacing: 0.04em;
}

.add {
  display: flex;
  gap: var(--space-2);
}

.add input {
  flex: 1;
  padding: var(--space-2) var(--space-3);
  color: var(--color-text);
  background: var(--color-bg-page);
  border: var(--border-thin) solid var(--color-border);
  border-radius: var(--radius-md);
}

.add input:focus {
  outline: none;
  border-color: var(--color-primary);
}

.invite {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  padding-top: var(--space-2);
  border-top: var(--border-thin) solid var(--color-border);
}

.invite-label {
  color: var(--color-text);
}

.code-row {
  display: flex;
  align-items: center;
  gap: var(--space-2);
}

.code {
  flex: 1;
  padding: var(--space-2) var(--space-3);
  font-family: var(--font-mono);
  color: var(--color-text);
  background: var(--color-bg-page);
  border: var(--border-thin) solid var(--color-border);
  border-radius: var(--radius-md);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.hint {
  color: var(--color-text-tertiary);
}

.hint code {
  font-family: var(--font-mono);
}

.error {
  color: var(--color-error);
}

.actions {
  display: flex;
  justify-content: flex-end;
  gap: var(--space-3);
}

</style>
