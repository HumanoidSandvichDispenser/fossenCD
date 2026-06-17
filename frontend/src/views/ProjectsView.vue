<script setup lang="ts">
import { onMounted, ref } from 'vue';

import { useProjectsStore } from '@/stores/projects';
import type { ProjectView } from '@/client/types.gen';
import CreateProjectDialog from '@/components/CreateProjectDialog.vue';
import ShareProjectDialog from '@/components/ShareProjectDialog.vue';

const projects = useProjectsStore();

const dialogOpen = ref(false);
const sharing = ref<ProjectView | null>(null);
const error = ref<string | null>(null);
const deleting = ref<string | null>(null);

onMounted(async () => {
  try {
    await projects.list();
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Could not load projects';
  }
});

function formatDate(iso: string) {
  return new Date(iso).toLocaleDateString(undefined, {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
  });
}

async function remove(project: ProjectView) {
  if (!window.confirm(`Delete "${project.name}"? This cannot be undone.`)) {
    return;
  }
  error.value = null;
  deleting.value = project.id;
  try {
    await projects.remove(project.id);
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Could not delete project';
  } finally {
    deleting.value = null;
  }
}
</script>

<template>
  <main class="projects">
    <header>
      <h1 class="h3">Projects</h1>
      <div class="spacer" />
      <button class="label-sm new" @click="dialogOpen = true">New project</button>
    </header>

    <p v-if="error" class="error text-sm">{{ error }}</p>

    <p v-if="!projects.loaded" class="state text-sm">Loading...</p>

    <p v-else-if="projects.projects.length === 0" class="state text-sm">
      No projects yet. Create your first one.
    </p>

    <ul v-else class="list">
      <li v-for="project in projects.projects" :key="project.id" class="card">
        <RouterLink :to="{ name: 'editor', params: { id: project.id } }" class="open">
          <span class="name label-md">{{ project.name }}</span>
          <span class="meta text-xs">Created {{ formatDate(project.created_at) }}</span>
        </RouterLink>
        <button class="label-sm share" @click="sharing = project">Share</button>
        <button
          class="label-sm delete"
          :disabled="deleting === project.id"
          @click="remove(project)"
        >
          {{ deleting === project.id ? 'Deleting...' : 'Delete' }}
        </button>
      </li>
    </ul>

    <CreateProjectDialog :open="dialogOpen" @close="dialogOpen = false" />
    <ShareProjectDialog :open="sharing !== null" :project="sharing" @close="sharing = null" />
  </main>
</template>

<style scoped>
.projects {
  max-width: var(--width-content);
  margin: 0 auto;
  padding: var(--space-8) var(--space-4);
}

header {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  padding-bottom: var(--space-4);
  border-bottom: var(--border-thin) solid var(--color-border);
}

.spacer {
  flex: 1;
}

.new {
  padding: var(--space-2) var(--space-4);
  color: var(--color-text-inverse);
  background: var(--color-primary);
  border: none;
  border-radius: var(--radius-md);
  cursor: pointer;
}

.new:hover {
  background: var(--color-accent-700);
}

.error {
  margin-top: var(--space-4);
  color: var(--color-error);
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
  transition: border-color var(--duration-fast) var(--ease-out);
}

.card:hover {
  border-color: var(--color-accent-300);
}

.open {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
  flex: 1;
  color: inherit;
  text-decoration: none;
}

.open:hover {
  text-decoration: none;
}

.name {
  color: var(--color-text);
}

.meta {
  color: var(--color-text-tertiary);
}

.delete {
  padding: var(--space-2) var(--space-3);
  color: var(--color-error);
  background: transparent;
  border: var(--border-thin) solid var(--color-border);
  border-radius: var(--radius-md);
  cursor: pointer;
}

.delete:hover:not(:disabled) {
  background: var(--color-error-light);
  border-color: var(--color-error-200);
}

.delete:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}
</style>
