import { ShareSegmentRequest } from "@session-recorder/protocols/ts/sessionsource";
import { sessionSourceClient } from "../sessionSourceClient";

export const shareSegment = async (args: {
  recorderId: string;
  sessionId: string;
  segmentId: string;
  recipientEmails: string[];
}) => {
  const request = ShareSegmentRequest.create({
    recorderID: args.recorderId,
    sessionID: args.sessionId,
    segmentID: args.segmentId,
    recipientEmails: args.recipientEmails,
  });

  const call = await sessionSourceClient.shareSegment(request);
  return call.response;
};
