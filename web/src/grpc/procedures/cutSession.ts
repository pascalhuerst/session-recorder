import { CutSessionRequest } from "@session-recorder/protocols/ts/sessionsource";
import { sessionSourceClient } from "../sessionSourceClient";

export const cutSession = async (args: { recorderID: string }) => {
  const request: CutSessionRequest = {
    recorderID: args.recorderID,
  };

  const call = await sessionSourceClient.cutSession(request);
  return call.response;
};