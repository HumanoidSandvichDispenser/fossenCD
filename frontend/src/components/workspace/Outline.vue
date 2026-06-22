<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import { TreeItem, TreeRoot } from 'reka-ui';
import { PhCaretRight } from '@phosphor-icons/vue';
import { nestOutline, type OutlineHeading, type OutlineNode } from '@/teamtype/outline';

const props = defineProps<{
  headings: OutlineHeading[];
}>();

const emit = defineEmits<{
  select: [line: number];
}>();

const tree = computed(() => nestOutline(props.headings));
const getKey = (node: OutlineNode) => String(node.line);
const getChildren = (node: OutlineNode) => (node.children.length > 0 ? node.children : undefined);

const expanded = ref<string[]>([]);
const seen = new Set<string>();
watch(
  () => props.headings,
  (headings) => {
    const additions = headings.map(getKey).filter((key) => !seen.has(key));
    if (additions.length > 0) {
      additions.forEach((key) => seen.add(key));
      expanded.value = [...expanded.value, ...additions];
    }
  },
  { immediate: true },
);

function onToggle(event: CustomEvent<{ originalEvent: Event }>) {
  if (event.detail.originalEvent.type !== 'keydown') {
    event.preventDefault();
  }
}
</script>

<template>
  <aside class="outline">
    <div class="outline-head label-xs">Outline</div>
    <TreeRoot
      v-if="tree.length"
      v-slot="{ flattenItems }"
      v-model:expanded="expanded"
      class="outline-tree"
      :items="tree"
      :get-key="getKey"
      :get-children="getChildren"
    >
      <TreeItem
        v-for="item in flattenItems"
        :key="item._id"
        v-slot="{ isExpanded, isSelected, handleToggle }"
        v-bind="item.bind"
        class="outline-item"
        :style="{ marginLeft: `calc(${item.level - 1} * var(--space-4))` }"
        @select="emit('select', (item.value as OutlineNode).line)"
        @toggle="onToggle"
      >
        <span class="outline-caret-slot">
          <button
            v-if="item.hasChildren"
            class="outline-caret"
            tabindex="-1"
            aria-hidden="true"
            @click.stop="handleToggle"
          >
            <PhCaretRight :size="12" :class="{ expanded: isExpanded }" />
          </button>
        </span>
        <span class="outline-label text-sm" :class="{ selected: isSelected }">
          {{ (item.value as OutlineNode).text }}
        </span>
      </TreeItem>
    </TreeRoot>
    <p v-else class="outline-empty text-xs">No headings.</p>
  </aside>
</template>

<style scoped>
.outline {
  display: flex;
  flex-direction: column;
  min-height: 0;
  padding: var(--space-3) var(--space-2);
  overflow: auto;
}

.outline-head {
  padding: var(--space-1) var(--space-2) var(--space-3);
  color: var(--color-text-tertiary);
}

.outline-tree {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.outline-item {
  display: flex;
  align-items: center;
  width: fit-content;
  max-width: 100%;
  cursor: pointer;
  outline: none;
}

.outline-caret-slot {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex: none;
  width: var(--space-4);
}

.outline-caret {
  display: inline-flex;
  align-items: center;
  padding: 0;
  color: var(--color-text-tertiary);
  background: none;
  border: none;
  cursor: pointer;
}

.outline-caret :deep(svg) {
  transition: transform var(--duration-fast) var(--ease-out);
}

.outline-caret :deep(svg.expanded) {
  transform: rotate(90deg);
}

.outline-label {
  min-width: 0;
  padding: var(--space-1) var(--space-2);
  color: var(--color-text-secondary);
  border-radius: var(--radius-sm);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.outline-label:hover {
  background: var(--color-surface-hover);
  color: var(--color-text);
}

.outline-label.selected {
  background: var(--color-primary-light);
  color: var(--color-primary);
}

.outline-item:focus-visible .outline-label {
  box-shadow: inset 0 0 0 var(--border-thin) var(--color-accent-400);
}

.outline-empty {
  padding: var(--space-2) var(--space-3);
  color: var(--color-text-tertiary);
}
</style>
