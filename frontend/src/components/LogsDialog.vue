<script setup lang="ts">
import { nextTick, ref, useTemplateRef, watch } from 'vue';

import { useProjectsStore } from '@/stores/projects';
import type { ProjectView } from '@/client/types.gen';
import BaseDialog from './BaseDialog.vue';

const POLL_MS = 3000;

const props = defineProps<{ open: boolean; project: ProjectView | null }>();
const emit = defineEmits<{ close: [] }>();

const projects = useProjectsStore();

const output = ref('');
const running = ref(false);
const error = ref<string | null>(null);
const loaded = ref(false);
const pre = useTemplateRef<HTMLPreElement>('pre');

let timer: ReturnType<typeof setInterval> | null = null;

async function poll() {
  if (!props.project) {
    return;
  }
  try {
    const atBottom = pre.value
      ? pre.value.scrollHeight - pre.value.scrollTop - pre.value.clientHeight < 40
      : true;
    const data = await projects.fetchLogs(props.project.id);
    output.value = data.output;
    running.value = data.running;
    error.value = null;
    loaded.value = true;
    if (atBottom) {
      await nextTick();
      if (pre.value) {
        pre.value.scrollTop = pre.value.scrollHeight;
      }
    }
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Could not load logs';
  }
}

function stop() {
  if (timer) {
    clearInterval(timer);
    timer = null;
  }
}

watch(
  () => props.open,
  (open) => {
    if (!open) {
      stop();
      return;
    }
    output.value = '';
    running.value = false;
    error.value = null;
    loaded.value = false;
    poll();
    timer = setInterval(poll, POLL_MS);
  },
);
</script>

<template>
  <BaseDialog :open="open" class="logs-dialog" @close="emit('close')">
    <header class="head">
      <h2 class="h4">Logs for {{ project?.name }}</h2>
      <span class="status text-xs" :class="{ live: running }">
        {{ running ? 'running' : 'stopped' }}
      </span>
    </header>

    <p v-if="!loaded && !error" class="state text-sm">Loading…</p>
    <pre v-else ref="pre" class="output code-sm">{{ output || '(no output)' }}</pre>

    <p v-if="error" class="error text-sm">{{ error }}</p>

    <div class="actions">
      <button type="button" class="label-md btn btn-secondary" @click="emit('close')">Done</button>
    </div>
  </BaseDialog>
</template>

<style scoped>
.logs-dialog {
  --dialog-width: var(--width-wide);
}

.head {
  display: flex;
  align-items: center;
  gap: var(--space-3);
}

.status {
  display: inline-flex;
  align-items: center;
  gap: var(--space-1);
  padding: var(--space-1) var(--space-2);
  color: var(--color-text-tertiary);
  background: var(--color-surface);
  border-radius: var(--radius-sm);
  text-transform: uppercase;
  letter-spacing: 0.04em;
}

.status::before {
  content: '';
  width: 6px;
  height: 6px;
  border-radius: var(--radius-full);
  background: var(--color-text-tertiary);
}

.status.live {
  color: var(--color-success);
}

.status.live::before {
  background: var(--color-success);
}

.state {
  color: var(--color-text-tertiary);
}

.output {
  margin: 0;
  max-height: 24rem;
  overflow: auto;
  padding: var(--space-3);
  color: var(--color-text);
  background: var(--color-code-bg);
  border: var(--border-thin) solid var(--color-border);
  border-radius: var(--radius-md);
  white-space: pre-wrap;
  word-break: break-word;
}

.error {
  color: var(--color-error);
}

.actions {
  display: flex;
  justify-content: flex-end;
  gap: var(--space-3);
}
</style>
