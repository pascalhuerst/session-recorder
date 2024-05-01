import { sessionSourceClient } from "../sessionSourceClient";

export const cutSession = async (args: { recorderID: string }) => {
  await sessionSourceClient.cutSession({
    recorderID: args.recorderID
  });
};
