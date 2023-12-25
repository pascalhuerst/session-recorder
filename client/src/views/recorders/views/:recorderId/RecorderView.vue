<script setup lang="ts">
import WaveformEditor from "../../../../lib/waveform/WaveformEditor.vue";
import waveFormUrl from "../../../../assets/waveform/07023003-2channel.dat?url";
import mp3FileUrl from "../../../../assets/waveform/07023003.mp3?url";
</script>

<template>
  <router-view />
  <WaveformEditor :waveform-url="waveFormUrl" :audio-urls="[{ type: 'audio/mp3', src: mp3FileUrl}]" />
</template>
