export interface Position {
  line: number; // 0-based
  character: number; // 0-based, code points
}

export interface Range {
  start: Position; // anchor
  end: Position; // active; backward selection has start > end
}

export interface CursorState {
  name: string | null;
  file_path: string;
  ranges: Range[];
}

export interface EphemeralMessage {
  cursor_id: string;
  sequence_number: number;
  cursor_state: CursorState;
}

export interface NodeInfo {
  node_id: string;
  passphrase: string;
}

// Internally-tagged on the Rust side (`#[serde(tag = "kind")]`).
export type DisconnectReason =
  | { kind: 'PeerDisconnected' }
  | { kind: 'ReceiveError'; error: string }
  | { kind: 'HandlerError'; error: string };

// Offset-space delta. Retain/Delete counts are in code points.
export type WireOp = { Retain: number } | { Insert: string } | { Delete: number };
export type WireDelta = WireOp[];
