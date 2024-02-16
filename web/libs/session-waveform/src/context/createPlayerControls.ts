import { ref } from 'vue';
import type { createPeaksCanvas } from './createPeaksCanvas';

export const createPlayerControls = ({
  peaks,
  eventEmitter,
  commandEmitter,
}: ReturnType<typeof createPeaksCanvas>) => {
  const isPlaying = ref(false);
  const currentTime = ref(0);
  const duration = ref(0);

  commandEmitter.on('play', () => {
    peaks.value?.player.play();
  });

  commandEmitter.on('pause', () => {
    peaks.value?.player.pause();
  });

  commandEmitter.on('seek', (seconds) => {
    peaks.value?.player.seek(seconds);
  });

  eventEmitter.on('ready', () => {
    // @todo: for some reason player.canplay doesn't fire unless you interact
    // with the player
    peaks.value?.player.seek(0);

    peaks.value?.on('player.canplay', () => {
      duration.value = peaks.value?.player.getDuration() || 0;
    });

    peaks.value?.on('player.playing', () => {
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

  return {
    isPlaying,
    currentTime,
    duration,
  };
};
