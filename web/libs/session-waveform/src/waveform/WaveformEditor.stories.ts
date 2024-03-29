import type { Meta, StoryObj } from '@storybook/vue3';
import WaveformEditor from './WaveformEditor.vue';
import { ref } from 'vue';
import type { Segment } from '../context/models/state';

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

export const Primary: StoryObj = {
  render: (args: Meta['args']) => ({
    components: { WaveformEditor },
    setup() {
      const segments = ref<Segment[]>([]);
      return {
        args: {
          waveformUrl: args?.waveformUrl,
          audioUrls: [
            {
              type: 'audio/mp3',
              src: args?.audioUrl,
            },
          ],
          permissions: {
            create: true,
            update: true,
            delete: true,
          },
        },
        segments,
      };
    },
    template: `
      <WaveformEditor v-bind="args" v-model:segments="segments" />
    `,
  }),
  args: {
    audioUrl: '/Free_Test_Data_1OMB_MP3.mp3',
  },
};
