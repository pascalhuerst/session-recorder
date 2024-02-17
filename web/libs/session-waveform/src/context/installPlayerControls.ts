import type { createPeaksModule } from './createPeaksModule';

export const installPlayerControls = ({
  state,
  peaks,
  eventEmitter,
  commandEmitter,
}: ReturnType<typeof createPeaksModule>) => {
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
      state.update((prev) => {
        return {
          ...prev,
          duration: peaks.value?.player.getDuration() || 0,
        };
      });
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

    peaks.value?.on('player.timeupdate', (currentTime) => {
      state.update((prev) => {
        return {
          ...prev,
          currentTime,
        };
      });
    });
  });

  eventEmitter.on('playbackStarted', () => {
    state.update((prev) => {
      return {
        ...prev,
        isPlaying: true,
      };
    });
  });

  eventEmitter.on('playbackPaused', () => {
    state.update((prev) => {
      return {
        ...prev,
        isPlaying: false,
      };
    });
  });

  eventEmitter.on('playbackEnded', () => {
    state.update((prev) => {
      return {
        ...prev,
        isPlaying: false,
      };
    });
  });
};
