import { createNanoEvents, type EventsMap } from 'nanoevents';

export const createEventEmitter = <T extends EventsMap>() => {
  return createNanoEvents<T>();
};
