import { ref, watch } from 'vue';
import type { createPeaksCanvas } from './createPeaksCanvas';

export const createZoomControls = ({
  peaks,
}: ReturnType<typeof createPeaksCanvas>) => {
  const zoomLevel = ref(300);
  const zoomStep = ref(100);

  const adjustZoom = (seconds: number) => {
    zoomLevel.value = Math.max(0, zoomLevel.value + seconds);
  };

  watch(
    [zoomLevel, peaks],
    () => {
      const zoomview = peaks.value?.views.getView('zoomview');
      zoomview?.setZoom({ seconds: zoomLevel.value });
    },
    { immediate: true }
  );

  return { zoomLevel, zoomStep, adjustZoom };
};
