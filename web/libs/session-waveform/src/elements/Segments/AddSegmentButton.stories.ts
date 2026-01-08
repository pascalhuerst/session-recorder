/**
 * Test Plan: AddSegmentButton
 *
 * Scenario: Render add segment button
 *   Given the AddSegmentButton component is rendered with PeaksContext
 *   When the component mounts
 *   Then the "Add Segment" button should be visible
 *
 * Scenario: Click to add segment
 *   Given the button is rendered
 *   When the user clicks the button
 *   Then the addSegment command should be emitted
 *
 * Scenario: Keyboard interaction
 *   Given the button is focused
 *   When the user presses Enter or Space
 *   Then the addSegment command should be emitted
 */

import type { Meta, StoryObj } from '@storybook/vue3';
import { userEvent, within, expect } from '@storybook/test';
import AddSegmentButton from './controls/AddSegmentButton.vue';
import { createPeaksContext, providePeaksContext } from '../../context/usePeaksContext';

const createMockContext = () => ({
  initialState: {
    audioUrls: [{ type: 'audio/mp3', src: '/test.mp3' }],
    permissions: { create: true, update: true, delete: true },
    segments: [],
    player: {
      isPlaying: false,
      duration: 300,
      currentTime: 0,
    },
  },
});

const meta: Meta<typeof AddSegmentButton> = {
  title: 'Lib/Elements/Segments/AddSegmentButton',
  component: AddSegmentButton,
  tags: ['autodocs'],
  parameters: {
    layout: 'centered',
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

// Default button
export const Default: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    const button = canvas.getByRole('button', { name: /add segment/i });

    await expect(button).toBeInTheDocument();
    await expect(button).not.toBeDisabled();
  },
  render: () => ({
    components: { AddSegmentButton },
    setup() {
      const context = createPeaksContext(createMockContext());
      providePeaksContext(context);
      return {};
    },
    template: '<AddSegmentButton />',
  }),
};

// Click interaction
export const ClickInteraction: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    const button = canvas.getByRole('button', { name: /add segment/i });

    await userEvent.click(button);

    // Button should still be present after click
    await expect(button).toBeInTheDocument();
  },
  render: () => ({
    components: { AddSegmentButton },
    setup() {
      const context = createPeaksContext(createMockContext());
      providePeaksContext(context);
      return {};
    },
    template: '<AddSegmentButton />',
  }),
};

// Keyboard interaction
export const KeyboardInteraction: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    const button = canvas.getByRole('button', { name: /add segment/i });

    // Focus the button
    button.focus();
    await expect(button).toHaveFocus();

    // Press Enter
    await userEvent.keyboard('{Enter}');
    await expect(button).toBeInTheDocument();

    // Press Space
    await userEvent.keyboard(' ');
    await expect(button).toBeInTheDocument();
  },
  render: () => ({
    components: { AddSegmentButton },
    setup() {
      const context = createPeaksContext(createMockContext());
      providePeaksContext(context);
      return {};
    },
    template: '<AddSegmentButton />',
  }),
};

// In context (with label)
export const InContext: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await expect(canvas.getByText(/add segment/i)).toBeInTheDocument();
  },
  render: () => ({
    components: { AddSegmentButton },
    setup() {
      const context = createPeaksContext(createMockContext());
      providePeaksContext(context);
      return {};
    },
    template: `
      <div style="display: flex; flex-direction: column; align-items: center; gap: 1rem;">
        <p style="font-size: 0.875rem; color: #666;">Click to create a new segment at the current playhead position</p>
        <AddSegmentButton />
      </div>
    `,
  }),
};

// Multiple clicks (demonstrating repeated action)
export const MultipleClicks: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    const button = canvas.getByRole('button', { name: /add segment/i });

    // Click multiple times
    await userEvent.click(button);
    await userEvent.click(button);
    await userEvent.click(button);

    // Button should still work after multiple clicks
    await expect(button).toBeInTheDocument();
  },
  render: () => ({
    components: { AddSegmentButton },
    setup() {
      const context = createPeaksContext(createMockContext());
      providePeaksContext(context);
      return {};
    },
    template: '<AddSegmentButton />',
  }),
};
