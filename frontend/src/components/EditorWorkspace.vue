<script setup lang="ts">
import { computed, onBeforeUnmount, ref } from 'vue';
import { PhFilePdf } from '@phosphor-icons/vue';

import { useTeamtypeStore } from '@/stores/teamtype';
import { useZoom } from '@/composables/useZoom';
import { useCompileStatus } from '@/composables/useCompileStatus';
import { useDiagnostics } from '@/composables/useDiagnostics';
import { useExportPdf } from '@/composables/useExportPdf';
import { useEditorJump } from '@/composables/useEditorJump';
import { useSplitDrag } from '@/composables/useSplitDrag';
import { parseOutline } from '@/teamtype/outline';
import TeamtypeEditor from '@/components/TeamtypeEditor.vue';
import TypstPreview from '@/components/TypstPreview.vue';
import FileList from '@/components/FileList.vue';
import ZoomBar from './workspace/ZoomBar.vue';
import CompileStatus from './workspace/CompileStatus.vue';
import Outline from './workspace/Outline.vue';

const teamtype = useTeamtypeStore();

// Typst rendering only makes sense for .typ files; the preview tracks the
// chosen build target
const isTypst = computed(() => teamtype.previewFile?.toLowerCase().endsWith('.typ') ?? false);

const zoomState = useZoom();
const compileStatus = useCompileStatus();
const diagnostics = useDiagnostics();
const jump = useEditorJump();
const { exporting: pdfExporting, error: pdfError, exportPdf } = useExportPdf();

// outline of the file being edited, derived from its source markup; recomputes
// reactively as the file's content (in the VFS) or the selection changes
const outline = computed(() => {
  const file = teamtype.currentFile;
  const text = file ? teamtype.read(file) : undefined;
  return text ? parseOutline(text) : [];
});

// editor pane width as a % of the split grid; files pane height as a % of the
// left sidebar. Both resolve a grid track basis.
const splitPercent = ref(55);
const split = ref<HTMLElement>();
const editorDrag = useSplitDrag('x', (percent) => (splitPercent.value = percent));

const filesPercent = ref(55);
const sidebar = ref<HTMLElement>();
const filesDrag = useSplitDrag('y', (percent) => (filesPercent.value = percent), 80);

function startEditorDrag() {
  if (split.value) {
    editorDrag.start(split.value);
  }
}

function startFilesDrag() {
  if (sidebar.value) {
    filesDrag.start(sidebar.value);
  }
}

onBeforeUnmount(() => {
  editorDrag.stop();
  filesDrag.stop();
});
</script>

<template>
  <div class="workspace">
    <div ref="sidebar" class="leftbar" :style="{ '--files-basis': `${filesPercent}%` }">
      <FileList
        class="leftbar-files"
        :files="teamtype.files"
        :current-file="teamtype.currentFile"
        :preview-file="teamtype.previewFile"
        @select="teamtype.selectFile($event)"
        @preview="teamtype.setPreviewFile($event)"
      />
      <div
        class="divider divider-h"
        role="separator"
        aria-orientation="horizontal"
        @pointerdown.prevent="startFilesDrag"
      >
        <span class="grip grip-h" />
      </div>
      <Outline class="leftbar-outline" :headings="outline" @select="jump.jumpTo($event)" />
    </div>

    <div ref="split" class="split" :style="{ '--editor-basis': `${splitPercent}%` }">
      <div class="toolbar toolbar-editor">
        <div class="toolbar-group">
          <!-- start: editor actions (formatting, etc.) -->
        </div>
        <div class="toolbar-group">
          <!-- middle -->
        </div>
        <div class="toolbar-group">
          <!-- end -->
        </div>
      </div>

      <div class="toolbar toolbar-preview">
        <div class="toolbar-group">
          <!-- start: preview controls -->
          <ZoomBar :zoomState="zoomState" />
        </div>
        <div class="toolbar-group">
          <!-- middle -->
        </div>
        <div class="toolbar-group">
          <!-- end -->
          <CompileStatus :status="compileStatus" />
          <button
            v-if="isTypst && teamtype.previewFile"
            class="label-sm btn btn-sm btn-secondary"
            :disabled="pdfExporting"
            :title="pdfError ?? 'Export the preview as PDF'"
            @click="teamtype.previewFile && exportPdf(teamtype.previewFile, teamtype.vfs)"
          >
            <PhFilePdf :size="14" />
            {{ pdfExporting ? 'Exporting…' : 'Export PDF' }}
          </button>
        </div>
      </div>

      <div class="divider" role="separator" aria-orientation="vertical" @pointerdown.prevent="startEditorDrag">
        <span class="grip" />
      </div>

      <section class="editor-pane">
        <TeamtypeEditor class="editor" :diagnostics="diagnostics" :jump="jump" />
      </section>

      <section class="preview-pane">
        <!-- unsure what pattern to use once we have multiple preview types -->
        <TypstPreview
          v-if="isTypst && teamtype.previewFile"
          :main-file="teamtype.previewFile"
          :vfs="teamtype.vfs"
          :zoom="zoomState.zoom.value"
          :compile-status="compileStatus"
          :diagnostics="diagnostics"
        />
        <div v-else class="preview-placeholder">
          <span class="preview-title serif-lg">Preview</span>
          <span class="preview-sub text-sm">
            <template v-if="teamtype.previewFile">
              Preview is currently not available for <code>{{ teamtype.previewFile }}</code>.
            </template>
            <template v-else>
              Select a file to preview.
            </template>
          </span>
        </div>
      </section>
    </div>
  </div>
