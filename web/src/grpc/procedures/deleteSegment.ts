import { sessionSourceClient } from '../sessionSourceClient';

export const deleteSegment = async (sessionId: string, segmentId: string) => {
  await sessionSourceClient.deleteSegment({
    sessionID: sessionId,
    segmentID: segmentId,
  });
};
