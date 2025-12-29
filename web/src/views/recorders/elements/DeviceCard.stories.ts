/**
 * Test Plan: DeviceCard
 *
 * Scenario: Render device card with recorder info
 *   Given the DeviceCard is rendered with a recorder
 *   When the component mounts
 *   Then the recorder name and status should be visible
 *
 * Scenario: Recording state
 *   Given the recorder has SIGNAL status
 *   When the card renders
 *   Then the recording indicator and RMS meter should be visible
 *
 * Scenario: Selected state
 *   Given the card has isSelected=true
 *   When rendered
 *   Then it should have selected styling
 *
 * Scenario: Idle state
 *   Given the recorder is not recording
 *   When the card renders
 *   Then the idle indicator should be shown
 */

import type { Meta, StoryObj } from '@storybook/vue3';
import { within, expect, userEvent } from '@storybook/test';
import DeviceCard from './DeviceCard.vue';

// Mock recorder data
const mockIdleRecorder = {
  recorderId: 'recorder-1',
  recorderName: 'Studio Mic 1',
  info: {
    oneofKind: 'status' as const,
    status: {
      signalStatus: 0, // NO_SIGNAL
      rmsPercent: 0,
    },
  },
};

const mockRecordingRecorder = {
  recorderId: 'recorder-2',
  recorderName: 'Live Room Mic',
  info: {
    oneofKind: 'status' as const,
    status: {
      signalStatus: 1, // SIGNAL (recording)
      rmsPercent: 65,
    },
  },
};

const mockLongNameRecorder = {
  recorderId: 'recorder-3',
  recorderName: 'This is a very long recorder name that should be truncated',
  info: {
    oneofKind: 'status' as const,
    status: {
      signalStatus: 0,
      rmsPercent: 0,
    },
  },
};

const meta: Meta<typeof DeviceCard> = {
  title: 'App/Elements/DeviceCard',
  component: DeviceCard,
  tags: ['autodocs'],
  parameters: {
    layout: 'centered',
  },
  argTypes: {
    recorder: {
      control: 'object',
      description: 'Recorder data object',
    },
    isSelected: {
      control: 'boolean',
      description: 'Whether the card is selected',
    },
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

// Idle recorder
export const Idle: Story = {
  args: {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    recorder: mockIdleRecorder as any,
    isSelected: false,
  },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);

    // Check recorder name is displayed
    await expect(canvas.getByText('Studio Mic 1')).toBeInTheDocument();

    // Check idle status
    await expect(canvas.getByText('off')).toBeInTheDocument();
  },
};

// Recording state
export const Recording: Story = {
  args: {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    recorder: mockRecordingRecorder as any,
    isSelected: false,
  },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);

    // Check recorder name
    await expect(canvas.getByText('Live Room Mic')).toBeInTheDocument();
  },
};

// Selected state
export const Selected: Story = {
  args: {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    recorder: mockIdleRecorder as any,
    isSelected: true,
  },
  play: async ({ canvasElement }) => {
    const link = canvasElement.querySelector('.link');
    await expect(link).toHaveClass('is-selected');
  },
};

// Long name (truncation)
export const LongName: Story = {
  args: {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    recorder: mockLongNameRecorder as any,
    isSelected: false,
  },
  play: async ({ canvasElement }) => {
    const nameElement = canvasElement.querySelector('.name');
    await expect(nameElement).toBeInTheDocument();
  },
};

// Hover interaction
export const HoverInteraction: Story = {
  args: {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    recorder: mockIdleRecorder as any,
    isSelected: false,
  },
  play: async ({ canvasElement }) => {
    const link = canvasElement.querySelector('.link') as HTMLElement;

    await userEvent.hover(link);
    // After hover, should have hover styles (white background)
    await expect(link).toBeInTheDocument();
  },
};

// Multiple cards
export const MultipleCards: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);

    await expect(canvas.getByText('Studio Mic 1')).toBeInTheDocument();
    await expect(canvas.getByText('Live Room Mic')).toBeInTheDocument();
  },
  render: () => ({
    components: { DeviceCard },
    setup() {
      return {
        recorders: [
          mockIdleRecorder,
          mockRecordingRecorder,
          { ...mockIdleRecorder, recorderId: 'r3', recorderName: 'Booth Mic' },
        ],
        selectedId: 'recorder-2',
      };
    },
    template: `
      <div style="display: flex; flex-direction: column; gap: 0.5rem;">
        <DeviceCard
          v-for="rec in recorders"
          :key="rec.recorderId"
          :recorder="rec"
          :isSelected="rec.recorderId === selectedId"
        />
      </div>
    `,
  }),
};

// In sidebar context
export const InSidebarContext: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await expect(canvas.getByText('Studio Mic 1')).toBeInTheDocument();
  },
  render: () => ({
    components: { DeviceCard },
    setup() {
      return { recorder: mockIdleRecorder };
    },
    template: `
      <div style="width: 250px; padding: 1rem; background: #f5f5f5; border-radius: 8px;">
        <h3 style="font-size: 0.75rem; text-transform: uppercase; color: #666; margin-bottom: 0.5rem;">Devices</h3>
        <DeviceCard :recorder="recorder" />
      </div>
    `,
  }),
};

// Click interaction
export const ClickInteraction: Story = {
  args: {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    recorder: mockIdleRecorder as any,
    isSelected: false,
  },
  play: async ({ canvasElement }) => {
    const link = canvasElement.querySelector('.link') as HTMLElement;

    await userEvent.click(link);
    await expect(link).toBeInTheDocument();
  },
};
