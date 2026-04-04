import { CustomSegmentMarker } from '../elements/CustomSegmentMarker';
import type { PeaksInstance, PeaksOptions, SegmentMarker } from 'peaks.js';
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

    const audioContext = waveformUrl ? undefined : new AudioContext();

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
              audioContext: audioContext!,
              multiChannel: false,
            },
          }),
      createSegmentMarker: (options) => {
        return new CustomSegmentMarker(options, {
          eventEmitter,
        }) as SegmentMarker;
      },
    } satisfies PeaksOptions;

    let peaksInstance: PeaksInstance | null = null;

    Peaks.init(options, function (err, peaks) {
      if (err || !peaks) {
        // When scrolling fast, containers may lose dimensions before Peaks.js
        // completes async initialization (XHR for waveform data). This is expected
        // and not an error - the component was simply unmounted during loading.
        const isContainerError =
          err instanceof Error &&
          err.message.includes('must be visible and have non-zero width');
        if (isContainerError) {
          audioContext?.close();
          return;
        }
        console.error(err);
        audioContext?.close();
        return;
      }

      peaksInstance = peaks;
      eventEmitter.emit('ready', peaks);
    });

    commandEmitter.on('destroy', () => {
      if (peaksInstance) {
        peaksInstance.destroy();
        peaksInstance = null;
      }
      audioContext?.close();
    });
  });
};
