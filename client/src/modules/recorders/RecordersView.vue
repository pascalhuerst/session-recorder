<script setup lang="ts">
import { useRouter } from "vue-router";
import { onBeforeMount } from "vue";
import { getRecordingDevices } from "../../client/getRecordingDevices.ts";

const router = useRouter();

onBeforeMount(() => {
  getRecordingDevices().then((devices) => {
    router.replace(`/recorders/${devices.at(0)}/sessions`);
  });
});

// @todo: Maybe we want to an intro page here ??
</script>

<template>

</template>

<style scoped>

</style>