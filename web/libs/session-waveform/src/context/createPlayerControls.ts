import { ref } from 'vue';
import { onClickOutside } from '@vueuse/core';
import type { createPeaksCanvas } from './createPeaksCanvas';

export const createPlayerControls = ({
  peaks,
  eventEmitter,
  commandEmitter,
  layout: { canvasElement },
}: ReturnType<typeof createPeaksCanvas>) => {
  const canPlay = ref(false);
  const isPlaying = ref(false);
  const currentTime = ref(0);
  const duration = ref(0);

  commandEmitter.on('play', () => {
    console.log(peaks.value?.player);
    peaks.value?.player.play();
  });

  commandEmitter.on('pause', () => {
    peaks.value?.player.pause();
  });

  commandEmitter.on('seek', (seconds) => {
    peaks.value?.player.seek(seconds);
  });

  eventEmitter.on('ready', () => {
    peaks.value?.on('player.canplay', () => {
      console.log('player.canplay', peaks.value);
      canPlay.value = true;
      duration.value = peaks.value?.player.getDuration() || 0;
    });

    peaks.value?.on('player.playing', () => {
      console.log('player.playing', peaks.value);
      eventEmitter.emit('playbackStarted');
    });

    peaks.value?.on('player.pause', () => {
      eventEmitter.emit('playbackPaused');
    });

    peaks.value?.on('player.ended', () => {
      eventEmitter.emit('playbackEnded');
    });

    peaks.value?.on('player.timeupdate', (time) => {
      currentTime.value = time;
    });
  });

  eventEmitter.on('playbackStarted', () => {
    isPlaying.value = true;
  });

  eventEmitter.on('playbackPaused', () => {
    isPlaying.value = false;
  });

  eventEmitter.on('playbackEnded', () => {
    isPlaying.value = false;
  });

  onClickOutside(canvasElement.value, () => {
    if (isPlaying.value) {
      peaks.value?.player.pause();
    }
  });

  return {
    canPlay,
    isPlaying,
    currentTime,
    duration,
  };
};
