<script setup lang="ts">
import { computed, defineComponent, h, markRaw, onBeforeUnmount, ref, watch } from 'vue';
import {
  PhArrowCounterClockwise,
  PhBookOpen,
  PhCaretDown,
  PhChatsCircle,
  PhPencilSimple,
} from '@phosphor-icons/vue';
import {
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuPortal,
  DropdownMenuRoot,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from 'reka-ui';
import { DockviewVue } from 'dockview-vue';
import type { DockviewApi, DockviewReadyEvent, DockviewTheme, VueComponent } from 'dockview-vue';
import 'dockview-vue/dist/styles/dockview.css';

import { useOkayegStore } from '@/stores/okayeg';
import { useZoom } from '@/composables/useZoom';
import { useCompileStatus } from '@/composables/useCompileStatus';
import { useDiagnostics } from '@/composables/useDiagnostics';
import { useEditorJump } from '@/composables/useEditorJump';
import { parseOutline } from '@/teamtype/outline';
import TeamtypeEditor from '@/components/TeamtypeEditor.vue';
import FileList from '@/components/FileList.vue';
import Outline from './workspace/Outline.vue';
import PreviewPane from './workspace/PreviewPane.vue';
import ReviewPanel, { type CommentDraft } from './workspace/ReviewPanel.vue';
import type { CommentAnchor, EditorViewport } from '@/teamtype/comments';

const okayeg = useOkayegStore();

const zoomState = useZoom();
const compileStatus = useCompileStatus();
const diagnostics = useDiagnostics();
const jump = useEditorJump();

// outline of the file being edited, derived from its source markup; recomputes
// reactively as the file's content (in the VFS) or the selection changes
const outline = computed(() => {
  const file = okayeg.currentFile;
  const text = file ? okayeg.read(file) : undefined;
  return text ? parseOutline(text) : [];
});

// a selection waiting to become a comment; owned here so the editor (where the
// selection happens) and the panel (where the body is written) both see it
const draft = ref<CommentDraft | null>(null);
// each comment anchor's document-space position and the editor's scroll
// mapping, reported by the editor; the current-file review pane hangs its
// cards against these
const anchors = ref<CommentAnchor[]>([]);
const viewport = ref<EditorViewport | null>(null);
const editor = ref<InstanceType<typeof TeamtypeEditor>>();

// count for the review tab title: unresolved threads on the file being edited
const commentCount = computed(
  () =>
    okayeg.comments.filter(
      (c) => !c.parent && c.file === okayeg.currentFile && c.fields.resolved !== true,
    ).length,
);

function startDraft(file: string, start: number, end: number, quote: string) {
  draft.value = { file, start, end, quote };
  dock.value?.getPanel('review-current')?.api.setActive();
}

// --- dockview layouts -------------------------------------------------------
//
// The workspace is a dockview dock. Panels are the workspace's fixed cast
// (files, outline, review, editor, preview); a layout is where they sit. Named
// layouts capture a workflow: writing wants the file tree at hand, reviewing
// puts the comment thread list first. Each layout persists the user's
// rearrangements separately, so switching back restores the dock as it was
// left, and a layout the user has never touched is built from its default.

type LayoutName = 'write' | 'review' | 'read';
const LAYOUT_NAMES: LayoutName[] = ['write', 'review', 'read'];
const LAYOUT_LABELS: Record<LayoutName, string> = { write: 'Write', review: 'Review', read: 'Read' };
const LAYOUT_ICONS = { write: PhPencilSimple, review: PhChatsCircle, read: PhBookOpen } as const;
const LAYOUT_HINTS: Record<LayoutName, string> = {
  write: 'View and edit files',
  review: 'Read and resolve comments',
  read: 'View the rendered document',
};
const storedLayout = localStorage.getItem('fossencd.workspace-layout.current') as LayoutName;
const layoutName = ref<LayoutName>(LAYOUT_NAMES.includes(storedLayout) ? storedLayout : 'write');
watch(layoutName, (name) => {
  localStorage.setItem('fossencd.workspace-layout.current', name);
});
const dock = ref<DockviewApi>();

// dockview looks panels up by name in this map. Each entry closes over the
// workspace's state, so the panels stay wired to the shared refs without any
// params plumbing; inheritAttrs off keeps dockview's injected props (params,
// api) from leaking onto the panel roots as DOM attributes.
function panel(render: () => ReturnType<typeof h>): VueComponent {
  return defineComponent({ inheritAttrs: false, setup: () => render }) as unknown as VueComponent;
}

const panelComponents = markRaw({
  files: panel(() =>
    h(FileList, {
      class: 'panel-fill',
      files: okayeg.files,
      currentFile: okayeg.currentFile,
      previewFile: okayeg.previewFile,
      onSelect: (f: string) => okayeg.selectFile(f),
      onPreview: (f: string) => okayeg.setPreviewFile(f),
    }),
  ),
  outline: panel(() =>
    h(Outline, {
      class: 'panel-fill',
      headings: outline.value,
      onSelect: (pos: number) => jump.jumpTo(pos),
    }),
  ),
  'review-current': panel(() =>
    h(ReviewPanel, {
      class: 'panel-fill',
      scope: 'file',
      draft: draft.value,
      anchors: anchors.value,
      viewport: viewport.value,
      onDraftDone: () => (draft.value = null),
      onScrollBy: (px: number) => editor.value?.scrollBy(px),
    }),
  ),
  'review-all': panel(() =>
    h(ReviewPanel, {
      class: 'panel-fill',
      scope: 'all',
      draft: null,
      anchors: [],
      viewport: null,
    }),
  ),
  editor: panel(() =>
    h(TeamtypeEditor, {
      ref: editor,
      class: 'panel-fill',
      diagnostics,
      jump,
      draft: draft.value,
      onDraft: startDraft,
      onAnchors: (a: CommentAnchor[]) => (anchors.value = a),
      onViewport: (v: EditorViewport) => (viewport.value = v),
    }),
  ),
  preview: panel(() => h(PreviewPane, { zoomState, compileStatus, diagnostics })),
});

const theme: DockviewTheme = {
  name: 'fossencd',
  // stack on the built-in light theme so every variable has a value; the
  // fossencd class overrides the ones our tokens care about
  className: 'dockview-theme-light dockview-theme-fossencd',
  colorScheme: 'light',
};

function storageKey(name: LayoutName) {
  return `fossencd.workspace-layout.${name}`;
}

function reviewTitle() {
  return commentCount.value ? `Review (${commentCount.value})` : 'Review';
}

// The default arrangement for each named layout. Write and review share the
// same shape (left column of tabs over the outline; editor in the middle; a
// right group holding the preview and the current file's review threads).
// Writing leads with the file tree and the preview; reviewing leads with the
// all-comments list and swaps the thread pane in front of the preview. Reading
// gives the rendered document the whole center, with the outline for
// navigation and everything else stacked behind.
function buildLayout(api: DockviewApi, name: LayoutName) {
  api.clear();

  if (name === 'read') {
    api.addPanel({ id: 'preview', component: 'preview', title: 'Preview' });
    // the editor keeps its DOM alive behind the preview so scroll and
    // selection survive a dip back into the source
    api.addPanel({
      id: 'editor',
      component: 'editor',
      title: 'Editor',
      renderer: 'always',
      position: { referencePanel: 'preview', direction: 'within' },
      inactive: true,
    });
    api.addPanel({
      id: 'review-current',
      component: 'review-current',
      title: reviewTitle(),
      position: { referencePanel: 'preview', direction: 'within' },
      inactive: true,
    });
    api.addPanel({
      id: 'outline',
      component: 'outline',
      title: 'Outline',
      position: { referencePanel: 'preview', direction: 'left' },
      initialWidth: 240,
    });
    api.addPanel({
      id: 'files',
      component: 'files',
      title: 'Files',
      position: { referencePanel: 'outline', direction: 'within' },
      inactive: true,
    });
    api.addPanel({
      id: 'review-all',
      component: 'review-all',
      title: 'Comments',
      position: { referencePanel: 'outline', direction: 'within' },
      inactive: true,
    });
    api.getPanel('preview')?.api.setActive();
    return;
  }

  api.addPanel({
    id: 'editor',
    component: 'editor',
    title: 'Editor',
    // keep CodeMirror's DOM alive when the panel is hidden so editor state
    // (scroll, selection) survives the user stacking it behind another tab
    renderer: 'always',
  });

  const reviewing = name === 'review';

  // right group: review-current and preview tabbed together, the layout's
  // focus first
  const rightFirst = reviewing ? 'review-current' : 'preview';
  const rightSecond = reviewing ? 'preview' : 'review-current';
  api.addPanel({
    id: rightFirst,
    component: rightFirst,
    title: rightFirst === 'review-current' ? reviewTitle() : 'Preview',
    position: { referencePanel: 'editor', direction: 'right' },
  });
  api.addPanel({
    id: rightSecond,
    component: rightSecond,
    title: rightSecond === 'review-current' ? reviewTitle() : 'Preview',
    position: { referencePanel: rightFirst, direction: 'within' },
    inactive: true,
  });

  // left group: file tree and the cross-file comment list
  const leftFirst = reviewing ? 'review-all' : 'files';
  const leftSecond = reviewing ? 'files' : 'review-all';
  api.addPanel({
    id: leftFirst,
    component: leftFirst,
    title: leftFirst === 'review-all' ? 'Comments' : 'Files',
    position: { referencePanel: 'editor', direction: 'left' },
    initialWidth: reviewing ? 304 : 208,
  });
  api.addPanel({
    id: leftSecond,
    component: leftSecond,
    title: leftSecond === 'review-all' ? 'Comments' : 'Files',
    position: { referencePanel: leftFirst, direction: 'within' },
    inactive: true,
  });
  api.addPanel({
    id: 'outline',
    component: 'outline',
    title: 'Outline',
    position: { referencePanel: leftFirst, direction: 'below' },
    inactive: true,
    initialHeight: 260,
  });

  api.getPanel('editor')?.api.setActive();
}

function restoreOrBuild(api: DockviewApi, name: LayoutName) {
  const saved = localStorage.getItem(storageKey(name));
  if (saved) {
    try {
      api.fromJSON(JSON.parse(saved));
      // a layout saved before the review split lacks the new panels; treat it
      // as invalid rather than stranding the user without them
      if (!api.getPanel('review-current') || !api.getPanel('review-all')) {
        throw new Error('stale layout');
      }
      // the serialized layout carries the review title from when it was saved
      api.getPanel('review-current')?.setTitle(reviewTitle());
      return;
    } catch {
      localStorage.removeItem(storageKey(name));
    }
  }
  buildLayout(api, name);
}

function saveLayout(api: DockviewApi, name: LayoutName) {
  try {
    localStorage.setItem(storageKey(name), JSON.stringify(api.toJSON()));
  } catch {
    // a full or unavailable storage only costs persistence
  }
}

function switchLayout(name: LayoutName) {
  const api = dock.value;
  if (!api || name === layoutName.value) {
    return;
  }
  saveLayout(api, layoutName.value);
  layoutName.value = name;
  restoreOrBuild(api, name);
}

function resetLayout() {
  const api = dock.value;
  if (!api) {
    return;
  }
  localStorage.removeItem(storageKey(layoutName.value));
  buildLayout(api, layoutName.value);
}

function onReady(event: DockviewReadyEvent) {
  dock.value = event.api;
  restoreOrBuild(event.api, layoutName.value);
}


watch(commentCount, () => {
  dock.value?.getPanel('review-current')?.setTitle(reviewTitle());
});

onBeforeUnmount(() => {
  if (dock.value) {
    saveLayout(dock.value, layoutName.value);
  }
});
</script>

<template>
  <!-- the layout switcher rides in the topbar rather than a strip of its own,
       so the modes read like Docs' editing/suggesting/viewing selector -->
  <Teleport to="#workspace-actions">
    <DropdownMenuRoot>
      <DropdownMenuTrigger
        class="layout-trigger label-sm btn btn-sm btn-secondary"
        aria-label="Workspace layout"
      >
        <component :is="LAYOUT_ICONS[layoutName]" :size="14" />
        {{ LAYOUT_LABELS[layoutName] }}
        <PhCaretDown :size="12" />
      </DropdownMenuTrigger>
      <DropdownMenuPortal>
        <DropdownMenuContent class="layout-list" align="end" :side-offset="4">
          <DropdownMenuItem
            v-for="name in LAYOUT_NAMES"
            :key="name"
            class="layout-option"
            :data-current="layoutName === name || undefined"
            @select="switchLayout(name)"
          >
            <span class="layout-icon">
              <component :is="LAYOUT_ICONS[name]" :size="16" />
            </span>
            <span class="layout-text">
              <span class="layout-name label-sm">{{ LAYOUT_LABELS[name] }}</span>
              <span class="layout-hint text-xs">{{ LAYOUT_HINTS[name] }}</span>
            </span>
          </DropdownMenuItem>
          <DropdownMenuSeparator class="layout-divider" />
          <DropdownMenuItem class="layout-option" @select="resetLayout">
            <span class="layout-icon">
              <PhArrowCounterClockwise :size="16" />
            </span>
            <span class="layout-text">
              <span class="layout-name label-sm">Reset layout</span>
              <span class="layout-hint text-xs">Back to this layout's default</span>
            </span>
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenuPortal>
    </DropdownMenuRoot>
  </Teleport>

  <div class="workspace">
    <DockviewVue class="dock" :components="panelComponents" :theme="theme" @ready="onReady" />
  </div>
</template>

<style scoped>
.workspace {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-height: 0;
}

/* the trigger teleports into the topbar; scoped styles still reach it there */
.layout-trigger {
  display: inline-flex;
  align-items: center;
  gap: var(--space-1);
}

:deep(.layout-list) {
  z-index: var(--z-dropdown);
  min-width: var(--width-xs);
  padding: var(--space-1);
  background: var(--color-bg-card);
  border: var(--border-thin) solid var(--color-border);
  border-radius: var(--radius-md);
  box-shadow: var(--shadow-md);
}

:deep(.layout-option) {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  padding: var(--space-2);
  color: var(--color-text-secondary);
  border-radius: var(--radius-sm);
  cursor: pointer;
  outline: none;
}

:deep(.layout-option[data-highlighted]) {
  color: var(--color-text);
  background: var(--color-surface-hover);
}

:deep(.layout-option[data-current]) {
  color: var(--color-primary);
}

:deep(.layout-option[data-current] .layout-hint) {
  color: var(--color-primary);
}

:deep(.layout-icon) {
  display: inline-flex;
  flex: none;
}

:deep(.layout-text) {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

:deep(.layout-hint) {
  color: var(--color-text-tertiary);
}

:deep(.layout-divider) {
  height: 1px;
  margin: var(--space-1) 0;
  background: var(--color-border-light);
}

:deep(span.layout-hint, span.layout-name) {
  line-height: 1;
}

.dock {
  flex: 1;
  min-height: 0;
}
</style>

<style>
.panel-fill {
  height: 100%;
}

.dockview-theme-fossencd {
  --dv-group-view-background-color: var(--color-bg-card);
  --dv-tabs-and-actions-container-background-color: var(--color-neutral-100);
  /*--dv-tabs-and-actions-container-height: 32px;*/
  --dv-tabs-and-actions-container-font-size: var(--label-sm);
  --dv-activegroup-visiblepanel-tab-background-color: var(--color-bg-card);
  --dv-activegroup-visiblepanel-tab-color: var(--color-text);
  --dv-activegroup-hiddenpanel-tab-background-color: transparent;
  --dv-activegroup-hiddenpanel-tab-color: var(--color-text-tertiary);
  --dv-inactivegroup-visiblepanel-tab-background-color: var(--color-bg-card);
  --dv-inactivegroup-visiblepanel-tab-color: var(--color-text-secondary);
  --dv-inactivegroup-hiddenpanel-tab-background-color: transparent;
  --dv-inactivegroup-hiddenpanel-tab-color: var(--color-text-tertiary);
  --dv-tab-divider-color: transparent;
  --dv-separator-border: var(--color-border);
  --dv-tab-border-radius: var(--radius-sm);
  --dv-tab-margin: 0;
  --dv-sash-color: transparent;
  --dv-active-sash-color: var(--color-accent-400);
  --dv-paneview-active-outline-color: var(--color-primary);
  --dv-drag-over-background-color: var(--color-primary-light);
  --dv-tabs-container-scrollbar-color: var(--color-neutral-300);
}

/* only the built-in "spaced" themes consume --dv-tab-border-radius, and tab
   padding is hardcoded upstream, so both need explicit rules here */
.dockview-theme-fossencd .dv-tab {
  /* attached cards: flush with the panel below, rounded on top only */
  border-radius: var(--dv-tab-border-radius) var(--dv-tab-border-radius) 0 0;
  padding: var(--space-1) var(--space-4);
}

/* the workspace panels are fixtures; closing one would strand it until a
   layout reset, so hide the per-tab close button */
.dockview-theme-fossencd .dv-tab .dv-default-tab .dv-default-tab-action {
  display: none;
}
</style>
