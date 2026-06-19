<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue';
import { useRoute } from 'vue-router';

import { useTeamtypeStore } from '@/stores/teamtype';
import { useProjectsStore } from '@/stores/projects';
import type { ProjectView } from '@/client/types.gen';
import TeamtypeEditor from '@/components/TeamtypeEditor.vue';
import TypstPreview from '@/components/TypstPreview.vue';
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

// Typst rendering only makes sense for .typ files
const isTypst = computed(() => teamtype.currentFile?.toLowerCase().endsWith('.typ') ?? false);

// editor/preview split percentage
const splitPercent = ref(55);
const workspace = ref<HTMLElement>();
// cached during a drag; the workspace doesn't move while dragging, so we
// measure it once instead of on every pointermove
let dragRect: DOMRect | null = null;

function onDragMove(event: PointerEvent) {
  if (!dragRect) {
    return;
  }
  const pct = ((event.clientX - dragRect.left) / dragRect.width) * 100;
  splitPercent.value = Math.min(80, Math.max(20, pct));
}

function stopDrag() {
  dragRect = null;
  document.body.style.cursor = '';
  document.body.style.userSelect = '';
  window.removeEventListener('pointermove', onDragMove);
  window.removeEventListener('pointerup', stopDrag);
}

function startDrag() {
  if (!workspace.value) {
    return;
  }
  dragRect = workspace.value.getBoundingClientRect();
  document.body.style.cursor = 'col-resize';
  document.body.style.userSelect = 'none';
  window.addEventListener('pointermove', onDragMove);
  window.addEventListener('pointerup', stopDrag);
}

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
  stopDrag();
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

    <div v-if="connected" ref="workspace" class="workspace" :style="{ '--editor-basis': `${splitPercent}%` }">
      <aside class="sidebar">
        <div class="sidebar-head label-xs">Files</div>
        <ul v-if="teamtype.files.length" class="files">
          <li v-for="file in teamtype.files" :key="file" class="file-item label-sm"
            :class="{ active: file === teamtype.currentFile }" @click="teamtype.selectFile(file)">
            {{ file }}
          </li>
        </ul>
        <p v-else class="sidebar-empty text-xs">No files yet.</p>
      </aside>

      <section class="editor-pane">
        <TeamtypeEditor class="editor" />
      </section>

      <div class="divider" role="separator" aria-orientation="vertical" @pointerdown.prevent="startDrag">
        <span class="grip" />
      </div>

      <section class="preview-pane">
        <TypstPreview v-if="isTypst" :source="teamtype.currentText" />
        <div v-else class="preview-placeholder">
          <span class="preview-title serif-lg">Preview</span>
          <span class="preview-sub text-sm">
            {{ teamtype.currentFile ? 'Preview is only available for .typ files.' : 'Select a file to preview.' }}
          </span>
        </div>
      </section>
    </div>
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

/* Workspace split */
.workspace {
  display: flex;
  flex: 1;
  min-height: 0;
}

.sidebar {
  width: 13rem;
  flex: none;
  display: flex;
  flex-direction: column;
  padding: var(--space-3) var(--space-2);
  background: var(--color-bg-card);
  border-right: var(--border-thin) solid var(--color-border);
  overflow: auto;
}

.sidebar-head {
  padding: var(--space-1) var(--space-2) var(--space-3);
  color: var(--color-text-tertiary);
}

.files {
  list-style: none;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.file-item {
  padding: var(--space-2) var(--space-3);
  color: var(--color-text-secondary);
  border-radius: var(--radius-md);
  cursor: pointer;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  transition: background var(--duration-fast) var(--ease-out);
}

.file-item:hover {
  background: var(--color-surface-hover);
  color: var(--color-text);
}

.file-item.active {
  background: var(--color-primary-light);
  color: var(--color-primary);
}

.sidebar-empty {
  padding: var(--space-2) var(--space-3);
  color: var(--color-text-tertiary);
}

.editor-pane {
  /* Width driven by the drag handle via --editor-basis (set on .workspace). */
  flex: 1 1 var(--editor-basis, 55%);
  min-width: 0;
  background: var(--color-bg-card);
  overflow: hidden;
}

.editor {
  height: 100%;
}

/* Resizable divider */
.divider {
  flex: none;
  width: var(--border-thin);
  position: relative;
  background: var(--color-border);
  cursor: col-resize;
}

.divider::before {
  /* Widen the hit target without shifting layout. */
  content: '';
  position: absolute;
  inset: 0 -4px;
}

.divider:hover,
.divider:active {
  background: var(--color-accent-400);
}

.grip {
  position: absolute;
  top: 50%;
  left: 50%;
  width: 4px;
  height: 28px;
  transform: translate(-50%, -50%);
  border-radius: var(--radius-full);
  background: var(--color-neutral-300);
}

.divider:hover .grip {
  background: var(--color-accent-400);
}

/* Preview pane */
.preview-pane {
  flex: 1 1 auto;
  min-width: 0;
  overflow: auto;
  background: var(--color-bg-page);
}

.preview-placeholder {
  height: 100%;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: var(--space-1);
  text-align: center;
  padding: var(--space-8);
}

.preview-title {
  color: var(--color-text-secondary);
}

.preview-sub {
  color: var(--color-text-tertiary);
}
</style>
