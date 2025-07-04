import { createSegment } from './procedures/createSegment';
import { Session } from '@session-recorder/protocols/ts/sessionsource';
import { updateSegment } from './procedures/updateSegment';
import { deleteSegment } from './procedures/deleteSegment';
import { renderSegment } from './procedures/renderSegment';
import type { PeaksContext } from '@session-recorder/session-waveform';

export const integrateSegments = (session: Session, ctx: PeaksContext) => {
  ctx.eventEmitter.on('segmentAdded', async (segment) => {
    await createSegment({
      recorderId: '', // TODO: Add recorderId parameter to integrateSegments
      sessionId: session.iD,
      segmentId: segment.id,
      segment: {
        timeStart: {
          seconds: Math.floor(segment.startTime / 1000).toString(),
          nanos: 0,
        },
        timeEnd: {
          seconds: Math.floor(segment.endTime / 1000).toString(),
          nanos: 0,
        },
        name: segment.labelText,
      },
    });
  });

  ctx.eventEmitter.on('segmentUpdated', async (segmentId, _, segment) => {
    await updateSegment({
      recorderId: '', // TODO: Add recorderId parameter to integrateSegments
      sessionId: session.iD,
      segmentId: segmentId,
      segment: {
        timeStart: {
          seconds: Math.floor(segment.startTime / 1000).toString(),
          nanos: 0,
        },
        timeEnd: {
          seconds: Math.floor(segment.endTime / 1000).toString(),
          nanos: 0,
        },
        name: segment.labelText,
      },
    });
  });

  ctx.eventEmitter.on('segmentRemoved', async (segmentId) => {
    await deleteSegment({
      recorderId: '', // TODO: Add recorderId parameter to integrateSegments
      sessionId: session.iD,
      segmentId,
    });
  });

  ctx.commandEmitter.on('renderSegment', async (segmentId) => {
    await renderSegment({
      recorderId: '', // TODO: Add recorderId parameter to integrateSegments
      sessionId: session.iD,
      segmentId,
    });
  });
};
