import { createChannel, createClient } from "nice-grpc-web";
import { env } from "../env.ts";
import { SessionSourceClient, SessionSourceDefinition } from "@session-recorder/protocols/ts/sessionsource.ts";

const channel = createChannel(env.VITE_GRPC_SERVER_URL);

export const sessionSourceClient: SessionSourceClient = createClient(
  SessionSourceDefinition,
  channel
);

