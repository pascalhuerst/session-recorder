<script setup lang="ts">
import { computed } from "vue";
import { SessionInfo } from "@session-recorder/protocols/ts/sessionsource.ts";
import SessionMenu from "./SessionMenu.vue";
import Button from "../../../lib/controls/Button.vue";
import { FontAwesomeIcon } from "@fortawesome/vue-fontawesome";
import { useDateFormat } from "@vueuse/core";

const props = defineProps<{
  session: SessionInfo,
  recorderId: string
}>();

const createdAt = computed(() => {
  if (!props.session.timeCreated) {
    return { iso: "", formatted: "" };
  }
  const format = props.session.timeCreated.getFullYear() === new Date().getFullYear() ? "ddd, MMM D, HH:mm" : "MMM D, YYYY HH:mm";
  return {
    iso: props.session.timeCreated.toISOString(),
    formatted: useDateFormat(props.session.timeCreated, format).value
  };
});
</script>

<template>
  <div class="header">
    <div class="back">
      <Button
        tag-name="router-link"
        :to="`/recorders/${recorderId}/sessions`"
        aria-label="Back to sessions"
        shape="circle"
      >
        <font-awesome-icon icon="fa-solid fa-arrow-left"></font-awesome-icon>
      </Button>
    </div>
    <div class="heading">
      <h1 class="name">Untitled</h1>
      <time class="timestamp" :datetime="createdAt.iso">{{ createdAt.formatted }}</time>
    </div>
    <div class="menu">
      <SessionMenu :session="session" :recorder-id="recorderId" />
    </div>
  </div>
</template>

<style scoped>
.header {
  display: flex;
  flex-direction: row;
  align-items: flex-end;
  flex-wrap: nowrap;
  gap: var(--size-4);
  border-bottom: 1px solid var(--color-grey-300);
  padding: var(--size-4) 0;
}

.back {
  flex: none;
}

.heading {
  display: flex;
  flex-direction: column;
  gap: var(--size-1);
}

.name {
  font-size: var(--scale-1);
  font-weight: var(--weight-bold);
  color: var(--color-purple-700);
  text-decoration: none;
}

.timestamp {
  font-size: var(--scale-2);
  font-weight: var(--weight-semibold);
}

.menu {
  margin-left: auto;
}
</style>