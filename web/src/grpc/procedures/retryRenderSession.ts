import { DeleteSessionRequest } from "@session-recorder/protocols/ts/sessionsource";
import { sessionSourceClient } from "../sessionSourceClient";

export const retryRenderSession = async (args: {
  recorderId: string;
  sessionId: string;
}) => {
  const request: DeleteSessionRequest = {
    recorderID: args.recorderId,
    sessionID: args.sessionId,
  };

  const call = await sessionSourceClient.retryRenderSession(request);
  return call.response;
};
