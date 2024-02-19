import type { createPeaksModule } from './createPeaksModule';

export const installZoomControls = ({
  state,
  commandEmitter,
  eventEmitter,
}: ReturnType<typeof createPeaksModule>) => {
  eventEmitter.on('ready', (peaks) => {
    const zoomLevel = state.select((st) => st.zoom.zoomLevel);
    const duration = peaks.player.getDuration();

    commandEmitter.emit(
      'setZoomLevel',
      Math.min(zoomLevel, Math.floor(duration))
    );

    commandEmitter.on('setZoomLevel', (seconds) => {
      const zoomview = peaks.views.getView('zoomview');
      zoomview?.setZoom({ seconds });

      // @todo: see how to convert peaks' zoom level to seconds and
      // hook into peaks zoom.update event
      eventEmitter.emit('zoomLevelChanged', seconds);
    });
  });
};
