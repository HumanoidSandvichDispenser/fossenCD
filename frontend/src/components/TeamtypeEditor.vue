<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref, watch } from 'vue';
import { EditorState } from '@codemirror/state';
import { drawSelection, EditorView, keymap, lineNumbers } from '@codemirror/view';
import { defaultKeymap, history, historyKeymap } from '@codemirror/commands';

import { useTeamtypeStore } from '@/stores/teamtype';
import { collab, remoteEdit, setRemoteCursors, wireDeltaToChanges } from '@/teamtype/collab';
import type { RemoteCursor } from '@/teamtype/collab';

const teamtype = useTeamtypeStore();
const host = ref<HTMLDivElement>();
let view: EditorView | null = null;
const remote = new Map<string, RemoteCursor>();
const cleanups: Array<() => void> = [];

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

onMounted(() => {
  const state = EditorState.create({
    extensions: [
      lineNumbers(),
      history(),
      drawSelection(),
      keymap.of([...defaultKeymap, ...historyKeymap]),
      EditorView.lineWrapping,
      collab({
        onEdit: (delta) => {
          const file = teamtype.currentFile;
          if (file) {
            teamtype.applyEdit(file, delta);
          }
        },
        onCursor: (ranges) => {
          const file = teamtype.currentFile;
          if (file) {
            teamtype.setCursor(file, ranges);
          }
        },
      }),
    ],
  });
  view = new EditorView({ state, parent: host.value });

  cleanups.push(
    teamtype.onContent((file, text) => {
      if (!view || file !== teamtype.currentFile) {
        return;
      }
      view.dispatch({
        changes: { from: 0, to: view.state.doc.length, insert: text },
        annotations: remoteEdit.of(true),
      });
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

watch(() => teamtype.currentFile, pushCursors);

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
