import { atom } from 'nanostores';
import { ZodSchema } from 'zod';

export const createContextStore = <T>(options: {
  initialState: T;
  schema: ZodSchema<T>;
}) => {
  const normalize = (state: T) => options.schema.parse(state);

  const $store = atom<T>(normalize(options.initialState));

  return {
    $store,
    update: (reduce: (prev: T) => T) => {
      $store.set(normalize(reduce($store.get())));
    },
  };
};
