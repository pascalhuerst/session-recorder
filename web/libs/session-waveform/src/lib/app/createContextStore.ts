import { atom } from 'nanostores';
import { z, ZodSchema } from 'zod';
import { useStore } from '@nanostores/vue';

export const createContextStore = <T extends ZodSchema>(options: {
  initialState: z.input<T>;
  schema: T;
}) => {
  const normalize = (state: T) => options.schema.parse(state);

  const $store = atom<z.output<T>>(normalize(options.initialState));

  return {
    $store,
    toRef: () => {
      return useStore($store);
    },
    get: () => $store.get(),
    select: <S>(fn: (st: z.output<T>) => S) => {
      return fn($store.get());
    },
    update: (reduce: (prev: z.output<T>) => z.input<T>) => {
      $store.set(normalize(reduce(Object.freeze($store.get()))));
    },
  };
};
