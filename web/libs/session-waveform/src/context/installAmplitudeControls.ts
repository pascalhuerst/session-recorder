import type { createPeaksModule } from './createPeaksModule';

export const installAmplitudeControls = ({
  state,
  peaks,
  commandEmitter,
  eventEmitter,
}: ReturnType<typeof createPeaksModule>) => {
  eventEmitter.on('ready', () => {
    commandEmitter.emit('setAmplitudeScale', state.$store.get().amplitudeScale);
  });

  commandEmitter.on('setAmplitudeScale', (value) => {
    const overview = peaks.value?.views.getView('overview');
    overview?.setAmplitudeScale(value);

    const zoomview = peaks.value?.views.getView('zoomview');
    zoomview?.setAmplitudeScale(value);

    eventEmitter.emit('amplitudeScaleChanged', value);
  });

  eventEmitter.on('amplitudeScaleChanged', (value: number) => {
    state.update((prev) => ({
      ...prev,
      amplitudeScale: value,
    }));
  });
};
