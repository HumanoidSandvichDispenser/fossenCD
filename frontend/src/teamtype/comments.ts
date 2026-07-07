import { StateEffect, StateField } from '@codemirror/state';
import type { EditorState, Extension } from '@codemirror/state';
import { Decoration, EditorView, showTooltip, WidgetType } from '@codemirror/view';
import type { DecorationSet, Tooltip } from '@codemirror/view';

/** A commented span in editor (UTF-16) offsets. */
export interface CommentSpan {
  id: string;
  from: number;
  to: number;
  orphaned: boolean;
}

/**
 * A comment anchor's position: the document-space y-coordinate of the line
 * holding its span start. The review panel lays cards out against these and
 * converts to screen space with [`EditorViewport`]. The id `__draft` marks
 * the pending draft's selection.
 */
export interface CommentAnchor {
  id: string;
  y: number;
}

/** Where the editor's scroll viewport sits, for document-to-screen mapping. */
export interface EditorViewport {
  /** The document top's screen y-coordinate (moves as the editor scrolls);
   * screen position of an anchor = `documentTop + anchor.y`. */
  documentTop: number;
}

/** Replace the highlighted spans (dispatched on every comments tick). */
export const setCommentSpans = StateEffect.define<CommentSpan[]>();

/** Change which thread renders as active. */
export const setActiveComment = StateEffect.define<string | null>();

interface CommentState {
  spans: CommentSpan[];
  active: string | null;
  deco: DecorationSet;
}

function buildDecorations(spans: CommentSpan[], active: string | null): DecorationSet {
  const marks = spans
    .filter((s) => s.from < s.to)
    .map((s) =>
      Decoration.mark({
        class:
          'cm-comment-span' +
          (s.id === active ? ' cm-comment-span-active' : '') +
          (s.orphaned ? ' cm-comment-span-orphaned' : ''),
      }).range(s.from, s.to),
    );
  // orphaned anchors collapse to a point; mark the character after the
  // surviving position so the thread stays discoverable
  for (const s of spans) {
    if (s.from >= s.to) {
      marks.push(
        Decoration.widget({
          widget: new OrphanWidget(s.id === active),
          side: 1,
        }).range(s.from),
      );
    }
  }
  return Decoration.set(
    marks.sort((a, b) => a.from - b.from || a.to - b.to),
    true,
  );
}

class OrphanWidget extends WidgetType {
  constructor(private active: boolean) {
    super();
  }

  toDOM(): HTMLElement {
    const el = document.createElement('span');
    el.className = 'cm-comment-orphan' + (this.active ? ' cm-comment-span-active' : '');
    el.textContent = '▏';
    return el;
  }

  override eq(other: OrphanWidget): boolean {
    return other.active === this.active;
  }
}

export interface CommentHandlers {
  /** A highlighted span was clicked. */
  onActivate: (id: string) => void;
  /** The user asked to comment on the selection (UTF-16 offsets). */
  onDraft: (from: number, to: number) => void;
}

/**
 * Highlight commented spans and offer a comment button on non-empty
 * selections. Span positions arrive via [`setCommentSpans`] and map through
 * local edits between ticks; the active thread is set via
 * [`setActiveComment`].
 */
export function comments(handlers: CommentHandlers): Extension {
  const field = StateField.define<CommentState>({
    create: () => ({ spans: [], active: null, deco: Decoration.none }),
    update(value, tr) {
      let { spans, active } = value;
      let changed = false;
      for (const effect of tr.effects) {
        if (effect.is(setCommentSpans)) {
          spans = effect.value;
          changed = true;
        } else if (effect.is(setActiveComment)) {
          active = effect.value;
          changed = true;
        }
      }
      if (tr.docChanged) {
        spans = spans.map((s) => ({
          ...s,
          from: tr.changes.mapPos(s.from),
          to: tr.changes.mapPos(s.to),
        }));
        changed = true;
      }
      if (!changed) {
        return value;
      }
      return { spans, active, deco: buildDecorations(spans, active) };
    },
    provide: (f) => EditorView.decorations.from(f, (v) => v.deco),
  });

  const clicks = EditorView.domEventHandlers({
    mousedown(event, view) {
      const pos = view.posAtCoords({ x: event.clientX, y: event.clientY });
      if (pos === null) {
        return false;
      }
      const { spans } = view.state.field(field);
      // the innermost (shortest) span wins when comments overlap
      const hit = spans
        .filter((s) => pos >= s.from && (pos < s.to || (s.from >= s.to && pos === s.from)))
        .sort((a, b) => a.to - a.from - (b.to - b.from))[0];
      if (hit) {
        handlers.onActivate(hit.id);
      }
      return false;
    },
  });

  const selectionButton = StateField.define<Tooltip | null>({
    create: (state) => draftTooltip(state, handlers),
    update(value, tr) {
      if (!tr.docChanged && !tr.selection) {
        return value;
      }
      return draftTooltip(tr.state, handlers);
    },
    provide: (f) => showTooltip.from(f),
  });

  return [field, clicks, selectionButton];
}

function draftTooltip(state: EditorState, handlers: CommentHandlers): Tooltip | null {
  const range = state.selection.main;
  if (range.empty) {
    return null;
  }
  return {
    pos: range.head,
    above: true,
    arrow: false,
    create: () => {
      const button = document.createElement('button');
      button.className = 'cm-comment-add';
      button.type = 'button';
      button.textContent = '+ Comment';
      // mousedown, so the button wins over the editor clearing the selection
      button.addEventListener('mousedown', (event) => {
        event.preventDefault();
        handlers.onDraft(range.from, range.to);
      });
      const dom = document.createElement('div');
      dom.className = 'cm-comment-add-tooltip';
      dom.appendChild(button);
      return { dom };
    },
  };
}
