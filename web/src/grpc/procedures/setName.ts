import { SetNameRequest } from "@session-recorder/protocols/ts/sessionsource";
import { sessionSourceClient } from "../sessionSourceClient";

export const setName = async (args: {
  recorderId: string;
  sessionId: string;
  name: string;
}) => {
  const request: SetNameRequest = {
    recorderID: args.recorderId,
    sessionID: args.sessionId,
    name: args.name,
  };

  const call = await sessionSourceClient.setName(request);
  return call.response;
};
