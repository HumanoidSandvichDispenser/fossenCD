<script setup lang="ts">
import { onMounted, ref } from 'vue';
import { useRoute } from 'vue-router';

import { useTeamtypeStore } from '@/stores/teamtype';
import { useProjectsStore } from '@/stores/projects';
import TeamtypeEditor from '@/components/TeamtypeEditor.vue';

const route = useRoute();
const teamtype = useTeamtypeStore();
const projects = useProjectsStore();

const error = ref<string | null>(null);

onMounted(async () => {
  await teamtype.start();
  teamtype.setName(`peer-${Math.floor(Math.random() * 1000)}`);

  try {
    const address = await projects.address(route.params.id as string);
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
    </header>

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
