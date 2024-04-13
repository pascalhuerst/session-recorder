import { computed } from 'vue';
import { env } from '@/env';

export const useSessionData = ({
  recorderId,
  sessionId,
}: {
  recorderId: string;
  sessionId: string;
}) => {
  // @todo: get this from backend
  const buildUrlPath = (fileName: string) => {
    return `/session-recorder/${recorderId}/sessions/${sessionId}/${fileName}`;
  };

  const waveformUrl = computed(() => {
    return new URL(
      buildUrlPath('waveform.dat'),
      env.VITE_FILE_SERVER_URL
    ).toString();
  });

  const audioUrls = computed(() => {
    return [
      {
        src: new URL(
          buildUrlPath('data.flac'),
          env.VITE_FILE_SERVER_URL
        ).toString(),
        type: 'audio/flac',
      },
      // {
      //   src: new URL(
      //     buildUrlPath('data.ogg'),
      //     env.VITE_FILE_SERVER_URL
      //   ).toString(),
      //   type: 'audio/ogg',
      // },
    ];
  });

  return { waveformUrl, audioUrls };
};
