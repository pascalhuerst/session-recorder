import type { createPeaksModule } from './createPeaksModule';

export const installPlayerControls = ({
  state,
  eventEmitter,
  commandEmitter,
}: ReturnType<typeof createPeaksModule>) => {
  let commandUnbinds: Array<() => void> = [];

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
          currentTime: peaks.player.getCurrentTime(),
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

    // Clean up previous command handlers before re-registering
    commandUnbinds.forEach((unbind) => unbind());
    commandUnbinds = [];

    commandUnbinds.push(commandEmitter.on('play', () => {
      player?.play();
    }));

    commandUnbinds.push(commandEmitter.on('pause', () => {
      player?.pause();
    }));

    commandUnbinds.push(commandEmitter.on('seek', (seconds) => {
      player?.seek(seconds);
    }));
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
