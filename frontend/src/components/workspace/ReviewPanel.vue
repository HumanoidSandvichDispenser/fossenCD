<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, reactive, ref, watch } from 'vue';

import { useOkayegStore, type OkayegComment } from '@/stores/okayeg';
import { useAuthStore } from '@/stores/auth';
import type { CommentAnchor, EditorViewport } from '@/teamtype/comments';
import CommentCard from './CommentCard.vue';

/** A pending comment: the selection captured in the editor, body not yet written. */
export interface CommentDraft {
  file: string;
  start: number;
  end: number;
  quote: string;
}

const props = defineProps<{
  /** 'file' hangs the current file's threads beside their source lines;
   * 'all' is a plain list grouped by file. */
  scope: 'file' | 'all';
  draft: CommentDraft | null;
  /** Document-space y-coordinates of the current file's anchors. */
  anchors: CommentAnchor[];
  /** The editor's document-to-screen mapping, updated as it scrolls. */
  viewport: EditorViewport | null;
}>();

const emit = defineEmits<{
  draftDone: [];
  /** Scrolling over the card lane forwards to the editor. */
  scrollBy: [delta: number];
}>();

const okayeg = useOkayegStore();
const auth = useAuthStore();

const showResolved = ref(false);
const draftBody = ref('');
const draftInput = ref<HTMLTextAreaElement>();
const lane = ref<HTMLElement>();

const self = computed(() => auth.user?.username ?? null);

function isResolved(c: OkayegComment): boolean {
  return c.fields.resolved === true;
}

/** Anchor order within a file; unresolved anchors sort last. */
function byRange(a: OkayegComment, b: OkayegComment): number {
  return (a.range?.[0] ?? Infinity) - (b.range?.[0] ?? Infinity) || a.createdAt - b.createdAt;
}

const replies = computed(() => {
  const map = new Map<string, OkayegComment[]>();
  for (const c of okayeg.comments) {
    if (c.parent) {
      const siblings = map.get(c.parent) ?? [];
      siblings.push(c);
      map.set(c.parent, siblings);
    }
  }
  for (const siblings of map.values()) {
    siblings.sort((a, b) => a.createdAt - b.createdAt);
  }
  return map;
});

const threads = computed(() =>
  okayeg.comments.filter((c) => !c.parent && (showResolved.value || !isResolved(c))),
);

const fileThreads = computed(() =>
  threads.value.filter((c) => c.file === okayeg.currentFile).sort(byRange),
);

/** All threads grouped per file, files in path order. */
const groupedThreads = computed(() => {
  const groups = new Map<string, OkayegComment[]>();
  for (const c of threads.value) {
    const key = c.file ?? '(no file)';
    const group = groups.get(key) ?? [];
    group.push(c);
    groups.set(key, group);
  }
  return [...groups.entries()]
    .sort(([a], [b]) => a.localeCompare(b))
    .map(([file, cs]) => ({ file, threads: cs.sort(byRange) }));
});

const resolvedCount = computed(
  () => okayeg.comments.filter((c) => !c.parent && isResolved(c)).length,
);

// ---- position-matched layout (file scope) ----
// Cards hang at the same height as their anchor line in the editor; when two
// would overlap, later ones get pushed down. The active thread (or the draft)
// wins its exact position and neighbors give way around it. Card positions
// are computed in document space, and the whole lane is translated to follow
// the editor's scroll, so scrolling never relayouts. The one exception is the
// priority card: when its anchor scrolls out of the pane, it pins to the
// nearer edge and slides back into alignment as the anchor returns.

const GAP = 8;
const FALLBACK_HEIGHT = 72;
// keep pinned cards clear of the lane's fade mask
const EDGE_PAD = 24;

const heights = reactive(new Map<string, number>());
const laneHeight = ref(0);
const observer = new ResizeObserver((entries) => {
  for (const entry of entries) {
    const el = entry.target as HTMLElement;
    if (el === lane.value) {
      laneHeight.value = el.clientHeight;
    } else if (el.dataset.commentId) {
      heights.set(el.dataset.commentId, el.getBoundingClientRect().height);
    }
  }
});

