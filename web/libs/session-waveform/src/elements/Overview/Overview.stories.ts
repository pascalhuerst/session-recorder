/**
 * Test Plan: Overview
 *
 * Scenario: Render overview layout
 *   Given the Overview component is rendered with providers
 *   When the component mounts
 *   Then the waveform container and controls should be visible
 *
 * Scenario: Layout structure
 *   Given the Overview component is rendered
 *   When examining the DOM
 *   Then it should have waveform area and controls area
 */

import type { Meta, StoryObj } from '@storybook/vue3';
import { ref } from 'vue';
import { within, expect } from '@storybook/test';
import Overview from './Overview.vue';
import { createPeaksContext, providePeaksContext } from '../../context/usePeaksContext';
import { useWaverformLayoutProvider } from '../../waveform/useWaverformLayoutProvider';

const createMockContext = () => ({
  initialState: {
    audioUrls: [{ type: 'audio/mp3', src: '/test.mp3' }],
    permissions: { create: true, update: true, delete: true },
    segments: [],
    player: {
      isPlaying: false,
      duration: 120,
      currentTime: 0,
    },
  },
});

const meta: Meta<typeof Overview> = {
  title: 'Lib/Elements/Overview/Overview',
  component: Overview,
  tags: ['autodocs'],
  parameters: {
    layout: 'padded',
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

// Default overview
export const Default: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);

    // Check layout structure
    const overview = canvasElement.querySelector('.overview');
    await expect(overview).toBeInTheDocument();

    const waveform = canvasElement.querySelector('.overview__waveform');
    await expect(waveform).toBeInTheDocument();

    const controls = canvasElement.querySelector('.overview__controls');
    await expect(controls).toBeInTheDocument();

    // Play button should be present
    const button = canvas.getByRole('button');
    await expect(button).toBeInTheDocument();
  },
  render: () => ({
    components: { Overview },
    setup() {
      // Setup layout provider
      const { provide: provideLayout } = useWaverformLayoutProvider();
      provideLayout({
        overviewRef: ref<HTMLElement>(),
        zoomviewRef: ref<HTMLElement>(),
        audioRef: ref<HTMLElement>(),
      });

      // Setup peaks context
      const context = createPeaksContext(createMockContext());
      providePeaksContext(context);

      return {};
    },
    template: '<Overview />',
  }),
};

// Overview with fixed width container
export const InContainer: Story = {
  play: async ({ canvasElement }) => {
    const overview = canvasElement.querySelector('.overview');
    await expect(overview).toBeInTheDocument();
  },
  render: () => ({
    components: { Overview },
    setup() {
      const { provide: provideLayout } = useWaverformLayoutProvider();
      provideLayout({
        overviewRef: ref<HTMLElement>(),
        zoomviewRef: ref<HTMLElement>(),
        audioRef: ref<HTMLElement>(),
      });

      const context = createPeaksContext(createMockContext());
      providePeaksContext(context);

      return {};
    },
    template: `
      <div style="width: 600px; border: 1px solid #ccc; border-radius: 8px; overflow: hidden;">
        <Overview />
      </div>
    `,
  }),
};

// Full width overview
export const FullWidth: Story = {
  play: async ({ canvasElement }) => {
    const overview = canvasElement.querySelector('.overview');
    await expect(overview).toBeInTheDocument();
  },
  render: () => ({
    components: { Overview },
    setup() {
      const { provide: provideLayout } = useWaverformLayoutProvider();
      provideLayout({
        overviewRef: ref<HTMLElement>(),
        zoomviewRef: ref<HTMLElement>(),
        audioRef: ref<HTMLElement>(),
      });

      const context = createPeaksContext(createMockContext());
      providePeaksContext(context);

      return {};
    },
    template: `
      <div style="width: 100%; background: #f5f5f5; padding: 1rem;">
        <Overview />
      </div>
    `,
  }),
};
