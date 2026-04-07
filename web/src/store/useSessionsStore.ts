import { computed, ref, watch } from 'vue';
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
        connect: (handlers) => {
          // Track which session IDs the server sends on this (re)connect
          // so we can prune any that no longer exist on the backend.
          const seen = new Set<string>();
          let pruneTimer: ReturnType<typeof setTimeout> | null = null;

          return streamSessions({
            request: { recorderID },
            onMessage: (msg) => {
              // Guard against messages from a stale stream arriving after
              // the user switched to a different recorder tab. The async
              // iterator in streamSessions may have buffered a message
              // before the AbortController signal was processed.
              if (selectedRecorderId.value !== recorderID) return;

              handlers.onMessage();

              switch (msg.type) {
                case 'deleted': {
                  sessions.value = sessions.value.filter(
                    (s) => s.id !== msg.id
                  );
                  break;
                }

                case 'updated': {
                  seen.add(msg.session.id);

                  // Debounced pruning: after a 100ms gap in messages, prune
                  // sessions the server didn't mention (deleted while we were
                  // disconnected). This is resilient to gRPC-Web transports
                  // that yield messages across multiple microtasks due to
                  // HTTP/2 framing or Envoy buffering.
                  if (pruneTimer !== null) {
                    clearTimeout(pruneTimer);
                  }
                  pruneTimer = setTimeout(() => {
                    pruneTimer = null;
                    sessions.value = sessions.value.filter((s) =>
                      seen.has(s.id)
                    );
                  }, 100);

                  const excludeUpdated = sessions.value.filter(
                    (s) => s.id !== msg.session.id
                  );
                  sessions.value = [...excludeUpdated, msg.session];
                }
              }
            },
            onError: handlers.onError,
            onEnd: handlers.onEnd,
          });
        },
      });
    },
    {
      immediate: true,
    }
  );

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
