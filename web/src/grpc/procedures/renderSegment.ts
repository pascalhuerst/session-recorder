import { RenderSegmentRequest } from "@session-recorder/protocols/ts/sessionsource";
import { sessionSourceClient } from "../sessionSourceClient";

export const renderSegment = async (args: {
  recorderId: string;
  sessionId: string;
  segmentId: string;
}) => {
  const request: RenderSegmentRequest = {
    recorderID: args.recorderId,
    sessionID: args.sessionId,
    segmentID: args.segmentId,
  };

  const call = await sessionSourceClient.renderSegment(request);
  return call.response;
};