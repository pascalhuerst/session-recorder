import { computed, ref } from "vue";
import { streamRecorders } from "../grpc/procedures/streamRecorders.ts";
import { RecorderInfo } from "@session-recorder/protocols/ts/sessionsource.ts";
import { useRoute, useRouter } from "vue-router";
import { defineStore } from "pinia";

export const useRecordersStore = defineStore("recorders", () => {
  const router = useRouter();
  const route = useRoute();

  const recorders = ref<Map<string, RecorderInfo>>(new Map);
  const selectedRecorderId = computed({
    get: () => {
      return route.params.recorderId as string;
    },
    set: (id: string) => {
      router.replace(`/recorders/${id}/sessions`);
    }
  });

  streamRecorders({
    onMessage: (recorderInfo) => {
      recorders.value.set(recorderInfo.ID, recorderInfo);
    }
  }).catch((err) => {
    console.error(err);
  });

  return { recorders, selectedRecorderId };
});