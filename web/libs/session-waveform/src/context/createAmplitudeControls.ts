import { ref } from 'vue';
import type { createPeaksCanvas } from './createPeaksCanvas';

export const createAmplitudeControls = ({
  peaks,
  commandEmitter,
  eventEmitter,
}: ReturnType<typeof createPeaksCanvas>) => {
  const amplitudeScale = ref(0.6);
  const amplitudeStep = ref(0.1);

  eventEmitter.on('ready', () => {
    commandEmitter.emit('setAmplitudeScale', amplitudeScale.value);
  });

  commandEmitter.on('setAmplitudeScale', (value) => {
    const overview = peaks.value?.views.getView('overview');
    overview?.setAmplitudeScale(amplitudeScale.value);

    const zoomview = peaks.value?.views.getView('zoomview');
    zoomview?.setAmplitudeScale(amplitudeScale.value);

    eventEmitter.emit('amplitudeScaleChanged', value);
  });

  eventEmitter.on('amplitudeScaleChanged', (value: number) => {
    amplitudeScale.value = value;
  });

  return {
    amplitudeScale,
    amplitudeStep,
  };
};
