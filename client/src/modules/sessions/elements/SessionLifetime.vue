<script setup lang="ts">
import { OpenSession } from "../../../grpc/procedures/streamSessions.ts";
import Switch from "../../../lib/forms/Switch.vue";
import { computed, ref } from "vue";

const props = defineProps<{
  session: OpenSession
}>();

const innerValue = ref(!!props.session.flagged);

const isFlagged = computed({
  get: () => innerValue.value,
  set: (val: boolean) => {
    console.log(val);
    innerValue.value = val;
  }
});

const ttl = computed(() => {
  const val = Math.floor(props.session.hours_to_live);
  if (val > 0) {
    return `${val} hours`;
  }

  const minutes = Math.floor(props.session.hours_to_live * 60);
  return `${minutes} minutes`;
});
</script>

<template>
  <div class="lifetime">
    <div v-if="!isFlagged" class="balance">
      {{ ttl }} until deleted
    </div>
    <Switch :model-value="isFlagged">
      <span>Keep session</span>
    </Switch>
  </div>
</template>

<style scoped>
.lifetime {
  display: flex;
  align-items: center;
  gap: var(--size-3);
  font-size: var(--scale-00);
}

.balance {
  color: var(--color-red-300);
}
</style>