import { inject, provide } from 'vue';

export const createProvider = <T>() => {
  const name = Symbol();

  return {
    provide: (value: T) => {
      provide<T>(name, value);
    },
    inject: (): T => {
      const value = inject<T>(name);
      if (typeof value === 'undefined') {
        throw new Error('You must provide the value');
      }
      return value;
    },
  };
};
