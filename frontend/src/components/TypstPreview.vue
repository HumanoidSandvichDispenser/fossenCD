<script setup lang="ts">
import { onBeforeUnmount, ref, watch } from 'vue';
import { renderTypstSvg } from '@/typst/compiler';

const props = defineProps<{ source: string }>();

const svg = ref('');
const error = ref<string | null>(null);
// true only until the first compile resolves; afterwards we keep showing the
// last good render while the next one compiles, to avoid an empty flicker
const loading = ref(true);

// recompiling on every keystroke is wasteful and Typst compiles are not free,
// so we wait for a short pause in edits before rendering
const DEBOUNCE_MS = 250;
let timer: ReturnType<typeof setTimeout> | null = null;
// bumped per request so a slow compile can't overwrite a newer one's result
let seq = 0;

async function compile(source: string) {
  const token = ++seq;
  try {
    const result = await renderTypstSvg(source);
    if (token === seq) {
      svg.value = result;
      error.value = null;
    }
  } catch (e) {
    if (token === seq) {
      error.value = e instanceof Error ? e.message : String(e);
    }
  } finally {
    if (token === seq) {
      loading.value = false;
    }
  }
}

watch(
  () => props.source,
  (source) => {
    if (timer !== null) {
      clearTimeout(timer);
    }
    timer = setTimeout(() => {
      timer = null;
      void compile(source);
    }, DEBOUNCE_MS);
  },
  { immediate: true },
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
