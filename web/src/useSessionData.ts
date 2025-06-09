import { computed } from 'vue';
import { type Session } from '@session-recorder/protocols/ts/sessionsource';

export const useSessionData = ({ session }: { session: Session }) => {
  const waveformUrl = computed(() => {
    return session.info.updated.waveformDataFile ?? '';
  });

  const audioUrls = computed(() => {
    return [
      {
        src: session.info.updated.audioFileName ?? '',
        type: 'audio/ogg',
      },
      {
        src: session.info.updated.losslessAudioFileName ?? '',
        type: 'audio/flac',
      },
    ];
  });

  return { waveformUrl, audioUrls };
};
