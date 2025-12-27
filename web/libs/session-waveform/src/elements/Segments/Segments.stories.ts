/**
 * Test Plan: Segments
 *
 * Scenario: Render empty segments list
 *   Given the Segments component is rendered with no segments
 *   When the component mounts
 *   Then an empty table should be displayed
 *
 * Scenario: Render segments list with checkboxes
 *   Given the Segments component is rendered with segments
 *   When the component mounts
 *   Then all segments should be displayed as rows with checkboxes
 *
 * Scenario: Select all via header checkbox
 *   Given multiple segments exist
 *   When the header checkbox is clicked
 *   Then all non-deleted segments should be selected
 *   And the bulk actions bar should appear
 *
 * Scenario: Bulk render selected segments
 *   Given multiple segments are selected
 *   When "Render Selected" is clicked
 *   Then renderSegment should be emitted for each selected segment
 *
 * Scenario: Bulk delete selected segments
 *   Given multiple segments are selected
 *   When "Delete" is clicked
 *   Then removeSegment should be emitted for each selected segment
 *   And selection should be cleared
 *
 * Scenario: Clear selection
 *   Given segments are selected
 *   When "Clear" is clicked
 *   Then all segments should be deselected
 *   And the bulk actions bar should hide
 */

import type { Meta, StoryObj } from '@storybook/vue3';
import { userEvent, within, expect } from '@storybook/test';
import Segments from './Segments.vue';
import {
  createPeaksContext,
  providePeaksContext,
} from '../../context/usePeaksContext';
import { intToChar } from '../../context/installSegmentsControls';

const createMockContext = (segments: any[] = []) => ({
  initialState: {
    audioUrls: [{ type: 'audio/mp3', src: '/test.mp3' }],
    permissions: { create: true, update: true, delete: true },
    segments,
    player: {
      isPlaying: false,
      duration: 300,
      currentTime: 0,
    },
  },
});

const mockSegments = [
  {
    id: '1',
    startIndex: 'A',
    endIndex: 'B',
    startTime: 0,
    endTime: 30,
    labelText: 'Introduction',
    renders: [],
  },
  {
    id: '2',
    startIndex: 'C',
    endIndex: 'D',
    startTime: 30,
    endTime: 90,
    labelText: 'Main Content',
    renders: [],
  },
  {
    id: '3',
    startIndex: 'E',
    endIndex: 'F',
    startTime: 90,
    endTime: 120,
    labelText: 'Conclusion',
    state: 'finished' as const,
    renders: [{ type: 'audio/flac', src: '/conclusion.flac' }],
  },
];

const meta: Meta<typeof Segments> = {
  title: 'Lib/Elements/Segments/Segments',
  component: Segments,
  tags: ['autodocs'],
  parameters: {
    layout: 'padded',
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

// Empty segments
export const Empty: Story = {
  play: async ({ canvasElement }) => {
    const table = canvasElement.querySelector('table');
    await expect(table).toBeInTheDocument();

    // Should have header but no data rows
    const tbody = canvasElement.querySelector('tbody');
    const rows = tbody?.querySelectorAll('tr') || [];
    await expect(rows).toHaveLength(0);
  },
  render: () => ({
    components: { Segments },
    setup() {
      const context = createPeaksContext(createMockContext([]));
      providePeaksContext(context);
      return {};
    },
    template: '<Segments />',
  }),
};

// With segments
export const WithSegments: Story = {
  play: async ({ canvasElement }) => {
    // Check that checkboxes exist
    const checkboxes = canvasElement.querySelectorAll('input[type="checkbox"]');
    // 1 header + 3 row checkboxes
    await expect(checkboxes.length).toBe(4);

    // Check segment labels
    const canvas = within(canvasElement);
    await expect(canvas.getByText('Introduction')).toBeInTheDocument();
    await expect(canvas.getByText('Main Content')).toBeInTheDocument();
    await expect(canvas.getByText('Conclusion')).toBeInTheDocument();
  },
  render: () => ({
    components: { Segments },
    setup() {
      const context = createPeaksContext(createMockContext(mockSegments));
      providePeaksContext(context);
      return {};
    },
    template: '<Segments />',
  }),
};

// Select all via header checkbox
export const SelectAll: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);

    // Find header checkbox (first one)
    const checkboxes = canvasElement.querySelectorAll('input[type="checkbox"]');
    const headerCheckbox = checkboxes[0];

    // Click to select all
    await userEvent.click(headerCheckbox);

    // Bulk actions bar should appear
    await expect(canvas.getByText(/segment\(s\) selected/)).toBeInTheDocument();
    await expect(canvas.getByText('Render Selected')).toBeInTheDocument();
    await expect(canvas.getByText('Delete')).toBeInTheDocument();
    await expect(canvas.getByText('Clear')).toBeInTheDocument();
  },
  render: () => ({
    components: { Segments },
    setup() {
      const context = createPeaksContext(createMockContext(mockSegments));
      providePeaksContext(context);
      return {};
    },
    template: '<Segments />',
  }),
};

