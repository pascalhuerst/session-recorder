import { type Recorder } from "@session-recorder/protocols/ts/sessionsource";
import { sessionSourceClient } from "../sessionSourceClient";

export const streamRecorders = async (args: {
  onMessage: (info: Recorder) => void;
}) => {
  for await (const response of sessionSourceClient.streamRecorders({})) {
    args.onMessage(response);
  }
};
