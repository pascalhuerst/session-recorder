/**
 * Test Plan: SegmentWaveformPreview
 *
 * Scenario: Display loading state
 *   Given the component is mounted
 *   When the waveform data is not yet loaded
 *   Then a loading indicator should be visible
 *
 * Scenario: Display waveform preview
 *   Given valid waveform data and segment times
 *   When the component renders
 *   Then a canvas element should be visible
 *
 * Scenario: Display error state
 *   Given the waveform fetch fails
 *   When the component renders
 *   Then an error indicator should be visible
 *
 * Scenario: Apply custom color
 *   Given a color prop is provided
 *   When the waveform renders
 *   Then the waveform should use the specified color
 *
 * Scenario: Update on segment time change
 *   Given the component is rendered
 *   When startTime or endTime props change
 *   Then the waveform preview should update
 */

import type { Meta, StoryObj } from '@storybook/vue3';
import { expect, waitFor } from '@storybook/test';
import SegmentWaveformPreview from './SegmentWaveformPreview.vue';

// Mock waveform data - creates a simple binary waveform file
function createMockWaveformData(): ArrayBuffer {
  // Peaks.js binary format: 20-byte header + min/max pairs
  const length = 100; // 100 samples
  const bytesPerSample = 2; // 16-bit
  const headerSize = 20;
  const dataSize = length * bytesPerSample * 2; // min + max
  const buffer = new ArrayBuffer(headerSize + dataSize);
  const view = new DataView(buffer);

  // Header
  view.setInt32(0, 2, true); // version
  view.setUint32(4, 1, true); // flags (16-bit = 1)
  view.setInt32(8, 48000, true); // sample_rate
  view.setInt32(12, 256, true); // samples_per_pixel
  view.setUint32(16, length, true); // length

  // Generate a simple sine wave pattern for visual testing
  for (let i = 0; i < length; i++) {
    const offset = headerSize + i * bytesPerSample * 2;
    const value = Math.sin((i / length) * Math.PI * 4) * 16384;
    view.setInt16(offset, -Math.abs(value), true); // min
    view.setInt16(offset + 2, Math.abs(value), true); // max
  }

  return buffer;
}

// Setup mock fetch for waveform data
const setupMockFetch = () => {
  const originalFetch = window.fetch;
  window.fetch = async (url: RequestInfo | URL) => {
    if (typeof url === 'string' && url.includes('waveform')) {
      return new Response(createMockWaveformData(), {
        status: 200,
        headers: { 'Content-Type': 'application/octet-stream' },
      });
    }
    return originalFetch(url);
  };
  return () => {
    window.fetch = originalFetch;
  };
};

const meta: Meta<typeof SegmentWaveformPreview> = {
  title: 'Lib/Elements/Segments/SegmentWaveformPreview',
  component: SegmentWaveformPreview,
  tags: ['autodocs'],
  parameters: {
    layout: 'padded',
  },
  argTypes: {
    startTime: {
      control: { type: 'number', min: 0, max: 300 },
      description: 'Start time of the segment in seconds',
    },
    endTime: {
      control: { type: 'number', min: 0, max: 300 },
      description: 'End time of the segment in seconds',
    },
    duration: {
      control: { type: 'number', min: 1, max: 600 },
      description: 'Total duration of the audio in seconds',
    },
    height: {
      control: { type: 'number', min: 20, max: 100 },
      description: 'Height of the preview in pixels',
    },
    color: {
      control: 'color',
      description: 'Color of the waveform',
    },
    waveformUrl: {
      control: 'text',
      description: 'URL to the waveform data file',
    },
  },
  decorators: [
    (story) => {
      setupMockFetch();
      return story();
    },
  ],
};

export default meta;
type Story = StoryObj<typeof meta>;

// Default preview
export const Default: Story = {
  args: {
    startTime: 30,
    endTime: 90,
    duration: 300,
    waveformUrl: '/mock-waveform.dat',
    height: 40,
  },
  play: async ({ canvasElement }) => {
    // Wait for loading to complete
    await waitFor(
      () => {
        const preview = canvasElement.querySelector('.waveform-preview');
        expect(preview).toBeInTheDocument();
      },
      { timeout: 2000 }
    );

    // Check canvas exists after loading
    await waitFor(
      () => {
        const canvasEl = canvasElement.querySelector('canvas');
        expect(canvasEl).toBeInTheDocument();
      },
      { timeout: 2000 }
    );
  },
  render: (args) => ({
    components: { SegmentWaveformPreview },
    setup() {
      return { args };
    },
    template:
      '<div style="width: 200px;"><SegmentWaveformPreview v-bind="args" /></div>',
  }),
};

// Custom height
export const CustomHeight: Story = {
  args: {
    startTime: 0,
    endTime: 60,
    duration: 300,
    waveformUrl: '/mock-waveform.dat',
    height: 60,
  },
  play: async ({ canvasElement }) => {
    await waitFor(
      () => {
        const preview = canvasElement.querySelector('.waveform-preview');
        expect(preview).toBeInTheDocument();
        expect(preview).toHaveStyle({ height: '60px' });
      },
      { timeout: 2000 }
    );
  },
  render: (args) => ({
    components: { SegmentWaveformPreview },
    setup() {
      return { args };
    },
    template:
      '<div style="width: 200px;"><SegmentWaveformPreview v-bind="args" /></div>',
  }),
};