// Select individual segments
export const SelectIndividual: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);

    // Find row checkboxes (skip header)
    const checkboxes = canvasElement.querySelectorAll('input[type="checkbox"]');
    const firstRowCheckbox = checkboxes[1];
    const secondRowCheckbox = checkboxes[2];

    // Select first segment
    await userEvent.click(firstRowCheckbox);

    // Should show 1 selected
    await expect(canvas.getByText('1 segment(s) selected')).toBeInTheDocument();

    // Select second segment
    await userEvent.click(secondRowCheckbox);

    // Should show 2 selected
    await expect(canvas.getByText('2 segment(s) selected')).toBeInTheDocument();
  },
  render: () => ({
    components: { Segments },
    setup() {
      const context = createPeaksContext(createMockContext(mockSegments));
      providePeaksContext(context);
      return {};
    },
    template: '<Segments />',
  }),
};

// Clear selection
export const ClearSelection: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);

    // Select all first
    const checkboxes = canvasElement.querySelectorAll('input[type="checkbox"]');
    await userEvent.click(checkboxes[0]);

    // Verify bulk bar is shown
    await expect(canvas.getByText(/segment\(s\) selected/)).toBeInTheDocument();

    // Click clear
    const clearButton = canvas.getByText('Clear');
    await userEvent.click(clearButton);

    // Bulk actions bar should be hidden
    await expect(
      canvas.queryByText(/segment\(s\) selected/)
    ).not.toBeInTheDocument();
  },
  render: () => ({
    components: { Segments },
    setup() {
      const context = createPeaksContext(createMockContext(mockSegments));
      providePeaksContext(context);
      return {};
    },
    template: '<Segments />',
  }),
};

// Single segment
export const SingleSegment: Story = {
  play: async ({ canvasElement }) => {
    const checkboxes = canvasElement.querySelectorAll('input[type="checkbox"]');
    // 1 header + 1 row checkbox
    await expect(checkboxes.length).toBe(2);
  },
  render: () => ({
    components: { Segments },
    setup() {
      const context = createPeaksContext(
        createMockContext([
          {
            id: '1',
            startIndex: 'X',
            endIndex: 'Y',
            startTime: 10,
            endTime: 50,
            labelText: 'Solo Segment',
            renders: [],
          },
        ])
      );
      providePeaksContext(context);
      return {};
    },
    template: '<Segments />',
  }),
};

// With rendered segments (FLAC downloads available)
export const WithRenderedSegments: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);

    // Rendered segment should have FLAC download button
    await expect(canvas.getByText('FLAC')).toBeInTheDocument();

    // Non-rendered should show Ready
    await expect(canvas.getByText('Ready')).toBeInTheDocument();
  },
  render: () => ({
    components: { Segments },
    setup() {
      const context = createPeaksContext(
        createMockContext([
          {
            id: '1',
            startIndex: 'A',
            endIndex: 'B',
            startTime: 0,
            endTime: 30,
            labelText: 'Rendered Segment',
            state: 'finished' as const,
            renders: [{ type: 'audio/flac', src: '/segment1.flac' }],
          },
          {
            id: '2',
            startIndex: 'C',
            endIndex: 'D',
            startTime: 30,
            endTime: 60,
            labelText: 'Not Rendered Yet',
            renders: [],
          },
        ])
      );
      providePeaksContext(context);
      return {};
    },
    template: '<Segments />',
  }),
};

