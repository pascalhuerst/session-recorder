import type { createPeaksModule } from './createPeaksModule';

export const installZoomControls = ({
  state,
  peaks,
  commandEmitter,
  eventEmitter,
}: ReturnType<typeof createPeaksModule>) => {
  commandEmitter.on('setZoomLevel', (seconds) => {
    const zoomview = peaks.value?.views.getView('zoomview');
    zoomview?.setZoom({ seconds });

    // @todo: see how to convert peaks' zoom level to seconds and
    // hook into peaks zoom.update event
    eventEmitter.emit('zoomLevelChanged', seconds);
  });

  eventEmitter.on('ready', () => {
    if (peaks.value?.player) {
      commandEmitter.emit(
        'setZoomLevel',
        Math.min(
          state.$store.get().zoomLevel,
          Math.floor(peaks.value.player.getDuration())
        )
      );
    }
  });
};
