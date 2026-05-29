import {
  type SegmentInfo,
  type SessionInfo,
  SessionState as ProtoSessionState,
  SegmentState as ProtoSegmentState,
  StreamSessionRequest,
} from '@session-recorder/protocols/ts/sessionsource';
import { sessionSourceClient } from '../sessionSourceClient';
import type { Segment, SegmentState, Session, SessionState } from '../../types';
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

const mapSessionState = (protoState: ProtoSessionState): SessionState => {
  switch (protoState) {
    case ProtoSessionState.RECORDING:
      return 'recording';
    case ProtoSessionState.PROCESSING:
      return 'processing';
    case ProtoSessionState.FINISHED:
      return 'finished';
    case ProtoSessionState.ERROR:
      return 'error';
    default:
      return 'recording';
  }
};

const mapSegmentState = (protoState: ProtoSegmentState): SegmentState => {
  switch (protoState) {
    case ProtoSegmentState.QUEUED:
      return 'queued';
    case ProtoSegmentState.RENDERING:
      return 'rendering';
    case ProtoSegmentState.FINISHED:
      return 'finished';
    case ProtoSegmentState.ERROR:
      return 'error';
    default:
      return 'unknown';
  }
};

export const normalizeSession = (id: string, info: SessionInfo): Session => {
  return fromFactory<Session>({
    id: () => id,
    name: () => info.name,
    keep: () => info.keep,
    state: () => mapSessionState(info.state),
    errorMessage: () => info.errorMessage || undefined,
    startedAt: () => {
      if (!info.timeCreated) {
        throw new Error('Missing startsAt');
      }
      return Timestamp.toDate(info.timeCreated);
    },
    finishedAt: () => {
      if (!info.timeFinished) {
        return null;
      }
      return Timestamp.toDate(info.timeFinished);
    },
    expiresAt: () => {
      if (!info.timeFinished || !info.lifetime) {
        return null;
      }
      const finishedAt = Timestamp.toDate(info.timeFinished);
      const expiresAt = addDurationToDate(finishedAt, info.lifetime);
      return expiresAt;
    },
    inlineFiles: () => {
      return info.inlineFiles || null;
    },
    downloadFiles: () => {
      return info.downloadFiles || null;
    },
    segments: () => {
      if (!info.segments) {
        return [];
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
        .filter(Boolean)
        .sort((a, b) => a.timeStart.getTime() - b.timeStart.getTime());
    },
  });
};

export const normalizeSegment = (id: string, info: SegmentInfo): Segment => {
  return fromFactory<Segment>({
    id: () => id,
    name: () => info.name,
    state: () => mapSegmentState(info.state),
    errorMessage: () => info.errorMessage || undefined,
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
    inlineFiles: () => {
      return info.inlineFiles || null;
    },
    downloadFiles: () => {
      return info.downloadFiles || null;
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

  const abortController = new AbortController();
  const call = sessionSourceClient.streamSessions(args.request, {
    abort: abortController.signal,
  });

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
      // Aborting the stream (recorder switch / component unmount) is intentional
      // teardown via the returned stop() — not a failure. The server stream is
      // fine; just stop quietly.
      if (abortController.signal.aborted) {
        return;
      }

      console.error('StreamSessions error:', error);

      if (args.onError) {
        args.onError(error as Error);
      }
    }
  })();

  return () => abortController.abort();
};
