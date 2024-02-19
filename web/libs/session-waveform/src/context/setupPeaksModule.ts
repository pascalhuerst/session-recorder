import { CustomSegmentMarker } from '../elements/CustomSegmentMarker';
import type { PeaksOptions, SegmentMarker } from 'peaks.js';
import Peaks from 'peaks.js';
import type { createPeaksModule } from './createPeaksModule';

export const setupPeaksModule = ({
  state,
  eventEmitter,
  commandEmitter,
}: ReturnType<typeof createPeaksModule>) => {
  commandEmitter.on('mount', (elements) => {
    const { theme, waveformUrl } = state.get();

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
        console.warn(err.message);
        return null;
      }

      state.update((prev) => ({
        ...prev,
        peaks,
      }));

      eventEmitter.emit('ready', peaks);
    });
  });
};
