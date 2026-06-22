export type SplitAxis = 'x' | 'y';

export function useSplitDrag(
  axis: SplitAxis,
  onPercent: (percent: number) => void,
  minPx = 160,
) {
  let geom: { start: number; size: number } | null = null;

  function onMove(event: PointerEvent) {
    if (geom === null) {
      return;
    }

    const { start, size } = geom;
    const min = Math.min(minPx, size / 2);
    const pointer = axis === 'x' ? event.clientX : event.clientY;
    const leading = Math.max(min, Math.min(size - min, pointer - start));

    onPercent((leading / size) * 100);
  }

  function stop() {
    geom = null;
    document.body.style.cursor = '';
    document.body.style.userSelect = '';
    window.removeEventListener('pointermove', onMove);
    window.removeEventListener('pointerup', stop);
  }

  function start(container: HTMLElement) {
    const rect = container.getBoundingClientRect();
    geom =
      axis === 'x'
        ? { start: rect.left, size: rect.width }
        : { start: rect.top, size: rect.height };
    document.body.style.cursor = axis === 'x' ? 'col-resize' : 'row-resize';
    document.body.style.userSelect = 'none';
    window.addEventListener('pointermove', onMove);
    window.addEventListener('pointerup', stop);
  }

  return { start, stop };
}
