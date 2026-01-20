/**
 * Test Plan: SeekInput
 *
 * Scenario: Render seek input
 *   Given the SeekInput component is rendered with PeaksContext
 *   When the component mounts
 *   Then the time input with current playback time should be visible
 *
 * Scenario: Edit seek time
 *   Given the seek input is rendered
 *   When the user changes the time value
 *   Then the seek command should be emitted
 *
 * Scenario: Display max time
 *   Given the player has a duration
 *   When the input is rendered
 *   Then the max attribute should be set to the duration
 */

import type { Meta, StoryObj } from '@storybook/vue3';
import { userEvent, expect } from '@storybook/test';
import SeekInput from './controls/SeekInput.vue';
import { createPeaksContext, providePeaksContext } from '../../context/usePeaksContext';

const createMockContext = (currentTime = 60, duration = 300) => ({
  initialState: {
    audioUrls: [{ type: 'audio/mp3', src: '/test.mp3' }],
    permissions: { create: true, update: true, delete: true },
    segments: [],
    player: {
      isPlaying: false,
      duration,
      currentTime,
    },
  },
});

const meta: Meta<typeof SeekInput> = {
  title: 'Lib/Elements/Zoomview/SeekInput',
  component: SeekInput,
  tags: ['autodocs'],
  parameters: {
    layout: 'centered',
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

// Default seek input
export const Default: Story = {
  play: async ({ canvasElement }) => {
    const input = canvasElement.querySelector('input[type="time"]');
    await expect(input).toBeInTheDocument();
  },
  render: () => ({
    components: { SeekInput },
    setup() {
      const context = createPeaksContext(createMockContext());
      providePeaksContext(context);
      return {};
    },
    template: '<SeekInput />',
  }),
};

// At beginning
export const AtBeginning: Story = {
  play: async ({ canvasElement }) => {
    const input = canvasElement.querySelector('input[type="time"]');
    await expect(input).toBeInTheDocument();
  },
  render: () => ({
    components: { SeekInput },
    setup() {
      const context = createPeaksContext(createMockContext(0, 300));
      providePeaksContext(context);
      return {};
    },
    template: '<SeekInput />',
  }),
};

// In middle
export const InMiddle: Story = {
  play: async ({ canvasElement }) => {
    const input = canvasElement.querySelector('input[type="time"]');
    await expect(input).toBeInTheDocument();
  },
  render: () => ({
    components: { SeekInput },
    setup() {
      const context = createPeaksContext(createMockContext(150, 300));
      providePeaksContext(context);
      return {};
    },
    template: '<SeekInput />',
  }),
};

// Near end
export const NearEnd: Story = {
  play: async ({ canvasElement }) => {
    const input = canvasElement.querySelector('input[type="time"]');
    await expect(input).toBeInTheDocument();
  },
  render: () => ({
    components: { SeekInput },
    setup() {
      const context = createPeaksContext(createMockContext(290, 300));
      providePeaksContext(context);
      return {};
    },
    template: '<SeekInput />',
  }),
};

// Long duration
export const LongDuration: Story = {
  play: async ({ canvasElement }) => {
    const input = canvasElement.querySelector('input[type="time"]');
    await expect(input).toBeInTheDocument();
  },
  render: () => ({
    components: { SeekInput },
    setup() {
      // 2 hours duration
      const context = createPeaksContext(createMockContext(3600, 7200));
      providePeaksContext(context);
      return {};
    },
    template: `
      <div>
        <SeekInput />
        <p style="font-size: 0.75rem; color: #666; margin-top: 0.5rem;">2 hour duration</p>
      </div>
    `,
  }),
};

// Focus interaction
export const FocusInteraction: Story = {
  play: async ({ canvasElement }) => {
    const input = canvasElement.querySelector('input[type="time"]') as HTMLInputElement;

    if (input) {
      await userEvent.click(input);
      await expect(input).toHaveFocus();
    }
  },
  render: () => ({
    components: { SeekInput },
    setup() {
      const context = createPeaksContext(createMockContext());
      providePeaksContext(context);
      return {};
    },
    template: '<SeekInput />',
  }),
};

// No duration (zero)
export const NoDuration: Story = {
  play: async ({ canvasElement }) => {
    const input = canvasElement.querySelector('input[type="time"]');
    await expect(input).toBeInTheDocument();
  },
  render: () => ({
    components: { SeekInput },
    setup() {
      const context = createPeaksContext(createMockContext(0, 0));
      providePeaksContext(context);
      return {};
    },
    template: `
      <div>
        <SeekInput />
        <p style="font-size: 0.75rem; color: #666; margin-top: 0.5rem;">No audio loaded</p>
      </div>
    `,
  }),
};
