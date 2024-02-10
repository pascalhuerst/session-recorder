import { computed, ref } from 'vue';
import type { PeaksOptions } from 'peaks.js';
import { onClickOutside } from '@vueuse/core';
import { usePeaksContext } from './usePeaksContext';

export const useAudioControls = () => {
  const { peaks, canvasElement } = usePeaksContext();

  const isPlaying = ref(false);

  const player = computed(() => {
    return peaks.value?.player as PeaksOptions['player'] | undefined;
  });

  const onToggle = () => {
    if (!player.value) {
      return;
    }

    if (isPlaying.value) {
      player.value.pause();
      isPlaying.value = false;
    } else {
      player.value.play();
      isPlaying.value = true;
    }

    onClickOutside(canvasElement.value, () => {
      if (isPlaying.value) {
        player.value?.pause();
        isPlaying.value = false;
      }
    });
  };

  return {
    onToggle,
    isPlaying,
    player,
  };
};
