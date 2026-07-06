<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue';
import { useRoute } from 'vue-router';

import { useOkayegStore } from '@/stores/okayeg';
import { useProjectsStore } from '@/stores/projects';
import type { ProjectView } from '@/client/types.gen';
import EditorWorkspace from '@/components/EditorWorkspace.vue';
import ShareProjectDialog from '@/components/ShareProjectDialog.vue';
import LogsDialog from '@/components/LogsDialog.vue';

const route = useRoute();
const okayeg = useOkayegStore();
const projects = useProjectsStore();

const error = ref<string | null>(null);
const project = ref<ProjectView | null>(null);
const sharing = ref(false);
const showLogs = ref(false);

// the workspace is shown once we've dialed the host and the live sync is
// running; until then (or on error/disconnect) we show nothing
const connected = computed(
  () =>
    !error.value &&
    !okayeg.lastDisconnect &&
    okayeg.ready &&
    okayeg.connected,
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
  okayeg.reset();
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
    // register first so the host's .eg/trust admits us before we dial; the host
    // re-reads trust on every incoming connection, so this is enough
    await projects.registerPeer(id, myId);
    const endpoint = await projects.endpoint(id);
    if (!isCurrent()) {
      return;
    }
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
        <span v-if="error" class="chip chip-error label-sm">{{ error }}</span>
        <span v-else-if="!okayeg.ready" class="chip label-sm">
          <span class="dot dot-pending" />
          starting&hellip;
        </span>
        <span v-else-if="!okayeg.connected" class="chip label-sm">
          <span class="dot dot-pending" />
          connecting&hellip;
        </span>
        <template v-else>
          <span v-if="okayeg.nodeId" class="chip label-sm" :title="okayeg.nodeId">
            <span class="dot dot-ok" />
            {{ okayeg.nodeId.slice(0, 8) }}&hellip;
          </span>
          <span v-if="okayeg.lastDisconnect" class="chip chip-error label-sm">
            dropped: {{ okayeg.lastDisconnect }}
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
