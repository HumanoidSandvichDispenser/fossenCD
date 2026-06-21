<script setup lang="ts">
import { computed, onBeforeUnmount, ref } from 'vue';

import { useTeamtypeStore } from '@/stores/teamtype';
import { useZoom } from '@/composables/useZoom';
import { useCompileStatus } from '@/composables/useCompileStatus';
import TeamtypeEditor from '@/components/TeamtypeEditor.vue';
import TypstPreview from '@/components/TypstPreview.vue';
import FileList from '@/components/FileList.vue';
import ZoomBar from './workspace/ZoomBar.vue';
import CompileStatus from './workspace/CompileStatus.vue';

const teamtype = useTeamtypeStore();

// Typst rendering only makes sense for .typ files; the preview tracks the
// chosen build target
const isTypst = computed(() => teamtype.previewFile?.toLowerCase().endsWith('.typ') ?? false);

// editor pane width as a percentage of the whole workspace (the value the
// editor column's grid track resolves against)
const splitPercent = ref(55);
const split = ref<HTMLElement>();

const zoomState = useZoom();
const compileStatus = useCompileStatus();

// smallest pixel width we let either the editor or the preview shrink to
const MIN_PANE_PX = 160;
// cached at drag start; the layout doesn't move while dragging, so we measure
// the geometry once instead of on every pointermove. The split grid (editor +
// divider + preview) is what `--editor-basis` resolves against, so all geometry
// is relative to it; the sidebar lives outside and doesn't enter the math.
let dragGeom: { left: number; width: number } | null = null;

function onDragMove(event: PointerEvent) {
  if (!dragGeom) {
    return;
  }
  const { left, width } = dragGeom;
  const min = Math.min(MIN_PANE_PX, width / 2);
  // editor width = cursor position relative to the split's left edge, clamped
  // so the preview keeps at least `min` px
  const editorWidth = Math.max(min, Math.min(width - min, event.clientX - left));
  splitPercent.value = (editorWidth / width) * 100;
}

function stopDrag() {
  dragGeom = null;
  document.body.style.cursor = '';
  document.body.style.userSelect = '';
  window.removeEventListener('pointermove', onDragMove);
  window.removeEventListener('pointerup', stopDrag);
}

function startDrag() {
  if (!split.value) {
    return;
  }
  const rect = split.value.getBoundingClientRect();
  dragGeom = { left: rect.left, width: rect.width };
  document.body.style.cursor = 'col-resize';
  document.body.style.userSelect = 'none';
  window.addEventListener('pointermove', onDragMove);
  window.addEventListener('pointerup', stopDrag);
}

// drop the window listeners (and reset the body cursor) if we unmount mid-drag
onBeforeUnmount(stopDrag);
</script>

<template>
  <div class="workspace">
    <FileList
      class="filelist"
      :files="teamtype.files"
      :current-file="teamtype.currentFile"
      :preview-file="teamtype.previewFile"
      @select="teamtype.selectFile($event)"
      @preview="teamtype.setPreviewFile($event)"
    />

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
        </div>
      </div>

      <div class="divider" role="separator" aria-orientation="vertical" @pointerdown.prevent="startDrag">
        <span class="grip" />
      </div>

      <section class="editor-pane">
        <TeamtypeEditor class="editor" />
      </section>

      <section class="preview-pane">
        <!-- unsure what pattern to use once we have multiple preview types -->
        <TypstPreview
          v-if="isTypst && teamtype.previewFile"
          :main-file="teamtype.previewFile"
          :vfs="teamtype.vfs"
          :zoom="zoomState.zoom.value"
          :compile-status="compileStatus"
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

.filelist {
  flex: none;
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

.zoom-value {
  min-width: 3.5em;
  padding: var(--space-1) var(--space-2);
  text-align: center;
  color: var(--color-text-secondary);
  background: none;
  border: none;
  border-radius: var(--radius-md);
  cursor: pointer;
}

.zoom-value:hover {
  background: var(--color-surface-hover);
  color: var(--color-text);
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
