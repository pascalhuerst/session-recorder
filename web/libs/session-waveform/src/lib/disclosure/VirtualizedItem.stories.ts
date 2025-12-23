/**
 * Test Plan: VirtualizedItem
 *
 * Scenario: Render content when visible
 *   Given the VirtualizedItem is in viewport
 *   When observed by IntersectionObserver
 *   Then the slot content should render
 *
 * Scenario: Hide content when not visible
 *   Given the VirtualizedItem is scrolled out of viewport
 *   When IntersectionObserver detects it's not visible
 *   Then the slot content should not render
 *   And the minHeight placeholder should remain
 *
 * Scenario: Custom component type
 *   Given an 'as' prop is provided
 *   When the component renders
 *   Then it should use the specified element type
 *
 * Scenario: Preload margin
 *   Given a preloadMargin prop is provided
 *   When items approach the viewport
 *   Then they should render before fully entering viewport
 */

import type { Meta, StoryObj } from '@storybook/vue3';
import { within, expect } from '@storybook/test';
import VirtualizedItem from './VirtualizedItem.vue';

const meta: Meta<typeof VirtualizedItem> = {
  title: 'Lib/Disclosure/VirtualizedItem',
  component: VirtualizedItem,
  tags: ['autodocs'],
  parameters: {
    layout: 'fullscreen',
  },
  argTypes: {
    as: {
      control: 'text',
      description: 'HTML element or component to render as',
    },
    minHeight: {
      control: 'number',
      description: 'Minimum height placeholder when content is not visible',
    },
    preloadMargin: {
      control: 'number',
      description: 'Margin for preloading before element enters viewport',
    },
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

// Single visible item
export const Default: Story = {
  args: {
    minHeight: 100,
  },
  play: async ({ canvasElement }) => {
    // Check component renders with minHeight style
    const wrapper = canvasElement.querySelector('[style*="min-height"]');
    await expect(wrapper).toBeInTheDocument();
  },
  render: (args) => ({
    components: { VirtualizedItem },
    setup() {
      return { args };
    },
    template: `
      <VirtualizedItem v-bind="args">
        <div style="padding: 1rem; background: #e9d5ff; border-radius: 8px;">
          Visible content - this renders when in viewport
        </div>
      </VirtualizedItem>
    `,
  }),
};

// List of virtualized items
export const VirtualizedList: Story = {
  play: async ({ canvasElement }) => {
    // Container should exist with proper structure
    const container = canvasElement.querySelector('[style*="overflow-y: auto"]');
    await expect(container).toBeInTheDocument();
  },
  render: () => ({
    components: { VirtualizedItem },
    setup() {
      const items = Array.from({ length: 50 }, (_, i) => ({
        id: i + 1,
        title: `Item ${i + 1}`,
        description: `Description for item ${i + 1}`,
      }));
      return { items };
    },
    template: `
      <div style="height: 400px; overflow-y: auto; border: 1px solid #ccc;">
        <VirtualizedItem
          v-for="item in items"
          :key="item.id"
          :minHeight="80"
          :preloadMargin="100"
        >
          <div style="padding: 1rem; border-bottom: 1px solid #eee;">
            <h4 style="margin: 0 0 0.5rem 0;">{{ item.title }}</h4>
            <p style="margin: 0; color: #666;">{{ item.description }}</p>
          </div>
        </VirtualizedItem>
      </div>
    `,
  }),
};

// Custom element type
export const AsListItem: Story = {
  args: {
    as: 'li',
    minHeight: 50,
  },
  play: async ({ canvasElement }) => {
    const listItems = canvasElement.querySelectorAll('li');
    await expect(listItems.length).toBeGreaterThan(0);
  },
  render: () => ({
    components: { VirtualizedItem },
    setup() {
      const items = ['First', 'Second', 'Third', 'Fourth', 'Fifth'];
      return { items };
    },
    template: `
      <ul style="list-style: none; padding: 0; margin: 0;">
        <VirtualizedItem
          v-for="(item, index) in items"
          :key="index"
          as="li"
          :minHeight="50"
        >
          <div style="padding: 1rem; border-bottom: 1px solid #eee;">
            {{ item }} item
          </div>
        </VirtualizedItem>
      </ul>
    `,
  }),
};

// As section element
export const AsSection: Story = {
  args: {
    as: 'section',
    minHeight: 200,
  },
  play: async ({ canvasElement }) => {
    const section = canvasElement.querySelector('section');
    await expect(section).toBeInTheDocument();
    await expect(section).toHaveStyle({ minHeight: '200px' });
  },
  render: (args) => ({
    components: { VirtualizedItem },
    setup() {
      return { args };
    },
    template: `
      <VirtualizedItem v-bind="args">
        <div style="padding: 2rem; background: #f3f4f6; border-radius: 8px;">
          <h2>Section Content</h2>
          <p>This is rendered inside a section element.</p>
        </div>
      </VirtualizedItem>
    `,
  }),
};

// With preload margin
export const WithPreloadMargin: Story = {
  args: {
    minHeight: 100,
    preloadMargin: 200,
  },
  play: async ({ canvasElement }) => {
    // Container should exist
    const container = canvasElement.querySelector('[style*="overflow-y: auto"]');
    await expect(container).toBeInTheDocument();
  },
  render: () => ({
    components: { VirtualizedItem },
    setup() {
      const items = Array.from({ length: 20 }, (_, i) => `Item ${i + 1}`);
      return { items };
    },
    template: `
      <div style="height: 300px; overflow-y: auto; border: 1px solid #ccc;">
        <VirtualizedItem
          v-for="(item, index) in items"
          :key="index"
          :minHeight="100"
          :preloadMargin="200"
        >
          <div style="padding: 1rem; background: #ddd5f9; margin: 0.5rem; border-radius: 4px;">
            {{ item }} (preloads 200px before viewport)
          </div>
        </VirtualizedItem>
      </div>
    `,
  }),
};

// Heavy content items
export const HeavyContentItems: Story = {
  play: async ({ canvasElement }) => {
    // Container should exist
    const container = canvasElement.querySelector('[style*="overflow-y: auto"]');
    await expect(container).toBeInTheDocument();
  },
  render: () => ({
    components: { VirtualizedItem },
    setup() {
      const cards = Array.from({ length: 30 }, (_, i) => ({
        id: i + 1,
        title: `Card ${i + 1}`,
        image: `https://picsum.photos/seed/${i}/300/200`,
        description: `This is the description for card ${i + 1}. It contains some longer text to simulate real content.`,
      }));
      return { cards };
    },
    template: `
      <div style="height: 500px; overflow-y: auto; padding: 1rem; background: #f5f5f5;">
        <VirtualizedItem
          v-for="card in cards"
          :key="card.id"
          :minHeight="280"
          :preloadMargin="100"
        >
          <div style="background: white; border-radius: 8px; margin-bottom: 1rem; overflow: hidden; box-shadow: 0 2px 4px rgba(0,0,0,0.1);">
            <div style="height: 150px; background: #e9d5ff; display: flex; align-items: center; justify-content: center;">
              {{ card.title }}
            </div>
            <div style="padding: 1rem;">
              <h3 style="margin: 0 0 0.5rem 0;">{{ card.title }}</h3>
              <p style="margin: 0; color: #666; font-size: 0.875rem;">{{ card.description }}</p>
            </div>
          </div>
        </VirtualizedItem>
      </div>
    `,
  }),
};

// Grid layout
export const GridLayout: Story = {
  play: async ({ canvasElement }) => {
    // Grid container should exist
    const grid = canvasElement.querySelector('[style*="display: grid"]');
    await expect(grid).toBeInTheDocument();
  },
  render: () => ({
    components: { VirtualizedItem },
    setup() {
      const items = Array.from({ length: 24 }, (_, i) => `Grid Item ${i + 1}`);
      return { items };
    },
    template: `
      <div style="height: 400px; overflow-y: auto; padding: 1rem;">
        <div style="display: grid; grid-template-columns: repeat(3, 1fr); gap: 1rem;">
          <VirtualizedItem
            v-for="(item, index) in items"
            :key="index"
            :minHeight="120"
            :preloadMargin="50"
          >
            <div style="padding: 1rem; background: #ddd5f9; border-radius: 8px; text-align: center;">
              {{ item }}
            </div>
          </VirtualizedItem>
        </div>
      </div>
    `,
  }),
};

// Placeholder visualization
export const PlaceholderDemo: Story = {
  args: {
    minHeight: 150,
  },
  play: async ({ canvasElement }) => {
    const items = canvasElement.querySelectorAll('[style*="min-height: 150px"]');
    await expect(items.length).toBeGreaterThan(0);
  },
  render: () => ({
    components: { VirtualizedItem },
    setup() {
      const items = Array.from({ length: 10 }, (_, i) => `Content ${i + 1}`);
      return { items };
    },
    template: `
      <div style="height: 300px; overflow-y: auto; border: 2px dashed #ccc; padding: 0.5rem;">
        <p style="margin: 0 0 1rem 0; font-size: 0.875rem; color: #666;">
          Scroll down to see items render lazily (minHeight: 150px placeholder)
        </p>
        <VirtualizedItem
          v-for="(item, index) in items"
          :key="index"
          :minHeight="150"
          style="border: 1px dashed #aaa; margin-bottom: 0.5rem;"
        >
          <div style="padding: 2rem; background: #e0f2fe; border-radius: 4px;">
            <h4>{{ item }}</h4>
            <p>This content only renders when in viewport</p>
          </div>
        </VirtualizedItem>
      </div>
    `,
  }),
};
