<script setup lang="ts">
import { computed, ref } from 'vue';
import { PhCheck, PhTrash, PhArrowUUpLeft } from '@phosphor-icons/vue';

import type { OkayegComment } from '@/stores/okayeg';

const props = defineProps<{
  comment: OkayegComment;
  replies: OkayegComment[];
  active: boolean;
  /** The signed-in username, for showing delete on own comments only. */
  self: string | null;
}>();

const emit = defineEmits<{
  activate: [];
  resolve: [];
  reopen: [];
  delete: [id: string];
  reply: [body: string];
}>();

const replyBody = ref('');
const replying = ref(false);

const resolved = computed(() => props.comment.fields.resolved === true);
const quote = computed(() => (props.comment.fields.quote as string | undefined) ?? '');

function author(c: OkayegComment): string {
  return (c.fields.author as string | undefined) ?? 'unknown';
}

function body(c: OkayegComment): string {
  return (c.fields.body as string | undefined) ?? '';
}

/** Compact relative age: "now", "5m", "3h", "2d", else a local date. */
function age(ms: number): string {
  const s = Math.max(0, (Date.now() - ms) / 1000);
  if (s < 60) {return 'now';}
  if (s < 3600) {return `${Math.floor(s / 60)}m`;}
  if (s < 86400) {return `${Math.floor(s / 3600)}h`;}
  if (s < 7 * 86400) {return `${Math.floor(s / 86400)}d`;}
  return new Date(ms).toLocaleDateString();
}

function submitReply() {
  const text = replyBody.value.trim();
  if (!text) {
    return;
  }
  emit('reply', text);
  replyBody.value = '';
  replying.value = false;
}
</script>

<template>
  <article class="card" :class="{ active }" @click="emit('activate')">
    <header class="head">
      <span class="author text-xs">{{ author(comment) }}</span>
      <span class="age text-xs">{{ age(comment.createdAt) }}</span>
      <span class="head-actions">
        <button
          v-if="!resolved"
          class="icon-btn"
          title="Resolve"
          @click.stop="emit('resolve')"
        >
          <PhCheck :size="13" />
        </button>
        <button v-else class="icon-btn" title="Reopen" @click.stop="emit('reopen')">
          <PhArrowUUpLeft :size="13" />
        </button>
        <button
          v-if="self && author(comment) === self"
          class="icon-btn"
          title="Delete thread"
          @click.stop="emit('delete', comment.id)"
        >
          <PhTrash :size="13" />
        </button>
      </span>
    </header>

    <p v-if="comment.orphaned" class="orphaned text-xs">Original text was deleted</p>
    <blockquote v-if="quote" class="quote text-xs">{{ quote }}</blockquote>
    <p class="body text-sm">{{ body(comment) }}</p>

    <template v-if="active">
      <div v-for="reply in replies" :key="reply.id" class="reply">
        <header class="head">
          <span class="author text-xs">{{ author(reply) }}</span>
          <span class="age text-xs">{{ age(reply.createdAt) }}</span>
          <button
            v-if="self && author(reply) === self"
            class="icon-btn head-actions"
            title="Delete reply"
            @click.stop="emit('delete', reply.id)"
          >
            <PhTrash :size="13" />
          </button>
        </header>
        <p class="body text-sm">{{ body(reply) }}</p>
      </div>

      <textarea
        v-if="replying"
        v-model="replyBody"
        class="reply-input text-sm"
        rows="2"
        placeholder="Reply…"
        @click.stop
        @keydown.enter.meta.prevent="submitReply"
        @keydown.enter.ctrl.prevent="submitReply"
      />
      <div class="actions">
        <template v-if="replying">
          <button class="btn btn-sm btn-primary label-sm" @click.stop="submitReply">
            Reply
          </button>
          <button class="btn btn-sm btn-secondary label-sm" @click.stop="replying = false">
            Cancel
          </button>
        </template>
        <button
          v-else
          class="btn btn-sm btn-secondary label-sm"
          @click.stop="replying = true"
        >
          Reply
        </button>
      </div>
    </template>
    <p v-else-if="replies.length" class="reply-count text-xs">
      {{ replies.length }} {{ replies.length === 1 ? 'reply' : 'replies' }}
    </p>
  </article>
</template>

<style scoped>
.card {
  padding: var(--space-2);
  background: var(--color-bg-card);
  border: var(--border-thin) solid var(--color-border);
  border-radius: var(--radius-sm);
  cursor: pointer;
}

.card.active {
  border-color: var(--color-warning-500);
}

.head {
  display: flex;
  align-items: center;
  gap: var(--space-1);
  min-width: 0;
}

.author {
  font-weight: 600;
  color: var(--color-text-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.age {
  flex: none;
  color: var(--color-text-tertiary);
}

.head-actions {
  display: flex;
  gap: 2px;
  margin-left: auto;
}

.icon-btn {
  display: inline-flex;
  padding: 2px;
  color: var(--color-text-tertiary);
  background: none;
  border: none;
  border-radius: var(--radius-sm);
  cursor: pointer;
}

.icon-btn:hover {
  background: var(--color-surface-hover);
  color: var(--color-text);
}

.orphaned {
  margin: var(--space-1) 0 0;
  color: var(--color-warning);
  font-style: italic;
}

.quote {
  margin: var(--space-1) 0 0;
  padding-left: var(--space-2);
  border-left: 2px solid var(--color-warning-200);
  color: var(--color-text-tertiary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.body {
  margin: var(--space-1) 0 0;
  color: var(--color-text);
  white-space: pre-wrap;
  overflow-wrap: break-word;
}

.reply {
  margin-top: var(--space-2);
  padding-left: var(--space-2);
  border-left: var(--border-thin) solid var(--color-border);
}

.reply-count {
  margin: var(--space-1) 0 0;
  color: var(--color-text-tertiary);
}

.reply-input {
  width: 100%;
  margin-top: var(--space-2);
  padding: var(--space-1) var(--space-2);
  font: inherit;
  border: var(--border-thin) solid var(--color-border);
  border-radius: var(--radius-sm);
  resize: vertical;
}

.actions {
  display: flex;
  gap: var(--space-1);
  margin-top: var(--space-2);
}
</style>
