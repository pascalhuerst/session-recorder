import { type SegmentInfo } from '@session-recorder/protocols/ts/sessionsource';
import { sessionSourceClient } from '../sessionSourceClient';

export const updateSegment = async (
  sessionId: string,
  segmentId: string,
  segment: Partial<SegmentInfo>
) => {
  await sessionSourceClient.createSegment({
    sessionID: sessionId,
    segmentID: segmentId,
    info: segment,
  });
};