// Read-only permissions
export const ReadOnlyPermissions: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);

    // Labels should be displayed as text, not inputs
    await expect(canvas.getByText('Read Only Segment')).toBeInTheDocument();

    // Checkboxes should still exist for selection
    const checkboxes = canvasElement.querySelectorAll('input[type="checkbox"]');
    await expect(checkboxes.length).toBeGreaterThan(0);
  },
  render: () => ({
    components: { Segments },
    setup() {
      const context = createPeaksContext({
        initialState: {
          audioUrls: [{ type: 'audio/mp3', src: '/test.mp3' }],
          permissions: { create: false, update: false, delete: false },
          segments: [
            {
              id: '1',
              startIndex: 'A',
              endIndex: 'B',
              startTime: 0,
              endTime: 30,
              labelText: 'Read Only Segment',
              renders: [],
            },
          ],
          player: { isPlaying: false, duration: 300, currentTime: 0 },
        },
      });
      providePeaksContext(context);
      return {};
    },
    template: '<Segments />',
  }),
};

// Many segments (stress test)
export const ManySegments: Story = {
  play: async ({ canvasElement }) => {
    const rows = canvasElement.querySelectorAll('tbody tr');
    await expect(rows.length).toBe(10);

    // Select all and verify bulk actions
    const checkboxes = canvasElement.querySelectorAll('input[type="checkbox"]');
    await userEvent.click(checkboxes[0]);

    const canvas = within(canvasElement);
    await expect(
      canvas.getByText('10 segment(s) selected')
    ).toBeInTheDocument();
  },
  render: () => ({
    components: { Segments },
    setup() {
      const manySegments = Array.from({ length: 10 }, (_, i) => ({
        id: `${i + 1}`,
        startIndex: intToChar(i * 2),
        endIndex: intToChar(i * 2 + 1),
        startTime: i * 30,
        endTime: (i + 1) * 30,
        labelText: `Segment ${i + 1}`,
        renders: [],
      }));

      const context = createPeaksContext(createMockContext(manySegments));
      providePeaksContext(context);
      return {};
    },
    template: '<Segments />',
  }),
};

// Mixed states with selection
export const MixedStates: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);

    // Verify different states are shown
    await expect(canvas.getByText('Ready')).toBeInTheDocument();
    await expect(canvas.getByText('Rendering...')).toBeInTheDocument();
    await expect(canvas.getByText('Error')).toBeInTheDocument();
    await expect(canvas.getByText('FLAC')).toBeInTheDocument();
  },
  render: () => ({
    components: { Segments },
    setup() {
      const mixedSegments = [
        {
          id: '1',
          startIndex: 'A',
          endIndex: 'B',
          startTime: 0,
          endTime: 30,
          labelText: 'Ready to Render',
          renders: [],
        },
        {
          id: '2',
          startIndex: 'C',
          endIndex: 'D',
          startTime: 30,
          endTime: 60,
          labelText: 'Currently Rendering',
          state: 'rendering' as const,
          renders: [],
        },
        {
          id: '3',
          startIndex: 'E',
          endIndex: 'F',
          startTime: 60,
          endTime: 90,
          labelText: 'Failed Render',
          state: 'error' as const,
          errorMessage: 'Encoding failed',
          renders: [],
        },
        {
          id: '4',
          startIndex: 'G',
          endIndex: 'H',
          startTime: 90,
          endTime: 120,
          labelText: 'Complete',
          state: 'finished' as const,
          renders: [{ type: 'audio/flac', src: '/segment.flac' }],
        },
      ];

      const context = createPeaksContext(createMockContext(mixedSegments));
      providePeaksContext(context);
      return {};
    },
    template: '<Segments />',
  }),
};
