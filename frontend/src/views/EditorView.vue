<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue';
import { useRoute } from 'vue-router';

import { useAuthStore } from '@/stores/auth';
import { useOkayegStore } from '@/stores/okayeg';
import { useProjectsStore } from '@/stores/projects';
import { useKeysStore } from '@/stores/keys';
import type { ProjectView } from '@/client/types.gen';
import BaseChip from '@/components/BaseChip.vue';
import EditorWorkspace from '@/components/EditorWorkspace.vue';
import LivePeers from '@/components/LivePeers.vue';
import ShareProjectDialog from '@/components/ShareProjectDialog.vue';
import LogsDialog from '@/components/LogsDialog.vue';

const route = useRoute();
const auth = useAuthStore();
const okayeg = useOkayegStore();
const projects = useProjectsStore();
const keys = useKeysStore();

const error = ref<string | null>(null);
const project = ref<ProjectView | null>(null);
const hostId = ref<string | null>(null);
const sharing = ref(false);
const showLogs = ref(false);

// the workspace is shown once we've dialed the host and the live sync is
// running; until then (or on error/disconnect) we show nothing
const connected = computed(
  () => !error.value && !okayeg.lastDisconnect && okayeg.ready && okayeg.connected,
);

/**
 * Opens a project by ID: spin up the wasm peer, register this browser's ed
 * identity with the host, fetch the host endpoint id, and dial it to start
 * live sync.
 * @param id The ID of the project to open.
 */
async function openProject(id: string) {
  // true while `id` is still the route's project; guards against applying
  // results from a project the user has already navigated away from
  const isCurrent = () => route.params.id === id;

  error.value = null;
  project.value = null;
  hostId.value = null;
  okayeg.reset();
  // claim the logged-in user's identity before dialing
  const user = await auth.getUser();
  okayeg.setIdentity(user?.username ?? null, user?.email ?? null);
  await okayeg.start();

  projects
    .get(id)
    .then((p) => {
      if (isCurrent()) {
        project.value = p;
      }
    })
    .catch(() => {});

  try {
    const myId = okayeg.nodeId;
    if (!myId) {
      throw new Error('peer identity unavailable');
    }
    // register this browser's key on the account before we dial; the host's
    // authz gate resolves the connecting endpoint to its owning account and
    // admits it on every project that account is a member of. Idempotent.
    await keys.ensure(myId, 'Browser');
    const endpoint = await projects.endpoint(id);
    if (!isCurrent()) {
      return;
    }
    hostId.value = endpoint;
    await okayeg.connect(endpoint);
  } catch (e) {
    if (isCurrent()) {
      error.value = e instanceof Error ? e.message : 'Could not connect to project';
    }
  }
}

watch(
  () => route.params.id,
  (id) => {
    if (typeof id === 'string') {
      void openProject(id);
    }
  },
  { immediate: true },
);

onBeforeUnmount(() => {
  okayeg.reset();
});
</script>

<template>
  <div class="editor-view">
    <header class="topbar">
      <RouterLink
        :to="{ name: 'projects' }"
        class="back"
        title="Back to projects"
        aria-label="Back to projects"
      >
        &lsaquo;
      </RouterLink>

      <nav class="breadcrumb">
        <RouterLink :to="{ name: 'projects' }" class="brand">fossenCD</RouterLink>
        <span class="sep">/</span>
        <span class="crumb">{{ project?.name ?? 'Project' }}</span>
        <template v-if="okayeg.currentFile">
          <span class="sep">/</span>
          <span class="crumb file code-md">{{ okayeg.currentFile }}</span>
        </template>
      </nav>

      <div class="spacer" />

      <div class="status">
        <BaseChip v-if="error" variant="error">{{ error }}</BaseChip>
        <BaseChip v-else-if="!okayeg.ready">
          <span class="dot dot-pending" />
          starting&hellip;
        </BaseChip>
        <BaseChip v-else-if="!okayeg.connected">
          <span class="dot dot-pending" />
          connecting&hellip;
        </BaseChip>
        <template v-else>
          <LivePeers
            v-if="typeof route.params.id === 'string'"
            :project-id="route.params.id"
            :self-id="okayeg.nodeId"
            :host-id="hostId"
          />
          <BaseChip v-if="okayeg.nodeId" :title="okayeg.nodeId">
            <span class="dot dot-ok" />
            {{ okayeg.nodeId.slice(0, 8) }}&hellip;
          </BaseChip>
          <BaseChip v-if="okayeg.lastDisconnect" variant="error">
            dropped: {{ okayeg.lastDisconnect }}
          </BaseChip>
        </template>
      </div>

      <!-- the workspace teleports its layout switcher here once connected -->
      <div id="workspace-actions" class="workspace-actions"></div>

      <button class="label-sm btn btn-sm btn-secondary" @click="showLogs = true">Logs</button>
      <button class="label-sm btn btn-sm btn-primary" @click="sharing = true">
        Share / Connect
      </button>
    </header>

    <ShareProjectDialog :open="sharing" :project="project" @close="sharing = false" />
    <LogsDialog :open="showLogs" :project="project" @close="showLogs = false" />

    <EditorWorkspace v-if="connected" />
  </div>
</template>

<style scoped>
.editor-view {
  display: flex;
  flex-direction: column;
  height: 100vh;
  background: var(--color-bg-page);
}

/* Top bar */
.topbar {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  padding: var(--space-2) var(--space-4);
  background: var(--color-bg-card);
  border-bottom: var(--border-thin) solid var(--color-border);
}

.back {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 1.75rem;
  height: 1.75rem;
  font-size: 22px;
  line-height: 1;
  color: var(--color-text-secondary);
  border-radius: var(--radius-md);
  text-decoration: none;
}

.back:hover {
  background: var(--color-surface-hover);
  color: var(--color-text);
  text-decoration: none;
}

.breadcrumb {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  min-width: 0;
}

.brand {
  font: 600 16px var(--font-serif);
  letter-spacing: -0.01em;
  color: var(--color-text);
  text-decoration: none;
}

.brand:hover {
  text-decoration: none;
}

.sep {
  color: var(--color-text-tertiary);
}

.crumb {
  color: var(--color-text-secondary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.crumb.file {
  color: var(--color-text);
}

.spacer {
  flex: 1;
}

/* the teleported switcher joins the topbar's flex row directly, and an empty
   holder claims no gap slot */
.workspace-actions {
  display: contents;
}

/* Connection status chips */
.status {
  display: flex;
  align-items: center;
  gap: var(--space-2);
}

.dot {
  width: 6px;
  height: 6px;
  border-radius: var(--radius-full);
  flex: none;
}

.dot-ok {
  background: var(--color-success);
}

.dot-pending {
  background: var(--color-warning);
  animation: pulse 1.4s var(--ease-in-out) infinite;
}
</style>
