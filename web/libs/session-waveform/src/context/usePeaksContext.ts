import { inject, provide } from 'vue';
import { installZoomControls } from './installZoomControls';
import { installAmplitudeControls } from './installAmplitudeControls';
import { installPlayerControls } from './installPlayerControls';
import { createPeaksModule } from './createPeaksModule';
import { installSegmentsControls } from './installSegmentsControls';
import { peaksModuleSchema } from './models/state';
import { setupPeaksModule } from './setupPeaksModule';
import { z } from 'zod';

const PeaksInjectionKey = Symbol();

export type PeaksContext = ReturnType<typeof createPeaksContext>;

export const createPeaksContext = (props: {
  initialState: z.input<typeof peaksModuleSchema>;
}) => {
  const module = createPeaksModule(props);

  setupPeaksModule(module);

  installZoomControls(module);
  installAmplitudeControls(module);
  installPlayerControls(module);
  installSegmentsControls(module);

  provide(PeaksInjectionKey, module);

  return module;
};

export const usePeaksContext = () => {
  const context = inject(PeaksInjectionKey);
  if (!context) {
    throw new Error('You must create a peaks context first');
  }
  return context as PeaksContext;
};
