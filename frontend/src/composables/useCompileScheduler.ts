// Coalescing scheduler for an expensive, non-reentrant async task.
//
// Two scheduling modes share one overlap guard, so the task never runs
// concurrently with itself:
//   - `whenIdle()` runs the task as soon as it is free; if a run is in flight,
//     one run is queued and fired when the current run finishes.
//   - `debounced()` waits for a quiet period, then runs via `whenIdle()`.

export interface CompileScheduler {
  /** Run as soon as the task is idle; coalesce if a run is in flight. */
  whenIdle(): void;
  /** Run after `debounceMs` of quiet, then via {@link whenIdle}. */
  debounced(): void;
  /** Cancel any pending timer. Call on teardown. */
  dispose(): void;
}

export function useCompileScheduler(
  task: () => Promise<void>,
  debounceMs = 250,
): CompileScheduler {
  let timer: ReturnType<typeof setTimeout> | null = null;
  // true while `task` is in flight; the next request is deferred behind it
  let running = false;
  // a request that arrived mid-run, to fire once the current run settles
  let pending = false;

  async function run(): Promise<void> {
    running = true;
    try {
      await task();
    } finally {
      running = false;
      if (pending) {
        pending = false;
        void run();
      }
    }
  }

  function clearTimer(): void {
    if (timer !== null) {
      clearTimeout(timer);
      timer = null;
    }
  }

  function whenIdle(): void {
    clearTimer();
    if (running) {
      pending = true;
    } else {
      void run();
    }
  }

  function debounced(): void {
    clearTimer();
    timer = setTimeout(() => {
      timer = null;
      whenIdle();
    }, debounceMs);
  }

  return { whenIdle, debounced, dispose: clearTimer };
}