function observeCard(el: unknown) {
  if (el instanceof HTMLElement) {
    observer.observe(el);
  }
}

watch(lane, (el) => {
  if (el) {
    observer.observe(el);
    laneHeight.value = el.clientHeight;
  }
});

onBeforeUnmount(() => observer.disconnect());

/** The card that wins alignment (and edge-pinning): the draft, else the active thread. */
const priorityId = computed(() =>
  props.draft ? '__draft' : (okayeg.activeCommentId ?? null),
);

const cardTops = computed(() => {
  const anchorY = new Map(props.anchors.map((a) => [a.id, a.y]));

  const ids = fileThreads.value.map((c) => c.id);
  if (props.draft) {
    // slot the draft among the threads by its anchor position
    const y = anchorY.get('__draft') ?? 0;
    const at = ids.findIndex((id) => (anchorY.get(id) ?? Infinity) > y);
    ids.splice(at < 0 ? ids.length : at, 0, '__draft');
  }

  const desired = ids.map((id) => anchorY.get(id) ?? 0);
  const height = (id: string) => heights.get(id) ?? FALLBACK_HEIGHT;
  const tops: number[] = Array.from({ length: ids.length }, () => 0);

  const priority = ids.indexOf(priorityId.value ?? '');

  // place the priority card exactly, stack the rest away from it; with no
  // priority card, a plain top-down sweep
  const from = priority >= 0 ? priority : 0;
  tops[from] = desired[from] ?? 0;
  for (let i = from + 1; i < ids.length; i++) {
    tops[i] = Math.max(desired[i] ?? 0, (tops[i - 1] ?? 0) + height(ids[i - 1] ?? '') + GAP);
  }
  for (let i = from - 1; i >= 0; i--) {
    tops[i] = Math.min(desired[i] ?? 0, (tops[i + 1] ?? 0) - height(ids[i] ?? '') - GAP);
  }

  return new Map(ids.map((id, i) => [id, tops[i] ?? 0]));
});

/** Shift the lane so document-space card positions land on screen. */
const laneShift = computed(() => {
  if (!props.viewport || !lane.value) {
    return 0;
  }
  return props.viewport.documentTop - lane.value.getBoundingClientRect().top;
});

/**
 * The priority card's pinned position, or null while its anchor is in view.
 * Pane-space bounds are converted back to document space so the card can stay
 * inside the translated track.
 */
const pinnedTop = computed(() => {
  const id = priorityId.value;
  if (!id || !laneHeight.value) {
    return null;
  }
  const top = cardTops.value.get(id);
  if (top === undefined) {
    return null;
  }
  const height = heights.get(id) ?? FALLBACK_HEIGHT;
  const shift = laneShift.value;
  const min = EDGE_PAD - shift;
  const max = Math.max(min, laneHeight.value - height - EDGE_PAD - shift);
  const clamped = Math.min(Math.max(top, min), max);
  return clamped === top ? null : clamped;
});

function cardTop(id: string): number {
  if (id === priorityId.value && pinnedTop.value !== null) {
    return pinnedTop.value;
  }
  return cardTops.value.get(id) ?? 0;
}

/** Pinned cards track the scroll frame by frame; easing would make them lag. */
function pinned(id: string): boolean {
  return id === priorityId.value && pinnedTop.value !== null;
}

/** Cards the pinned card floats over fade back so it reads as a layer above. */
function obscured(id: string): boolean {
  const pin = pinnedTop.value;
  const pinId = priorityId.value;
  if (pin === null || !pinId || id === pinId) {
    return false;
  }
  const pinBottom = pin + (heights.get(pinId) ?? FALLBACK_HEIGHT);
  const top = cardTops.value.get(id) ?? 0;
  const bottom = top + (heights.get(id) ?? FALLBACK_HEIGHT);
  return top < pinBottom + GAP && bottom > pin - GAP;
}

function activate(c: OkayegComment) {
  if (c.file && c.file !== okayeg.currentFile) {
    okayeg.selectFile(c.file);
  }
  okayeg.setActiveComment(c.id);
}

