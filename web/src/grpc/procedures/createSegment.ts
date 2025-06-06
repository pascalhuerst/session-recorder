import { CreateSegmentRequest, SegmentInfo } from "@session-recorder/protocols/ts/sessionsource";
import { sessionSourceClient } from "../sessionSourceClient";

export const createSegment = async (args: {
  recorderId: string;
  sessionId: string;
  segmentId: string;
  segment: Partial<SegmentInfo>;
}) => {
  const request: CreateSegmentRequest = {
    recorderID: args.recorderId,
    sessionID: args.sessionId,
    segmentID: args.segmentId,
    info: args.segment,
  };

  const call = await sessionSourceClient.createSegment(request);
  return call.response;
};