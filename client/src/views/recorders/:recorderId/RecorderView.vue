<script setup lang="ts">
import Container from "../../../lib/layout/Container.vue";
import DevicePicker from "../../../shared/DevicePicker.vue";
import { storeToRefs } from "pinia";
import { useRecordersStore } from "../../../store/useRecordersStore";

const { recorders, selectedRecorderId } = storeToRefs(useRecordersStore());
</script>

<template>
  <router-view />
</template>
