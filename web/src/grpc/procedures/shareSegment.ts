import { ShareSegmentRequest } from "@session-recorder/protocols/ts/sessionsource";
import { sessionSourceClient } from "../sessionSourceClient";

export const shareSegment = async (args: {
  recorderId: string;
  sessionId: string;
  segmentId: string;
  recipientEmail: string;
}) => {
  const request: ShareSegmentRequest = {
    recorderID: args.recorderId,
    sessionID: args.sessionId,
    segmentID: args.segmentId,
    recipientEmail: args.recipientEmail,
  };

  const call = await sessionSourceClient.shareSegment(request);
  return call.response;
};
