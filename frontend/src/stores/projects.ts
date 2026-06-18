import { ref } from 'vue';
import { defineStore } from 'pinia';

import {
  listProjects,
  getProject,
  createProject,
  deleteProject,
  projectAddress,
  projectJoinCode,
  listMembers as listMembersReq,
  addMember as addMemberReq,
  removeMember as removeMemberReq,
  projectLogs,
} from '@/client/sdk.gen';
import type { ProjectView, MemberView, LogsBody, JoinCodeBody } from '@/client/types.gen';
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

  /** Fetch a single project the caller has access to. */
  async function get(id: string): Promise<ProjectView> {
    const { data, error } = await getProject({ client, path: { id } });
    if (error || !data) {
      throw new Error(errorMessage(error, 'Could not load project'));
    }
    return data;
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

  /** Mint (or reuse and extend) a wormhole join code for terminal peers. */
  async function joinCode(id: string): Promise<JoinCodeBody> {
    const { data, error } = await projectJoinCode({ client, path: { id } });
    if (error || !data) {
      throw new Error(errorMessage(error, 'Could not mint join code'));
    }
    return data;
  }

  /** List a project's collaborators. */
  async function members(id: string): Promise<MemberView[]> {
    const { data, error } = await listMembersReq({ client, path: { id } });
    if (error || !data) {
      throw new Error(errorMessage(error, 'Could not load collaborators'));
    }
    return data;
  }

  /** Add a collaborator by username or email; returns the updated member list. */
  async function addMember(id: string, login: string): Promise<MemberView[]> {
    const { data, error } = await addMemberReq({ client, path: { id }, body: { login } });
    if (error || !data) {
      throw new Error(errorMessage(error, 'Could not add collaborator'));
    }
    return data;
  }

  /** Fetch recent daemon output and whether a host is running. */
  async function fetchLogs(id: string): Promise<LogsBody> {
    const { data, error } = await projectLogs({ client, path: { id } });
    if (error || !data) {
      throw new Error(errorMessage(error, 'Could not load logs'));
    }
    return data;
  }

  /** Remove a collaborator by user id. */
  async function removeMember(id: string, userId: number) {
    const { error } = await removeMemberReq({ client, path: { id, userId } });
    if (error) {
      throw new Error(errorMessage(error, 'Could not remove collaborator'));
    }
  }

  return {
    projects,
    loaded,
    list,
    get,
    create,
    remove,
    address,
    joinCode,
    members,
    addMember,
    removeMember,
    fetchLogs,
  };
});
