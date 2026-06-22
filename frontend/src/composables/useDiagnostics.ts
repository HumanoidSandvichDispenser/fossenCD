import { ref } from 'vue';
import type { TypstDiagnostic } from '@/typst/diagnostics';

export function useDiagnostics() {
  const diagnostics = ref<TypstDiagnostic[]>([]);

  function set(next: TypstDiagnostic[]) {
    diagnostics.value = next;
  }

  function clear() {
    diagnostics.value = [];
  }

  return { diagnostics, set, clear };
}
