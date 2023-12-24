import { ref, watch } from "vue";
import { streamSessions } from "../grpc/procedures/streamSessions.ts";
import { useRecordersStore } from "./useRecordersStore.ts";
import { SessionInfo } from "@session-recorder/protocols/ts/sessionsource.ts";
import { defineStore, storeToRefs } from "pinia";

export const useSessionsStore = defineStore("sessions", () => {
  const { selectedRecorderId } = storeToRefs(useRecordersStore());
  const sessions = ref<Map<string, SessionInfo>>(new Map);

  watch(selectedRecorderId, () => {
    if (!selectedRecorderId.value) {
      return;
    }

    sessions.value.clear();

    streamSessions({
      request: {
        recorderID: selectedRecorderId.value
      },
      onMessage: (session) => {
        sessions.value.set(session.ID, session);
      }
    });
  }, {
    immediate: true
  });

  return { sessions };
});
