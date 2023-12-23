<script setup lang="ts">
import { useRouter } from "vue-router";
import { onBeforeMount } from "vue";
import { streamRecorders } from "../../grpc/procedures/streamRecorders.ts";

const router = useRouter();

onBeforeMount(() => {
  streamRecorders({
    onMessage: (recorderInfo) => {
      router.replace(`/recorders/${recorderInfo.ID}/sessions`);
    }
  });
});

// @todo: Maybe we want to an intro page here ??
</script>

<template>

</template>

<style scoped>

</style>