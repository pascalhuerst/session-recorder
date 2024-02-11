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
}: CreateSegmentControlsProps & ReturnType<typeof createPeaksCanvas>) => {
  watch(
    [segments, peaks],
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
      deep: true,
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
    segments.value.push({
      id: segmentId,
      startTime: Number(peaks.value?.player.getCurrentTime()),
      endTime: Number(peaks.value?.player.getCurrentTime()) + 5,
      color: '#ed64a6',
      labelText: '0 to 10.5 seconds non-editable demo segment',
      editable: true,
      startIndex: intToChar(size),
      endIndex: intToChar(size + 1),
    });
  };

  const selectSegment = (segmentId: string) => {
    const segment = peaks.value?.segments.getSegment(segmentId);
    if (segment) {
      peaks.value?.player.seek(segment.startTime);
    }
  };

  return { segments, addSegment, selectSegment };
};
