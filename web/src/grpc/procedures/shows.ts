import type { ShowInfo } from '@session-recorder/protocols/ts/sessionsource';
import { sessionSourceClient } from '../sessionSourceClient';
import type { Show, Act } from '../../types';
import { ShowState } from '@session-recorder/protocols/ts/sessionsource';
import { Timestamp } from '@session-recorder/protocols/ts/google/protobuf/timestamp';

export const listShows = async (): Promise<Show[]> => {
  const call = await sessionSourceClient.listShows({});
  return call.response.shows.map(normalizeShow);
};

export const createShow = async (show: {
  name: string;
  date: Date;
  recorderId: string;
  acts: Omit<Act, 'id' | 'segmentId' | 'actualStart' | 'actualEnd'>[];
}) => {
  const call = await sessionSourceClient.createShow({
    show: {
      showID: '',
      name: show.name,
      date: Timestamp.fromDate(show.date),
      state: ShowState.DRAFT,
      recorderID: show.recorderId,
      sessionID: '',
      acts: show.acts.map((a) => ({
        actID: '',
        name: a.name,
        plannedStart: Timestamp.fromDate(a.plannedStart),
        plannedEnd: Timestamp.fromDate(a.plannedEnd),
        emails: a.emails,
        segmentID: '',
        actualStart: undefined,
        actualEnd: undefined,
      })),
    },
  });
  return call.response;
};

export const updateShow = async (show: {
  id: string;
  name?: string;
  date?: Date;
  recorderId?: string;
  acts?: Act[];
}) => {
  const call = await sessionSourceClient.updateShow({
    show: {
      showID: show.id,
      name: show.name ?? '',
      date: show.date ? Timestamp.fromDate(show.date) : undefined,
      state: ShowState.DRAFT,
      recorderID: show.recorderId ?? '',
      sessionID: '',
      acts: show.acts
        ? show.acts.map((a) => ({
            actID: a.id,
            name: a.name,
            plannedStart: Timestamp.fromDate(a.plannedStart),
            plannedEnd: Timestamp.fromDate(a.plannedEnd),
            emails: a.emails,
            segmentID: a.segmentId ?? '',
            actualStart: a.actualStart
              ? Timestamp.fromDate(a.actualStart)
              : undefined,
            actualEnd: a.actualEnd
              ? Timestamp.fromDate(a.actualEnd)
              : undefined,
          }))
        : undefined,
    },
  });
  return call.response;
};

export const deleteShow = async (showId: string) => {
  const call = await sessionSourceClient.deleteShow({ showID: showId });
  return call.response;
};

export const startShow = async (showId: string) => {
  const call = await sessionSourceClient.startShow({ showID: showId });
  return call.response;
};

export const renderAll = async (showId: string) => {
  const call = await sessionSourceClient.renderAll({ showID: showId });
  return call.response;
};

export const distributeAll = async (showId: string) => {
  const call = await sessionSourceClient.distributeAll({ showID: showId });
  return call.response;
};

// Normalize proto ShowInfo to app Show type
function normalizeShow(info: ShowInfo): Show {
  const stateMap: Record<number, Show['state']> = {
    [ShowState.DRAFT]: 'draft',
    [ShowState.LIVE]: 'live',
    [ShowState.ENDED]: 'ended',
    [ShowState.ARCHIVED]: 'archived',
  };

  return {
    id: info.showID,
    name: info.name,
    date: info.date ? Timestamp.toDate(info.date) : new Date(),
    state: stateMap[info.state] ?? 'draft',
    recorderId: info.recorderID,
    sessionId: info.sessionID || null,
    acts: info.acts.map((a) => ({
      id: a.actID,
      name: a.name,
      plannedStart: a.plannedStart
        ? Timestamp.toDate(a.plannedStart)
        : new Date(),
      plannedEnd: a.plannedEnd ? Timestamp.toDate(a.plannedEnd) : new Date(),
      emails: a.emails,
      segmentId: a.segmentID || null,
      actualStart: a.actualStart ? Timestamp.toDate(a.actualStart) : null,
      actualEnd: a.actualEnd ? Timestamp.toDate(a.actualEnd) : null,
    })),
  };
}
