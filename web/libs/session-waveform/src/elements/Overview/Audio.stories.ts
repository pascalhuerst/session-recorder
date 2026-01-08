/**
 * Test Plan: Audio
 *
 * Scenario: Render audio element with sources
 *   Given the Audio component is rendered with PeaksContext
 *   When audioUrls are provided in state
 *   Then the audio element should have source elements
 *
 * Scenario: Multiple audio sources
 *   Given multiple audioUrls are configured
 *   When the component renders
 *   Then all sources should be present for browser fallback
 */

import type { Meta, StoryObj } from '@storybook/vue3';
import { ref } from 'vue';
import { within, expect } from '@storybook/test';
import Audio from './Audio.vue';
import { createPeaksContext, providePeaksContext } from '../../context/usePeaksContext';
import { useWaverformLayoutProvider } from '../../waveform/useWaverformLayoutProvider';

const meta: Meta<typeof Audio> = {
  title: 'Lib/Elements/Overview/Audio',
  component: Audio,
  tags: ['autodocs'],
  parameters: {
    layout: 'centered',
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

// Single audio source
export const SingleSource: Story = {
  play: async ({ canvasElement }) => {
    const audio = canvasElement.querySelector('audio');
    await expect(audio).toBeInTheDocument();

    const sources = audio?.querySelectorAll('source');
    await expect(sources).toHaveLength(1);
  },
  render: () => ({
    components: { Audio },
    setup() {
      const { provide: provideLayout } = useWaverformLayoutProvider();
      provideLayout({
        overviewRef: ref<HTMLElement>(),
        zoomviewRef: ref<HTMLElement>(),
        audioRef: ref<HTMLElement>(),
      });

      const context = createPeaksContext({
        initialState: {
          audioUrls: [{ type: 'audio/mp3', src: '/test.mp3' }],
          permissions: { create: true, update: true, delete: true },
          segments: [],
        },
      });
      providePeaksContext(context);

      return {};
    },
    template: '<Audio />',
  }),
};

// Multiple audio sources (format fallback)
export const MultipleSources: Story = {
  play: async ({ canvasElement }) => {
    const audio = canvasElement.querySelector('audio');
    await expect(audio).toBeInTheDocument();

    const sources = audio?.querySelectorAll('source');
    await expect(sources).toHaveLength(3);
  },
  render: () => ({
    components: { Audio },
    setup() {
      const { provide: provideLayout } = useWaverformLayoutProvider();
      provideLayout({
        overviewRef: ref<HTMLElement>(),
        zoomviewRef: ref<HTMLElement>(),
        audioRef: ref<HTMLElement>(),
      });

      const context = createPeaksContext({
        initialState: {
          audioUrls: [
            { type: 'audio/flac', src: '/audio.flac' },
            { type: 'audio/ogg', src: '/audio.ogg' },
            { type: 'audio/mp3', src: '/audio.mp3' },
          ],
          permissions: { create: true, update: true, delete: true },
          segments: [],
        },
      });
      providePeaksContext(context);

      return {};
    },
    template: `
      <div>
        <Audio />
        <p style="font-size: 0.875rem; color: #666; margin-top: 1rem;">
          Audio element with FLAC, OGG, and MP3 sources for browser compatibility
        </p>
      </div>
    `,
  }),
};

// With visible controls (for demo)
export const WithControls: Story = {
  play: async ({ canvasElement }) => {
    const audio = canvasElement.querySelector('audio');
    await expect(audio).toBeInTheDocument();
  },
  render: () => ({
    components: { Audio },
    setup() {
      const { provide: provideLayout } = useWaverformLayoutProvider();
      provideLayout({
        overviewRef: ref<HTMLElement>(),
        zoomviewRef: ref<HTMLElement>(),
        audioRef: ref<HTMLElement>(),
      });

      const context = createPeaksContext({
        initialState: {
          audioUrls: [{ type: 'audio/mp3', src: 'https://www.soundhelix.com/examples/mp3/SoundHelix-Song-1.mp3' }],
          permissions: { create: true, update: true, delete: true },
          segments: [],
        },
      });
      providePeaksContext(context);

      return {};
    },
    template: `
      <div>
        <p style="font-size: 0.875rem; color: #666; margin-bottom: 0.5rem;">
          Audio element (hidden by default, shown here for demo)
        </p>
        <audio controls style="width: 300px;">
          <source type="audio/mp3" src="https://www.soundhelix.com/examples/mp3/SoundHelix-Song-1.mp3" />
        </audio>
      </div>
    `,
  }),
};
