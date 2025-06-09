import { computed } from 'vue';
import { type Session } from '@session-recorder/protocols/ts/sessionsource';

export const useSessionData = ({ session }: { session: Session }) => {
  const waveformUrl = computed(() => {
    return session.info.updated.waveformUrl;
  });

  const audioUrls = computed(() => {
    return [
      {
        src: session.info.updated.audioUrl,
        type: 'audio/ogg',
      },
      {
        src: session.info.updated.losslessAudioUrl,
        type: 'audio/flac',
      },
    ];
  });

  return { waveformUrl, audioUrls };
};
