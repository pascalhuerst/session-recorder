import type { createPeaksModule } from './createPeaksModule';
import type { PeaksInstance } from 'peaks.js';

export const installViewControls = ({
  state,
  commandEmitter,
  eventEmitter,
}: ReturnType<typeof createPeaksModule>) => {
  let zoomviewContainer: HTMLElement | null = null;
  let overviewContainer: HTMLElement | null = null;
  let isExpanded = true;
  let resizeObserver: ResizeObserver | null = null;
  let peaksInstance: PeaksInstance | null = null;

  // Debounce resize handling to avoid excessive redraws
  let resizeTimeout: ReturnType<typeof setTimeout> | null = null;
  const handleResize = () => {
    if (resizeTimeout) {
      clearTimeout(resizeTimeout);
    }
    resizeTimeout = setTimeout(() => {
      if (!peaksInstance) return;

      const overviewView = peaksInstance.views.getView('overview');
      const zoomView = peaksInstance.views.getView('zoomview');

      if (overviewView) {
        overviewView.fitToContainer();
      }
      if (zoomView && isExpanded) {
        zoomView.fitToContainer();
      }
    }, 100);
  };

  // Capture container references on mount
  commandEmitter.on('mount', (elements) => {
    zoomviewContainer = elements.zoomview;
    overviewContainer = elements.overview;
  });

  eventEmitter.on('ready', (peaks) => {
    peaksInstance = peaks;

    // Set up ResizeObserver to handle window/container resizes
    if (overviewContainer) {
      resizeObserver = new ResizeObserver(handleResize);
      resizeObserver.observe(overviewContainer);
    }

    // Start collapsed if initial state requests it
    const initialExpanded = state.select((st) => st.expanded);
    if (!initialExpanded) {
      peaks.views.destroyZoomview();
      isExpanded = false;
      eventEmitter.emit('expandedChanged', false);
    }

    commandEmitter.on('setExpanded', (expanded) => {
      if (expanded === isExpanded) return;

      if (expanded) {
        // Expanding: need to create zoomview
        if (zoomviewContainer) {
          peaks.views.createZoomview(zoomviewContainer);
          isExpanded = true;
          eventEmitter.emit('expandedChanged', true);
          // Fit to container after creating zoomview
          requestAnimationFrame(() => {
            const zoomView = peaks.views.getView('zoomview');
            if (zoomView) {
              zoomView.fitToContainer();
            }
          });
        }
      } else {
        // Collapsing: destroy zoomview
        peaks.views.destroyZoomview();
        isExpanded = false;
        eventEmitter.emit('expandedChanged', false);
      }
    });

    // Clean up on destroy
    commandEmitter.on('destroy', () => {
      if (resizeObserver) {
        resizeObserver.disconnect();
        resizeObserver = null;
      }
      if (resizeTimeout) {
        clearTimeout(resizeTimeout);
        resizeTimeout = null;
      }
      peaksInstance = null;
    });
  });

  eventEmitter.on('expandedChanged', (expanded) => {
    state.update((prev) => ({
      ...prev,
      expanded,
    }));
  });
};
