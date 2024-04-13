import { sessionSourceClient } from '../sessionSourceClient';

export const setKeepSession = async (args: { sessionId: string }) => {
  await sessionSourceClient.setKeepSession({
    sessionID: args.sessionId,
  });
};
