import { Annotation, StateEffect, StateField } from '@codemirror/state';
import type { ChangeSet, ChangeSpec, Extension, Text } from '@codemirror/state';
import { Decoration, EditorView, WidgetType } from '@codemirror/view';
import type { DecorationSet, ViewUpdate } from '@codemirror/view';

import { advanceCodePoints, codePointLength } from './codepoints';
import type { Position, Range, WireDelta } from './types';

export const remoteEdit = Annotation.define<boolean>();

function advance(doc: Text, from: number, n: number): number {
  if (n <= 0) {
    return from;
  }
  return from + advanceCodePoints(doc.sliceString(from, Math.min(doc.length, from + 2 * n)), 0, n);
}

export function offsetToPosition(doc: Text, offset: number): Position {
  const line = doc.lineAt(offset);
  return {
    line: line.number - 1,
    character: codePointLength(doc.sliceString(line.from, offset)),
  };
}

export function positionToOffset(doc: Text, pos: Position): number {
  const line = doc.line(Math.min(pos.line + 1, doc.lines));
  return advance(doc, line.from, pos.character);
}

export function changesToWireDelta(startDoc: Text, changes: ChangeSet): WireDelta {
  const ops: WireDelta = [];
  let posA = 0;
  changes.iterChanges((fromA, toA, _fromB, _toB, inserted) => {
    const retain = codePointLength(startDoc.sliceString(posA, fromA));
    if (retain > 0) {
      ops.push({ Retain: retain });
    }
    const del = codePointLength(startDoc.sliceString(fromA, toA));
    if (del > 0) {
      ops.push({ Delete: del });
    }
    const ins = inserted.toString();
    if (ins.length > 0) {
      ops.push({ Insert: ins });
    }
    posA = toA;
  });
  return ops;
}

export function wireDeltaToChanges(doc: Text, ops: WireDelta): ChangeSpec[] {
  const changes: ChangeSpec[] = [];
  let pos = 0;
  for (const op of ops) {
    if ('Retain' in op) {
      pos = advance(doc, pos, op.Retain);
    } else if ('Delete' in op) {
      const to = advance(doc, pos, op.Delete);
      changes.push({ from: pos, to });
      pos = to;
    } else {
      changes.push({ from: pos, insert: op.Insert });
    }
  }
  return changes;
}

export interface RemoteCursor {
  id: string;
  name: string;
  color: string;
  file: string;
  ranges: Range[];
}

export const setRemoteCursors = StateEffect.define<RemoteCursor[]>();

class CaretWidget extends WidgetType {
  constructor(
    readonly color: string,
    readonly name: string,
  ) {
    super();
  }
  eq(other: CaretWidget) {
    return other.color === this.color && other.name === this.name;
  }
  toDOM() {
    const el = document.createElement('span');
    el.className = 'tt-caret';
    el.style.color = this.color;
    el.style.setProperty('--tt-color', this.color);
    el.dataset.name = this.name;
    return el;
  }
  ignoreEvent() {
    return true;
  }
}

function buildCursors(doc: Text, cursors: RemoteCursor[]): DecorationSet {
  const ranges = [];
  for (const cursor of cursors) {
    for (const range of cursor.ranges) {
      const from = positionToOffset(doc, range.start);
      const to = positionToOffset(doc, range.end);
      const a = Math.min(from, to);
      const b = Math.max(from, to);
      if (b > a) {
        ranges.push(
          Decoration.mark({
            attributes: {
              style: `background-color:color-mix(in srgb, ${cursor.color} 22%, transparent)`,
            },
          }).range(a, b),
        );
      }
      ranges.push(
        Decoration.widget({ widget: new CaretWidget(cursor.color, cursor.name), side: 1 }).range(
          to,
        ),
      );
    }
  }
  return Decoration.set(ranges, true);
}

export const remoteCursorField = StateField.define<DecorationSet>({
  create() {
    return Decoration.none;
  },
  update(deco, tr) {
    deco = deco.map(tr.changes);
    for (const effect of tr.effects) {
      if (effect.is(setRemoteCursors)) {
        deco = buildCursors(tr.state.doc, effect.value);
      }
    }
    return deco;
  },
  provide: (field) => EditorView.decorations.from(field),
});

const cursorTheme = EditorView.baseTheme({
  '.tt-caret': {
    borderLeft: '2px solid',
    marginLeft: '-1px',
    position: 'relative',
  },
  '.tt-caret::after': {
    content: 'attr(data-name)',
    position: 'absolute',
    top: '-1.15em',
    left: '-1px',
    fontSize: '0.7em',
    lineHeight: '1.3',
    padding: '0 3px',
    borderRadius: '3px',
    whiteSpace: 'nowrap',
    color: 'white',
    background: 'var(--tt-color)',
  },
});

export interface CollabHandlers {
  onEdit: (delta: WireDelta) => void;
  onCursor: (ranges: Range[]) => void;
}

export function collab(handlers: CollabHandlers): Extension {
  const listener = EditorView.updateListener.of((update: ViewUpdate) => {
    const isRemote = update.transactions.some((tr) => tr.annotation(remoteEdit));
    if (isRemote) {
      return;
    }

    if (update.docChanged) {
      const delta = changesToWireDelta(update.startState.doc, update.changes);
      if (delta.length > 0) {
        handlers.onEdit(delta);
      }
    }
    if (update.selectionSet || update.docChanged) {
      const sel = update.state.selection.main;
      handlers.onCursor([
        {
          start: offsetToPosition(update.state.doc, sel.anchor),
          end: offsetToPosition(update.state.doc, sel.head),
        },
      ]);
    }
  });

  return [listener, remoteCursorField, cursorTheme];
}
