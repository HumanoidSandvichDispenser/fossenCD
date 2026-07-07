<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref, watch } from 'vue';
import { EditorState } from '@codemirror/state';
import {
  drawSelection,
  EditorView,
  highlightActiveLine,
  highlightActiveLineGutter,
  keymap,
  lineNumbers,
} from '@codemirror/view';
import { defaultKeymap, history, historyKeymap } from '@codemirror/commands';
import { lintGutter, setDiagnostics } from '@codemirror/lint';

import { useOkayegStore } from '@/stores/okayeg';
import { collab } from '@/teamtype/collab';
import {
  comments as commentsExtension,
  setActiveComment,
  setCommentSpans,
  type CommentAnchor,
  type CommentSpan,
  type EditorViewport,
} from '@/teamtype/comments';
import { advanceCodePoints, codePointLength } from '@/teamtype/codepoints';
import { editorPresentation, languageForFile } from '@/teamtype/editor-theme';
import { toEditorDiagnostics } from '@/teamtype/editor-diagnostics';
import type { useDiagnostics } from '@/composables/useDiagnostics';
import type { useEditorJump } from '@/composables/useEditorJump';

const props = defineProps<{
  diagnostics: ReturnType<typeof useDiagnostics>;
  jump: ReturnType<typeof useEditorJump>;
  /** The pending comment draft, so its selection gets an anchor too. */
  draft: { file: string; start: number } | null;
}>();

const emit = defineEmits<{
  /** The user asked to comment on a selection (code point offsets + text). */
  draft: [file: string, start: number, end: number, quote: string];
  /** Document-space positions of the file's comment anchors. */
  anchors: [anchors: CommentAnchor[]];
  /** The scroll viewport moved or resized. */
  viewport: [viewport: EditorViewport];
}>();

const okayeg = useOkayegStore();
const host = ref<HTMLDivElement>();
let view: EditorView | null = null;
const cleanups: Array<() => void> = [];

// map from file path to EditorState.
const states = new Map<string, EditorState>();

/**
 * Create a new EditorState for the given file and document text, with the
 * appropriate extensions for line numbers, history, keymaps, language mode,
 * and collaboration.
 */
function makeState(file: string | null, doc: string): EditorState {
  return EditorState.create({
    doc,
    extensions: [
      lineNumbers(),
      highlightActiveLine(),
      highlightActiveLineGutter(),
      history(),
      drawSelection(),
      keymap.of([...defaultKeymap, ...historyKeymap]),
      lintGutter(),
      EditorView.lineWrapping,
      editorPresentation,
      languageForFile(file),
      collab({
        onEdit: (delta) => {
          const current = okayeg.currentFile;
          if (current) {
            okayeg.applyEdit(current, delta);
          }
        },
      }),
      EditorView.updateListener.of((update) => {
        if (update.docChanged || update.geometryChanged) {
          scheduleAnchors();
        }
      }),
      commentsExtension({
        onActivate: (id) => okayeg.setActiveComment(id),
        onDraft: (from, to) => {
          const current = okayeg.currentFile;
          if (!view || !current) {
            return;
          }
          const doc = view.state.doc;
          emit(
            'draft',
            current,
            codePointLength(doc.sliceString(0, from)),
            codePointLength(doc.sliceString(0, to)),
            doc.sliceString(from, to),
          );
        },
      }),
    ],
  });
}

/**
 * The current file's comment anchors as editor spans, converted from the
 * wasm's code point ranges to UTF-16 offsets against `text`.
 */
function commentSpans(text: string): CommentSpan[] {
  const file = okayeg.currentFile;
  if (!file) {
    return [];
  }
  return okayeg.comments
    .filter((c) => c.file === file && c.range && c.fields.resolved !== true)
    .map((c) => {
      const [start, end] = c.range!;
      const from = advanceCodePoints(text, 0, start);
      return {
        id: c.id,
        from,
        to: advanceCodePoints(text, from, end - start),
        orphaned: c.orphaned,
      };
    });
}

// Reassert spans and the active thread from store state, e.g. after a reseed
// wiped the extension's state or a comments tick re-resolved anchors.
function pushComments() {
  if (!view) {
    return;
  }
  view.dispatch({
    effects: [
      setCommentSpans.of(commentSpans(view.state.doc.toString())),
      setActiveComment.of(okayeg.activeCommentId),
    ],
  });
  emitAnchors();
}

/**
 * Report each anchor's document-space y-coordinate to the review panel. Uses
 * line block geometry rather than DOM coordinates so anchors outside the
 * rendered viewport still get a (estimated) position. Scroll does not change
 * these; the panel maps them to the screen with the viewport report.
 */
