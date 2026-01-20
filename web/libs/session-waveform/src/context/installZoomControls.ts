import type { createPeaksModule } from './createPeaksModule';
import type { PeaksInstance } from 'peaks.js';

export const installZoomControls = ({
  state,
  commandEmitter,
  eventEmitter,
}: ReturnType<typeof createPeaksModule>) => {
  let peaksInstance: PeaksInstance | null = null;

  const applyZoom = (seconds: number) => {
    if (!peaksInstance) return;
    const zoomview = peaksInstance.views.getView('zoomview');
    zoomview?.setZoom({ seconds });
    eventEmitter.emit('zoomLevelChanged', seconds);
  };

  eventEmitter.on('ready', (peaks) => {
    peaksInstance = peaks;

    commandEmitter.on('setZoomLevel', applyZoom);

    // Set initial zoom to fit entire waveform, capped at 2 hours (7200 seconds)
    const duration = Math.floor(peaks.player.getDuration() || 256);
    const maxZoom = 7200; // 2 hours in seconds
    const initialZoom = Math.min(duration, maxZoom);

    // Update state with initial zoom
    state.update((prev) => ({
      ...prev,
      zoom: {
        ...prev.zoom,
        zoomLevel: initialZoom,
      },
    }));

    // Apply zoom if zoomview exists
    applyZoom(initialZoom);
  });

  // Re-apply zoom when view is expanded (zoomview is recreated)
  eventEmitter.on('expandedChanged', (expanded) => {
    if (expanded) {
      // Small delay to ensure zoomview is fully created
      requestAnimationFrame(() => {
        const currentZoom = state.select((st) => st.zoom.zoomLevel);
        applyZoom(currentZoom);
      });
    }
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
