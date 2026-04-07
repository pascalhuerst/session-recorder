import { createSegment } from './procedures/createSegment';
import { updateSegment } from './procedures/updateSegment';
import { deleteSegment } from './procedures/deleteSegment';
import { renderSegment } from './procedures/renderSegment';
import type { PeaksContext } from '@session-recorder/session-waveform';
import type { Session } from '../types';
import { SegmentState } from '@session-recorder/protocols/ts/sessionsource';
import { toastService } from '../services/Toaster/ToastService';

export const integrateSegments = (
  session: Session,
  recorderId: string,
  ctx: PeaksContext
) => {
  const unbinds: Array<() => void> = [];

  unbinds.push(ctx.eventEmitter.on('segmentAdded', async (segment) => {
    try {
      // Peaks.js uses seconds (float), convert to protobuf Timestamp (seconds + nanos)
      await createSegment({
        recorderId,
        sessionId: session.id,
        segmentId: segment.id,
        segment: {
          timeStart: {
            seconds: Math.floor(segment.startTime).toString(),
            nanos: Math.floor((segment.startTime % 1) * 1e9),
          },
          timeEnd: {
            seconds: Math.floor(segment.endTime).toString(),
            nanos: Math.floor((segment.endTime % 1) * 1e9),
          },
          name: segment.labelText,
          state: SegmentState.UNKNOWN,
        },
      });
    } catch (error) {
      console.error('Failed to create segment:', error);
      toastService.error('Failed to create segment');
    }
  }));

  unbinds.push(ctx.eventEmitter.on('segmentUpdated', async (segmentId, _, segment) => {
    try {
      // Peaks.js uses seconds (float), convert to protobuf Timestamp (seconds + nanos)
      await updateSegment({
        recorderId,
        sessionId: session.id,
        segmentId: segmentId,
        segment: {
          timeStart: {
            seconds: Math.floor(segment.startTime).toString(),
            nanos: Math.floor((segment.startTime % 1) * 1e9),
          },
          timeEnd: {
            seconds: Math.floor(segment.endTime).toString(),
            nanos: Math.floor((segment.endTime % 1) * 1e9),
          },
          name: segment.labelText,
          state: SegmentState.UNKNOWN,
        },
      });
    } catch (error) {
      console.error('Failed to update segment:', error);
      toastService.error('Failed to update segment');
    }
  }));

  unbinds.push(ctx.eventEmitter.on('segmentRemoved', async (segmentId) => {
    try {
      await deleteSegment({
        recorderId,
        sessionId: session.id,
        segmentId,
      });
    } catch (error) {
      console.error('Failed to delete segment:', error);
      toastService.error('Failed to delete segment');
    }
  }));

  unbinds.push(ctx.eventEmitter.on('segmentsBulkDeleted', (count) => {
    const label = count === 1 ? 'segment' : 'segments';
    toastService.success(`${count} ${label} deleted`);
  }));

  unbinds.push(ctx.commandEmitter.on('renderSegment', async (segmentId) => {
    try {
      await renderSegment({
        recorderId,
        sessionId: session.id,
        segmentId,
      });
      // Don't show success toast here - status update comes from server stream
    } catch (error) {
      console.error('Failed to render segment:', error);
      toastService.error('Failed to render segment');
    }
  }));

  return () => unbinds.forEach((unbind) => unbind());
};
