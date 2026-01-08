/**
 * Test Plan: EmptyScreen
 *
 * Scenario: Render empty screen with illustration slot
 *   Given the EmptyScreen component is rendered
 *   When an illustration slot is provided
 *   Then the illustration should be displayed
 *
 * Scenario: Render empty screen with text slot
 *   Given the EmptyScreen component is rendered
 *   When a text slot is provided
 *   Then the text should be displayed
 *
 * Scenario: Render empty screen with both slots
 *   Given the EmptyScreen component is rendered
 *   When both illustration and text slots are provided
 *   Then both should be displayed in correct order
 */

import type { Meta, StoryObj } from '@storybook/vue3';
import { within, expect } from '@storybook/test';
import EmptyScreen from './EmptyScreen.vue';

const meta: Meta<typeof EmptyScreen> = {
  title: 'Lib/Display/EmptyScreen',
  component: EmptyScreen,
  tags: ['autodocs'],
  parameters: {
    layout: 'fullscreen',
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

// Default empty screen with both slots
export const Default: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);

    await expect(canvas.getByText('📭')).toBeInTheDocument();
    await expect(canvas.getByText('No items found')).toBeInTheDocument();
  },
  render: () => ({
    components: { EmptyScreen },
    template: `
      <EmptyScreen>
        <template #illustration>📭</template>
        <template #text>No items found</template>
      </EmptyScreen>
    `,
  }),
};

// With illustration only
export const IllustrationOnly: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await expect(canvas.getByText('🔍')).toBeInTheDocument();
  },
  render: () => ({
    components: { EmptyScreen },
    template: `
      <EmptyScreen>
        <template #illustration>🔍</template>
      </EmptyScreen>
    `,
  }),
};

// With text only
export const TextOnly: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await expect(canvas.getByText('Nothing to display')).toBeInTheDocument();
  },
  render: () => ({
    components: { EmptyScreen },
    template: `
      <EmptyScreen>
        <template #text>Nothing to display</template>
      </EmptyScreen>
    `,
  }),
};

// No data state
export const NoDataState: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await expect(canvas.getByText('📊')).toBeInTheDocument();
    await expect(canvas.getByText('No data available')).toBeInTheDocument();
    await expect(canvas.getByText('Try adjusting your filters or adding new data.')).toBeInTheDocument();
  },
  render: () => ({
    components: { EmptyScreen },
    template: `
      <EmptyScreen>
        <template #illustration>📊</template>
        <template #text>
          <div>No data available</div>
          <div style="font-size: 0.8em; margin-top: 0.5rem;">Try adjusting your filters or adding new data.</div>
        </template>
      </EmptyScreen>
    `,
  }),
};

// Search results empty
export const SearchResultsEmpty: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await expect(canvas.getByText('🔎')).toBeInTheDocument();
    await expect(canvas.getByText('No results found')).toBeInTheDocument();
  },
  render: () => ({
    components: { EmptyScreen },
    template: `
      <EmptyScreen>
        <template #illustration>🔎</template>
        <template #text>No results found</template>
      </EmptyScreen>
    `,
  }),
};

// Error state
export const ErrorState: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await expect(canvas.getByText('⚠️')).toBeInTheDocument();
    await expect(canvas.getByText('Something went wrong')).toBeInTheDocument();
  },
  render: () => ({
    components: { EmptyScreen },
    template: `
      <EmptyScreen>
        <template #illustration>⚠️</template>
        <template #text>Something went wrong</template>
      </EmptyScreen>
    `,
  }),
};

// Loading complete, no items
export const LoadingCompleteNoItems: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await expect(canvas.getByText('📁')).toBeInTheDocument();
    await expect(canvas.getByText('This folder is empty')).toBeInTheDocument();
  },
  render: () => ({
    components: { EmptyScreen },
    template: `
      <EmptyScreen>
        <template #illustration>📁</template>
        <template #text>This folder is empty</template>
      </EmptyScreen>
    `,
  }),
};

// With SVG illustration
export const WithSVGIllustration: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    const svg = canvasElement.querySelector('svg');

    await expect(svg).toBeInTheDocument();
    await expect(canvas.getByText('Custom illustration')).toBeInTheDocument();
  },
  render: () => ({
    components: { EmptyScreen },
    template: `
      <EmptyScreen>
        <template #illustration>
          <svg width="100" height="100" viewBox="0 0 100 100">
            <circle cx="50" cy="50" r="40" fill="#e9d5ff" />
            <text x="50" y="55" text-anchor="middle" fill="#7c3aed">?</text>
          </svg>
        </template>
        <template #text>Custom illustration</template>
      </EmptyScreen>
    `,
  }),
};

// Recordings empty state (domain-specific)
export const RecordingsEmptyState: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await expect(canvas.getByText('🎙️')).toBeInTheDocument();
    await expect(canvas.getByText('No recordings yet')).toBeInTheDocument();
  },
  render: () => ({
    components: { EmptyScreen },
    template: `
      <EmptyScreen>
        <template #illustration>🎙️</template>
        <template #text>
          <div>No recordings yet</div>
          <div style="font-size: 0.8em; margin-top: 0.5rem;">Start recording to see your sessions here.</div>
        </template>
      </EmptyScreen>
    `,
  }),
};
