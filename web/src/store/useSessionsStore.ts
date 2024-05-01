import { reactive, watch } from 'vue';
import { streamSessions } from '../grpc/procedures/streamSessions';
import { useRecordersStore } from './useRecordersStore';
import { type Session } from '@session-recorder/protocols/ts/sessionsource';
import { defineStore, storeToRefs } from 'pinia';

export const useSessionsStore = defineStore('sessions', () => {
  const { selectedRecorderId } = storeToRefs(useRecordersStore());
  const sessions = reactive<Map<string, Session>>(new Map());

  watch(
    selectedRecorderId,
    () => {
      if (!selectedRecorderId.value) {
        return;
      }

      sessions.clear();

      streamSessions({
        request: {
          recorderID: selectedRecorderId.value,
        },
        onMessage: (session) => {
          if (session.removed) {
            sessions.delete(session.ID);
          } else {
            sessions.set(session.ID, session);
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
