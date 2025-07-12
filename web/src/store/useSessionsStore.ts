import { computed, onBeforeUnmount, ref, watch } from 'vue';
import { streamSessions } from '../grpc/procedures/streamSessions';
import { useRecordersStore } from './useRecordersStore';
import { defineStore, storeToRefs } from 'pinia';
import type { Session } from '../types';
import { noop } from '@vueuse/core';

export const useSessionsStore = defineStore('sessions', () => {
  const { selectedRecorderId } = storeToRefs(useRecordersStore());
  const sessions = ref<Session[]>([]);

  let stop = noop;

  watch(
    selectedRecorderId,
    () => {
      stop();
      stop = noop;

      if (!selectedRecorderId.value) {
        return;
      }

      sessions.value = [];

      stop = streamSessions({
        request: {
          recorderID: selectedRecorderId.value,
        },
        onMessage: (msg) => {
          console.log('Received message:', msg);

          switch (msg.type) {
            case 'deleted': {
              sessions.value = sessions.value.filter((s) => s.id !== msg.id);
              break;
            }

            case 'updated': {
              const excludeUpdated = sessions.value.filter(
                (s) => s.id !== msg.session.id
              );
              sessions.value = [...excludeUpdated, msg.session];
            }
          }
        },
        onError: (err) => {
          console.error('Sessions stream error:', err);
        },
        onEnd: () => {
          console.log('Sessions stream ended');
        },
      });
    },
    {
      immediate: true,
    }
  );

  onBeforeUnmount(() => {
    stop();
  });

  const sortedSessions = computed(() => {
    return sessions.value.sort((a, b) => {
      return (
        new Date(b.finishedAt ?? 0).getTime() -
        new Date(a.finishedAt ?? 0).getTime()
      );
    });
  });

  return { sessions: sortedSessions };
});
