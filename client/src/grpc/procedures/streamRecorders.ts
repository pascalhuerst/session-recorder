import { sessionSourceClient } from "../sessionSourceClient.ts";
import { type RecorderInfo } from "@session-recorder/protocols/ts/sessionsource";

export const streamRecorders = async (args: { onMessage: (info: RecorderInfo) => void }) => {
  for await (const response of sessionSourceClient.streamRecorders({})) {
    args.onMessage(response);
  }
};