// import { sessionSourceClient } from "../sessionSourceClient.ts";
import { type RecorderInfo } from "@session-recorder/protocols/ts/sessionsource";
import { AudioInputStatus } from "@session-recorder/protocols/ts/chunksink.ts";
import { env } from "../../env.ts";
//
// export const streamRecorders = async (args: { onMessage: (info: RecorderInfo) => void }) => {
//   for await (const response of sessionSourceClient.streamRecorders({})) {
//     args.onMessage(response);
//   }
// };

export const streamRecorders = async (args: { onMessage: (info: RecorderInfo) => void }) => {
  await fetch(new URL("introspect", env.VITE_SERVER_URL))
    .then((res) => res.json())
    .then((data) => {
      Object.keys(data).forEach((recorderId) => {
        args.onMessage({
          ID: recorderId,
          name: recorderId,
          audioInputStatus: AudioInputStatus.SIGNAL
        });
      });
    }).catch((err) => {
      console.error(err);
      return;
    });
};