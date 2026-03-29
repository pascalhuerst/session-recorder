import { computed, onBeforeUnmount, ref, watch } from 'vue';
import { streamSessions } from '../grpc/procedures/streamSessions';
import { useRecordersStore } from './useRecordersStore';
import { defineStore, storeToRefs } from 'pinia';
import type { Session } from '../types';
import {
  reconnectingStream,
  type ReconnectingStreamHandle,
} from '../grpc/reconnectingStream';

export const useSessionsStore = defineStore('sessions', () => {
  const { selectedRecorderId } = storeToRefs(useRecordersStore());
  const sessions = ref<Session[]>([]);

  let handle: ReconnectingStreamHandle | null = null;

  watch(
    selectedRecorderId,
    () => {
      handle?.stop();
      handle = null;

      if (!selectedRecorderId.value) {
        return;
      }

      sessions.value = [];

      const recorderID = selectedRecorderId.value;

      handle = reconnectingStream({
        name: `sessions(${recorderID})`,
        connect: (handlers) =>
          streamSessions({
            request: { recorderID },
            onMessage: (msg) => {
              handlers.onMessage();

              switch (msg.type) {
                case 'deleted': {
                  sessions.value = sessions.value.filter(
                    (s) => s.id !== msg.id
                  );
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
            onError: handlers.onError,
            onEnd: handlers.onEnd,
          }),
        onReconnecting: () => {
          // Clear stale sessions on reconnect so we get fresh state
          sessions.value = [];
        },
      });
    },
    {
      immediate: true,
    }
  );

  onBeforeUnmount(() => {
    handle?.stop();
  });

  /** Force-reconnect the sessions stream to fetch fresh state */
  const reconnect = () => {
    handle?.reconnect();
  };

  const sortedSessions = computed(() => {
    return [...sessions.value].sort((a, b) => {
      // Recording/processing sessions (no finishedAt) go to top
      if (!a.finishedAt && b.finishedAt) return -1;
      if (a.finishedAt && !b.finishedAt) return 1;
      if (!a.finishedAt && !b.finishedAt) {
        // Both in-progress - sort by startedAt descending
        return b.startedAt.getTime() - a.startedAt.getTime();
      }
      return b.finishedAt.getTime() - a.finishedAt.getTime();
    });
  });

  return { sessions: sortedSessions, reconnect };
});
