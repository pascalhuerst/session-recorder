import type { PeaksContext } from '../../libs/session-waveform/src/context/usePeaksContext';
import { createSegment } from './procedures/createSegment';
import { type Session } from '@session-recorder/protocols/ts/sessionsource';
import { updateSegment } from './procedures/updateSegment';
import { deleteSegment } from './procedures/deleteSegment';
import { renderSegment } from './procedures/renderSegment';

export const integrateSegments = (session: Session, ctx: PeaksContext) => {
  ctx.eventEmitter.on('segmentAdded', async (segment) => {
    await createSegment({
      sessionId: session.ID,
      segmentId: segment.id,
      segment: {
        timeStart: new Date(segment.startTime),
        timeEnd: new Date(segment.endTime),
        name: segment.labelText,
      },
    });
  });

  ctx.eventEmitter.on('segmentUpdated', async (segmentId, _, segment) => {
    await updateSegment({
      sessionId: session.ID,
      segmentId: segmentId,
      segment: {
        timeStart: new Date(segment.startTime),
        timeEnd: new Date(segment.endTime),
        name: segment.labelText,
      },
    });
  });

  ctx.eventEmitter.on('segmentRemoved', async (segmentId) => {
    await deleteSegment({
      sessionId: session.ID,
      segmentId,
    });
  });

  ctx.commandEmitter.on('renderSegment', async (segmentId) => {
    await renderSegment({
      sessionId: session.ID,
      segmentId,
    });
  });
};
