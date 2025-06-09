import { reactive, watch } from 'vue';
import { streamSessions } from '../grpc/procedures/streamSessions';
import { useRecordersStore } from './useRecordersStore';
import { defineStore, storeToRefs } from 'pinia';
import type { Session } from '@session-recorder/protocols/ts/sessionsource';

export const useSessionsStore = defineStore('sessions', () => {
  const { selectedRecorderId } = storeToRefs(useRecordersStore());
  const sessions = reactive<Session[]>([]);

  watch(
    selectedRecorderId,
    () => {
      if (!selectedRecorderId.value) {
        return;
      }

      sessions.splice(0, sessions.length);

      streamSessions({
        request: {
          recorderID: selectedRecorderId.value,
        },
        onMessage: (session) => {
          console.log('Received session:', session);

          if (session.info.oneofKind === 'removed') {
            const index = sessions.findIndex((s) => s.iD === session.iD);
            if (index !== -1) {
              sessions.splice(index, 1);
            } else {
              console.warn('❌ Session not found for removal:', session.iD);
            }
          } else if (session.info.oneofKind === 'updated') {
            const existingIndex = sessions.findIndex(
              (s) => s.iD === session.iD
            );
            if (existingIndex !== -1) {
              sessions[existingIndex] = session;
            } else {
              sessions.push(session);
            }
          } else {
            console.warn('Unknown session info type:', session.info);
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

  return { sessions };
});
