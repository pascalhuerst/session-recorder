import { computed, ref } from 'vue';
import { Recorder } from '@session-recorder/protocols/ts/sessionsource';
import { useRoute, useRouter } from 'vue-router';
import { defineStore } from 'pinia';
import { streamRecorders } from '../grpc/procedures/streamRecorders';

export const useRecordersStore = defineStore('recorders', () => {
  const router = useRouter();
  const route = useRoute();

  const recorders = ref<Map<string, Recorder>>(new Map());
  const selectedRecorderId = computed({
    get: () => {
      return route.params.recorderId as string;
    },
    set: (id: string) => {
      router.replace(`/recorders/${id}/sessions`);
    },
  });

  streamRecorders({
    onMessage: (recorderInfo) => {
      recorders.value.set(recorderInfo.recorderID, recorderInfo);
    },
    onError: (err) => {
      console.error('Recorders stream error:', err);
    },
    onEnd: () => {
      console.log('Recorders stream ended');
    },
  });

  return { recorders, selectedRecorderId };
});
