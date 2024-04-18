import { sessionSourceClient } from '../sessionSourceClient';

export const deleteSession = async (args: { recorderId: string, sessionId: string }) => {
  await sessionSourceClient.deleteSession({
    recorderID: args.recorderId,
    sessionID: args.sessionId,
  });
};
