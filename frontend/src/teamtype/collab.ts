import { Annotation } from '@codemirror/state';
import type { ChangeSet, ChangeSpec, Extension, Text } from '@codemirror/state';
import { EditorView } from '@codemirror/view';
import type { ViewUpdate } from '@codemirror/view';

import { advanceCodePoints, codePointLength } from './codepoints';
import type { WireDelta } from '@sandvichxyz/okayeg-wasm';

export const remoteEdit = Annotation.define<boolean>();

function advance(doc: Text, from: number, n: number): number {
  if (n <= 0) {
    return from;
  }
  return from + advanceCodePoints(doc.sliceString(from, Math.min(doc.length, from + 2 * n)), 0, n);
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

/**
 * Apply a wire delta to a plain string, mirroring okayeg's
 * `WireDelta::apply_to`. Offsets are counted in Unicode code points.
 */
export function applyWireDelta(text: string, ops: WireDelta): string {
  let pos = 0;
  let out = '';
  for (const op of ops) {
    if ('Retain' in op) {
      const to = advanceCodePoints(text, pos, op.Retain);
      out += text.slice(pos, to);
      pos = to;
    } else if ('Delete' in op) {
      pos = advanceCodePoints(text, pos, op.Delete);
    } else {
      out += op.Insert;
    }
  }
  return out + text.slice(pos);
}

// Convert a wire delta into CodeMirror changes against the current doc. Kept for
// the day okayeg-wasm emits incremental onEdit deltas; pass 1 syncs full
// snapshots via onContent instead, so this is unused for now.
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

export interface CollabHandlers {
  onEdit: (delta: WireDelta) => void;
}

// A CodeMirror extension that turns local document edits into wire deltas and
// hands them to `onEdit`. Remote edits arrive as full snapshots via the store's
// onContent, which reseed the editor directly, so no remote-delta dispatch
// happens here.
export function collab(handlers: CollabHandlers): Extension {
  const listener = EditorView.updateListener.of((update: ViewUpdate) => {
    // ignore updates without any transactions (e.g. a programmatic setState
    // reseed from a remote snapshot)
    if (update.transactions.length === 0) {
      return;
    }

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
  });

  return listener;
}