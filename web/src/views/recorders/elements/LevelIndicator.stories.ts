/**
 * Test Plan: LevelIndicator
 *
 * Scenario: Render with different values
 *   Given the LevelIndicator is rendered with a value
 *   When the component mounts
 *   Then it should display the percentage and visual indicator
 *
 * Scenario: Value ranges
 *   Given different value percentages
 *   When the indicator renders
 *   Then the clip-path should adjust accordingly
 *   And the color gradient should be visible
 *
 * Scenario: Edge cases
 *   Given edge case values (0, 100)
 *   When rendered
 *   Then it should handle them correctly
 */

import type { Meta, StoryObj } from '@storybook/vue3';
import { within, expect } from '@storybook/test';
import LevelIndicator from './LevelIndicator.vue';

const meta: Meta<typeof LevelIndicator> = {
  title: 'App/Elements/LevelIndicator',
  component: LevelIndicator,
  tags: ['autodocs'],
  parameters: {
    layout: 'centered',
  },
  argTypes: {
    value: {
      control: { type: 'range', min: 0, max: 100, step: 1 },
      description: 'Level in percent of full scale (0-100)',
    },
    label: {
      control: 'text',
      description: 'Short name in front of the meter',
    },
    warnAbove: {
      control: { type: 'number', min: 0, max: 100 },
      description: 'The percentage turns red above this value',
    },
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

// Zero value
export const Zero: Story = {
  args: {
    value: 0,
  },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await expect(canvas.getByText('0%')).toBeInTheDocument();
  },
};

// Low value
export const Low: Story = {
  args: {
    value: 20,
  },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await expect(canvas.getByText('20%')).toBeInTheDocument();
  },
};

// Medium value
export const Medium: Story = {
  args: {
    value: 50,
  },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await expect(canvas.getByText('50%')).toBeInTheDocument();
  },
};

// High value
export const High: Story = {
  args: {
    value: 80,
  },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await expect(canvas.getByText('80%')).toBeInTheDocument();
  },
};

// Maximum value
export const Maximum: Story = {
  args: {
    value: 100,
  },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await expect(canvas.getByText('100%')).toBeInTheDocument();
  },
};

// Decimal value (should ceil)
export const DecimalValue: Story = {
  args: {
    value: 45.7,
  },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    // Math.ceil(45.7) = 46
    await expect(canvas.getByText('46%')).toBeInTheDocument();
  },
};

// All levels comparison
export const AllLevels: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);

    await expect(canvas.getByText('0%')).toBeInTheDocument();
    await expect(canvas.getByText('25%')).toBeInTheDocument();
    await expect(canvas.getByText('50%')).toBeInTheDocument();
    await expect(canvas.getByText('75%')).toBeInTheDocument();
    await expect(canvas.getByText('100%')).toBeInTheDocument();
  },
  render: () => ({
    components: { LevelIndicator },
    template: `
      <div style="display: flex; flex-direction: column; gap: 1rem; width: 200px;">
        <div>
          <p style="font-size: 0.75rem; color: #666; margin-bottom: 0.25rem;">Silent</p>
          <LevelIndicator :value="0" />
        </div>
        <div>
          <p style="font-size: 0.75rem; color: #666; margin-bottom: 0.25rem;">Quiet</p>
          <LevelIndicator :value="25" />
        </div>
        <div>
          <p style="font-size: 0.75rem; color: #666; margin-bottom: 0.25rem;">Normal</p>
          <LevelIndicator :value="50" />
        </div>
        <div>
          <p style="font-size: 0.75rem; color: #666; margin-bottom: 0.25rem;">Loud</p>
          <LevelIndicator :value="75" />
        </div>
        <div>
          <p style="font-size: 0.75rem; color: #666; margin-bottom: 0.25rem;">Peak</p>
          <LevelIndicator :value="100" />
        </div>
      </div>
    `,
  }),
};

// In card context
export const InCardContext: Story = {
  args: {
    value: 65,
  },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await expect(canvas.getByText('65%')).toBeInTheDocument();
  },
  render: (args) => ({
    components: { LevelIndicator },
    setup() {
      return { args };
    },
    template: `
      <div style="display: flex; align-items: center; gap: 1rem; padding: 1rem; background: white; border: 1px solid #eee; border-radius: 8px; width: 200px;">
        <span style="font-size: 0.875rem;">Level:</span>
        <div style="flex: 1;">
          <LevelIndicator v-bind="args" />
        </div>
      </div>
    `,
  }),
};

// RMS and peak of the same signal, as the device card shows them
export const RmsAndPeak: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await expect(canvas.getByText('RMS')).toBeInTheDocument();
    await expect(canvas.getByText('Peak')).toBeInTheDocument();
    await expect(canvas.getByText('86%')).toHaveClass('hot');
    await expect(canvas.getByText('32%')).not.toHaveClass('hot');
  },
  render: () => ({
    components: { LevelIndicator },
    template: `
      <div style="display: flex; flex-direction: column; gap: 0.25rem; width: 220px;">
        <LevelIndicator label="RMS" :value="32" />
        <LevelIndicator label="Peak" :value="86" :warn-above="80" />
      </div>
    `,
  }),
};

// Color gradient visualization
export const ColorGradient: Story = {
  play: async ({ canvasElement }) => {
    const indicator = canvasElement.querySelector('.indicator');
    await expect(indicator).toBeInTheDocument();
  },
  render: () => ({
    components: { LevelIndicator },
    template: `
      <div style="width: 300px;">
        <p style="font-size: 0.75rem; color: #666; margin-bottom: 0.5rem;">
          Green (safe) → Yellow (warning) → Red (clipping)
        </p>
        <LevelIndicator :value="100" />
      </div>
    `,
  }),
};
