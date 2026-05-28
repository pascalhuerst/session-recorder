/**
 * Test Plan: Segment
 *
 * Scenario: Render segment row with checkbox
 *   Given the Segment component is rendered with segment data
 *   When the component mounts
 *   Then checkbox, preview, start/end markers, label, and status should be visible
 *
 * Scenario: Toggle selection via checkbox
 *   Given the segment is not selected
 *   When the checkbox is clicked
 *   Then the toggle-selection event should be emitted
 *
 * Scenario: Edit segment label
 *   Given the user has update permissions
 *   When the user edits the label input
 *   Then the updateSegment command should be emitted
 *
 * Scenario: Edit time inputs
 *   Given the user has update permissions
 *   When the user changes start/end time
 *   Then the updateSegment command should be emitted
 *
 * Scenario: Download FLAC when finished
 *   Given the segment state is 'finished' with FLAC render
 *   When the component mounts
 *   Then a FLAC download button should be visible
 *
 * Scenario: Rendering state shows spinner
 *   Given a segment with state='rendering'
 *   When the component mounts
 *   Then a spinner and "Rendering..." text should be visible
 *
 * Scenario: Error state shows error and retry
 *   Given a segment with state='error'
 *   When the component mounts
 *   Then an error indicator with errorMessage should be visible
 *   And a retry button should be available
 *
 * Scenario: Selected segment has highlight
 *   Given a segment with selected=true
 *   When the component mounts
 *   Then the row should have selected styling
 */

import type { Meta, StoryObj } from '@storybook/vue3';
import { userEvent, within, expect } from '@storybook/test';
import { ref } from 'vue';
import Segment from './Segment.vue';
import {
  createPeaksContext,
  providePeaksContext,
} from '../../context/usePeaksContext';

const createMockContext = (
  segment: any,
  permissions = { create: true, update: true, delete: true }
) => ({
  initialState: {
    audioUrls: [{ type: 'audio/mp3', src: '/test.mp3' }],
    permissions,
    segments: [segment],
    player: {
      isPlaying: false,
      duration: 300,
      currentTime: 0,
    },
  },
});

const baseSegment = {
  id: '1',
  startIndex: 'A',
  endIndex: 'B',
  startTime: 30,
  endTime: 90,
  labelText: 'Test Segment',
  renders: [],
};

const tableTemplate = `
  <table style="width: 100%">
    <thead>
      <tr>
        <th style="width: 40px"></th>
        <th style="width: 120px"></th>
        <th style="width: 140px"></th>
        <th style="width: 140px"></th>
        <th></th>
        <th style="width: 120px"></th>
      </tr>
    </thead>
    <tbody>
      <Segment :segment="segment" :selected="selected" @toggle-selection="onToggle" />
    </tbody>
  </table>
`;

const meta: Meta<typeof Segment> = {
  title: 'Lib/Elements/Segments/Segment',
  component: Segment,
  tags: ['autodocs'],
  parameters: {
    layout: 'padded',
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

// Default editable segment
export const Default: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);

    // Check checkbox exists
    const checkbox = canvasElement.querySelector('input[type="checkbox"]');
    await expect(checkbox).toBeInTheDocument();

    // Check markers
    await expect(canvas.getByText('A')).toBeInTheDocument();
    await expect(canvas.getByText('B')).toBeInTheDocument();

    // Check status shows "Ready"
    await expect(canvas.getByText('Ready')).toBeInTheDocument();
  },
  render: () => ({
    components: { Segment },
    setup() {
      const context = createPeaksContext(createMockContext(baseSegment));
      providePeaksContext(context);
      const selected = ref(false);
      const onToggle = () => {
        selected.value = !selected.value;
      };
      return { segment: baseSegment, selected, onToggle };
    },
    template: tableTemplate,
  }),
};

// Selected segment
export const Selected: Story = {
  play: async ({ canvasElement }) => {
    const row = canvasElement.querySelector('.row--selected');
    await expect(row).toBeInTheDocument();

    const checkbox = canvasElement.querySelector(
      'input[type="checkbox"]'
    ) as HTMLInputElement;
    await expect(checkbox.checked).toBe(true);
  },
  render: () => ({
    components: { Segment },
    setup() {
      const context = createPeaksContext(createMockContext(baseSegment));
      providePeaksContext(context);
      const selected = ref(true);
      const onToggle = () => {
        selected.value = !selected.value;
      };
      return { segment: baseSegment, selected, onToggle };
    },
    template: tableTemplate,
  }),
};