</template>

<style scoped>
.workspace {
  display: flex;
  flex: 1;
  min-height: 0;
}

/* Left sidebar: files (top) and outline (bottom), vertically resizable. */
.leftbar {
  flex: none;
  width: 13rem;
  min-height: 0;
  display: grid;
  grid-template-rows: var(--files-basis, 55%) var(--border-thin) 1fr;
  background: var(--color-bg-card);
  border-right: var(--border-thin) solid var(--color-border);
}

.leftbar-files,
.leftbar-outline {
  min-height: 0;
}

.split {
  flex: 1;
  min-width: 0;
  display: grid;
  grid-template-columns: var(--editor-basis, 55%) var(--border-thin) 1fr;
  grid-template-rows: auto 1fr;
  grid-template-areas:
    'editor-tools editor-tools preview-tools'
    'editor       divider      preview';
}

.toolbar {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  min-height: 2.5rem;
  padding: var(--space-1) var(--space-3);
  background: var(--color-bg-card);
  border-bottom: var(--border-thin) solid var(--color-border);
}

.toolbar-editor {
  grid-area: editor-tools;
  justify-content: space-between;
}

.toolbar-preview {
  grid-area: preview-tools;
  justify-content: space-between;
}

.toolbar-group {
  display: flex;
  align-items: center;
  gap: var(--space-1);
}

.editor-pane {
  grid-area: editor;
  min-width: 0;
  min-height: 0;
  background: var(--color-bg-card);
  overflow: hidden;
}

.editor {
  height: 100%;
}

.divider {
  grid-area: divider;
  position: relative;
  background: var(--color-border);
  cursor: col-resize;
}

.divider::before {
  /* Widen the hit target without shifting layout. */
  content: '';
  position: absolute;
  inset: 0 -4px;
}

.divider:hover,
.divider:active {
  background: var(--color-accent-400);
}

.grip {
  position: absolute;
  top: 50%;
  left: 50%;
  width: 4px;
  height: 28px;
  transform: translate(-50%, -50%);
  border-radius: var(--radius-full);
  background: var(--color-neutral-300);
}

.divider:hover .grip {
  background: var(--color-accent-400);
}

.divider-h {
  grid-area: auto;
  cursor: row-resize;
}

.divider-h::before {
  inset: -4px 0;
}

.grip-h {
  width: 28px;
  height: 4px;
}

/* Preview pane */
.preview-pane {
  grid-area: preview;
  min-width: 0;
  min-height: 0;
  overflow: auto;
  background: var(--color-bg-page);
}

.preview-placeholder {
  height: 100%;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: var(--space-1);
  text-align: center;
  padding: var(--space-8);
}

.preview-title {
  color: var(--color-text-secondary);
}

.preview-sub {
  color: var(--color-text-tertiary);
}
</style>
