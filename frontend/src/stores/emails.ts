import { ref } from 'vue';
import { defineStore } from 'pinia';

import { listEmails, addEmail, removeEmail } from '@/client/sdk.gen';
import type { EmailView } from '@/client/types.gen';
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

/** The user's email set. */
export const useEmailsStore = defineStore('emails', () => {
  const clientStore = useClientStore();
  const client = clientStore.client;

  const emails = ref<EmailView[]>([]);
  const loaded = ref(false);

  /**
   * Fetch the user's email set.
   */
  async function list(): Promise<EmailView[]> {
    const { data, error } = await listEmails({ client });
    if (error || !data) {
      throw new Error(errorMessage(error, 'Could not load emails'));
    }
    emails.value = data;
    loaded.value = true;
    return emails.value;
  }

  /** Add an address to the account. */
  async function add(email: string): Promise<void> {
    const { error } = await addEmail({ client, body: { email } });
    if (error) {
      throw new Error(errorMessage(error, 'Could not add email'));
    }
    await list();
  }

  /** Remove a non-primary address from the account. */
  async function remove(email: string): Promise<void> {
    const { error } = await removeEmail({ client, path: { email } });
    if (error) {
      throw new Error(errorMessage(error, 'Could not remove email'));
    }
    emails.value = emails.value.filter((e) => e.email !== email);
  }

  return { emails, loaded, list, add, remove };
});
