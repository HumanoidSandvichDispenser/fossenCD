<script setup lang="ts">
import { computed, onBeforeUnmount, ref } from 'vue';
import { PhFilePdf } from '@phosphor-icons/vue';

import { useOkayegStore } from '@/stores/okayeg';
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
import ReviewPanel, { type CommentDraft } from './workspace/ReviewPanel.vue';
import type { CommentAnchor, EditorViewport } from '@/teamtype/comments';

const okayeg = useOkayegStore();

// Typst rendering only makes sense for .typ files; the preview tracks the
// chosen build target
const isTypst = computed(() => okayeg.previewFile?.toLowerCase().endsWith('.typ') ?? false);

const zoomState = useZoom();
const compileStatus = useCompileStatus();
const diagnostics = useDiagnostics();
const jump = useEditorJump();
const { exporting: pdfExporting, error: pdfError, exportPdf } = useExportPdf();

// outline of the file being edited, derived from its source markup; recomputes
// reactively as the file's content (in the VFS) or the selection changes
const outline = computed(() => {
  const file = okayeg.currentFile;
  const text = file ? okayeg.read(file) : undefined;
  return text ? parseOutline(text) : [];
});

// editor pane width as a % of the split grid; files pane height as a % of the
// left sidebar. Both resolve a grid track basis.
const splitPercent = ref(55);
const split = ref<HTMLElement>();
const editorDrag = useSplitDrag('x', (percent) => (splitPercent.value = percent));

// the left pane: project files (with the outline) or the review panel
const sidebarTab = ref<'files' | 'review'>('files');
// a selection waiting to become a comment; owned here so the editor (where the
// selection happens) and the panel (where the body is written) both see it
const draft = ref<CommentDraft | null>(null);
// each comment anchor's document-space position and the editor's scroll
// mapping, reported by the editor; the review panel lays its cards out
// against these
const anchors = ref<CommentAnchor[]>([]);
const viewport = ref<EditorViewport | null>(null);
const editor = ref<InstanceType<typeof TeamtypeEditor>>();

// count for the review tab: unresolved threads on the file being edited
const commentCount = computed(
  () =>
    okayeg.comments.filter(
      (c) => !c.parent && c.file === okayeg.currentFile && c.fields.resolved !== true,
    ).length,
);

function startDraft(file: string, start: number, end: number, quote: string) {
  draft.value = { file, start, end, quote };
  sidebarTab.value = 'review';
}

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
    <div
      ref="sidebar"
      class="leftbar"
      :class="{ wide: sidebarTab === 'review' }"
      :style="{ '--files-basis': `${filesPercent}%` }"
    >
      <div class="tabbar">
        <button
          class="tab label-xs"
          :class="{ on: sidebarTab === 'files' }"
          @click="sidebarTab = 'files'"
        >
          Files
        </button>
        <button
          class="tab label-xs"
          :class="{ on: sidebarTab === 'review' }"
          @click="sidebarTab = 'review'"
        >
          Review
          <span v-if="commentCount" class="tab-badge">{{ commentCount }}</span>
        </button>
      </div>
      <div v-if="sidebarTab === 'files'" class="leftbar-files-tab">
        <FileList
          class="leftbar-section"
          :files="okayeg.files"
          :current-file="okayeg.currentFile"
          :preview-file="okayeg.previewFile"
          @select="okayeg.selectFile($event)"
          @preview="okayeg.setPreviewFile($event)"
        />
        <div
          class="divider divider-h"
          role="separator"
          aria-orientation="horizontal"
          @pointerdown.prevent="startFilesDrag"
        >
          <span class="grip grip-h" />
        </div>
        <Outline class="leftbar-section" :headings="outline" @select="jump.jumpTo($event)" />
      </div>
      <ReviewPanel
        v-else
        class="leftbar-review"
        :draft="draft"
        :anchors="anchors"
        :viewport="viewport"
        @draft-done="draft = null"
        @scroll-by="editor?.scrollBy($event)"
      />
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
            v-if="isTypst && okayeg.previewFile"
            class="label-sm btn btn-sm btn-secondary"
            :disabled="pdfExporting"
            :title="pdfError ?? 'Export the preview as PDF'"
            @click="okayeg.previewFile && exportPdf(okayeg.previewFile, okayeg.vfs)"
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
        <TeamtypeEditor
          ref="editor"
          class="editor"
          :diagnostics="diagnostics"
          :jump="jump"
          :draft="draft"
          @draft="startDraft"
          @anchors="anchors = $event"
          @viewport="viewport = $event"
        />
      </section>

      <section class="preview-pane">
        <!-- unsure what pattern to use once we have multiple preview types -->
        <TypstPreview
          v-if="isTypst && okayeg.previewFile"
          :main-file="okayeg.previewFile"
          :vfs="okayeg.vfs"
          :zoom="zoomState.zoom.value"
          :compile-status="compileStatus"
          :diagnostics="diagnostics"
        />
        <div v-else class="preview-placeholder">
          <span class="preview-title serif-lg">Preview</span>
          <span class="preview-sub text-sm">
            <template v-if="okayeg.previewFile">
              Preview is currently not available for <code>{{ okayeg.previewFile }}</code>.
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

/* Left sidebar: a Files tab (file list over outline, vertically resizable)
   and a Review tab (comment cards tracking the source). */
.leftbar {
  flex: none;
  width: 13rem;
  min-height: 0;
  display: flex;
  flex-direction: column;
  background: var(--color-bg-card);
  border-right: var(--border-thin) solid var(--color-border);
  transition: width var(--duration-fast) var(--ease-out);
}

.leftbar.wide {
  width: 19rem;
}

.leftbar-files-tab {
  flex: 1;
  min-height: 0;
  display: grid;
  grid-template-rows: var(--files-basis, 55%) var(--border-thin) 1fr;
}

.leftbar-section {
  min-height: 0;
}

.leftbar-review {
  flex: 1;
  min-height: 0;
}

.tabbar {
  display: flex;
  flex: none;
  gap: var(--space-1);
  padding: var(--space-2) var(--space-2) var(--space-1);
  border-bottom: var(--border-thin) solid var(--color-border);
}

.tab {
  display: inline-flex;
  align-items: center;
  gap: var(--space-1);
  padding: var(--space-1) var(--space-2);
  color: var(--color-text-tertiary);
  background: none;
  border: none;
  border-radius: var(--radius-sm);
  cursor: pointer;
}

.tab:hover {
  color: var(--color-text);
}

.tab.on {
  background: var(--color-primary-light);
  color: var(--color-primary);
}

.tab-badge {
  padding: 0 var(--space-1);
  font-size: 10px;
  color: var(--color-primary);
  background: var(--color-bg-card);
  border-radius: var(--radius-full);
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
