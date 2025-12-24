import type { createPeaksModule } from './createPeaksModule';

export const installViewControls = ({
  state,
  commandEmitter,
  eventEmitter,
}: ReturnType<typeof createPeaksModule>) => {
  let zoomviewContainer: HTMLElement | null = null;
  let isExpanded = true;

  // Capture zoomview container reference on mount
  commandEmitter.on('mount', (elements) => {
    zoomviewContainer = elements.zoomview;
  });

  eventEmitter.on('ready', (peaks) => {
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
          const { theme } = state.get();
          peaks.views.createZoomview(zoomviewContainer, theme.zoomviewTheme);
          isExpanded = true;
          eventEmitter.emit('expandedChanged', true);
        }
      } else {
        // Collapsing: destroy zoomview
        peaks.views.destroyZoomview();
        isExpanded = false;
        eventEmitter.emit('expandedChanged', false);
      }
    });
  });

  eventEmitter.on('expandedChanged', (expanded) => {
    state.update((prev) => ({
      ...prev,
      expanded,
    }));
  });
};
