import { CustomSegmentMarker } from '../elements/CustomSegmentMarker';
import type { PeaksOptions, SegmentMarker } from 'peaks.js';
import Peaks from 'peaks.js';
import type { createPeaksModule } from './createPeaksModule';

/**
 * Checks if an element has visible dimensions
 */
const hasVisibleDimensions = (element: HTMLElement): boolean => {
  const rect = element.getBoundingClientRect();
  return rect.width > 0 && rect.height > 0;
};

/**
 * Waits for an element to have visible dimensions using ResizeObserver.
 * Resolves immediately if element already has dimensions.
 * Times out after specified duration to prevent hanging.
 */
const waitForDimensions = (
  element: HTMLElement,
  timeoutMs = 2000
): Promise<void> => {
  return new Promise((resolve) => {
    if (hasVisibleDimensions(element)) {
      resolve();
      return;
    }

    let resolved = false;
    const observer = new ResizeObserver((entries) => {
      for (const entry of entries) {
        if (entry.contentRect.width > 0 && entry.contentRect.height > 0) {
          if (!resolved) {
            resolved = true;
            observer.disconnect();
            resolve();
          }
          return;
        }
      }
    });

    observer.observe(element);

    // Timeout to prevent hanging forever
    setTimeout(() => {
      if (!resolved) {
        resolved = true;
        observer.disconnect();
        resolve();
      }
    }, timeoutMs);
  });
};

export const setupPeaksModule = ({
  state,
  eventEmitter,
  commandEmitter,
}: ReturnType<typeof createPeaksModule>) => {
  commandEmitter.on('mount', async (elements) => {
    const { theme, waveformUrl } = state.get();

    // Wait for containers to have visible dimensions before initializing.
    // The zoomview uses position:absolute+visibility:hidden when collapsed,
    // which maintains dimensions for Peaks.js initialization.
    await Promise.all([
      waitForDimensions(elements.overview),
      waitForDimensions(elements.zoomview),
    ]);

    // Skip initialization if containers still have no dimensions (e.g., still hidden)
    if (
      !hasVisibleDimensions(elements.overview) ||
      !hasVisibleDimensions(elements.zoomview)
    ) {
      console.warn(
        'Peaks.js initialization skipped: containers have no visible dimensions'
      );
      return;
    }

    const options = {
      overview: {
        container: elements.overview,
        ...theme.overviewTheme,
      },
      zoomview: {
        container: elements.zoomview,
        ...theme.zoomviewTheme,
      },
      mediaElement: elements.audio,
      ...(waveformUrl
        ? {
            dataUri: {
              arraybuffer: waveformUrl,
            },
          }
        : {
            webAudio: {
              audioContext: new AudioContext(),
              multiChannel: false,
            },
          }),
      createSegmentMarker: (options) => {
        return new CustomSegmentMarker(options, {
          eventEmitter,
        }) as SegmentMarker;
      },
    } satisfies PeaksOptions;

    Peaks.init(options, function (err, peaks) {
      if (err || !peaks) {
        console.error(err);
        throw err;
      }

      eventEmitter.emit('ready', peaks);
    });
  });
};
