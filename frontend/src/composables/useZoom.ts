import { computed, ref } from "vue";

export function useZoom(initialZoom = 1) {
  const zoom = ref(initialZoom);

  const minZoom = ref(0.5);

  const maxZoom = ref(3);

  const zoomStep = ref(0.1);

  const zoomLabel = computed(() => `${Math.round(zoom.value * 100)}%`);

  const canZoomOut = computed(() => zoom.value > minZoom.value);

  const canZoomIn = computed(() => zoom.value < maxZoom.value);

  function zoomBy(delta: number) {
    const targetZoom = Math.round((zoom.value + delta) * 100) / 100;
    zoom.value = Math.min(Math.max(targetZoom, minZoom.value), maxZoom.value);
  }

  function zoomIn() {
    zoomBy(zoomStep.value);
  }

  function zoomOut() {
    zoomBy(-zoomStep.value);
  }

  function resetZoom() {
    zoom.value = initialZoom;
  }

  return {
    zoom,
    minZoom,
    maxZoom,
    zoomStep,
    zoomLabel,
    canZoomOut,
    canZoomIn,
    zoomBy,
    zoomIn,
    zoomOut,
    resetZoom,
  }
}
