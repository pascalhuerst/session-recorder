import type { ComputedRef, MaybeRefOrGetter } from 'vue';

export type MaybeReactive<T> = MaybeRefOrGetter<T> | ComputedRef<T>;
