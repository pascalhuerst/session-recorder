import { type SegmentInfo } from '@session-recorder/protocols/ts/sessionsource';
import { sessionSourceClient } from '../sessionSourceClient';

export const updateSegment = async (args: {
  sessionId: string;
  segmentId: string;
  segment: Partial<SegmentInfo>;
}) => {
  await sessionSourceClient.updateSegment({
    sessionID: args.sessionId,
    segmentID: args.segmentId,
    info: args.segment,
  });
};