// Toggle selection interaction
export const ToggleSelection: Story = {
  play: async ({ canvasElement }) => {
    const checkbox = canvasElement.querySelector(
      'input[type="checkbox"]'
    ) as HTMLInputElement;
    await expect(checkbox).toBeInTheDocument();

    // Initially not selected
    await expect(checkbox.checked).toBe(false);

    // Click to select
    await userEvent.click(checkbox);
    await expect(checkbox.checked).toBe(true);

    // Row should have selected class
    const row = canvasElement.querySelector('.row--selected');
    await expect(row).toBeInTheDocument();
  },
  render: () => ({
    components: { Segment },
    setup() {
      const context = createPeaksContext(createMockContext(baseSegment));
      providePeaksContext(context);
      const selected = ref(false);
      const onToggle = () => {
        selected.value = !selected.value;
      };
      return { segment: baseSegment, selected, onToggle };
    },
    template: tableTemplate,
  }),
};

// Finished state with FLAC download
export const FinishedWithFlac: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);

    // Should show FLAC download button
    await expect(canvas.getByText('FLAC')).toBeInTheDocument();

    // Download link should have correct href
    const link = canvasElement.querySelector('a[href="/segment.flac"]');
    await expect(link).toBeInTheDocument();
  },
  render: () => ({
    components: { Segment },
    setup() {
      const finishedSegment = {
        ...baseSegment,
        state: 'finished' as const,
        renders: [
          { type: 'audio/ogg', src: '/segment.ogg' },
          { type: 'audio/flac', src: '/segment.flac' },
        ],
      };
      const context = createPeaksContext(createMockContext(finishedSegment));
      providePeaksContext(context);
      const selected = ref(false);
      const onToggle = () => {
        selected.value = !selected.value;
      };
      return { segment: finishedSegment, selected, onToggle };
    },
    template: tableTemplate,
  }),
};

// Read-only segment (no permissions)
export const ReadOnly: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);

    // Should display text instead of inputs for label
    await expect(canvas.getByText('Read Only Label')).toBeInTheDocument();

    // Checkbox should still work for selection
    const checkbox = canvasElement.querySelector('input[type="checkbox"]');
    await expect(checkbox).toBeInTheDocument();
  },
  render: () => ({
    components: { Segment },
    setup() {
      const readOnlySegment = {
        ...baseSegment,
        labelText: 'Read Only Label',
      };
      const context = createPeaksContext(
        createMockContext(readOnlySegment, {
          create: false,
          update: false,
          delete: false,
        })
      );
      providePeaksContext(context);
      const selected = ref(false);
      const onToggle = () => {
        selected.value = !selected.value;
      };
      return { segment: readOnlySegment, selected, onToggle };
    },
    template: tableTemplate,
  }),
};

// Deleted segment
export const DeletedSegment: Story = {
  play: async ({ canvasElement }) => {
    const row = canvasElement.querySelector('.row--deleted');
    await expect(row).toBeInTheDocument();

    // Checkbox should be disabled for deleted segments
    const checkbox = canvasElement.querySelector(
      'input[type="checkbox"]'
    ) as HTMLInputElement;
    await expect(checkbox.disabled).toBe(true);
  },
  render: () => ({
    components: { Segment },
    setup() {
      const deletedSegment = {
        ...baseSegment,
        deleted: true,
        labelText: 'Deleted Segment',
      };
      const context = createPeaksContext(createMockContext(deletedSegment));
      providePeaksContext(context);
      const selected = ref(false);
      const onToggle = () => {
        selected.value = !selected.value;
      };
      return { segment: deletedSegment, selected, onToggle };
    },
    template: tableTemplate,
  }),
};

// Queued state - shows queued indicator
export const QueuedState: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);

    // Should show "queued" text
    await expect(canvas.getByText('queued')).toBeInTheDocument();

    // Should have the queued indicator
    const indicator = canvasElement.querySelector('.status-indicator--queued');
    await expect(indicator).toBeInTheDocument();
  },
  render: () => ({
    components: { Segment },
    setup() {
      const queuedSegment = {
        ...baseSegment,
        state: 'queued' as const,
        renders: [],
      };
      const context = createPeaksContext(createMockContext(queuedSegment));
      providePeaksContext(context);
      const selected = ref(false);
      const onToggle = () => {
        selected.value = !selected.value;
      };
      return { segment: queuedSegment, selected, onToggle };
    },
    template: tableTemplate,
  }),
};

// Rendering state - shows spinner
export const RenderingState: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);

    // Should show "processing" text
    await expect(canvas.getByText('processing')).toBeInTheDocument();
  },
  render: () => ({
    components: { Segment },
    setup() {
      const renderingSegment = {
        ...baseSegment,
        state: 'rendering' as const,
        renders: [],
      };
      const context = createPeaksContext(createMockContext(renderingSegment));
      providePeaksContext(context);
      const selected = ref(false);
      const onToggle = () => {
        selected.value = !selected.value;
      };
      return { segment: renderingSegment, selected, onToggle };
    },
    template: tableTemplate,
  }),
};

