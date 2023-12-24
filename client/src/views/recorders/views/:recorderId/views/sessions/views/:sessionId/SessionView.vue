<script setup lang="ts">
import { useSessionsStore } from "../../../../../../../../store/useSessionsStore";
import { useRoute } from "vue-router";
import { computed } from "vue";
import { storeToRefs } from "pinia";
import SessionHeader from "../../../../../../elements/SessionHeader.vue";
import { useRecordersStore } from "../../../../../../../../store/useRecordersStore.ts";
import SessionEditor from "../../../../../../elements/SessionEditor.vue";

const route = useRoute();
const sessionId = computed(() => route.params.sessionId as string);

const { selectedRecorderId } = storeToRefs(useRecordersStore());
const { sessions } = storeToRefs(useSessionsStore());

const session = computed(() => {
  console.log(sessions.value);
  return sessions.value.get(sessionId.value);
});
</script>

<template>
  <div v-if="session">
    <SessionHeader :session="session" :recorder-id="selectedRecorderId" />
    <SessionEditor :session="session" :recorder-id="selectedRecorderId" />
  </div>
</template>

<style scoped>

</style>