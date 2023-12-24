// import { sessionSourceClient } from "../sessionSourceClient.ts";
import {
  type SessionInfo,
  SessionState,
  type StreamSeesionRequst
} from "@session-recorder/protocols/ts/sessionsource.ts";
import { env } from "../../env.ts";
import { Duration } from "@session-recorder/protocols/ts/google/protobuf/duration.ts";

// export const streamSessions = async (args: {
//   request: StreamSeesionRequst,
//   onMessage: (info: SessionInfo) => void
// }) => {
//   for await (const response of sessionSourceClient.streamSessions(args.request)) {
//     args.onMessage(response);
//   }
// };

export const streamSessions = async (args: {
  request: StreamSeesionRequst,
  onMessage: (info: SessionInfo) => void
}) => {
  await fetch(new URL("introspect", env.VITE_SERVER_URL))
    .then((res) => res.json())
    .then((data) => {
      (data[args.request.recorderID].open_sessions || []).forEach((session: any) => {
        args.onMessage({
          ID: session.id,
          state: SessionState.SESSION_STATE_CREATED,
          timeCreated: new Date(session.timestamp),
          waveformDataFile: "",
          audioFileName: "",
          keepSession: session.flagged,
          lifetime: Duration.create({
            seconds: session.hours_to_live * 60,
            nanos: 0
          }),
          timeFinished: new Date(session.timestamp)
        });
      });
    });
};
