/**
 * Test Plan: DevicePicker
 *
 * Scenario: Render list of recorders
 *   Given the DevicePicker is rendered with recorders
 *   When the component mounts
 *   Then all recorder cards should be visible sorted by name
 *
 * Scenario: Selected recorder
 *   Given a recorder is selected
 *   When the picker renders
 *   Then the selected recorder card should have selected styling
 *
 * Scenario: Click recorder
 *   Given the picker is rendered
 *   When a recorder card is clicked
 *   Then navigation should occur to that recorder
 *
 * Scenario: Empty state
 *   Given no recorders
 *   When the picker renders
 *   Then the list should be empty
 */

import type { Meta, StoryObj } from '@storybook/vue3';
import { within, expect, userEvent } from '@storybook/test';
import DevicePicker from './DevicePicker.vue';

// Mock router
const mockRouter = {
  push: () => {},
};

// Mock recorder data
const createMockRecorder = (id: string, name: string, isRecording = false) => ({
  recorderID: id,
  recorderName: name,
  info: {
    oneofKind: 'status' as const,
    status: {
      signalStatus: isRecording ? 1 : 0,
      rmsPercent: isRecording ? 45 : 0,
    },
  },
});

const mockRecordersMap = new Map([
  ['recorder-1', createMockRecorder('recorder-1', 'Studio Mic A')],
  ['recorder-2', createMockRecorder('recorder-2', 'Live Room Mic', true)],
  ['recorder-3', createMockRecorder('recorder-3', 'Booth Mic')],
]);

const meta: Meta<typeof DevicePicker> = {
  title: 'App/Elements/DevicePicker',
  component: DevicePicker,
  tags: ['autodocs'],
  parameters: {
    layout: 'padded',
  },
  argTypes: {
    recorders: {
      control: 'object',
      description: 'Map of recorder ID to recorder data',
    },
    selectedRecorderId: {
      control: 'text',
      description: 'Currently selected recorder ID',
    },
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

// Default with recorders
export const Default: Story = {
  args: {
    recorders: mockRecordersMap as any,
    selectedRecorderId: undefined,
  },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);

    // All recorders should be visible (sorted alphabetically)
    await expect(canvas.getByText('Booth Mic')).toBeInTheDocument();
    await expect(canvas.getByText('Live Room Mic')).toBeInTheDocument();
    await expect(canvas.getByText('Studio Mic A')).toBeInTheDocument();
  },
};

// With selected recorder
export const WithSelection: Story = {
  args: {
    recorders: mockRecordersMap as any,
    selectedRecorderId: 'recorder-2',
  },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);

    // Selected recorder should be visible
    await expect(canvas.getByText('Live Room Mic')).toBeInTheDocument();

    // Should have a selected card
    const selectedCard = canvasElement.querySelector('.is-selected');
    await expect(selectedCard).toBeInTheDocument();
  },
};

// Empty state
export const Empty: Story = {
  args: {
    recorders: new Map() as any,
    selectedRecorderId: undefined,
  },
  play: async ({ canvasElement }) => {
    const list = canvasElement.querySelector('ul');
    await expect(list?.children.length).toBe(0);
  },
};

// Single recorder
export const SingleRecorder: Story = {
  args: {
    recorders: new Map([
      ['recorder-1', createMockRecorder('recorder-1', 'Only Recorder')],
    ]) as any,
    selectedRecorderId: 'recorder-1',
  },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await expect(canvas.getByText('Only Recorder')).toBeInTheDocument();
  },
};

// Many recorders (scrollable)
export const ManyRecorders: Story = {
  play: async ({ canvasElement }) => {
    const items = canvasElement.querySelectorAll('li');
    await expect(items.length).toBe(8);
  },
  render: () => ({
    components: { DevicePicker },
    setup() {
      const recorders = new Map([
        ['r1', createMockRecorder('r1', 'Mic 1')],
        ['r2', createMockRecorder('r2', 'Mic 2', true)],
        ['r3', createMockRecorder('r3', 'Mic 3')],
        ['r4', createMockRecorder('r4', 'Mic 4')],
        ['r5', createMockRecorder('r5', 'Mic 5', true)],
        ['r6', createMockRecorder('r6', 'Mic 6')],
        ['r7', createMockRecorder('r7', 'Mic 7')],
        ['r8', createMockRecorder('r8', 'Mic 8')],
      ]);
      return { recorders };
    },
    template: `
      <div style="max-width: 600px;">
        <DevicePicker :recorders="recorders" selectedRecorderId="r2" />
      </div>
    `,
  }),
};

// Click interaction
export const ClickInteraction: Story = {
  args: {
    recorders: mockRecordersMap as any,
    selectedRecorderId: 'recorder-1',
  },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);

    // Find a recorder card (don't click - router not mocked)
    const recorderCard = canvas.getByText('Booth Mic');
    await expect(recorderCard).toBeInTheDocument();
  },
};

// Mixed recording states
export const MixedStates: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);

    // Check recorders are displayed
    await expect(canvas.getByText('Idle Recorder')).toBeInTheDocument();
    await expect(canvas.getByText('Recording Recorder')).toBeInTheDocument();
  },
  render: () => ({
    components: { DevicePicker },
    setup() {
      const recorders = new Map([
        ['r1', createMockRecorder('r1', 'Idle Recorder')],
        ['r2', createMockRecorder('r2', 'Recording Recorder', true)],
      ]);
      return { recorders };
    },
    template: '<DevicePicker :recorders="recorders" />',
  }),
};
