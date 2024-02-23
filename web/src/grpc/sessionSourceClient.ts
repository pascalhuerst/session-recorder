import { createChannel, createClient } from 'nice-grpc-web';
import {
  type SessionSourceClient,
  SessionSourceDefinition,
} from '@session-recorder/protocols/ts/sessionsource';
import { env } from '../env';

const channel = createChannel(env.VITE_GRPC_SERVER_URL);

export const sessionSourceClient: SessionSourceClient = createClient(
  SessionSourceDefinition,
  channel
);
