/**
 * Test Plan: Zoomview
 *
 * Scenario: Render zoomview layout
 *   Given the Zoomview component is rendered with providers
 *   When the component mounts
 *   Then the waveform area and controls panel should be visible
 *
 * Scenario: Controls display
 *   Given the Zoomview is rendered
 *   When examining the controls area
 *   Then SeekInput, ZoomInput, AmplitudeInput should be present
 *
 * Scenario: Add segment button visibility
 *   Given the user has create permissions
 *   When the Zoomview renders
 *   Then the AddSegmentButton should be visible
 */

import type { Meta, StoryObj } from '@storybook/vue3';
import { ref } from 'vue';
import { within, expect } from '@storybook/test';
import Zoomview from './Zoomview.vue';
import { createPeaksContext, providePeaksContext } from '../../context/usePeaksContext';
import { useWaverformLayoutProvider } from '../../waveform/useWaverformLayoutProvider';

const createMockContext = (permissions = { create: true, update: true, delete: true }) => ({
  initialState: {
    audioUrls: [{ type: 'audio/mp3', src: '/test.mp3' }],
    permissions,
    segments: [],
    player: {
      isPlaying: false,
      duration: 300,
      currentTime: 60,
    },
    zoom: {
      zoomLevel: 300,
      zoomStep: 60,
    },
    amplitude: {
      amplitudeScale: 0.6,
      amplitudeStep: 0.1,
    },
  },
});

const meta: Meta<typeof Zoomview> = {
  title: 'Lib/Elements/Zoomview/Zoomview',
  component: Zoomview,
  tags: ['autodocs'],
  parameters: {
    layout: 'padded',
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

// Default zoomview
export const Default: Story = {
  play: async ({ canvasElement }) => {
    // Check layout structure
    const zoomview = canvasElement.querySelector('.zoomview');
    await expect(zoomview).toBeInTheDocument();

    const waveform = canvasElement.querySelector('.zoomview__waveform');
    await expect(waveform).toBeInTheDocument();

    const controls = canvasElement.querySelector('.zoomview__controls');
    await expect(controls).toBeInTheDocument();

    // Check for input controls
    const inputs = canvasElement.querySelectorAll('input');
    await expect(inputs.length).toBeGreaterThanOrEqual(3);
  },
  render: () => ({
    components: { Zoomview },
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
    template: '<Zoomview />',
  }),
};

// With add segment button
export const WithAddSegmentButton: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    const addButton = canvas.getByRole('button', { name: /add segment/i });
    await expect(addButton).toBeInTheDocument();
  },
  render: () => ({
    components: { Zoomview },
    setup() {
      const { provide: provideLayout } = useWaverformLayoutProvider();
      provideLayout({
        overviewRef: ref<HTMLElement>(),
        zoomviewRef: ref<HTMLElement>(),
        audioRef: ref<HTMLElement>(),
      });

      const context = createPeaksContext(createMockContext({ create: true, update: true, delete: true }));
      providePeaksContext(context);

      return {};
    },
    template: '<Zoomview />',
  }),
};

// Without add segment button (no create permission)
export const WithoutAddSegmentButton: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    const addButton = canvas.queryByRole('button', { name: /add segment/i });
    await expect(addButton).not.toBeInTheDocument();
  },
  render: () => ({
    components: { Zoomview },
    setup() {
      const { provide: provideLayout } = useWaverformLayoutProvider();
      provideLayout({
        overviewRef: ref<HTMLElement>(),
        zoomviewRef: ref<HTMLElement>(),
        audioRef: ref<HTMLElement>(),
      });

      const context = createPeaksContext(createMockContext({ create: false, update: true, delete: true }));
      providePeaksContext(context);

      return {};
    },
    template: '<Zoomview />',
  }),
};

// In container
export const InContainer: Story = {
  play: async ({ canvasElement }) => {
    const zoomview = canvasElement.querySelector('.zoomview');
    await expect(zoomview).toBeInTheDocument();
  },
  render: () => ({
    components: { Zoomview },
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
      <div style="width: 800px; border: 1px solid #ccc; border-radius: 8px; overflow: hidden;">
        <Zoomview />
      </div>
    `,
  }),
};
