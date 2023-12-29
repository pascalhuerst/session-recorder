<script setup lang="ts">
import WaveformEditor from '../../../../../../web/session-waveform/src/waveform/WaveformEditor.vue';
import waveFormUrl from '../../../../assets/waveform/07030039.dat?url';
import mp3FileUrl from '../../../../assets/waveform/Symphony No.6 (1st movement).mp3?url';
</script>

<template>
  <router-view />
  <WaveformEditor :audio-urls="[{ type: 'audio/mp3', src: mp3FileUrl }]" />
</template>
