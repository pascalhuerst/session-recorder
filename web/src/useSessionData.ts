import { computed } from 'vue';
import { env } from '@/env';

export const useSessionData = ({
  recorderId,
  sessionId,
}: {
  recorderId: string;
  sessionId: string;
}) => {
  const waveformUrl = computed(() => {
    return new URL(
      `${recorderId}/${sessionId}/waveform.dat`,
      env.VITE_FILE_SERVER_URL
    ).toString();
  });

  const audioUrls = computed(() => {
    return [
      {
        src: new URL(
          `${recorderId}/${sessionId}/data.ogg`,
          env.VITE_FILE_SERVER_URL
        ).toString(),
        type: 'audio/ogg',
      },
      {
        src: new URL(
          `${recorderId}/${sessionId}/data.flac`,
          env.VITE_FILE_SERVER_URL
        ).toString(),
        type: 'audio/flac',
      },
    ];
  });

  return { waveformUrl, audioUrls };
};
