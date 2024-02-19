import type { createPeaksModule } from './createPeaksModule';
import uuid from 'uuidv4';
import type { Segment } from './models/state';

const intToChar = (int: number) => {
  const start = 'a'.charCodeAt(0);
  return String.fromCharCode(start + int).toUpperCase();
};

export const installSegmentsControls = ({
  state,
  eventEmitter,
  commandEmitter,
}: ReturnType<typeof createPeaksModule>) => {
  eventEmitter.on('ready', (peaks) => {
    const { segments } = state.get();

    segments.forEach((segment) => {
      peaks.segments.add(segment);
    });

    commandEmitter.on('addSegment', () => {
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
        startTime: Number(peaks.player.getCurrentTime()),
        endTime: Math.min(
          Number(peaks.player.getDuration()),
          Number(peaks.player.getCurrentTime()) + 60
        ),
        color: '#ed64a6',
        labelText: `Segment ${startIndex}-${endIndex}`,
        editable: permissions.update,
        startIndex,
        endIndex,
      } satisfies Segment;

      console.log(segment);
      peaks.segments.add(segment);
    });

    commandEmitter.on('updateSegment', (segmentId, patch) => {
      const { permissions } = state.get();
      if (!permissions.update) {
        return;
      }

      const segment = peaks?.segments.getSegment(segmentId);
      if (!segment) {
        return;
      }

      segment.update({ ...patch } as any);
    });

    commandEmitter.on('removeSegment', (segmentId) => {
      const { permissions } = state.get();
      if (!permissions.delete) {
        return;
      }

      peaks.segments.removeById(segmentId);
    });

    peaks.on('segments.add', (event) => {
      const { permissions } = state.get();
      event.segments.forEach((segment) => {
        eventEmitter.emit('segmentAdded', {
          id: String(segment.id),
          startTime: segment.startTime,
          endTime: segment.endTime,
          color: '#ed64a6',
          labelText: segment.labelText,
          editable: permissions.update,
          startIndex: String(segment.startIndex),
          endIndex: String(segment.endIndex),
        });
      });
    });

    peaks.on('segments.remove', (event) => {
      event.segments.forEach((segment) => {
        eventEmitter.emit('segmentRemoved', segment.id!);
      });
    });
  });

  eventEmitter.on('segmentAdded', (segment) => {
    state.update((prev) => ({
      ...prev,
      segments: [...prev.segments, segment],
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
      return { ...prev, segments };
    });
  });
};
