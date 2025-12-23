/**
 * Test Plan: StatusIndicator
 *
 * Scenario: Render idle state
 *   Given the StatusIndicator is rendered with isRecording=false
 *   When the component mounts
 *   Then it should show "off" text and grey indicator
 *
 * Scenario: Render recording state
 *   Given the StatusIndicator is rendered with isRecording=true
 *   When the component mounts
 *   Then it should show "rec" text and red pulsing indicator
 *
 * Scenario: Toggle state
 *   Given the StatusIndicator is rendered
 *   When isRecording changes
 *   Then the visual state should update accordingly
 */

import type { Meta, StoryObj } from '@storybook/vue3';
import { within, expect } from '@storybook/test';
import StatusIndicator from './StatusIndicator.vue';

const meta: Meta<typeof StatusIndicator> = {
  title: 'App/Elements/StatusIndicator',
  component: StatusIndicator,
  tags: ['autodocs'],
  parameters: {
    layout: 'centered',
  },
  argTypes: {
    isRecording: {
      control: 'boolean',
      description: 'Whether the recorder is currently recording',
    },
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

// Idle state (not recording)
export const Idle: Story = {
  args: {
    isRecording: false,
  },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);

    // Should show "off" text
    await expect(canvas.getByText('off')).toBeInTheDocument();

    // Indicator should not have recording class
    const indicator = canvasElement.querySelector('.indicator');
    await expect(indicator).not.toHaveClass('is-recording');
  },
};

// Recording state
export const Recording: Story = {
  args: {
    isRecording: true,
  },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);

    // Should show "rec" text
    await expect(canvas.getByText('rec')).toBeInTheDocument();

    // Indicator should have recording class
    const indicator = canvasElement.querySelector('.indicator');
    await expect(indicator).toHaveClass('is-recording');

    // Text should have recording class
    const text = canvasElement.querySelector('.text');
    await expect(text).toHaveClass('is-recording');
  },
};

// Side by side comparison
export const Comparison: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);

    await expect(canvas.getByText('off')).toBeInTheDocument();
    await expect(canvas.getByText('rec')).toBeInTheDocument();
  },
  render: () => ({
    components: { StatusIndicator },
    template: `
      <div style="display: flex; gap: 2rem; align-items: center;">
        <div style="text-align: center;">
          <StatusIndicator :isRecording="false" />
          <p style="margin-top: 0.5rem; font-size: 0.75rem; color: #666;">Idle</p>
        </div>
        <div style="text-align: center;">
          <StatusIndicator :isRecording="true" />
          <p style="margin-top: 0.5rem; font-size: 0.75rem; color: #666;">Recording</p>
        </div>
      </div>
    `,
  }),
};

// In card context
export const InCardContext: Story = {
  args: {
    isRecording: true,
  },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await expect(canvas.getByText('rec')).toBeInTheDocument();
  },
  render: (args) => ({
    components: { StatusIndicator },
    setup() {
      return { args };
    },
    template: `
      <div style="display: flex; align-items: center; gap: 1rem; padding: 1rem; background: white; border: 1px solid #eee; border-radius: 8px;">
        <StatusIndicator v-bind="args" />
        <span>Device Name</span>
      </div>
    `,
  }),
};

// Animation showcase
export const AnimationShowcase: Story = {
  args: {
    isRecording: true,
  },
  play: async ({ canvasElement }) => {
    const indicator = canvasElement.querySelector('.indicator.is-recording');
    await expect(indicator).toBeInTheDocument();
  },
  render: () => ({
    components: { StatusIndicator },
    template: `
      <div style="display: flex; flex-direction: column; gap: 1rem; align-items: center;">
        <StatusIndicator :isRecording="true" />
        <p style="font-size: 0.75rem; color: #666;">Watch the pulsing animation</p>
      </div>
    `,
  }),
};
