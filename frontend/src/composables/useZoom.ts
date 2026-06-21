import { computed, ref, watch } from "vue";

export function useZoom(initialZoom = 1) {
  const zoom = ref(initialZoom);

  const minZoom = ref(0.5);

  const maxZoom = ref(3);

  const zoomStep = ref(0.1);

  const zoomLabel = computed(() => `${Math.round(zoom.value * 100)}%`);

  const canZoomOut = computed(() => zoom.value > minZoom.value);

  const canZoomIn = computed(() => zoom.value < maxZoom.value);

  watch(zoom, (value) => {
    const clamped = Math.min(Math.max(value, minZoom.value), maxZoom.value);
    const rounded = Math.round(clamped * 100) / 100;
    if (rounded !== value) {
      zoom.value = rounded;
    }
  });

  function zoomBy(delta: number) {
    zoom.value += delta;
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
