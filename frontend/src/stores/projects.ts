import { ref } from 'vue';
import { defineStore } from 'pinia';

import { listProjects, createProject, deleteProject, projectAddress } from '@/client/sdk.gen';
import type { ProjectView } from '@/client/types.gen';
import { useClientStore } from './client';

/**
 * Pull a human-readable message out of a huma ErrorModel response.
 */
function errorMessage(error: unknown, fallback: string): string {
  if (error && typeof error === 'object') {
    const e = error as { detail?: string; title?: string };
    return e.detail ?? e.title ?? fallback;
  }
  return fallback;
}

export const useProjectsStore = defineStore('projects', () => {
  const clientStore = useClientStore();
  const client = clientStore.client;

  const projects = ref<ProjectView[]>([]);
  const loaded = ref(false);

  /**
   * Fetch the caller's projects. Deduped so concurrent callers share one request.
   */
  async function list() {
    const { data } = await clientStore.wrap(listProjects)({ client });
    projects.value = data ?? [];
    loaded.value = true;
    return projects.value;
  }

  async function create(name: string) {
    const { data, error } = await createProject({ client, body: { name } });
    if (error || !data) {
      throw new Error(errorMessage(error, 'Could not create project'));
    }
    projects.value = [data, ...projects.value];
    return data;
  }

  async function remove(id: string) {
    const { error } = await deleteProject({ client, path: { id } });
    if (error) {
      throw new Error(errorMessage(error, 'Could not delete project'));
    }
    projects.value = projects.value.filter((p) => p.id !== id);
  }

  /**
   * Fetch a project's secret address for web peers to connect by.
   */
  async function address(id: string) {
    const { data, error } = await projectAddress({ client, path: { id } });
    if (error || !data) {
      throw new Error(errorMessage(error, 'Could not load project address'));
    }
    return data.address;
  }

  return {
    projects,
    loaded,
    list,
    create,
    remove,
    address
  };
});
