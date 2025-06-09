import {
  UpdateSegmentRequest,
  SegmentInfo,
} from '@session-recorder/protocols/ts/sessionsource';
import { sessionSourceClient } from '../sessionSourceClient';

export const updateSegment = async (args: {
  recorderId: string;
  sessionId: string;
  segmentId: string;
  segment?: SegmentInfo;
}) => {
  const request: UpdateSegmentRequest = {
    recorderID: args.recorderId,
    sessionID: args.sessionId,
    segmentID: args.segmentId,
    info: args.segment,
  };

  const call = await sessionSourceClient.updateSegment(request);
  return call.response;
};
