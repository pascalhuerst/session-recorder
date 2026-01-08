import type { createPeaksModule } from './createPeaksModule';
import uuid from 'uuidv4';
import type { Segment } from './models/state';
import { getSegmentColor } from '../lib/utils/segmentColors';

/**
 * Convert an integer to a letter label (A-Z, then AA-AZ, BA-BZ, etc.)
 * Used for segment marker naming.
 */
export const intToChar = (int: number): string => {
  const A = 'A'.charCodeAt(0);
  if (int < 26) {
    return String.fromCharCode(A + int);
  }
  // After Z, use double letters: AA, AB, ..., AZ, BA, BB, ...
  const adjusted = int - 26;
  const first = Math.floor(adjusted / 26);
  const second = adjusted % 26;
  return String.fromCharCode(A + first) + String.fromCharCode(A + second);
};

export const installSegmentsControls = ({
  state,
  eventEmitter,
  commandEmitter,
}: ReturnType<typeof createPeaksModule>) => {
  let peaksInstance: Parameters<Parameters<typeof eventEmitter.on<'ready'>>[1]>[0] | null = null;

  // Register command handlers once, outside the ready event
  commandEmitter.on('addSegment', () => {
    if (!peaksInstance) return;

    const { permissions, segments } = state.get();
    if (!permissions.create) {
      return;
    }

    const segmentId = uuid();
    const size = segments.length * 2;

    const startIndex = intToChar(size);
    const endIndex = intToChar(size + 1);

    const segment = {
      id: segmentId,
      startTime: Number(peaksInstance.player.getCurrentTime()),
      endTime: Math.min(
        Number(peaksInstance.player.getDuration()),
        Number(peaksInstance.player.getCurrentTime()) + 60
      ),
      color: getSegmentColor(segments.length),
      labelText: `Segment ${startIndex}-${endIndex}`,
      editable: permissions.update,
      startIndex,
      endIndex,
      renders: [],
    } satisfies Segment;

    peaksInstance.segments.add(segment);

    // Try to show the entire segment in the zoomview
    const zoomview = peaksInstance.views.getView('zoomview');
    if (zoomview) {
      const viewDuration = zoomview.getEndTime() - zoomview.getStartTime();
      const segmentDuration = segment.endTime - segment.startTime;

      if (segmentDuration <= viewDuration) {
        // Segment fits in view - center it
        const centerTime = (segment.startTime + segment.endTime) / 2;
        const newStartTime = Math.max(0, centerTime - viewDuration / 2);
        zoomview.setStartTime(newStartTime);
      } else {
        // Segment too long - focus on end marker
        zoomview.setStartTime(Math.max(0, segment.endTime - viewDuration * 0.8));
      }
    }

    // Move playhead to end of the added segment
    peaksInstance.player.seek(segment.endTime);
  });

  commandEmitter.on('updateSegment', (segmentId, patch) => {
    if (!peaksInstance) return;

    const { permissions, segments: stateSegments } = state.get();
    if (!permissions.update) {
      return;
    }

    const segment = peaksInstance.segments.getSegment(segmentId);
    if (!segment) {
      return;
    }

    segment.update({ ...patch } as any);

    // Emit segmentUpdated after the update (not from CustomSegmentMarker.update
    // to avoid duplicate events during drag)
    const stateSegment = stateSegments.find((s) => s.id === segmentId);
    if (stateSegment) {
      eventEmitter.emit('segmentUpdated', segmentId, patch, {
        ...stateSegment,
        ...patch,
      });
    }
  });

  commandEmitter.on('removeSegment', (segmentId) => {
    if (!peaksInstance) return;

    const { permissions } = state.get();
    if (!permissions.delete) {
      return;
    }

    peaksInstance.segments.removeById(segmentId);
  });

  eventEmitter.on('ready', (peaks) => {
    peaksInstance = peaks;

    const { segments } = state.get();

    segments.forEach((segment) => {
      peaks.segments.add(segment);
    });

    peaks.on('segments.add', (event) => {
      const { permissions, segments: existingSegments } = state.get();
      event.segments.forEach((segment, index) => {
        // Skip segments that already exist in state (e.g., initial segments)
        if (existingSegments.some((s) => s.id === segment.id)) {
          return;
        }

        eventEmitter.emit('segmentAdded', {
          id: String(segment.id),
          startTime: segment.startTime,
          endTime: segment.endTime,
          color: typeof segment.color === 'string' ? segment.color : getSegmentColor(existingSegments.length + index),
          labelText: segment.labelText,
          editable: permissions.update,
          startIndex: String(segment.startIndex),
          endIndex: String(segment.endIndex),
          renders: [],
        });
      });
    });

    peaks.on('segments.remove', (event) => {
      event.segments.forEach((segment) => {
        eventEmitter.emit('segmentRemoved', segment.id!);
      });
    });

    peaks.on('segments.dragend', (event) => {
      const segment = event.segment;
      const stateSegment = state.get().segments.find((s) => s.id === segment.id);
      if (!stateSegment) {
        return;
      }

      eventEmitter.emit('segmentUpdated', segment.id!, {
        startTime: segment.startTime,
        endTime: segment.endTime,
      }, {
        ...stateSegment,
        startTime: segment.startTime,
        endTime: segment.endTime,
      });
    });
  });

  eventEmitter.on('segmentAdded', (segment) => {
    state.update((prev) => ({
      ...prev,
      segments: [...prev.segments, segment].sort(
        (a, b) => a.startTime - b.startTime
      ),
    }));
  });

  eventEmitter.on('segmentRemoved', (segmentId) => {
    state.update((prev) => {
      const segments = [...prev.segments];
      const index = segments.findIndex((el) => el.id === segmentId);
      if (index > -1) {
        segments.splice(index, 1, {
          ...segments[index],
          deleted: true,
        });
      }
      return { ...prev, segments };
    });
  });

  eventEmitter.on('segmentUpdated', (segmentId, patch) => {
    state.update((prev) => {
      const segments = [...prev.segments];
      const index = segments.findIndex((el) => el.id === segmentId);
      if (index > -1) {
        segments.splice(index, 1, {
          ...segments[index],
          ...patch,
        });
      }
      return {
        ...prev,
        segments: segments.sort((a, b) => a.startTime - b.startTime),
      };
    });
  });
};
