import type { createPeaksModule } from './createPeaksModule';

export const installZoomControls = ({
  state,
  commandEmitter,
  eventEmitter,
}: ReturnType<typeof createPeaksModule>) => {
  eventEmitter.on('ready', (peaks) => {
    commandEmitter.on('setZoomLevel', (seconds) => {
      const zoomview = peaks.views.getView('zoomview');
      zoomview?.setZoom({ seconds });

      // @todo: see how to convert peaks' zoom level to seconds and
      // hook into peaks zoom.update event
      eventEmitter.emit('zoomLevelChanged', seconds);
    });

    commandEmitter.emit(
      'setZoomLevel',
      Math.min(
        state.select((st) => st.zoom.zoomLevel),
        Math.floor(peaks.player.getDuration() || 256)
      )
    );
  });

  eventEmitter.on('zoomLevelChanged', (seconds) => {
    state.update((prev) => ({
      ...prev,
      zoom: {
        ...prev.zoom,
        zoomLevel: seconds,
      },
    }));
  });
};
