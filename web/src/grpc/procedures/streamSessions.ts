import { StreamSessionRequest, Session } from "@session-recorder/protocols/ts/sessionsource";
import { sessionSourceClient } from "../sessionSourceClient";

export const streamSessions = (args: {
  request: StreamSessionRequest;
  onMessage: (info: Session) => void;
  onError?: (error: Error) => void;
  onEnd?: () => void;
}) => {
  console.log('Starting streamSessions for recorder:', args.request.recorderID);
  const call = sessionSourceClient.streamSessions(args.request);
  
  // Handle streaming responses
  (async () => {
    try {
      let sessionCount = 0;
      for await (const response of call.responses) {
        sessionCount++;
        console.log(`StreamSessions received session #${sessionCount}:`, response);
        args.onMessage(response);
      }
      console.log(`StreamSessions ended. Total sessions received: ${sessionCount}`);
      if (args.onEnd) {
        args.onEnd();
      }
    } catch (error) {
      console.error('StreamSessions error:', error);
      if (args.onError) {
        args.onError(error as Error);
      }
    }
  })();
  
  return call;
};