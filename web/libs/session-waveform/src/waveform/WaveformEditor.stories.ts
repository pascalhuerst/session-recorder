import type { Meta, StoryObj } from '@storybook/vue3';
import WaveformEditor from './WaveformEditor.vue';
import {
  createPeaksContext,
  providePeaksContext,
} from '../context/usePeaksContext';

const meta: Meta = {
  title: 'Waveform Editor',
  argTypes: {
    waveformUrl: {
      type: 'string',
    },
    audioUrl: {
      type: 'string',
    },
  },
};

export default meta;

const render = (args: Meta['args']) => ({
  components: { WaveformEditor },
  setup() {
    const context = createPeaksContext({
      initialState: {
        waveformUrl: args?.waveformUrl,
        audioUrls: [
          {
            type: args?.audioFormat,
            src: args?.audioUrl,
          },
        ],
        permissions: {
          create: true,
          update: true,
          delete: true,
        },
        segments: [
          {
            id: '1',
            startIndex: 'A',
            endIndex: 'B',
            startTime: 0,
            endTime: 20,
            labelText: 'Hello',
            renders: [
              {
                type: 'audio/mp3',
                src: 'hello.mpd',
              },
            ],
          },
          {
            id: '2',
            startIndex: 'A',
            endIndex: 'B',
            startTime: 0,
            endTime: 20,
            labelText: 'Hello',
            renders: [],
          },
        ],
      },
    });

    providePeaksContext(context);
  },
  template: `
    <WaveformEditor />
  `,
});

export const Local: StoryObj = {
  render,
  args: {
    audioFormat: 'audio/mp3',
    audioUrl: '/Free_Test_Data_1OMB_MP3.mp3',
  },
};

export const Remote: StoryObj = {
  render,
  args: {
    waveformUrl:
      'http://192.168.52.154:9000/session-recorder/cdd40c26-5b62-465d-8014-e239fda909ba/sessions/07c6fa13-44f3-4cbf-8d3d-d41524a81ec4/waveform.dat',
    audioUrl:
      'http://192.168.52.154:9000/session-recorder/cdd40c26-5b62-465d-8014-e239fda909ba/sessions/07c6fa13-44f3-4cbf-8d3d-d41524a81ec4/data.flac',
    audioFormat: 'audio/flac',
    // audioUrl: '/data.flac',
    // audioUrl: '/Free_Test_Data_1OMB_MP3.mp3',
  },
};
