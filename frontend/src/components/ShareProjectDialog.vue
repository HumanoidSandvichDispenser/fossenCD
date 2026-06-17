<script setup lang="ts">
import { computed, ref, watch } from 'vue';

import { useProjectsStore } from '@/stores/projects';
import { useAuthStore } from '@/stores/auth';
import type { MemberView, ProjectView } from '@/client/types.gen';
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

// Whether the current user owns this project, gating the add/remove controls.
const isOwner = computed(
  () => members.value.find((m) => m.user_id === auth.user?.id)?.role === 'owner',
);

// Load the member list each time the dialog opens.
watch(
  () => props.open,
  async (open) => {
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
  <BaseDialog :open="open" @close="emit('close')">
    <h2 class="h4">Share {{ project?.name }}</h2>

    <p v-if="loading" class="state text-sm">Loading…</p>

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
          class="label-sm remove"
          :disabled="removing === member.user_id"
          @click="remove(member)"
        >
          {{ removing === member.user_id ? 'Removing…' : 'Remove' }}
        </button>
      </li>
    </ul>

    <form v-if="isOwner" class="add" @submit.prevent="add">
      <input v-model="login" placeholder="username or email" aria-label="username or email" />
      <button type="submit" class="label-sm primary" :disabled="adding">
        {{ adding ? 'Adding…' : 'Add' }}
      </button>
    </form>

    <p v-if="error" class="error text-sm">{{ error }}</p>

    <div class="actions">
      <button type="button" class="label-md secondary" @click="emit('close')">Done</button>
    </div>
  </BaseDialog>
</template>

<style scoped>
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

.remove {
  padding: var(--space-1) var(--space-3);
  color: var(--color-error);
  background: transparent;
  border: var(--border-thin) solid var(--color-border);
  border-radius: var(--radius-md);
  cursor: pointer;
}

.remove:hover:not(:disabled) {
  background: var(--color-error-light);
  border-color: var(--color-error-200);
}

.remove:disabled {
  opacity: 0.6;
  cursor: not-allowed;
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

.error {
  color: var(--color-error);
}

.actions {
  display: flex;
  justify-content: flex-end;
  gap: var(--space-3);
}

.primary {
  padding: var(--space-2) var(--space-4);
  color: var(--color-text-inverse);
  background: var(--color-primary);
  border: none;
  border-radius: var(--radius-md);
  cursor: pointer;
}

.primary:hover:not(:disabled) {
  background: var(--color-accent-700);
}

.primary:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.secondary {
  padding: var(--space-2) var(--space-4);
  color: var(--color-text);
  background: var(--color-surface);
  border: var(--border-thin) solid var(--color-border);
  border-radius: var(--radius-md);
  cursor: pointer;
}

.secondary:hover {
  background: var(--color-surface-hover);
}
</style>
