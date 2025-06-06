import { GrpcWebFetchTransport } from "@protobuf-ts/grpcweb-transport";
import { SessionSourceClient } from "@session-recorder/protocols/ts/sessionsource.client";
import { env } from "../env";

const transport = new GrpcWebFetchTransport({
  baseUrl: env.VITE_GRPC_SERVER_URL,
});

export const sessionSourceClient = new SessionSourceClient(transport);