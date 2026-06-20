<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref, watch } from 'vue';
import { renderTypstFile, resetSources, syncSource } from '@/typst/compiler';

const props = defineProps<{
  // the file to render (the build target)
  mainFile: string;
  // live text of every opened project file, so cross-file imports resolve
  sources: Map<string, string>;
  // bumped whenever any source changes; our recompile trigger
  revision: number;
  // subset of `revision` counting only our own edits — those compile eagerly,
  // peer edits are debounced
  localRevision: number;
}>();

const svg = ref('');
const error = ref<string | null>(null);
// true only until the first compile resolves; afterwards we keep showing the
// last good render while the next one compiles, to avoid an empty flicker
const loading = ref(true);

// Peer edits can arrive in bursts we don't control, so we coalesce them behind
// a short quiet period. Our own typing instead recompiles as fast as the
// compiler goes idle (see `requestCompile`).
const DEBOUNCE_MS = 250;
let timer: ReturnType<typeof setTimeout> | null = null;
// guards against overlapping compiles: while one runs, the next is deferred and
// fired once it finishes, so the editor never queues a backlog of compiles
let compiling = false;
let pending = false;

async function runCompile() {
  compiling = true;
  loading.value = true;
  try {
    // push the latest text of every file into the compiler's shadow FS
    // (unchanged files are skipped internally) before rendering the target
    for (const [file, text] of props.sources) {
      await syncSource(file, text);
    }
    svg.value = await renderTypstFile(props.mainFile);
    error.value = null;
  } catch (e) {
    error.value = e instanceof Error ? e.message : String(e);
  } finally {
    loading.value = false;
    compiling = false;
    if (pending) {
      pending = false;
      void runCompile();
    }
  }
}

// Compile as soon as the compiler is free; if it's busy, run again right after.
// Used for our own edits and build-target switches — no artificial delay.
function compileWhenIdle() {
  if (timer !== null) {
    clearTimeout(timer);
    timer = null;
  }
  if (compiling) {
    pending = true;
  } else {
    void runCompile();
  }
}

// Wait for a pause before compiling. Used for peer edits.
function compileDebounced() {
  if (timer !== null) {
    clearTimeout(timer);
  }
  timer = setTimeout(() => {
    timer = null;
    compileWhenIdle();
  }, DEBOUNCE_MS);
}

onMounted(async () => {
  // start from a clean shadow FS so a previously-previewed project's files
  // don't linger in this one
  await resetSources();
  compileWhenIdle();
});

// Switching the build target is a deliberate user action — render it at once.
watch(() => props.mainFile, compileWhenIdle);

// A text change: if our own edit count moved with it, it's local (eager);
// otherwise it came from a peer (debounced).
let lastLocal = props.localRevision;
watch(
  () => props.revision,
  () => {
    const isLocal = props.localRevision !== lastLocal;
    lastLocal = props.localRevision;
    if (isLocal) {
      compileWhenIdle();
    } else {
      compileDebounced();
    }
  },
);

onBeforeUnmount(() => {
  if (timer !== null) {
    clearTimeout(timer);
  }
});
</script>

<template>
  <div class="typst-preview">
    <pre v-if="error" class="preview-error code-sm">{{ error }}</pre>
    <!-- eslint-disable-next-line vue/no-v-html -- SVG is produced by the Typst compiler from the user's own document -->
    <div v-else-if="svg" class="preview-doc" v-html="svg" />
    <div v-else-if="loading" class="preview-hint text-sm">Compiling&hellip;</div>
    <div v-else class="preview-hint text-sm">Nothing to preview.</div>
  </div>
</template>

<style scoped>
.typst-preview {
  height: 100%;
  padding: var(--space-6);
}

.preview-doc {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: var(--space-4);
}

.preview-doc :deep(svg) {
  max-width: 100%;
  height: auto;
  background: var(--color-bg-card);
  box-shadow: var(--shadow-md);
}

.preview-error {
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
