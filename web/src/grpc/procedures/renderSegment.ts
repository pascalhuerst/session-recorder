import { sessionSourceClient } from '../sessionSourceClient';

export const renderSegment = async (args: {
  sessionId: string;
  segmentId: string;
}) => {
  await sessionSourceClient.renderSegment({
    sessionID: sessionId,
    segmentID: segmentId,
  });
};
