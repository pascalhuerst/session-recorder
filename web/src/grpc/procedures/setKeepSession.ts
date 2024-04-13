import { sessionSourceClient } from '../sessionSourceClient';

export const setKeepSession = async (args: { sessionId: string }) => {
  console.log(args);
  await sessionSourceClient.setKeepSession({
    sessionID: args.sessionId,
  });
};
