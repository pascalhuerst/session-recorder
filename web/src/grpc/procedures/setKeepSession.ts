import { SetKeepSessionRequest } from "@session-recorder/protocols/ts/sessionsource";
import { sessionSourceClient } from "../sessionSourceClient";

export const setKeepSession = async (args: { 
  recorderId: string; 
  sessionId: string; 
  keep: boolean; 
}) => {
  const request: SetKeepSessionRequest = {
    recorderID: args.recorderId,
    sessionID: args.sessionId,
    keep: args.keep,
  };

  const call = await sessionSourceClient.setKeepSession(request);
  return call.response;
};