<script setup lang="ts">
import { ToastContainer } from './services/Toaster';
</script>

<template>
  <router-view />
  <div id="modals" />
  <ToastContainer />
</template>

<style>
@import '../assets/styles/app.css';
</style>
