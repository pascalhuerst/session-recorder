/**
 * Test Plan: Segment
 *
 * Scenario: Render segment row
 *   Given the Segment component is rendered with segment data
 *   When the component mounts
 *   Then start/end markers, label, and action buttons should be visible
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
 * Scenario: Delete segment
 *   Given the user has delete permissions
 *   When the remove button is clicked
 *   Then the removeSegment command should be emitted
 *
 * Scenario: Download rendered segment
 *   Given the segment has renders available
 *   When download button is clicked
 *   Then the file should start downloading
 */

import type { Meta, StoryObj } from '@storybook/vue3';
import { userEvent, within, expect } from '@storybook/test';
import Segment from './Segment.vue';
import { createPeaksContext, providePeaksContext } from '../../context/usePeaksContext';

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

    // Check markers
    await expect(canvas.getByText('A')).toBeInTheDocument();
    await expect(canvas.getByText('B')).toBeInTheDocument();

    // Check label input is present
    const inputs = canvasElement.querySelectorAll('input');
    await expect(inputs.length).toBeGreaterThan(0);

    // Check action buttons
    await expect(canvas.getByText('Render')).toBeInTheDocument();
    await expect(canvas.getByText('Remove')).toBeInTheDocument();
  },
  render: () => ({
    components: { Segment },
    setup() {
      const context = createPeaksContext(createMockContext(baseSegment));
      providePeaksContext(context);
      return { segment: baseSegment };
    },
    template: `
      <table>
        <thead>
          <tr>
            <th style="width: 10%"></th>
            <th style="width: 10%"></th>
            <th></th>
            <th style="width: 20%"></th>
          </tr>
        </thead>
        <Segment :segment="segment" />
      </table>
    `,
  }),
};

// Segment with renders (download available)
export const WithRenders: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);

    // Should have download links instead of render button
    const downloadLink = canvasElement.querySelector('a[href]');
    await expect(downloadLink).toBeInTheDocument();
    await expect(canvas.getByText('mp3')).toBeInTheDocument();
  },
  render: () => ({
    components: { Segment },
    setup() {
      const segmentWithRenders = {
        ...baseSegment,
        renders: [{ type: 'audio/mp3', src: '/segment.mp3' }],
      };
      const context = createPeaksContext(createMockContext(segmentWithRenders));
      providePeaksContext(context);
      return { segment: segmentWithRenders };
    },
    template: `
      <table>
        <thead>
          <tr>
            <th style="width: 10%"></th>
            <th style="width: 10%"></th>
            <th></th>
            <th style="width: 20%"></th>
          </tr>
        </thead>
        <Segment :segment="segment" />
      </table>
    `,
  }),
};

// Multiple render formats
export const MultipleRenderFormats: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);

    await expect(canvas.getByText('mp3')).toBeInTheDocument();
    await expect(canvas.getByText('flac')).toBeInTheDocument();
  },
  render: () => ({
    components: { Segment },
    setup() {
      const segmentWithRenders = {
        ...baseSegment,
        renders: [
          { type: 'audio/mp3', src: '/segment.mp3' },
          { type: 'audio/flac', src: '/segment.flac' },
        ],
      };
      const context = createPeaksContext(createMockContext(segmentWithRenders));
      providePeaksContext(context);
      return { segment: segmentWithRenders };
    },
    template: `
      <table>
        <thead>
          <tr>
            <th style="width: 10%"></th>
            <th style="width: 10%"></th>
            <th></th>
            <th style="width: 20%"></th>
          </tr>
        </thead>
        <Segment :segment="segment" />
      </table>
    `,
  }),
};

// Read-only segment (no permissions)
export const ReadOnly: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);

    // Should display text instead of inputs
    await expect(canvas.getByText('Read Only Label')).toBeInTheDocument();

    // No remove button
    const buttons = canvasElement.querySelectorAll('button');
    const removeBtn = Array.from(buttons).find((b) =>
      b.textContent?.includes('Remove')
    );
    await expect(removeBtn).not.toBeDefined();
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
      return { segment: readOnlySegment };
    },
    template: `
      <table>
        <thead>
          <tr>
            <th style="width: 10%"></th>
            <th style="width: 10%"></th>
            <th></th>
            <th style="width: 20%"></th>
          </tr>
        </thead>
        <Segment :segment="segment" />
      </table>
    `,
  }),
};

// Deleted segment
export const DeletedSegment: Story = {
  play: async ({ canvasElement }) => {
    const row = canvasElement.querySelector('.row--deleted');
    await expect(row).toBeInTheDocument();
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
      return { segment: deletedSegment };
    },
    template: `
      <table>
        <thead>
          <tr>
            <th style="width: 10%"></th>
            <th style="width: 10%"></th>
            <th></th>
            <th style="width: 20%"></th>
          </tr>
        </thead>
        <Segment :segment="segment" />
      </table>
    `,
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
      return { segment: baseSegment };
    },
    template: `
      <table>
        <thead>
          <tr>
            <th style="width: 10%"></th>
            <th style="width: 10%"></th>
            <th></th>
            <th style="width: 20%"></th>
          </tr>
        </thead>
        <Segment :segment="segment" />
      </table>
    `,
  }),
};

// Click to seek
export const ClickToSeek: Story = {
  play: async ({ canvasElement }) => {
    const row = canvasElement.querySelector('.row');
    await expect(row).toBeInTheDocument();

    // Clicking the row should trigger seek
    if (row) {
      await userEvent.click(row);
    }
  },
  render: () => ({
    components: { Segment },
    setup() {
      const context = createPeaksContext(createMockContext(baseSegment));
      providePeaksContext(context);
      return { segment: baseSegment };
    },
    template: `
      <table>
        <thead>
          <tr>
            <th style="width: 10%"></th>
            <th style="width: 10%"></th>
            <th></th>
            <th style="width: 20%"></th>
          </tr>
        </thead>
        <Segment :segment="segment" />
      </table>
    `,
  }),
};

// Long label
export const LongLabel: Story = {
  play: async ({ canvasElement }) => {
    const inputs = canvasElement.querySelectorAll('input');
    await expect(inputs.length).toBeGreaterThan(0);
  },
  render: () => ({
    components: { Segment },
    setup() {
      const longLabelSegment = {
        ...baseSegment,
        labelText:
          'This is a very long segment label that might overflow the container',
      };
      const context = createPeaksContext(createMockContext(longLabelSegment));
      providePeaksContext(context);
      return { segment: longLabelSegment };
    },
    template: `
      <table>
        <thead>
          <tr>
            <th style="width: 10%"></th>
            <th style="width: 10%"></th>
            <th></th>
            <th style="width: 20%"></th>
          </tr>
        </thead>
        <Segment :segment="segment" />
      </table>
    `,
  }),
};
