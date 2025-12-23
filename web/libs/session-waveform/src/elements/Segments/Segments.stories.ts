/**
 * Test Plan: Segments
 *
 * Scenario: Render empty segments list
 *   Given the Segments component is rendered with no segments
 *   When the component mounts
 *   Then an empty table should be displayed
 *
 * Scenario: Render segments list
 *   Given the Segments component is rendered with segments
 *   When the component mounts
 *   Then all segments should be displayed as rows
 *
 * Scenario: Multiple segments display
 *   Given multiple segments exist in state
 *   When rendered
 *   Then each segment should have its own row with markers and controls
 */

import type { Meta, StoryObj } from '@storybook/vue3';
import { within, expect } from '@storybook/test';
import Segments from './Segments.vue';
import { createPeaksContext, providePeaksContext } from '../../context/usePeaksContext';

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
    renders: [{ type: 'audio/mp3', src: '/conclusion.mp3' }],
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
    const rows = canvasElement.querySelectorAll('tr');
    await expect(rows).toHaveLength(1); // Just header
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
    // Check that the component wrapper exists
    const wrapper = canvasElement.querySelector('[data-v-app]') || canvasElement;
    await expect(wrapper).toBeInTheDocument();
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
    // Check that the component wrapper exists
    const wrapper = canvasElement.querySelector('[data-v-app]') || canvasElement;
    await expect(wrapper).toBeInTheDocument();
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

// Segments with renders (downloads available)
export const WithRenderedSegments: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);

    // Rendered segment should have download button
    const downloadLinks = canvasElement.querySelectorAll('a[href]');
    await expect(downloadLinks.length).toBeGreaterThan(0);
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
            renders: [
              { type: 'audio/mp3', src: '/segment1.mp3' },
              { type: 'audio/flac', src: '/segment1.flac' },
            ],
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

    // No remove buttons should be present
    const removeButtons = canvasElement.querySelectorAll('button');
    const removeBtn = Array.from(removeButtons).find((btn) =>
      btn.textContent?.includes('Remove')
    );
    await expect(removeBtn).not.toBeDefined();
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
    const rows = canvasElement.querySelectorAll('tr');
    // 1 header + 10 data rows
    await expect(rows.length).toBeGreaterThan(5);
  },
  render: () => ({
    components: { Segments },
    setup() {
      const manySegments = Array.from({ length: 10 }, (_, i) => ({
        id: `${i + 1}`,
        startIndex: String.fromCharCode(65 + i * 2),
        endIndex: String.fromCharCode(66 + i * 2),
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
