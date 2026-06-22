<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref, watch } from 'vue';
import { compilerPort, TypstCompileError } from '@/typst/compiler';
import { dropSource, resetSources, syncAll } from '@/typst/shadow-fs';
import { CanvasPreview } from '@/typst/canvas-preview';
import { isLogging, measure, recordSample } from '@/typst/perf';
import { useCompileScheduler } from '@/composables/useCompileScheduler';
import { useCompileStatus } from '@/composables/useCompileStatus';
import { useDiagnostics } from '@/composables/useDiagnostics';
import type { VirtualFs } from '@/vfs';

const props = withDefaults(
  defineProps<{
    // the file to render (the build target)
    mainFile: string;
    // the project filesystem to read sources from and watch for changes
    vfs: VirtualFs;
    // display zoom: 1 = page rendered at its natural point size (A4 -> 595x842 CSS px)
    zoom?: number;
    // compilation status, surfaced in the preview toolbar
    compileStatus: ReturnType<typeof useCompileStatus>;
    // compiler diagnostics, surfaced as inline markers in the editor
    diagnostics: ReturnType<typeof useDiagnostics>;
  }>(),
  { zoom: 1 },
);

// The scrolling viewport (used as the IntersectionObserver root for culling) and
// the centering host the preview renders into. Page display size is driven by
// `zoom`.
const scroller = ref<HTMLElement | null>(null);
const host = ref<HTMLElement | null>(null);
const error = ref<string | null>(null);
// true only until the first compile resolves; afterwards we keep showing the
// last good render while the next one compiles, to avoid an empty flicker
const loading = ref(true);
// becomes true once a render has actually painted into the host, so we can hide
// the "Compiling"/"Nothing to preview" hints
const hasRendered = ref(false);

let preview: CanvasPreview | null = null;

// One compile + render cycle: sync the shadow FS, then rasterize the target.
// Scheduling (debounce + overlap coalescing) is owned by `useCompileScheduler`;
// this function is just the work, and is never called directly.
async function compile() {
  if (preview === null) {
    return;
  }
  loading.value = true;
  props.compileStatus.startCompile();
  const cycleStart = performance.now();
  try {
    // push the latest text of every file into the compiler's shadow FS
    // (unchanged files are skipped internally) before rendering the target.
    await measure('sync', () => syncAll(props.vfs));
    // compile + cull + raster are measured inside the renderer as 'compile',
    // 'manip', 'raster' and 'rasterPage'. The compile also yields any warnings.
    const diagnostics = await preview.render(props.mainFile);
    props.diagnostics.set(diagnostics);
    hasRendered.value = true;
    error.value = null;
  } catch (e) {
    // A failed compile still carries structured diagnostics (the errors); other
    // failures have none to show in the editor.
    props.diagnostics.set(e instanceof TypstCompileError ? e.diagnostics : []);
    error.value = e instanceof Error ? e.message : String(e);
  } finally {
    // Record the whole cycle (sync + compile + raster, incl. any error path) as
    // its own metric so report() shows end-to-end edit-to-pixels latency.
    const total = performance.now() - cycleStart;
    recordSample('total', total);
    props.compileStatus.finishCompile(total);
    if (isLogging()) {
      console.debug(`[typst] compile cycle ${Math.round(total * 10) / 10}ms`);
    }
    loading.value = false;
  }
}

// Edits coalesce behind a short quiet period (rendering is main-thread work, so
// we render once typing pauses rather than per keystroke); deliberate actions
// (build-target switch, zoom) run via `whenIdle` with no delay.
const scheduler = useCompileScheduler(compile);

let unsubscribe: () => void = () => {};

onMounted(async () => {
  if (scroller.value !== null && host.value !== null) {
    preview = new CanvasPreview(scroller.value, host.value, compilerPort, props.zoom);
  }
  // start from a clean shadow FS so a previously-previewed project's files
  // don't linger in this one
  await resetSources();
  // watch the VFS: edits are debounced, and removed files are evicted from the
  // shadow FS
  unsubscribe = props.vfs.subscribe((change) => {
    if (change.kind === 'delete') {
      void dropSource(change.path);
    }
    scheduler.debounced();
  });
  scheduler.whenIdle();
});

// render whenever the build target changes
watch(() => props.mainFile, scheduler.whenIdle);

// Zoom changes rebuild at a new resolution; route through the compile path so
// session access stays serialized with edits.
watch(
  () => props.zoom,
  (z) => {
    preview?.setZoom(z);
    scheduler.whenIdle();
  },
);

onBeforeUnmount(() => {
  unsubscribe();
  preview?.destroy();
  scheduler.dispose();
  props.compileStatus.reset();
  props.diagnostics.clear();
});
</script>

<template>
  <div ref="scroller" class="typst-preview">
    <pre v-if="error" class="preview-error code-sm">{{ error }}</pre>
    <div ref="host" class="preview-doc" />
    <div v-if="!error && loading && !hasRendered" class="preview-hint text-sm">
      Compiling&hellip;
    </div>
    <div v-else-if="!error && !hasRendered" class="preview-hint text-sm">
      Nothing to preview.
    </div>
  </div>
</template>

<style scoped>
.typst-preview {
  height: 100%;
  /* This element is the scroll viewport (and the culling observer's root). At
     high zoom a page can exceed the pane in both axes, so allow both scrolls. */
  overflow: auto;
  padding: var(--space-6);
}

.preview-doc {
  display: flex;
  flex-direction: column;
  align-items: center;
  /* Grow to the (zoomed) page width so a page wider than the pane scrolls from
     the left edge instead of being clipped; stay >= pane width so a narrow page
     stays centered. */
  width: fit-content;
  min-width: 100%;
}

/* ts lowk buns just dont render this for now since it lags */
.preview-doc :deep(.typst-html-semantics) {
  display: none;
}

/* Page chrome lives on the renderer's per-page wrappers, never on the <canvas>
   itself */
.preview-doc :deep(.typst-page) {
  background: var(--color-bg-card);
  box-shadow: var(--shadow-md);
}

/* Space between stacked pages (the library stacks them with no gap). */
.preview-doc :deep(.typst-page:not(:last-child)) {
  margin-bottom: var(--space-4);
}

.preview-error {
  position: sticky;
  top: 0;
  left: 0;
  z-index: 2;
  margin: 0;
  padding: var(--space-4);
  white-space: pre-wrap;
  word-break: break-word;
  color: var(--color-error);
  background: var(--color-error-light);
  border-radius: var(--radius-md);
}

.preview-hint {
  display: flex;
  height: 100%;
  align-items: center;
  justify-content: center;
  color: var(--color-text-tertiary);
}
</style>
