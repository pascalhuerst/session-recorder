import { ref, watch } from 'vue';
import type { createPeaksCanvas } from './createPeaksCanvas';

export const createAmplitudeControls = ({
  peaks,
}: ReturnType<typeof createPeaksCanvas>) => {
  const amplitudeScale = ref(0.6);
  const amplitudeStep = ref(0.1);

  const adjustAmplitudeScale = (step: number) => {
    amplitudeScale.value = Math.max(
      0,
      (amplitudeScale.value * 100 + step * 100) / 100
    );
  };

  // const zoomview = peaks.value?.views.getView('zoomview');
  // zoomview?.setAmplitudeScale(zoomviewAmpZoom.value);

  watch(
    [amplitudeScale, peaks],
    () => {
      const overview = peaks.value?.views.getView('overview');
      overview?.setAmplitudeScale(amplitudeScale.value);

      const zoomview = peaks.value?.views.getView('zoomview');
      zoomview?.setAmplitudeScale(amplitudeScale.value);
    },
    { immediate: true }
  );

  return {
    amplitudeScale,
    amplitudeStep,
    adjustAmplitudeScale,
  };
};
