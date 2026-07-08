<script setup lang="ts">
import { computed } from 'vue';
import { PhFilePdf } from '@phosphor-icons/vue';

import { useOkayegStore } from '@/stores/okayeg';
import type { useZoom } from '@/composables/useZoom';
import type { useCompileStatus } from '@/composables/useCompileStatus';
import type { useDiagnostics } from '@/composables/useDiagnostics';
import { useExportPdf } from '@/composables/useExportPdf';
import TypstPreview from '@/components/TypstPreview.vue';
import ZoomBar from './ZoomBar.vue';
import CompileStatus from './CompileStatus.vue';

// zoom, compile status and diagnostics are shared with the editor pane, so
// the workspace owns them and hands them down
const props = defineProps<{
  zoomState: ReturnType<typeof useZoom>;
  compileStatus: ReturnType<typeof useCompileStatus>;
  diagnostics: ReturnType<typeof useDiagnostics>;
}>();

const okayeg = useOkayegStore();
const { exporting: pdfExporting, error: pdfError, exportPdf } = useExportPdf();

// Typst rendering only makes sense for .typ files; the preview tracks the
// chosen build target
const isTypst = computed(() => okayeg.previewFile?.toLowerCase().endsWith('.typ') ?? false);
</script>

<template>
  <div class="preview-panel">
    <div class="toolbar">
      <div class="toolbar-group">
        <ZoomBar :zoomState="props.zoomState" />
      </div>
      <div class="toolbar-group">
        <CompileStatus :status="props.compileStatus" />
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
    <div class="preview-body">
      <TypstPreview
        v-if="isTypst && okayeg.previewFile"
        :main-file="okayeg.previewFile"
        :vfs="okayeg.vfs"
        :zoom="props.zoomState.zoom.value"
        :compile-status="props.compileStatus"
        :diagnostics="props.diagnostics"
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
    </div>
  </div>
</template>

<style scoped>
.preview-panel {
  display: flex;
  flex-direction: column;
  height: 100%;
}

.toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-2);
  min-height: 2.5rem;
  padding: var(--space-1) var(--space-3);
  background: var(--color-bg-card);
  border-bottom: var(--border-thin) solid var(--color-border);
}

.toolbar-group {
  display: flex;
  align-items: center;
  gap: var(--space-1);
}

.preview-body {
  flex: 1;
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
