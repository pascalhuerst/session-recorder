import { sessionSourceClient } from "../sessionSourceClient.ts";
import { type SessionInfo, StreamSeesionRequst } from "@session-recorder/protocols/ts/sessionsource.ts";

export const streamSessions = async (args: {
  request: StreamSeesionRequst,
  onMessage: (info: SessionInfo) => void
}) => {
  for await (const response of sessionSourceClient.streamSessions(args.request)) {
    args.onMessage(response);
  }
};