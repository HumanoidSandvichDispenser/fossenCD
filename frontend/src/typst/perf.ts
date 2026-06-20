// Lightweight render-timing harness for the Typst preview: measures where
// wall-clock time goes on each compile, per phase.

/** How many recent samples to keep per phase for the rolling summary. */
const MAX_SAMPLES = 200;

interface PerfState {
  /** Phase label -> recent durations in milliseconds (most recent last). */
  samples: Map<string, number[]>;
  /** When true, each completed cycle is also logged to the console. */
  logToConsole: boolean;
}

// Anchor all mutable state on globalThis. Under Vite HMR (and
// when './perf' vs '@/typst/perf' resolve to distinct module URLs) this file can
// be evaluated more than once; a module-local `samples` Map would then split
// into several stores, and `window.__typstPerf` would point at whichever copy
// loaded last while `measure()` wrote to a different one. Sharing one object
// across all instances keeps every writer and reader on the same store.
const globalKey = '__typstPerfState';
const g = globalThis as unknown as Record<string, PerfState | undefined>;
const state: PerfState = g[globalKey] ?? (g[globalKey] = {
  samples: new Map<string, number[]>(),
  logToConsole: true,
});
const samples = state.samples;

function record(label: string, ms: number): void {
  let arr = samples.get(label);
  if (arr === undefined) {
    arr = [];
    samples.set(label, arr);
  }
  arr.push(ms);
  if (arr.length > MAX_SAMPLES) {
    arr.shift();
  }
}

/**
 * Record a pre-measured duration under `label`. Use when the timed region
 * can't be expressed as a single callback (e.g. spans a try/finally that also
 * needs its own error handling).
 */
export function recordSample(label: string, ms: number): void {
  record(label, ms);
}

/**
 * Time an async operation and record it under `label`. Returns whatever the
 * operation returns, so call sites can wrap inline:
 *
 *   const out = await measure('compile', () => compiler.compile(...));
 */
export async function measure<T>(label: string, fn: () => Promise<T> | T): Promise<T> {
  const start = performance.now();
  try {
    return await fn();
  } finally {
    const ms = performance.now() - start;
    record(label, ms);
    // Surface the phase on the devtools Performance timeline.
    try {
      performance.measure(`typst ${label}`, { start: start, end: start + ms });
    } catch {
      // performance.measure isn't supported everywhere; the in-memory sample is
      // the source of truth regardless.
    }
  }
}

/** Compute summary statistics for one phase's samples. */
function summarize(arr: number[]) {
  const sorted = [...arr].sort((a, b) => a - b);
  const n = sorted.length;
  const pct = (p: number) => sorted[Math.min(n - 1, Math.floor((p / 100) * n))] ?? 0;
  const sum = sorted.reduce((a, b) => a + b, 0);
  return {
    count: n,
    last: arr[arr.length - 1] ?? 0,
    mean: sum / n,
    p50: pct(50),
    p95: pct(95),
    max: sorted[n - 1] ?? 0,
  };
}

/** Round numbers in a summary for readable console output. */
function round(s: ReturnType<typeof summarize>) {
  const r = (x: number) => Math.round(x * 10) / 10;
  return {
    count: s.count,
    last: r(s.last),
    mean: r(s.mean),
    p50: r(s.p50),
    p95: r(s.p95),
    max: r(s.max),
  };
}

/** Print a table of every recorded phase to the console. */
export function report(): Record<string, ReturnType<typeof round>> {
  const out: Record<string, ReturnType<typeof round>> = {};
  for (const [label, arr] of samples) {
    if (arr.length > 0) {
      out[label] = round(summarize(arr));
    }
  }
  // console.table renders this as a sortable grid in devtools.
  console.table(out);
  return out;
}

/** Discard all recorded samples (e.g. before a fresh measurement run). */
export function reset(): void {
  samples.clear();
}

/** Toggle per-cycle console logging. */
export function setLogging(on: boolean): void {
  state.logToConsole = on;
}

export function isLogging(): boolean {
  return state.logToConsole;
}

// Expose a handle for ad-hoc measurement from the devtools console.
if (typeof window !== 'undefined') {
  (window as unknown as { __typstPerf: unknown }).__typstPerf = {
    report,
    reset,
    setLogging,
    samples,
  };
}
