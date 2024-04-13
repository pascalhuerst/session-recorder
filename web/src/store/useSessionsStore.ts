import { ref, watch } from 'vue';
import { streamSessions } from '../grpc/procedures/streamSessions';
import { useRecordersStore } from './useRecordersStore';
import { type Session } from '@session-recorder/protocols/ts/sessionsource';
import { defineStore, storeToRefs } from 'pinia';

export const useSessionsStore = defineStore('sessions', () => {
  const { selectedRecorderId } = storeToRefs(useRecordersStore());
  const sessions = ref<Map<string, Session>>(new Map());

  watch(
    selectedRecorderId,
    () => {
      if (!selectedRecorderId.value) {
        return;
      }

      sessions.value.clear();

      streamSessions({
        request: {
          recorderID: selectedRecorderId.value,
        },
        onMessage: (session) => {
          if (session.removed) {
            sessions.value.delete(session.ID);
          } else {
            sessions.value.set(session.ID, session);
          }
        },
      });
    },
    {
      immediate: true,
    }
  );

  return { sessions };
});
