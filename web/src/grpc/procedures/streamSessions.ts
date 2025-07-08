import {
  type SegmentInfo,
  type SessionInfo,
  SessionState,
  StreamSessionRequest,
} from '@session-recorder/protocols/ts/sessionsource';
import { sessionSourceClient } from '../sessionSourceClient';
import type { Segment, Session } from '../../types';
import { Timestamp } from '@session-recorder/protocols/ts/google/protobuf/timestamp';
import { Duration } from '@session-recorder/protocols/ts/google/protobuf/duration';

export type SessionMessage =
  | {
      type: 'updated';
      session: Session;
    }
  | {
      type: 'deleted';
      id: string;
    };

function durationToMilliseconds(duration: Duration): number {
  const seconds = parseInt(duration.seconds, 10);
  const milliseconds = Math.floor(duration.nanos / 1000000);
  return seconds * 1000 + milliseconds;
}

function addDurationToDate(date: Date, duration: Duration): Date {
  const durationMs = durationToMilliseconds(duration);
  return new Date(date.getTime() + durationMs);
}

type Factory<T> = { [K in keyof T]: () => T[K] };

const fromFactory = <T>(factory: Factory<T>): T => {
  return Object.keys(factory).reduce((acc, key) => {
    return { ...acc, [key]: factory[key]() };
  }, {} as T);
};

export const normalizeSession = (id: string, info: SessionInfo): Session => {
  return fromFactory<Session>({
    id: () => id,
    name: () => info.name,
    keep: () => info.keep,
    startedAt: () => {
      if (!info.timeCreated) {
        throw new Error('Missing startsAt');
      }
      return Timestamp.toDate(info.timeCreated);
    },
    finishedAt: () => {
      if (!info.timeFinished) {
        throw new Error('Missing finishedAt');
      }
      return Timestamp.toDate(info.timeFinished);
    },
    expiresAt: () => {
      if (!info.timeFinished || !info.lifetime) {
        throw new Error('Missing time info');
      }
      const finishedAt = Timestamp.toDate(info.timeFinished);
      const expiresAt = addDurationToDate(finishedAt, info.lifetime);
      return expiresAt;
    },
    inlineFiles: () => {
      if (!info.inlineFiles) {
        throw new Error('Missing inlineFiles');
      }
      return info.inlineFiles;
    },
    downloadFiles: () => {
      if (!info.downloadFiles) {
        throw new Error('Missing downloadFiles');
      }
      return info.downloadFiles;
    },
    segments: () => {
      if (!info.segments) {
        throw new Error('Missing segments');
      }
      return info.segments
        .map((s) => {
          if ('updated' in s.info) {
            try {
              return normalizeSegment(s.segmentID, s.info.updated);
            } catch (error) {
              console.error(`Error normalizing segment ${s.segmentID}:`, error);
            }
          }
        })
        .filter(Boolean);
    },
  });
};

export const normalizeSegment = (id: string, info: SegmentInfo): Segment => {
  return fromFactory<Segment>({
    id: () => id,
    name: () => info.name,
    timeStart: () => {
      if ('timeStart' in info) {
        return Timestamp.toDate(info.timeStart);
      }
      throw new Error('Missing timeStart');
    },
    timeEnd: () => {
      if ('timeEnd' in info) {
        return Timestamp.toDate(info.timeEnd);
      }
      throw new Error('Missing timeEnd');
    },
  });
};

export const streamSessions = (args: {
  request: StreamSessionRequest;
  onMessage: (msg: SessionMessage) => void;
  onError?: (error: Error) => void;
  onEnd?: () => void;
}) => {
  console.log('Starting streamSessions for recorder:', args.request.recorderID);
  const call = sessionSourceClient.streamSessions(args.request);

  // Handle streaming responses
  (async () => {
    try {
      let sessionCount = 0;

      for await (const response of call.responses) {
        sessionCount++;
        console.log(
          `StreamSessions received session #${sessionCount}:`,
          response
        );
        switch (response.info.oneofKind) {
          case 'updated': {
            if ('updated' in response.info) {
              if (response.info.updated.state !== SessionState.FINISHED) {
                break;
              }

              try {
                const session = normalizeSession(
                  response.iD,
                  response.info.updated
                );
                args.onMessage({
                  type: 'updated',
                  session,
                });
              } catch (error) {
                console.error('Error normalizing session:', error);
              }
            } else {
              console.error('Missing updated session info', response);
            }
            break;
          }
          case 'removed': {
            if ('removed' in response.info) {
              args.onMessage({
                type: 'deleted',
                id: response.iD,
              });
            } else {
              console.error('Missing deleted session info', response);
            }
            break;
          }
        }
      }

      console.log(
        `StreamSessions ended. Total sessions received: ${sessionCount}`
      );

      if (args.onEnd) {
        args.onEnd();
      }
    } catch (error) {
      console.error('StreamSessions error:', error);

      if (args.onError) {
        args.onError(error as Error);
      }
    }
  })();

  return call;
};
