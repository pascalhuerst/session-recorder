import { createNanoEvents, type EventsMap } from 'nanoevents';

export const createCommandEmitter = <T extends EventsMap>() => {
  return createNanoEvents<T>();
};
