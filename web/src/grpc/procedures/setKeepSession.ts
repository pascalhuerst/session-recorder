import { sessionSourceClient } from '../sessionSourceClient';

export const setKeepSession = async (args: { recorderId: string, sessionId: string, keep: boolean }) => {
  await sessionSourceClient.setKeepSession({
    recorderID: args.recorderId,
    sessionID: args.sessionId,
    keep: args.keep,
  });
};
