import type { createPeaksModule } from './createPeaksModule';

export const installAmplitudeControls = ({
  state,
  commandEmitter,
  eventEmitter,
}: ReturnType<typeof createPeaksModule>) => {
  eventEmitter.on('ready', (peaks) => {
    commandEmitter.on('setAmplitudeScale', (value) => {
      const overview = peaks.views.getView('overview');
      overview?.setAmplitudeScale(value);

      const zoomview = peaks.views.getView('zoomview');
      zoomview?.setAmplitudeScale(value);

      eventEmitter.emit('amplitudeScaleChanged', value);
    });

    commandEmitter.emit(
      'setAmplitudeScale',
      state.select((st) => st.amplitude.amplitudeScale)
    );
  });

  eventEmitter.on('amplitudeScaleChanged', (value: number) => {
    state.update((prev) => ({
      ...prev,
      amplitude: {
        ...prev.amplitude,
        amplitudeScale: value,
      },
    }));
  });
};
