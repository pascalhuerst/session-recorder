import type { AudioChunk } from '@session-recorder/protocols/ts/sessionsource';
import { sessionSourceClient } from '../sessionSourceClient';

export type AudioChunkMessage = {
  sessionID: string;
  samples: Int16Array;
  chunkNumber: number;
  timestamp: Date;
};

export const streamSessionAudio = (args: {
  sessionID: string;
  onChunk: (chunk: AudioChunkMessage) => void;
  onError?: (error: Error) => void;
  onEnd?: () => void;
}) => {
  console.log('Starting streamSessionAudio for session:', args.sessionID);

  const abortController = new AbortController();
  const request = { sessionID: args.sessionID };

  const call = sessionSourceClient.streamSessionAudio(request, {
    abort: abortController.signal,
  });

  // Handle streaming responses
  (async () => {
    try {
      let chunkCount = 0;

      for await (const response of call.responses) {
        chunkCount++;

        // Convert int32 samples back to Int16Array
        const samples = new Int16Array(response.samples.length);
        for (let i = 0; i < response.samples.length; i++) {
          samples[i] = response.samples[i];
        }

        const chunk: AudioChunkMessage = {
          sessionID: response.sessionID,
          samples,
          chunkNumber: response.chunkNumber,
          timestamp: response.timestamp
            ? new Date(
                Number(response.timestamp.seconds) * 1000 +
                  response.timestamp.nanos / 1000000
              )
            : new Date(),
        };

        args.onChunk(chunk);
      }

      console.log(
        `StreamSessionAudio ended. Total chunks received: ${chunkCount}`
      );

      if (args.onEnd) {
        args.onEnd();
      }
    } catch (error) {
      console.error('StreamSessionAudio error:', error);

      if (args.onError) {
        args.onError(error as Error);
      }
    }
  })();

  return () => abortController.abort();
};
