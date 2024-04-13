import type { PeaksContext } from '../../libs/session-waveform/src/context/usePeaksContext';

export const integrateSegments = (ctx: PeaksContext) => {
  ctx.eventEmitter.on('segmentAdded', () => {});
};
