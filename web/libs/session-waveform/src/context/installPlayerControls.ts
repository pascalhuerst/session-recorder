import type { createPeaksModule } from './createPeaksModule';

export const installPlayerControls = ({
  state,
  eventEmitter,
  commandEmitter,
}: ReturnType<typeof createPeaksModule>) => {
  eventEmitter.on('ready', (peaks) => {
    const player = peaks.player;

    // @todo: for some reason player.canplay doesn't fire unless you interact
    // with the player
    player.seek(0);

    peaks.on('player.canplay', () => {
      state.update((prev) => ({
        ...prev,
        player: {
          ...peaks.player,
          duration: peaks.player.getDuration(),
        },
      }));
    });

    peaks.on('player.playing', () => {
      eventEmitter.emit('playbackStarted');
    });

    peaks.on('player.pause', () => {
      eventEmitter.emit('playbackPaused');
    });

    peaks.on('player.ended', () => {
      eventEmitter.emit('playbackEnded');
    });

    peaks.on('player.timeupdate', (currentTime: number) => {
      state.update((prev) => ({
        ...prev,
        player: {
          ...prev.player,
          currentTime,
        },
      }));
    });

    commandEmitter.on('play', () => {
      player?.play();
    });

    commandEmitter.on('pause', () => {
      player?.pause();
    });

    commandEmitter.on('seek', (seconds) => {
      player?.seek(seconds);
    });
  });

  eventEmitter.on('playbackStarted', () => {
    state.update((prev) => {
      return {
        ...prev,
        player: {
          ...prev.player,
          isPlaying: true,
        },
      };
    });
  });

  eventEmitter.on('playbackPaused', () => {
    state.update((prev) => {
      return {
        ...prev,
        player: {
          ...prev.player,
          isPlaying: false,
        },
      };
    });
  });

  eventEmitter.on('playbackEnded', () => {
    state.update((prev) => {
      return {
        ...prev,
        player: {
          ...prev.player,
          isPlaying: false,
        },
      };
    });
  });

  eventEmitter.on('clickOutsideCanvas', () => {
    if (state.select((st) => st.player.isPlaying)) {
      commandEmitter.emit('pause');
    }
  });
};
