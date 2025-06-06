import { StreamRecordersRequest, Recorder } from "@session-recorder/protocols/ts/sessionsource";
import { sessionSourceClient } from "../sessionSourceClient";

export const streamRecorders = (args: {
  onMessage: (info: Recorder) => void;
  onError?: (error: Error) => void;
  onEnd?: () => void;
}) => {
  const request: StreamRecordersRequest = {};
  
  const call = sessionSourceClient.streamRecorders(request);
  
  // Handle streaming responses
  (async () => {
    try {
      for await (const response of call.responses) {
        args.onMessage(response);
      }
      if (args.onEnd) {
        args.onEnd();
      }
    } catch (error) {
      console.error('StreamRecorders error:', error);
      if (args.onError) {
        args.onError(error as Error);
      }
    }
  })();
  
  return call;
};