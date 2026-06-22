import { ref } from 'vue';

export function useEditorJump() {
  const target = ref<{ line: number } | null>(null);

  function jumpTo(line: number) {
    target.value = { line };
  }

  return { target, jumpTo };
}
