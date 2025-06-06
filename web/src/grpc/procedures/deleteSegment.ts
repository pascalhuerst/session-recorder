import { DeleteSegmentRequest } from "@session-recorder/protocols/ts/sessionsource";
import { sessionSourceClient } from "../sessionSourceClient";

export const deleteSegment = async (args: {
  recorderId: string;
  sessionId: string;
  segmentId: string;
}) => {
  const request: DeleteSegmentRequest = {
    recorderID: args.recorderId,
    sessionID: args.sessionId,
    segmentID: args.segmentId,
  };

  const call = await sessionSourceClient.deleteSegment(request);
  return call.response;
};