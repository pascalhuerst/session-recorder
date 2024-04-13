import { sessionSourceClient } from '../sessionSourceClient';

export const renderSegment = async (sessionId: string, segmentId: string) => {
  await sessionSourceClient.renderSegment({
    sessionID: sessionId,
    segmentID: segmentId,
  });
};