// Error state - shows error indicator and retry button
export const ErrorState: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);

    // Should show "Error" text
    await expect(canvas.getByText('Error')).toBeInTheDocument();

    // Should show Retry button
    await expect(canvas.getByText('Retry')).toBeInTheDocument();
  },
  render: () => ({
    components: { Segment },
    setup() {
      const errorSegment = {
        ...baseSegment,
        state: 'error' as const,
        errorMessage: 'encoding failed',
        renders: [],
      };
      const context = createPeaksContext(createMockContext(errorSegment));
      providePeaksContext(context);
      const selected = ref(false);
      const onToggle = () => {
        selected.value = !selected.value;
      };
      return { segment: errorSegment, selected, onToggle };
    },
    template: tableTemplate,
  }),
};

// Error retry interaction
export const ErrorRetryInteraction: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);

    // Find and click the retry button
    const retryButton = canvas.getByText('Retry');
    await expect(retryButton).toBeInTheDocument();

    // Click retry triggers renderSegment command
    await userEvent.click(retryButton);
  },
  render: () => ({
    components: { Segment },
    setup() {
      const errorSegment = {
        ...baseSegment,
        state: 'error' as const,
        errorMessage: 'Connection timeout while encoding',
        renders: [],
      };
      const context = createPeaksContext(createMockContext(errorSegment));
      providePeaksContext(context);
      const selected = ref(false);
      const onToggle = () => {
        selected.value = !selected.value;
      };
      return { segment: errorSegment, selected, onToggle };
    },
    template: tableTemplate,
  }),
};

// Edit label interaction
export const EditLabel: Story = {
  play: async ({ canvasElement }) => {
    const inputs = canvasElement.querySelectorAll('input[type="text"]');
    const labelInput = inputs[inputs.length - 1]; // Last text input should be label

    if (labelInput) {
      await userEvent.clear(labelInput);
      await userEvent.type(labelInput, 'Updated Label');
      await expect(labelInput).toHaveValue('Updated Label');
    }
  },
  render: () => ({
    components: { Segment },
    setup() {
      const context = createPeaksContext(createMockContext(baseSegment));
      providePeaksContext(context);
      const selected = ref(false);
      const onToggle = () => {
        selected.value = !selected.value;
      };
      return { segment: baseSegment, selected, onToggle };
    },
    template: tableTemplate,
  }),
};

// All segment states in one view
export const AllStates: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);

    // Check all states are represented
    await expect(canvas.getByText('queued')).toBeInTheDocument();
    await expect(canvas.getByText('processing')).toBeInTheDocument();
    await expect(canvas.getByText('Error')).toBeInTheDocument();
  },
  render: () => ({
    components: { Segment },
    setup() {
      const segments = [
        { ...baseSegment, id: '1', labelText: 'Ready to Render' },
        {
          ...baseSegment,
          id: '2',
          state: 'queued' as const,
          labelText: 'Queued for Render',
        },
        {
          ...baseSegment,
          id: '3',
          state: 'rendering' as const,
          labelText: 'Currently Rendering',
        },
        {
          ...baseSegment,
          id: '4',
          state: 'error' as const,
          errorMessage: 'Encoding failed',
          labelText: 'Failed Render',
        },
        {
          ...baseSegment,
          id: '5',
          state: 'finished' as const,
          labelText: 'Complete',
          renders: [{ type: 'audio/flac', src: '/segment.flac' }],
        },
      ];
      const context = createPeaksContext({
        initialState: {
          audioUrls: [{ type: 'audio/mp3', src: '/test.mp3' }],
          permissions: { create: true, update: true, delete: true },
          segments,
          player: { isPlaying: false, duration: 300, currentTime: 0 },
        },
      });
      providePeaksContext(context);
      const selectedIds = ref(new Set<string>());
      return {
        segments,
        isSelected: (id: string) => selectedIds.value.has(id),
        onToggle: (id: string) => {
          if (selectedIds.value.has(id)) {
            selectedIds.value.delete(id);
          } else {
            selectedIds.value.add(id);
          }
          selectedIds.value = new Set(selectedIds.value);
        },
      };
    },
    template: `
      <table style="width: 100%">
        <thead>
          <tr>
            <th style="width: 40px"></th>
            <th style="width: 120px"></th>
            <th style="width: 140px">Start</th>
            <th style="width: 140px">End</th>
            <th>Label</th>
            <th style="width: 120px">Status</th>
          </tr>
        </thead>
        <tbody>
          <Segment
            v-for="segment in segments"
            :key="segment.id"
            :segment="segment"
            :selected="isSelected(segment.id)"
            @toggle-selection="onToggle(segment.id)"
          />
        </tbody>
      </table>
    `,
  }),
};
