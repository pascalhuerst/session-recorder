import type { Permissions, Segment } from '../types';
import { type Ref } from 'vue';
import type { createPeaksCanvas } from './createPeaksCanvas';
import uuid from 'uuidv4';

export type CreateSegmentControlsProps = {
  segments: Ref<Segment[]>;
  permissions: Ref<Permissions>;
};

const intToChar = (int: number) => {
  const start = 'a'.charCodeAt(0);
  return String.fromCharCode(start + int).toUpperCase();
};

export const createSegmentControls = ({
  peaks,
  segments,
  permissions,
  eventEmitter,
  commandEmitter,
}: CreateSegmentControlsProps & ReturnType<typeof createPeaksCanvas>) => {
  commandEmitter.on('addSegment', () => {
    if (!permissions.value.create) {
      return;
    }

    const segmentId = uuid();
    const size = segments.value.length * 2;

    const startIndex = intToChar(size);
    const endIndex = intToChar(size + 1);

    const segment = {
      id: segmentId,
      startTime: Number(peaks.value?.player.getCurrentTime()),
      endTime: Math.min(
        Number(peaks.value?.player.getDuration()),
        Number(peaks.value?.player.getCurrentTime()) + 60
      ),
      color: '#ed64a6',
      labelText: `Segment ${startIndex}-${endIndex}`,
      editable: permissions.value.update,
      startIndex,
      endIndex,
    } satisfies Segment;

    peaks.value?.segments.add(segment);
  });

  commandEmitter.on('updateSegment', () => {
    // @todo: implement
    if (!permissions.value.update) {
      return;
    }
  });

  commandEmitter.on('removeSegment', (segmentId) => {
    if (!permissions.value.delete) {
      return;
    }

    peaks.value?.segments.removeById(segmentId);
  });

  eventEmitter.on('ready', () => {
    segments.value.forEach((segment) => {
      console.log(segment);
      peaks.value?.segments.add(segment);
    });

    peaks.value?.on('segments.add', (event) => {
      event.segments.forEach((segment) => {
        eventEmitter.emit('segmentAdded', {
          id: String(segment.id),
          startTime: segment.startTime,
          endTime: segment.endTime,
          color: '#ed64a6',
          labelText: segment.labelText,
          editable: permissions.value.update,
          startIndex: String(segment.startIndex),
          endIndex: String(segment.endIndex),
        });
      });
    });

    peaks.value?.on('segments.remove', (event) => {
      event.segments.forEach((segment) => {
        eventEmitter.emit('segmentRemoved', segment.id!);
      });
    });
  });

  eventEmitter.on('segmentAdded', (segment) => {
    segments.value.push(segment);
  });

  eventEmitter.on('segmentRemoved', (segmentId) => {
    const index = segments.value.findIndex((el) => el.id === segmentId);
    if (index > -1) {
      segments.value.splice(index, 1);
    }
  });

  eventEmitter.on('segmentUpdated', (segmentId, patch) => {
    const index = segments.value.findIndex((el) => el.id === segmentId);
    if (index > -1) {
      segments.value.splice(index, 1, {
        ...segments.value[index],
        ...patch,
      });
    }
  });

  return { segments };
};
