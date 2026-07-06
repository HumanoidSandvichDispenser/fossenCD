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
import { editorPresentation, languageForFile } from '@/teamtype/editor-theme';
import { toEditorDiagnostics } from '@/teamtype/editor-diagnostics';
import type { useDiagnostics } from '@/composables/useDiagnostics';
import type { useEditorJump } from '@/composables/useEditorJump';

const props = defineProps<{
  diagnostics: ReturnType<typeof useDiagnostics>;
  jump: ReturnType<typeof useEditorJump>;
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
    ],
  });
}

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
      }
    }),
  );
});

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
