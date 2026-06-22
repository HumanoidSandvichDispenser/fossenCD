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

import { useTeamtypeStore } from '@/stores/teamtype';
import { collab, remoteEdit, setRemoteCursors, wireDeltaToChanges } from '@/teamtype/collab';
import type { RemoteCursor } from '@/teamtype/collab';
import { editorPresentation, languageForFile } from '@/teamtype/editor-theme';
import { toEditorDiagnostics } from '@/teamtype/editor-diagnostics';
import type { useDiagnostics } from '@/composables/useDiagnostics';

const props = defineProps<{
  diagnostics: ReturnType<typeof useDiagnostics>;
}>();

const teamtype = useTeamtypeStore();
const host = ref<HTMLDivElement>();
let view: EditorView | null = null;
const remote = new Map<string, RemoteCursor>();
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
          const current = teamtype.currentFile;
          if (current) {
            teamtype.applyEdit(current, delta);
          }
        },
        onCursor: (ranges) => {
          const current = teamtype.currentFile;
          if (current) {
            teamtype.setCursor(current, ranges);
          }
        },
      }),
    ],
  });
}

/**
 * Swap the editor to a new state and mirror its text to the store. Emits a
 * `docChanged` event. This can be used for file switches.
 */
function showState(state: EditorState) {
  if (!view) {
    return;
  }
  view.setState(state);
}

function colorFor(id: string): string {
  let hash = 0;
  for (const ch of id) {
    hash = (hash * 31 + ch.charCodeAt(0)) >>> 0;
  }
  return `hsl(${hash % 360} 70% 45%)`;
}

function pushCursors() {
  if (!view) {
    return;
  }
  const file = teamtype.currentFile;
  const visible = [...remote.values()].filter((cursor) => cursor.file === file);
  view.dispatch({ effects: setRemoteCursors.of(visible) });
}

// Replace the editor's inline markers with the current file's diagnostics.
function pushDiagnostics() {
  if (!view) {
    return;
  }
  const file = teamtype.currentFile;
  const mapped = file
    ? toEditorDiagnostics(view.state.doc, props.diagnostics.diagnostics.value, file)
    : [];
  view.dispatch(setDiagnostics(view.state, mapped));
}

onMounted(() => {
  view = new EditorView({ state: makeState(teamtype.currentFile, ''), parent: host.value });

  cleanups.push(
    teamtype.onContent((file, text) => {
      // Seed the editor if the focused file's content arrives after it was
      // selected. Edits to other opened files stream into the store's text map
      // (the WASM peer now emits onEdit for every opened file), so switching to
      // one rebuilds its state from there.
      if (!view || file !== teamtype.currentFile) {
        return;
      }

      if (view.state.doc.toString() !== text) {
        showState(makeState(file, text));
        pushCursors();
        pushDiagnostics();
      }
    }),
    teamtype.onEdit((file, delta) => {
      if (!view || file !== teamtype.currentFile) {
        return;
      }

      view.dispatch({
        changes: wireDeltaToChanges(view.state.doc, delta),
        annotations: remoteEdit.of(true),
      });
    }),
    teamtype.onEphemeral((message) => {
      const state = message.cursor_state;
      if (state.ranges.length === 0 || state.file_path === '') {
        remote.delete(message.cursor_id);
      } else {
        remote.set(message.cursor_id, {
          id: message.cursor_id,
          name: state.name ?? message.cursor_id,
          color: colorFor(message.cursor_id),
          file: state.file_path,
          ranges: state.ranges,
        });
      }
      pushCursors();
    }),
    teamtype.onEphemeralRemoved((cursorId) => {
      remote.delete(cursorId);
      pushCursors();
    }),
  );
});

watch(() => teamtype.currentFile, (file, prev) => {
  if (!view) {
    return;
  }

  if (!file) {
    // on deselect (project switch), clear editor state
    states.clear();
    showState(makeState(null, ''));
    pushCursors();
    pushDiagnostics();
    return;
  }

  // save the current state before switching, so we can restore it later if
  // needed
  if (prev) {
    states.set(prev, view.state);
  }

  const text = teamtype.read(file) ?? '';
  const cached = states.get(file);
  showState(cached && cached.doc.toString() === text ? cached : makeState(file, text));
  pushCursors();
  pushDiagnostics();
});

// re-render markers when a new compile reports diagnostics for the open file
watch(() => props.diagnostics.diagnostics.value, pushDiagnostics);

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
