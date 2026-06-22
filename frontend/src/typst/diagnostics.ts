export type DiagnosticSeverity = 'error' | 'warning';

/**
 * Normalized diagnostics emitted by the Typst compiler, with parsed source
 * ranges.
 */
export interface TypstDiagnostic {
  package: string;
  path: string;
  severity: DiagnosticSeverity;
  fromLine: number;
  fromColumn: number;
  toLine: number;
  toColumn: number;
  message: string;
}

interface RawDiagnostic {
  package: string;
  path: string;
  severity: string;
  range: string;
  message: string;
}

function parsePoint(point: string): [line: number, column: number] | null {
  const [line, column] = point.split(':').map(Number);
  if (line === undefined || column === undefined || Number.isNaN(line) || Number.isNaN(column)) {
    return null;
  }
  return [line, column];
}

function parseRange(range: string): Pick<
  TypstDiagnostic,
  'fromLine' | 'fromColumn' | 'toLine' | 'toColumn'
> | null {
  const dash = range.indexOf('-');
  const startStr = dash === -1 ? range : range.slice(0, dash);
  const endStr = dash === -1 ? range : range.slice(dash + 1);
  const start = parsePoint(startStr);
  const end = parsePoint(endStr);

  if (start === null || end === null) {
    return null;
  }

  return {
    fromLine: start[0],
    fromColumn: start[1],
    toLine: end[0],
    toColumn: end[1]
  };
}

/**
 * Convert raw diagnostics from the compiler (with string ranges) into
 * TypstDiagnostics with parsed ranges. Invalid diagnostics (e.g. with unparsable
 * ranges) are dropped.
 *
 * @param raw The raw diagnostics to convert.
 */
export function toTypstDiagnostics(raw: RawDiagnostic[] | undefined): TypstDiagnostic[] {
  if (raw === undefined) {
    return [];
  }
  const out: TypstDiagnostic[] = [];
  for (const d of raw) {
    const range = parseRange(d.range);
    if (range === null) {
      continue;
    }
    out.push({
      package: d.package,
      path: d.path,
      severity: d.severity === 'error' ? 'error' : 'warning',
      message: d.message,
      ...range,
    });
  }
  return out;
}

/**
 * Format a Typst diagnostic into a human-readable string.
 */
export function formatDiagnostic(d: TypstDiagnostic): string {
  const loc = d.package ? `${d.package}@${d.path}` : d.path;
  return `${loc}:${d.fromLine + 1}:${d.fromColumn + 1}: ${d.severity}: ${d.message}`;
}
