/**
 * Test Plan: Marker
 *
 * Scenario: Render marker with index
 *   Given the Marker component is rendered
 *   When an index prop is passed
 *   Then the badge should display the index value
 *
 * Scenario: Render marker with slot content
 *   Given the Marker component is rendered with slot content
 *   When the component mounts
 *   Then both the badge and slot content should be visible
 */

import type { Meta, StoryObj } from '@storybook/vue3';
import { within, expect } from '@storybook/test';
import Marker from './Marker.vue';

const meta: Meta<typeof Marker> = {
  title: 'Lib/Controls/Marker',
  component: Marker,
  tags: ['autodocs'],
  argTypes: {
    index: {
      control: 'text',
      description: 'Index displayed in the badge',
    },
    default: {
      control: 'text',
      description: 'Slot content displayed next to the badge',
    },
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

// Default marker
export const Default: Story = {
  args: {
    index: 'A',
  },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    const badge = canvas.getByText('A');

    await expect(badge).toBeInTheDocument();
    await expect(badge).toHaveClass('badge');
  },
  render: (args) => ({
    components: { Marker },
    setup() {
      return { args };
    },
    template: '<Marker :index="args.index" />',
  }),
};

// Marker with slot content
export const WithContent: Story = {
  args: {
    index: 'B',
    default: 'Marker content text',
  },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);

    await expect(canvas.getByText('B')).toBeInTheDocument();
    await expect(canvas.getByText('Marker content text')).toBeInTheDocument();
  },
  render: (args) => ({
    components: { Marker },
    setup() {
      return { args };
    },
    template: '<Marker :index="args.index">{{ args.default }}</Marker>',
  }),
};

// Numeric index
export const NumericIndex: Story = {
  args: {
    index: '1',
  },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await expect(canvas.getByText('1')).toBeInTheDocument();
  },
  render: (args) => ({
    components: { Marker },
    setup() {
      return { args };
    },
    template: '<Marker :index="args.index">First item</Marker>',
  }),
};

// Multiple markers
export const MultipleMarkers: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);

    await expect(canvas.getByText('A')).toBeInTheDocument();
    await expect(canvas.getByText('B')).toBeInTheDocument();
    await expect(canvas.getByText('C')).toBeInTheDocument();
    await expect(canvas.getByText('Start point')).toBeInTheDocument();
    await expect(canvas.getByText('Middle point')).toBeInTheDocument();
    await expect(canvas.getByText('End point')).toBeInTheDocument();
  },
  render: () => ({
    components: { Marker },
    template: `
      <div style="display: flex; flex-direction: column; gap: 1rem;">
        <Marker index="A">Start point</Marker>
        <Marker index="B">Middle point</Marker>
        <Marker index="C">End point</Marker>
      </div>
    `,
  }),
};

// Long index text
export const LongIndex: Story = {
  args: {
    index: '99',
  },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await expect(canvas.getByText('99')).toBeInTheDocument();
  },
  render: (args) => ({
    components: { Marker },
    setup() {
      return { args };
    },
    template: '<Marker :index="args.index">Item number 99</Marker>',
  }),
};

// Rich slot content
export const RichSlotContent: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);

    await expect(canvas.getByText('X')).toBeInTheDocument();
    await expect(canvas.getByText('Main Title')).toBeInTheDocument();
    await expect(canvas.getByText('Subtitle text')).toBeInTheDocument();
  },
  render: () => ({
    components: { Marker },
    template: `
      <Marker index="X">
        <div>
          <strong>Main Title</strong>
          <small style="display: block; color: gray;">Subtitle text</small>
        </div>
      </Marker>
    `,
  }),
};
