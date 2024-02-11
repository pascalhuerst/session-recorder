import { computed, onMounted, onUnmounted, ref, shallowRef, watch } from 'vue';
import type { PeaksOptions } from 'peaks.js';
import { onClickOutside } from '@vueuse/core';
import type { createPeaksCanvas } from './createPeaksCanvas';

export const createPlayerControls = (
  props: ReturnType<typeof createPeaksCanvas>
) => {
  const {
    peaks,
    layout: { canvasElement },
  } = props;

  const isPlaying = ref(false);
  const currentTime = ref(0);

  const player = computed(() => {
    return peaks.value?.player as PeaksOptions['player'] | undefined;
  });

  const duration = computed(() => {
    return player.value?.getDuration();
  });

  watch(
    player,
    () => {
      if (player.value) {
        currentTime.value = player.value.getCurrentTime();
      }
    },
    { immediate: true }
  );

  const play = () => {
    player.value?.play();
    isPlaying.value = true;
  };

  const pause = () => {
    isPlaying.value = false;
    player.value?.pause();
  };

  const seek = (seconds: number) => {
    player.value?.seek(seconds);
  };

  const interval = shallowRef();

  onMounted(() => {
    currentTime.value = interval.value = setInterval(() => {
      if (peaks.value) {
        currentTime.value = peaks.value.player.getCurrentTime();
      }
    }, 100);

    onClickOutside(canvasElement.value, () => {
      if (isPlaying.value) {
        player.value?.pause();
        isPlaying.value = false;
      }
    });
  });

  onUnmounted(() => {
    clearInterval(interval.value);
  });

  return {
    player,
    isPlaying,
    currentTime,
    duration,
    play,
    pause,
    seek,
  };
};
