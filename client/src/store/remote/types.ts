import { Session } from 'alsa2fifo_indexer/proto/ts/sessionsource_pb.d';

export interface RemoteState {
    openSessions: Session[];
    version: string;
}