function emitAnchors() {
  if (!view) {
    return;
  }
  const text = view.state.doc.toString();
  const docLength = view.state.doc.length;
  const anchorAt = (id: string, pos: number): CommentAnchor => ({
    id,
    y: view!.lineBlockAt(Math.min(pos, docLength)).top,
  });
  const anchors = commentSpans(text).map((s) => anchorAt(s.id, s.from));
  if (props.draft && props.draft.file === okayeg.currentFile) {
    anchors.push(anchorAt('__draft', advanceCodePoints(text, 0, props.draft.start)));
  }
  emit('anchors', anchors);
  emitViewport();
}

function emitViewport() {
  if (view) {
    emit('viewport', { documentTop: view.documentTop });
  }
}

let anchorFrame = 0;
function scheduleAnchors() {
  cancelAnimationFrame(anchorFrame);
  anchorFrame = requestAnimationFrame(emitAnchors);
}

let viewportFrame = 0;
function scheduleViewport() {
  cancelAnimationFrame(viewportFrame);
  viewportFrame = requestAnimationFrame(emitViewport);
}

/** Scroll the editor by `delta` pixels; the review pane forwards its wheel here. */
function scrollBy(delta: number) {
  view?.scrollDOM.scrollBy({ top: delta });
}

defineExpose({ scrollBy });

/**
 * Swap the editor to a new state and mirror its text to the store.
 */
function showState(state: EditorState) {
  if (!view) {
    return;
  }
  view.setState(state);
}

// Replace the editor's inline markers with the current file's diagnostics.
function pushDiagnostics() {
  if (!view) {
    return;
  }
  const file = okayeg.currentFile;
  const mapped = file
    ? toEditorDiagnostics(view.state.doc, props.diagnostics.diagnostics.value, file)
    : [];
  view.dispatch(setDiagnostics(view.state, mapped));
}

onMounted(() => {
  view = new EditorView({ state: makeState(okayeg.currentFile, ''), parent: host.value });

  // scrolling only shifts the document-to-screen mapping, not the anchors
  // themselves; the scroller element survives setState, so one listener
  // covers every file
  view.scrollDOM.addEventListener('scroll', scheduleViewport, { passive: true });
  cleanups.push(() => {
    view?.scrollDOM.removeEventListener('scroll', scheduleViewport);
    cancelAnimationFrame(anchorFrame);
    cancelAnimationFrame(viewportFrame);
  });

  // Remote edits arrive as full snapshots via onContent (pass-1 okayeg-wasm
  // reflects the whole doc per change tick). Reseed only when the snapshot
  // differs from the editor's current text, so local typing doesn't feed back
  // into the buffer.
  cleanups.push(
    okayeg.onContent((file, text) => {
      if (!view || file !== okayeg.currentFile) {
        return;
      }

      if (view.state.doc.toString() !== text) {
        showState(makeState(file, text));
        pushDiagnostics();
        pushComments();
      }
    }),
  );
});

// re-highlight whenever anchors re-resolve (every change tick) or the active
// thread changes
watch(() => okayeg.comments, pushComments);
watch(() => props.draft, scheduleAnchors);
watch(
  () => okayeg.activeCommentId,
  (id) => {
    pushComments();
    if (!view || !id) {
      return;
    }
    // bring the anchor into view when the thread was picked from the panel
    const span = commentSpans(view.state.doc.toString()).find((s) => s.id === id);
    if (span) {
      view.dispatch({
        effects: EditorView.scrollIntoView(span.from, { y: 'nearest', yMargin: 24 }),
      });
    }
  },
);

watch(() => okayeg.currentFile, (file, prev) => {
  if (!view) {
    return;
  }

  if (!file) {
    // on deselect (project switch), clear editor state
    states.clear();
    showState(makeState(null, ''));
    pushDiagnostics();
    return;
  }

  // save the current state before switching, so we can restore it later if
  // needed
  if (prev) {
    states.set(prev, view.state);
  }

  const text = okayeg.read(file) ?? '';
  const cached = states.get(file);
  showState(cached && cached.doc.toString() === text ? cached : makeState(file, text));
  pushDiagnostics();
  pushComments();
});

// re-render markers when a new compile reports diagnostics for the open file
watch(() => props.diagnostics.diagnostics.value, pushDiagnostics);

// scroll to a line when the outline (or any producer) requests a jump
watch(() => props.jump.target.value, (jumpTarget) => {
  if (!view || !jumpTarget) {
    return;
  }
  const lineNumber = Math.min(Math.max(jumpTarget.line, 1), view.state.doc.lines);
  const pos = view.state.doc.line(lineNumber).from;
  view.dispatch({
    selection: { anchor: pos },
    effects: EditorView.scrollIntoView(pos, { y: 'start', yMargin: 8 }),
  });
  view.focus();
});

onBeforeUnmount(() => {
  for (const cleanup of cleanups) {
    cleanup();
  }
  view?.destroy();
});
</script>

<template>
  <div ref="host" class="editor"></div>
</template>

<style scoped>
.editor {
  height: 100%;
  overflow: auto;
}

.editor :deep(.cm-editor) {
  height: 100%;
}
</style>
