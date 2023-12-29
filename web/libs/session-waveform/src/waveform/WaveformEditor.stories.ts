import type { Meta, StoryObj } from '@storybook/vue3';
import WaveformEditor from './WaveformEditor.vue';

const meta: Meta = {
  title: 'Waveform Editor',
  argTypes: {
    waveformUrl: {
      type: 'string'
    },
    audioUrl: {
      type: 'string'
    }
  }
};

export default meta;

export const Primary: StoryObj = {
  render: (args: Meta['args']) => ({
    components: { WaveformEditor },
    setup() {
      return {
        args: {
          waveformUrl: args?.waveformUrl,
          audioUrls: [
            {
              type: 'audio/mp3',
              src: args?.audioUrl
            }
          ]
        }
      };
    },
    template: `
      <WaveformEditor v-bind="args">

      </WaveformEditor>`
  }),
  args: {
    audioUrl: 'https://cdn.pixabay.com/audio/2023/10/17/audio_65ea4766f8.mp3'
  }
};
