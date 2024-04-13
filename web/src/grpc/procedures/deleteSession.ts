import { sessionSourceClient } from '../sessionSourceClient';

export const deleteSession = async (args: { sessionId: string }) => {
  await sessionSourceClient.deleteSession({
    sessionID: args.sessionId,
  });
};
