import { ShareSessionRequest } from "@session-recorder/protocols/ts/sessionsource";
import { sessionSourceClient } from "../sessionSourceClient";

export const shareSession = async (args: {
  recorderId: string;
  sessionId: string;
  recipientEmail: string;
}) => {
  const request: ShareSessionRequest = {
    recorderID: args.recorderId,
    sessionID: args.sessionId,
    recipientEmail: args.recipientEmail,
  };

  const call = await sessionSourceClient.shareSession(request);
  return call.response;
};
