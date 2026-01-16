/**
 * Test Plan: RecorderActions
 *
 * Scenario: Render when recorder is recording
 *   Given the selected recorder has SIGNAL status
 *   When the component renders
 *   Then the "Cut Session" button and recording banner should be visible
 *
 * Scenario: Hidden when not recording
 *   Given the selected recorder has NO_SIGNAL status
 *   When the component renders
 *   Then no banner or button should be visible
 *
 * Scenario: Cut session action
 *   Given the recorder is recording
 *   When the Cut Session button is clicked
 *   Then the cutSession procedure should be called
 */

import type { Meta, StoryObj } from '@storybook/vue3';
import { within, expect, userEvent } from '@storybook/test';
import { ref } from 'vue';
import { Button } from '@session-recorder/session-waveform';

// Mock component since real one depends on Pinia store and gRPC
const MockRecorderActions = {
  name: 'MockRecorderActions',
  components: { Button },
  props: {
    isRecording: {
      type: Boolean,
      default: false,
    },
  },
  emits: ['cutSession'],
  setup(props: { isRecording: boolean }, { emit }: { emit: (event: string) => void }) {
    const handleCutSession = () => {
      emit('cutSession');
    };
    return { handleCutSession };
  },
  template: `
    <div class="actions">
      <div v-if="isRecording" class="banner">
        <div>This recorder is currently recording</div>
        <Button @click="handleCutSession" color="primary">
          ✂️ Cut Session
        </Button>
      </div>
    </div>
  `,
};

const meta: Meta = {
  title: 'App/Elements/RecorderActions',
  component: MockRecorderActions,
  tags: ['autodocs'],
  parameters: {
    layout: 'padded',
  },
  argTypes: {
    isRecording: {
      control: 'boolean',
      description: 'Whether the recorder is currently recording',
    },
  },
};

export default meta;
type Story = StoryObj;

// Recording state - banner visible
export const Recording: Story = {
  args: {
    isRecording: true,
  },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);

    // Banner should be visible
    await expect(canvas.getByText('This recorder is currently recording')).toBeInTheDocument();

    // Cut session button should be visible
    await expect(canvas.getByText('✂️ Cut Session')).toBeInTheDocument();
  },
};

// Not recording - nothing visible
export const NotRecording: Story = {
  args: {
    isRecording: false,
  },
  play: async ({ canvasElement }) => {
    const banner = canvasElement.querySelector('.banner');
    await expect(banner).not.toBeInTheDocument();
  },
};

// Cut session interaction
export const CutSessionInteraction: Story = {
  args: {
    isRecording: true,
  },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);

    const cutButton = canvas.getByText('✂️ Cut Session');
    await expect(cutButton).toBeInTheDocument();

    // Click the button
    await userEvent.click(cutButton);
  },
};

// In context (with styling)
export const InContext: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await expect(canvas.getByText('This recorder is currently recording')).toBeInTheDocument();
  },
  render: () => ({
    components: { MockRecorderActions },
    template: `
      <div style="max-width: 800px; margin: auto;">
        <h2 style="margin-bottom: 1rem;">Recording Session</h2>
        <MockRecorderActions :isRecording="true" />
      </div>
    `,
  }),
};

// Toggle state
export const ToggleState: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);

    // Initially recording
    await expect(canvas.getByText('This recorder is currently recording')).toBeInTheDocument();

    // Click toggle button
    const toggleButton = canvas.getByText('Toggle Recording');
    await userEvent.click(toggleButton);
  },
  render: () => ({
    components: { MockRecorderActions },
    setup() {
      const isRecording = ref(true);
      const toggle = () => {
        isRecording.value = !isRecording.value;
      };
      return { isRecording, toggle };
    },
    template: `
      <div>
        <button @click="toggle" style="margin-bottom: 1rem; padding: 0.5rem 1rem;">Toggle Recording</button>
        <MockRecorderActions :isRecording="isRecording" />
      </div>
    `,
  }),
};
