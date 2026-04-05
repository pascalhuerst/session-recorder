import { sessionSourceClient } from '../sessionSourceClient';

export type WaveformPeakMessage = {
  sessionID: string;
  peaks: Int8Array; // interleaved [min, max, min, max, ...]
  isInitial: boolean;
  totalPeakPairs: number;
  peakLevel: number; // 0.0 - 1.0
  clipping: boolean;
};

export const streamWaveformPeaks = (args: {
  sessionID: string;
  onPeaks: (msg: WaveformPeakMessage) => void;
  onError?: (error: Error) => void;
  onEnd?: () => void;
}) => {
  const abortController = new AbortController();
  const request = { sessionID: args.sessionID };

  const call = sessionSourceClient.streamWaveformPeaks(request, {
    abort: abortController.signal,
  });

  (async () => {
    try {
      for await (const response of call.responses) {
        // Convert int32 peaks to Int8Array
        const peaks = new Int8Array(response.peaks.length);
        for (let i = 0; i < response.peaks.length; i++) {
          peaks[i] = response.peaks[i];
        }

        args.onPeaks({
          sessionID: response.sessionID,
          peaks,
          isInitial: response.isInitial,
          totalPeakPairs: response.totalPeakPairs,
          peakLevel: response.peakLevel,
          clipping: response.clipping,
        });
      }

      args.onEnd?.();
    } catch (error) {
      args.onError?.(error as Error);
    }
  })();

  return () => abortController.abort();
};