function submitDraft() {
  const draft = props.draft;
  const author = self.value;
  const body = draftBody.value.trim();
  if (!draft || !author || !body) {
    return;
  }

  const id = okayeg.addComment(draft.file, draft.start, draft.end, {
    body,
    author,
    quote: draft.quote,
  });
  if (id) {
    okayeg.setActiveComment(id);
  }

  draftBody.value = '';
  emit('draftDone');
}

function cancelDraft() {
  draftBody.value = '';
  emit('draftDone');
}

// which file groups the user collapsed; groups default to open so the
// panel never hides threads unasked
const collapsed = reactive(new Set<string>());

function setCollapsed(file: string, value: boolean) {
  if (value) {
    collapsed.add(file);
  } else {
    collapsed.delete(file);
  }
}

function reply(parent: string, body: string) {
  const author = self.value;
  if (author) {
    okayeg.replyComment(parent, { body, author });
  }
}

// focus the draft box when a selection turns into a draft
watch(
  () => props.draft,
  async (draft) => {
    if (draft) {
      await nextTick();
      draftInput.value?.focus();
    }
  },
  { immediate: true },
);
</script>

<template>
  <aside class="panel">
    <!-- file scope: cards absolutely positioned to track the source; the
         track translates with the editor's scroll and wheel input over the
         lane scrolls the editor -->
    <div
      v-if="scope === 'file'"
      ref="lane"
      class="lane"
      @wheel.prevent="emit('scrollBy', $event.deltaY)"
    >
      <div class="lane-track" :style="{ transform: `translateY(${laneShift}px)` }">
      <div
        v-if="draft"
        :ref="observeCard"
        class="lane-card draft"
        :class="{ pinned: pinned('__draft'), obscured: obscured('__draft') }"
        data-comment-id="__draft"
        :style="{ top: `${cardTop('__draft')}px` }"
      >
        <blockquote class="draft-quote text-xs">{{ draft.quote }}</blockquote>
        <textarea
          ref="draftInput"
          v-model="draftBody"
          class="draft-input text-sm"
          rows="3"
          placeholder="Comment…"
          @keydown.enter.meta.prevent="submitDraft"
          @keydown.enter.ctrl.prevent="submitDraft"
          @keydown.esc="cancelDraft"
        />
        <div class="draft-actions">
          <button class="btn btn-sm btn-primary label-sm" @click="submitDraft">Comment</button>
          <button class="btn btn-sm btn-secondary label-sm" @click="cancelDraft">Cancel</button>
        </div>
      </div>

      <div
        v-for="thread in fileThreads"
        :key="thread.id"
        :ref="observeCard"
        class="lane-card"
        :class="{ pinned: pinned(thread.id), obscured: obscured(thread.id) }"
        :data-comment-id="thread.id"
        :style="{ top: `${cardTop(thread.id)}px` }"
      >
        <CommentCard
          :comment="thread"
          :replies="replies.get(thread.id) ?? []"
          :active="thread.id === okayeg.activeCommentId"
          :self="self"
          @activate="activate(thread)"
          @resolve="okayeg.setComment(thread.id, 'resolved', true)"
          @reopen="okayeg.setComment(thread.id, 'resolved', false)"
          @delete="okayeg.removeComment($event)"
          @reply="reply(thread.id, $event)"
        />
      </div>

      </div>

      <p v-if="!fileThreads.length && !draft" class="empty text-xs">
        No comments on this file. Select text to add one.
      </p>
    </div>

    <!-- all scope: threads grouped by file, each group collapsible -->
    <div v-else class="threads">
      <details
        v-for="group in groupedThreads"
        :key="group.file"
        class="group"
        :open="!collapsed.has(group.file)"
        @toggle="setCollapsed(group.file, !($event.target as HTMLDetailsElement).open)"
      >
        <summary class="group-head text-xs">
          <span class="group-file">{{ group.file }}</span>
          <span class="group-count">{{ group.threads.length }}</span>
        </summary>
        <div class="group-threads">
          <CommentCard
            v-for="thread in group.threads"
            :key="thread.id"
            :comment="thread"
            :replies="replies.get(thread.id) ?? []"
            :active="thread.id === okayeg.activeCommentId"
            :self="self"
            @activate="activate(thread)"
            @resolve="okayeg.setComment(thread.id, 'resolved', true)"
            @reopen="okayeg.setComment(thread.id, 'resolved', false)"
            @delete="okayeg.removeComment($event)"
            @reply="reply(thread.id, $event)"
          />
        </div>
      </details>
      <p v-if="!groupedThreads.length" class="empty text-xs">No comments in this project yet.</p>
    </div>

    <button
      v-if="resolvedCount"
      class="resolved-toggle text-xs"
      @click="showResolved = !showResolved"
    >
      {{ showResolved ? 'Hide' : 'Show' }} {{ resolvedCount }} resolved
    </button>
  </aside>
