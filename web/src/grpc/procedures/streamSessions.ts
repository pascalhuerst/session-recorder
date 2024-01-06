import {
  Session,
  StreamSessionRequest,
} from '@session-recorder/protocols/ts/sessionsource';
import { sessionSourceClient } from '@/grpc/sessionSourceClient';

export const streamSessions = async (args: {
  request: StreamSessionRequest;
  onMessage: (info: Session) => void;
}) => {
  for await (const response of sessionSourceClient.streamSessions(
    args.request
  )) {
    args.onMessage(response);
  }
};
