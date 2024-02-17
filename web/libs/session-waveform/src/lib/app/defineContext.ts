import { createContextStore } from './createContextStore';
import { createEventEmitter } from './createEventEmitter';
import { createCommandEmitter } from './createCommandEmitter';
import type { ZodSchema } from 'zod';
import type { EventsMap } from 'nanoevents';

export const defineContext = <
  E extends EventsMap,
  C extends EventsMap,
  T
>(options: {
  schema: ZodSchema<T>;
  resolveInitialState: () => T;
}) => {
  return () => ({
    state: createContextStore({
      initialState: options.resolveInitialState(),
      schema: options.schema,
    }),
    eventEmitter: createEventEmitter<E>(),
    commandEmitter: createCommandEmitter<C>(),
  });
};