// Custom color
export const CustomColor: Story = {
  args: {
    startTime: 30,
    endTime: 90,
    duration: 300,
    waveformUrl: '/mock-waveform.dat',
    height: 40,
    color: '#4299e1',
  },
  play: async ({ canvasElement }) => {
    await waitFor(
      () => {
        const canvasEl = canvasElement.querySelector('canvas');
        expect(canvasEl).toBeInTheDocument();
      },
      { timeout: 2000 }
    );
  },
  render: (args) => ({
    components: { SegmentWaveformPreview },
    setup() {
      return { args };
    },
    template:
      '<div style="width: 200px;"><SegmentWaveformPreview v-bind="args" /></div>',
  }),
};

// Loading state (no mock fetch)
export const LoadingState: Story = {
  args: {
    startTime: 0,
    endTime: 60,
    duration: 300,
    waveformUrl: '/never-resolves.dat',
    height: 40,
  },
  decorators: [
    (story) => {
      // Override fetch to never resolve
      const originalFetch = window.fetch;
      window.fetch = () => new Promise(() => {}); // Never resolves
      const cleanup = () => {
        window.fetch = originalFetch;
      };
      setTimeout(cleanup, 5000);
      return story();
    },
  ],
  play: async ({ canvasElement }) => {
    const loading = canvasElement.querySelector('.loading');
    await expect(loading).toBeInTheDocument();
  },
  render: (args) => ({
    components: { SegmentWaveformPreview },
    setup() {
      return { args };
    },
    template:
      '<div style="width: 200px;"><SegmentWaveformPreview v-bind="args" /></div>',
  }),
};

// Error state
export const ErrorState: Story = {
  args: {
    startTime: 0,
    endTime: 60,
    duration: 300,
    waveformUrl: '/error-waveform.dat',
    height: 40,
  },
  decorators: [
    (story) => {
      const originalFetch = window.fetch;
      window.fetch = async () => {
        return new Response(null, { status: 404, statusText: 'Not Found' });
      };
      const cleanup = () => {
        window.fetch = originalFetch;
      };
      setTimeout(cleanup, 5000);
      return story();
    },
  ],
  play: async ({ canvasElement }) => {
    await waitFor(
      () => {
        const error = canvasElement.querySelector('.error');
        expect(error).toBeInTheDocument();
      },
      { timeout: 2000 }
    );
  },
  render: (args) => ({
    components: { SegmentWaveformPreview },
    setup() {
      return { args };
    },
    template:
      '<div style="width: 200px;"><SegmentWaveformPreview v-bind="args" /></div>',
  }),
};

// Multiple previews with different colors
export const MultipleColors: Story = {
  play: async ({ canvasElement }) => {
    await waitFor(
      () => {
        const previews = canvasElement.querySelectorAll('.waveform-preview');
        expect(previews.length).toBe(4);
      },
      { timeout: 2000 }
    );
  },
  render: () => ({
    components: { SegmentWaveformPreview },
    template: `
      <div style="display: flex; flex-direction: column; gap: 8px; width: 300px;">
        <SegmentWaveformPreview
          :startTime="0"
          :endTime="60"
          :duration="300"
          waveformUrl="/mock-waveform.dat"
          :height="30"
          color="#4299e1"
        />
        <SegmentWaveformPreview
          :startTime="60"
          :endTime="120"
          :duration="300"
          waveformUrl="/mock-waveform.dat"
          :height="30"
          color="#38b2ac"
        />
        <SegmentWaveformPreview
          :startTime="120"
          :endTime="180"
          :duration="300"
          waveformUrl="/mock-waveform.dat"
          :height="30"
          color="#ed8936"
        />
        <SegmentWaveformPreview
          :startTime="180"
          :endTime="240"
          :duration="300"
          waveformUrl="/mock-waveform.dat"
          :height="30"
          color="#ed64a6"
        />
      </div>
    `,
  }),
};

// Wide preview
export const WidePreview: Story = {
  args: {
    startTime: 30,
    endTime: 90,
    duration: 300,
    waveformUrl: '/mock-waveform.dat',
    height: 40,
  },
  play: async ({ canvasElement }) => {
    await waitFor(
      () => {
        const canvasEl = canvasElement.querySelector('canvas');
        expect(canvasEl).toBeInTheDocument();
      },
      { timeout: 2000 }
    );
  },
  render: (args) => ({
    components: { SegmentWaveformPreview },
    setup() {
      return { args };
    },
    template:
      '<div style="width: 500px;"><SegmentWaveformPreview v-bind="args" /></div>',
  }),
};

// Small segment (short duration)
export const SmallSegment: Story = {
  args: {
    startTime: 100,
    endTime: 105,
    duration: 300,
    waveformUrl: '/mock-waveform.dat',
    height: 40,
  },
  play: async ({ canvasElement }) => {
    await waitFor(
      () => {
        const canvasEl = canvasElement.querySelector('canvas');
        expect(canvasEl).toBeInTheDocument();
      },
      { timeout: 2000 }
    );
  },
  render: (args) => ({
    components: { SegmentWaveformPreview },
    setup() {
      return { args };
    },
    template:
      '<div style="width: 200px;"><SegmentWaveformPreview v-bind="args" /></div>',
  }),
};
