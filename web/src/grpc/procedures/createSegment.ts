import { type SegmentInfo } from '@session-recorder/protocols/ts/sessionsource';
import { sessionSourceClient } from '../sessionSourceClient';

export const createSegment = async (args: {
  sessionId: string;
  segmentId: string;
  segment: Partial<SegmentInfo>;
}) => {
  await sessionSourceClient.createSegment({
    sessionID: args.sessionId,
    segmentID: args.segmentId,
    info: args.segment,
  });
};
