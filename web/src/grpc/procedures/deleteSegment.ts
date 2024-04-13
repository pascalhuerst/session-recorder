import { sessionSourceClient } from '../sessionSourceClient';

export const deleteSegment = async (args: {
  sessionId: string;
  segmentId: string;
}) => {
  await sessionSourceClient.deleteSegment({
    sessionID: args.sessionId,
    segmentID: args.segmentId,
  });
};
