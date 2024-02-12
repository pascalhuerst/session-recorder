import type { Permissions, Segment } from '../types';
import { type Ref, watch } from 'vue';
import type { createPeaksCanvas } from './createPeaksCanvas';
import uuid from 'uuidv4';

export type CreateSegmentControlsProps = {
  segments: Ref<Segment[]>;
  permissions: Ref<Permissions>;
};

export const createSegmentControls = ({
  peaks,
  segments,
  permissions,
  emitter,
}: CreateSegmentControlsProps & ReturnType<typeof createPeaksCanvas>) => {
  watch(
    peaks,
    () => {
      if (peaks.value && segments.value.length) {
        try {
          segments.value.forEach((segment) => {
            peaks.value?.segments.removeById(segment.id);
            peaks.value?.segments.add(segment);
          });
        } catch (err) {
          console.log(err);
        }
        // const overview = peaks.value?.views.getView("overview");
        // overview?.setPlayedWaveformColor("#767c89");
        //
        // const zoomview = peaks.value?.views.getView("zoomview");
        // zoomview?.setPlayedWaveformColor("#767c89");
      }
    },
    {
      immediate: true,
    }
  );

  const intToChar = (int: number) => {
    const start = 'a'.charCodeAt(0);
    return String.fromCharCode(start + int).toUpperCase();
  };

  const addSegment = () => {
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
      endTime: Number(peaks.value?.player.getCurrentTime()) + 5,
      color: '#ed64a6',
      labelText: `Segment ${startIndex}-${endIndex}`,
      editable: true,
      startIndex,
      endIndex,
    } satisfies Segment;

    segments.value.push(segment);
    emitter.emit('ui.segments.add', segment);
  };

  const selectSegment = (segmentId: string) => {
    emitter.emit('ui.segment.selectById', segmentId);
  };

  emitter.on('ui.segment.selectById', (segmentId) => {
    const segment = peaks.value?.segments.getSegment(segmentId);
    if (segment) {
      peaks.value?.player.seek(segment.startTime);
    }
  });

  emitter.on('ui.segments.add', (segment) => {
    peaks.value?.segments.add(segment);
  });

  emitter.on('peaks.segment.updateById', (segmentId, patch) => {
    const index = segments.value.findIndex((el) => el.id === segmentId);
    if (index > -1) {
      segments.value.splice(index, 1, {
        ...segments.value[index],
        ...patch,
      });
    }
  });

  return { segments, addSegment, selectSegment };
};
