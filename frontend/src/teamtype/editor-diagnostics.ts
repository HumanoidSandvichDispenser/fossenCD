import type { Diagnostic } from '@codemirror/lint';
import type { Text } from '@codemirror/state';
import type { TypstDiagnostic } from '@/typst/diagnostics';

function toOffset(doc: Text, line: number, column: number): number {
  const lineNumber = Math.min(Math.max(line + 1, 1), doc.lines);
  const docLine = doc.line(lineNumber);
  return Math.min(docLine.from + Math.max(column, 0), docLine.to);
}

function isFor(diagnostic: TypstDiagnostic, file: string): boolean {
  if (diagnostic.package !== '') {
    return false;
  }
  const strip = (path: string) => path.replace(/^\/+/, '');
  return strip(diagnostic.path) === strip(file);
}

/**
 * Map the compiler's diagnostics into CodeMirror lint diagnostics for one
 * file.
 *
 * Diagnostics for other files and for package sources (which have no editable
 * source here) are dropped; positions are resolved against `doc` and clamped
 * to it so a diagnostic from a stale compile can never produce an out-of-range
 * mark.
 *
 * @param doc The document the markers will be applied to.
 * @param diagnostics The compiler's diagnostics for the whole build.
 * @param file The path of the file shown in `doc`.
 * @returns The lint diagnostics belonging to `file`.
 */
export function toEditorDiagnostics(
  doc: Text,
  diagnostics: TypstDiagnostic[],
  file: string,
): Diagnostic[] {
  const out: Diagnostic[] = [];
  for (const d of diagnostics) {
    if (!isFor(d, file)) {
      continue;
    }
    const from = toOffset(doc, d.fromLine, d.fromColumn);
    const to = Math.max(from, toOffset(doc, d.toLine, d.toColumn));
    out.push({ from, to, severity: d.severity, message: d.message });
  }
  return out;
}
