import { computed, ref } from "vue";

export type CompileState = "idle" | "compiling" | "done";

function formatDuration(ms: number): string {
  if (ms < 1000) {
    return `${ms.toPrecision(3)}ms`;
  }
  return `${(ms / 1000).toPrecision(3)} seconds`;
}

export function useCompileStatus() {
  const state = ref<CompileState>("idle");
  const durationMs = ref(0);

  const label = computed(() => {
    switch (state.value) {
      case "compiling":
        return "Compiling…";
      case "done":
        return `Compiled in ${formatDuration(durationMs.value)}`;
      default:
        return "";
    }
  });

  function startCompile() {
    state.value = "compiling";
  }

  function finishCompile(ms: number) {
    durationMs.value = ms;
    state.value = "done";
  }

  function reset() {
    state.value = "idle";
  }

  return { state, durationMs, label, startCompile, finishCompile, reset };
}
