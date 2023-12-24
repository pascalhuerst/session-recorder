import { computed, inject, ref } from "vue";
import { streamRecorders } from "../grpc/procedures/streamRecorders.ts";
import { RecorderInfo } from "@session-recorder/protocols/ts/sessionsource.ts";
import { useRoute } from "vue-router";

const recordersInjectionKey = Symbol("recorders");

export const createRecordersContext = () => {
  const route = useRoute();

  const recorders = ref<Map<string, RecorderInfo>>(new Map);
  inject(recordersInjectionKey, recorders);

  const selectedRecorderId = computed(() => route.params.recorderId as string | undefined);

  streamRecorders({
    onMessage: (recorderInfo) => {
      recorders.value.set(recorderInfo.ID, recorderInfo);
    }
  });

  return { recorders, selectedRecorderId };
};