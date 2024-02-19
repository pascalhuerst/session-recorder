import { createProvider } from '../lib/app/createProvider';
import type { Ref } from 'vue';

const provider = createProvider<{
  overviewRef: Ref<HTMLElement | undefined>;
  zoomviewRef: Ref<HTMLElement | undefined>;
  audioRef: Ref<HTMLElement | undefined>;
}>();

export const useWaverformLayoutProvider = () => provider;
