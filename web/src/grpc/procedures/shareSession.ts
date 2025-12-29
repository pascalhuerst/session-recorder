import { ShareSessionRequest } from "@session-recorder/protocols/ts/sessionsource";
import { sessionSourceClient } from "../sessionSourceClient";

export const shareSession = async (args: {
  recorderId: string;
  sessionId: string;
  recipientEmails: string[];
}) => {
  const request = ShareSessionRequest.create({
    recorderID: args.recorderId,
    sessionID: args.sessionId,
    recipientEmails: args.recipientEmails,
  });

  const call = await sessionSourceClient.shareSession(request);
  return call.response;
};
