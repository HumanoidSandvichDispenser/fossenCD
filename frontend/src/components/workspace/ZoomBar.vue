<script setup lang="ts">
import { PhCaretDown, PhMinus, PhPlus } from '@phosphor-icons/vue';
import {
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuPortal,
  DropdownMenuRoot,
  DropdownMenuTrigger,
  NumberFieldDecrement,
  NumberFieldIncrement,
  NumberFieldInput,
  NumberFieldRoot,
} from 'reka-ui';

import { useZoom } from '@/composables/useZoom';

const props = defineProps<{
  zoomState: ReturnType<typeof useZoom>;
}>();

const PRESETS = [0.5, 0.75, 0.9, 1, 1.1, 1.25, 1.5, 2, 3];

const FORMAT: Intl.NumberFormatOptions = { style: 'percent' };

function setZoom(value: number | null) {
  if (value != null) {
    props.zoomState.zoom.value = value;
  }
}
</script>

<template>
  <NumberFieldRoot
    class="zoom-field"
    :model-value="props.zoomState.zoom.value"
    :min="props.zoomState.minZoom.value"
    :max="props.zoomState.maxZoom.value"
    :step="props.zoomState.zoomStep.value"
    :format-options="FORMAT"
    @update:model-value="setZoom"
  >
    <NumberFieldDecrement class="btn btn-sm btn-secondary zoom-btn zoom-btn-start" aria-label="Zoom out">
      <PhMinus :size="12" />
    </NumberFieldDecrement>

    <DropdownMenuRoot>
      <DropdownMenuTrigger as-child>
        <div class="zoom-combo" aria-label="Zoom level and presets">
          <NumberFieldInput
            class="zoom-input label-sm"
            aria-label="Zoom level"
            @pointerdown.stop
            @keydown.stop
          />
          <span class="zoom-caret">
            <PhCaretDown :size="12" />
          </span>
        </div>
      </DropdownMenuTrigger>
      <DropdownMenuPortal>
        <DropdownMenuContent class="zoom-presets" align="start" :side-offset="4">
          <DropdownMenuItem
            v-for="preset in PRESETS"
            :key="preset"
            class="zoom-preset label-sm"
            @select="setZoom(preset)"
          >
            {{ Math.round(preset * 100) }}%
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenuPortal>
    </DropdownMenuRoot>

    <NumberFieldIncrement class="btn btn-sm btn-secondary zoom-btn zoom-btn-end" aria-label="Zoom in">
      <PhPlus :size="12" />
    </NumberFieldIncrement>
  </NumberFieldRoot>
</template>

<style scoped>
.zoom-field {
  display: flex;
  align-items: stretch;
  gap: 0;
  height: 1.75rem;
}

.zoom-btn {
  height: 100%;
}

.zoom-btn:disabled {
  background-color: var(--color-surface-hover);
}

.zoom-btn-start {
  border-right: none;
  border-radius: var(--radius-md) 0 0 var(--radius-md);
}

.zoom-btn-end {
  border-left: none;
  border-radius: 0 var(--radius-md) var(--radius-md) 0;
}

.zoom-combo {
  position: relative;
  display: inline-flex;
  align-items: stretch;
  background: var(--color-bg-card);
  border: var(--border-thin) solid var(--color-border);
}

.zoom-combo:focus-within {
  border-color: var(--color-accent-400);
}

.zoom-input {
  width: 4em;
  padding: 0 var(--space-2);
  text-align: center;
  color: var(--color-text);
  background: none;
  border: none;
  outline: none;
}

.zoom-btn {
  background-color: var(--color-bg-card);
}

.zoom-caret {
  display: flex;
  align-items: center;
  padding: 0 var(--space-1);
  font-size: 0.7em;
  color: var(--color-text-secondary);
  background: none;
  border: none;
  border-left: var(--border-thin) solid var(--color-border);
  cursor: pointer;
}

.zoom-caret:hover {
  background: var(--color-surface-hover);
  color: var(--color-text);
}

:deep(.zoom-presets) {
  z-index: var(--z-dropdown);
  width: var(--reka-popper-anchor-width);
  padding: var(--space-1);
  background: var(--color-bg-card);
  border: var(--border-thin) solid var(--color-border);
  border-radius: var(--radius-md);
  box-shadow: var(--shadow-md);
}

:deep(.zoom-preset) {
  display: block;
  width: 100%;
  padding: var(--space-1) var(--space-3);
  text-align: center;
  color: var(--color-text-secondary);
  border-radius: var(--radius-sm);
  cursor: pointer;
  outline: none;
}

:deep(.zoom-preset[data-highlighted]) {
  color: var(--color-text);
  background: var(--color-surface-hover);
}
</style>
