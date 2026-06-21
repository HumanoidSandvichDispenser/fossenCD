<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue';
import { useRoute } from 'vue-router';

import { useTeamtypeStore } from '@/stores/teamtype';
import { useProjectsStore } from '@/stores/projects';
import type { ProjectView } from '@/client/types.gen';
import EditorWorkspace from '@/components/EditorWorkspace.vue';
import ShareProjectDialog from '@/components/ShareProjectDialog.vue';
import LogsDialog from '@/components/LogsDialog.vue';

const route = useRoute();
const teamtype = useTeamtypeStore();
const projects = useProjectsStore();

const error = ref<string | null>(null);
const project = ref<ProjectView | null>(null);
const sharing = ref(false);
const showLogs = ref(false);

// the workspace is shown once we're connected to the project with at least
// one peer; until then (or on error/disconnect) we show nothing
const connected = computed(
  () =>
    !error.value &&
    !teamtype.lastDisconnect &&
    teamtype.ready &&
    teamtype.peers.length > 0,
);

/**
 * Opens a project by ID, fetching its details and connecting to its teamtype
 * address.
 * @param id The ID of the project to open.
 */
async function openProject(id: string) {
  // true while `id` is still the route's project; guards against applying
  // results from a project the user has already navigated away from
  const isCurrent = () => route.params.id === id;

  error.value = null;
  project.value = null;
  teamtype.reset();
  await teamtype.start();
  teamtype.setName(`peer-${Math.floor(Math.random() * 1000)}`);

  projects
    .get(id)
    .then((p) => {
      if (isCurrent()) {
        project.value = p;
      }
    })
    .catch(() => {});

  try {
    const address = await projects.address(id);
    if (!isCurrent()) {
      return;
    }
    teamtype.connectByAddress(address);
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
  teamtype.reset();
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
        <template v-if="teamtype.currentFile">
          <span class="sep">/</span>
          <span class="crumb file code-md">{{ teamtype.currentFile }}</span>
        </template>
      </nav>

      <div class="spacer" />

      <div class="status">
        <span v-if="error" class="chip chip-error label-sm">{{ error }}</span>
        <span v-else-if="!teamtype.ready" class="chip label-sm">
          <span class="dot dot-pending" />
          connecting&hellip;
        </span>
        <template v-else>
          <span v-if="teamtype.nodeInfo" class="chip label-sm" :title="teamtype.nodeInfo.node_id">
            <span class="dot dot-ok" />
            {{ teamtype.nodeInfo.node_id.slice(0, 8) }}&hellip;
          </span>
          <span class="chip label-sm">
            {{ teamtype.peers.length }} peer{{ teamtype.peers.length === 1 ? '' : 's' }}
          </span>
          <span v-if="teamtype.lastDisconnect" class="chip chip-error label-sm">
            dropped: {{ teamtype.lastDisconnect.kind }}
          </span>
        </template>
      </div>

      <button class="label-sm btn btn-sm btn-secondary" @click="showLogs = true">
        Logs
      </button>
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

/* Connection status chips */
.status {
  display: flex;
  align-items: center;
  gap: var(--space-2);
}

.chip {
  display: inline-flex;
  align-items: center;
  gap: var(--space-1);
  padding: var(--space-1) var(--space-2);
  color: var(--color-text-secondary);
  background: var(--color-surface);
  border-radius: var(--radius-full);
}

.chip-error {
  color: var(--color-error);
  background: var(--color-error-light);
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
