import { createSegment } from './procedures/createSegment';
import { updateSegment } from './procedures/updateSegment';
import { deleteSegment } from './procedures/deleteSegment';
import { renderSegment } from './procedures/renderSegment';
import type { PeaksContext } from '@session-recorder/session-waveform';
import type { Session } from '../types';
import { SegmentState } from '@session-recorder/protocols/ts/sessionsource';
import { toastService } from '../services/Toaster/ToastService';

export const integrateSegments = (session: Session, ctx: PeaksContext) => {
  ctx.eventEmitter.on('segmentAdded', async (segment) => {
    try {
      await createSegment({
        recorderId: '', // TODO: Add recorderId parameter to integrateSegments
        sessionId: session.id,
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
          state: SegmentState.UNKNOWN,
        },
      });
      toastService.success('Segment created successfully');
    } catch (error) {
      console.error('Failed to create segment:', error);
      toastService.error('Failed to create segment');
    }
  });

  ctx.eventEmitter.on('segmentUpdated', async (segmentId, _, segment) => {
    try {
      await updateSegment({
        recorderId: '', // TODO: Add recorderId parameter to integrateSegments
        sessionId: session.id,
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
          state: SegmentState.UNKNOWN,
        },
      });
      toastService.success('Segment updated successfully');
    } catch (error) {
      console.error('Failed to update segment:', error);
      toastService.error('Failed to update segment');
    }
  });

  ctx.eventEmitter.on('segmentRemoved', async (segmentId) => {
    try {
      await deleteSegment({
        recorderId: '', // TODO: Add recorderId parameter to integrateSegments
        sessionId: session.id,
        segmentId,
      });
      toastService.success('Segment deleted successfully');
    } catch (error) {
      console.error('Failed to delete segment:', error);
      toastService.error('Failed to delete segment');
    }
  });

  ctx.commandEmitter.on('renderSegment', async (segmentId) => {
    try {
      await renderSegment({
        recorderId: '', // TODO: Add recorderId parameter to integrateSegments
        sessionId: session.id,
        segmentId,
      });
      toastService.success('Segment rendered successfully');
    } catch (error) {
      console.error('Failed to render segment:', error);
      toastService.error('Failed to render segment');
    }
  });
};
