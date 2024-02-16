import { ref } from 'vue';
import type { createPeaksCanvas } from './createPeaksCanvas';

export const createZoomControls = ({
  peaks,
  commandEmitter,
  eventEmitter,
}: ReturnType<typeof createPeaksCanvas>) => {
  const zoomLevel = ref(300);
  const zoomStep = ref(60);

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
        Math.min(zoomLevel.value, Math.floor(peaks.value.player.getDuration()))
      );
    }
  });

  eventEmitter.on('zoomLevelChanged', (value) => {
    zoomLevel.value = value;
  });

  return { zoomLevel, zoomStep };
};
