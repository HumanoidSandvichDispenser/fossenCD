import { StateEffect, StateField } from '@codemirror/state';
import type { Extension } from '@codemirror/state';
import { Decoration, EditorView, WidgetType } from '@codemirror/view';
import type { DecorationSet } from '@codemirror/view';

/** One peer's cursor in editor (UTF-16) offsets, for the current file. */
export interface PeerCursorSpan {
  /** The peer's endpoint id, used as a stable key and color seed. */
  id: string;
  /** Display name, or null to fall back to a shortened id. */
  name: string | null;
  /** Selection anchor and head; `from`/`to` are the ordered span. */
  from: number;
  to: number;
  head: number;
}

/** Replace the rendered peer cursors (dispatched on every cursor tick). */
export const setPeerCursors = StateEffect.define<PeerCursorSpan[]>();

/** A stable hue (0-359) from a peer id, so each peer keeps one color. */
function hueOf(id: string): number {
  let hash = 0;
  for (let i = 0; i < id.length; i++) {
    hash = (hash * 31 + id.charCodeAt(i)) | 0;
  }
  return Math.abs(hash) % 360;
}

class CaretWidget extends WidgetType {
  constructor(
    private readonly label: string,
    private readonly hue: string,
  ) {
    super();
  }

  toDOM(): HTMLElement {
    const caret = document.createElement('span');
    caret.className = 'cm-peer-caret';
    caret.style.borderLeftColor = this.hue;

    const label = document.createElement('span');
    label.className = 'cm-peer-caret-label';
    label.style.backgroundColor = this.hue;
    label.textContent = this.label;
    caret.appendChild(label);

    return caret;
  }

  override eq(other: CaretWidget): boolean {
    return other.label === this.label && other.hue === this.hue;
  }

  // the caret sits between characters, so it never covers document text
  override get estimatedHeight(): number {
    return -1;
  }
}

function buildDecorations(spans: PeerCursorSpan[]): DecorationSet {
  const decos = [];
  for (const s of spans) {
    const h = hueOf(s.id);
    const solid = `hsl(${h} 70% 45%)`;
    if (s.from < s.to) {
      decos.push(
        Decoration.mark({
          class: 'cm-peer-selection',
          attributes: { style: `background-color: hsl(${h} 70% 45% / 0.25)` },
        }).range(s.from, s.to),
      );
    }
    // the caret marks the head; side +1 keeps it after any selection mark
    decos.push(
      Decoration.widget({
        widget: new CaretWidget(s.name ?? s.id.slice(0, 6), solid),
        side: 1,
      }).range(s.head),
    );
  }
  return Decoration.set(
    decos.sort((a, b) => a.from - b.from),
    true,
  );
}

/**
 * Render peer cursors and selections. Spans arrive via [`setPeerCursors`] and
 * map through local edits between ticks, so a caret drifts with your own typing
 * until the next resolved snapshot corrects it.
 */
export function peerCursors(): Extension {
  return StateField.define<DecorationSet>({
    create: () => Decoration.none,
    update(value, tr) {
      for (const effect of tr.effects) {
        if (effect.is(setPeerCursors)) {
          return buildDecorations(effect.value);
        }
      }
      return tr.docChanged ? value.map(tr.changes) : value;
    },
    provide: (f) => EditorView.decorations.from(f),
  });
}
