<script setup lang="ts">
import { onMounted, ref } from 'vue';
import { useRoute } from 'vue-router';

import { useTeamtypeStore } from '@/stores/teamtype';
import { useProjectsStore } from '@/stores/projects';
import type { ProjectView } from '@/client/types.gen';
import TeamtypeEditor from '@/components/TeamtypeEditor.vue';
import ShareProjectDialog from '@/components/ShareProjectDialog.vue';
import LogsDialog from '@/components/LogsDialog.vue';

const route = useRoute();
const teamtype = useTeamtypeStore();
const projects = useProjectsStore();

const error = ref<string | null>(null);
const project = ref<ProjectView | null>(null);
const sharing = ref(false);
const showLogs = ref(false);

onMounted(async () => {
  const id = route.params.id as string;
  await teamtype.start();
  teamtype.setName(`peer-${Math.floor(Math.random() * 1000)}`);

  projects
    .get(id)
    .then((p) => (project.value = p))
    .catch(() => {});

  try {
    const address = await projects.address(id);
    teamtype.connectByAddress(address);
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Could not connect to project';
  }
});
</script>

<template>
  <div class="editor-view">
    <header>
      <span v-if="error" class="drop">{{ error }}</span>
      <span v-else-if="!teamtype.ready" class="node">connecting…</span>
      <span v-if="teamtype.nodeInfo" class="node">
        node {{ teamtype.nodeInfo.node_id.slice(0, 12) }}…
      </span>
      <span class="peers">peers: {{ teamtype.peers.length }}</span>
      <span v-if="teamtype.lastDisconnect" class="drop">
        dropped: {{ teamtype.lastDisconnect.kind }}
      </span>
      <span class="spacer" />
      <button class="ghost" @click="showLogs = true">Logs</button>
      <button class="share" @click="sharing = true">Share</button>
    </header>

    <ShareProjectDialog :open="sharing" :project="project" @close="sharing = false" />
    <LogsDialog :open="showLogs" :project="project" @close="showLogs = false" />

    <main>
      <ul class="files">
        <li
          v-for="file in teamtype.files"
          :key="file"
          :class="{ active: file === teamtype.currentFile }"
          @click="teamtype.selectFile(file)"
        >
          {{ file }}
        </li>
      </ul>
      <TeamtypeEditor class="pane" />
    </main>
  </div>
</template>

<style scoped>
.editor-view {
  display: flex;
  flex-direction: column;
  height: 100vh;
  font-family: system-ui, sans-serif;
}

header {
  display: flex;
  gap: 0.5rem;
  align-items: center;
  padding: 0.5rem;
  border-bottom: 1px solid #ddd;
}

.node,
.peers {
  color: #777;
  font-size: 0.85rem;
}

.drop {
  color: #c0392b;
  font-size: 0.85rem;
}

.spacer {
  flex: 1;
}

.share {
  padding: 0.25rem 0.75rem;
  font-size: 0.85rem;
  color: var(--color-text-inverse);
  background: var(--color-primary);
  border: none;
  border-radius: var(--radius-md);
  cursor: pointer;
}

.share:hover {
  background: var(--color-accent-700);
}

.ghost {
  padding: 0.25rem 0.75rem;
  font-size: 0.85rem;
  color: var(--color-text);
  background: var(--color-surface);
  border: var(--border-thin) solid var(--color-border);
  border-radius: var(--radius-md);
  cursor: pointer;
}

.ghost:hover {
  background: var(--color-surface-hover);
}

main {
  display: flex;
  flex: 1;
  min-height: 0;
}

.files {
  width: 12rem;
  margin: 0;
  padding: 0.5rem;
  list-style: none;
  border-right: 1px solid #ddd;
  overflow: auto;
}

.files li {
  padding: 0.25rem 0.5rem;
  border-radius: 4px;
  cursor: pointer;
}

.files li.active {
  background: #eef;
}

.pane {
  flex: 1;
  min-width: 0;
}
</style>