</template>

<style scoped>
.panel {
  display: flex;
  flex-direction: column;
  min-height: 0;
  height: 100%;
}

/* The card lane mirrors the editor's scroll space; cards are pinned to their
   anchor's height and clipped when scrolled out. */
.lane {
  position: relative;
  flex: 1;
  min-height: 0;
  overflow: hidden;
  /* cards pushed past the edges fade out instead of clipping mid-line */
  mask-image: linear-gradient(
    to bottom,
    transparent 0,
    black var(--space-6),
    black calc(100% - var(--space-6)),
    transparent 100%
  );
}

.lane-track {
  position: absolute;
  inset: 0;
  /* the track only translates; height comes from the lane */
  will-change: transform;
}

.lane-card {
  position: absolute;
  left: var(--space-2);
  right: var(--space-2);
  transition: top var(--duration-fast) var(--ease-out);
}

.lane-card.pinned {
  transition: none;
  z-index: 1;
  filter: drop-shadow(0 2px 8px rgb(0 0 0 / 0.12));
}

.lane-card.obscured {
  opacity: 0.25;
  transition:
    top var(--duration-fast) var(--ease-out),
    opacity var(--duration-fast) var(--ease-out);
}

.threads {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  padding: var(--space-2);
  overflow: auto;
}

.group-head {
  display: flex;
  align-items: center;
  gap: var(--space-1);
  margin-top: var(--space-1);
  padding: 0 var(--space-1);
  color: var(--color-text-tertiary);
  cursor: pointer;
  user-select: none;
}

.group-head::marker {
  content: none;
}

.group-head::before {
  content: '▸';
  flex: none;
  transition: transform var(--duration-fast) var(--ease-out);
}

.group[open] > .group-head::before {
  transform: rotate(90deg);
}

.group-file {
  font-family: var(--font-mono);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.group-count {
  flex: none;
  color: var(--color-text-tertiary);
}

.group-threads {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  margin-top: var(--space-2);
}

.draft {
  padding: var(--space-2);
  border: var(--border-thin) solid var(--color-warning-500);
  border-radius: var(--radius-sm);
  background: var(--color-bg-card);
}

.draft-quote {
  margin: 0 0 var(--space-1);
  padding-left: var(--space-2);
  border-left: 2px solid var(--color-warning-200);
  color: var(--color-text-tertiary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.draft-input {
  width: 100%;
  padding: var(--space-1) var(--space-2);
  font: inherit;
  border: var(--border-thin) solid var(--color-border);
  border-radius: var(--radius-sm);
  resize: vertical;
}

.draft-actions {
  display: flex;
  gap: var(--space-1);
  margin-top: var(--space-1);
}

.empty {
  padding: var(--space-2) var(--space-3);
  color: var(--color-text-tertiary);
}

.resolved-toggle {
  flex: none;
  margin: var(--space-2);
  padding: var(--space-1);
  color: var(--color-text-tertiary);
  background: none;
  border: var(--border-thin) dashed var(--color-border);
  border-radius: var(--radius-sm);
  cursor: pointer;
}

.resolved-toggle:hover {
  color: var(--color-text);
}
</style>
