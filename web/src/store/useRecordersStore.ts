import { computed, ref } from 'vue';
import { Recorder } from '@session-recorder/protocols/ts/sessionsource';
import { useRoute, useRouter } from 'vue-router';
import { defineStore } from 'pinia';
import { streamRecorders } from '../grpc/procedures/streamRecorders';
import { reconnectingStream } from '../grpc/reconnectingStream';

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

  const { stop } = reconnectingStream({
    name: 'recorders',
    connect: (handlers) =>
      streamRecorders({
        onMessage: (recorderInfo) => {
          handlers.onMessage();
          recorders.value.set(recorderInfo.recorderID, recorderInfo);
        },
        onError: handlers.onError,
        onEnd: handlers.onEnd,
      }),
  });

  return { recorders, selectedRecorderId, dispose: stop };
});
