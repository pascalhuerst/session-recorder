/**
 * Test Plan: Button
 *
 * Scenario: Render button with different variants
 *   Given the Button component is rendered
 *   When different props are passed (color, variant, shape, size)
 *   Then the button should display with correct CSS classes
 *   And the button should be clickable
 *
 * Scenario: Button loading state
 *   Given the Button component has isLoading=true
 *   When rendered
 *   Then the button should display loading indicator
 *
 * Scenario: Button disabled state
 *   Given the Button component is disabled
 *   When the user tries to click
 *   Then the click handler should not be called
 *
 * Scenario: Button as anchor tag
 *   Given the Button component has tagName="a"
 *   When rendered
 *   Then it should render as an anchor element
 */

import type { Meta, StoryObj } from '@storybook/vue3';
import { userEvent, within, expect, fn } from '@storybook/test';
import Button from './Button.vue';

const meta: Meta<typeof Button> = {
  title: 'Lib/Controls/Button',
  component: Button,
  tags: ['autodocs'],
  argTypes: {
    color: {
      control: 'select',
      options: ['primary', 'neutral'],
      description: 'Button color scheme',
    },
    variant: {
      control: 'select',
      options: ['ghost', 'solid', 'outlined'],
      description: 'Button variant style',
    },
    shape: {
      control: 'select',
      options: ['normal', 'circle', 'square'],
      description: 'Button shape',
    },
    size: {
      control: 'select',
      options: ['lg', 'md', 'sm', 'xs'],
      description: 'Button size',
    },
    isLoading: {
      control: 'boolean',
      description: 'Shows loading spinner',
    },
    tagName: {
      control: 'text',
      description: 'HTML tag to render (button, a)',
    },
    default: {
      control: 'text',
      description: 'Button content',
    },
  },
  args: {
    onClick: fn(),
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

// Default button
export const Default: Story = {
  args: {
    default: 'Button',
  },
  play: async ({ canvasElement, args }) => {
    const canvas = within(canvasElement);
    const button = canvas.getByRole('button');

    await expect(button).toBeInTheDocument();
    await expect(button).toHaveTextContent('Button');
    await expect(button).toHaveClass('button', 'is-ghost', 'is-neutral', 'is-md', 'is-normal');

    await userEvent.click(button);
    await expect(args.onClick).toHaveBeenCalledOnce();
  },
  render: (args) => ({
    components: { Button },
    setup() {
      return { args };
    },
    template: '<Button v-bind="args" @click="args.onClick">{{ args.default }}</Button>',
  }),
};

// Primary solid button
export const PrimarySolid: Story = {
  args: {
    color: 'primary',
    variant: 'solid',
    default: 'Primary Action',
  },
  play: async ({ canvasElement, args }) => {
    const canvas = within(canvasElement);
    const button = canvas.getByRole('button');

    await expect(button).toHaveClass('is-primary', 'is-solid');
    await userEvent.click(button);
    await expect(args.onClick).toHaveBeenCalled();
  },
  render: (args) => ({
    components: { Button },
    setup() {
      return { args };
    },
    template: '<Button v-bind="args" @click="args.onClick">{{ args.default }}</Button>',
  }),
};

// Primary outlined button
export const PrimaryOutlined: Story = {
  args: {
    color: 'primary',
    variant: 'outlined',
    default: 'Outlined',
  },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    const button = canvas.getByRole('button');

    await expect(button).toHaveClass('is-primary', 'is-outlined');
  },
  render: (args) => ({
    components: { Button },
    setup() {
      return { args };
    },
    template: '<Button v-bind="args">{{ args.default }}</Button>',
  }),
};

// All sizes
export const Sizes: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    const buttons = canvas.getAllByRole('button');

    await expect(buttons).toHaveLength(4);
    await expect(buttons[0]).toHaveClass('is-xs');
    await expect(buttons[1]).toHaveClass('is-sm');
    await expect(buttons[2]).toHaveClass('is-md');
    await expect(buttons[3]).toHaveClass('is-lg');
  },
  render: () => ({
    components: { Button },
    template: `
      <div style="display: flex; gap: 1rem; align-items: center;">
        <Button size="xs">Extra Small</Button>
        <Button size="sm">Small</Button>
        <Button size="md">Medium</Button>
        <Button size="lg">Large</Button>
      </div>
    `,
  }),
};

// All variants
export const Variants: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    const buttons = canvas.getAllByRole('button');

    await expect(buttons).toHaveLength(3);
    await expect(buttons[0]).toHaveClass('is-ghost');
    await expect(buttons[1]).toHaveClass('is-outlined');
    await expect(buttons[2]).toHaveClass('is-solid');
  },
  render: () => ({
    components: { Button },
    template: `
      <div style="display: flex; gap: 1rem; align-items: center;">
        <Button variant="ghost">Ghost</Button>
        <Button variant="outlined">Outlined</Button>
        <Button variant="solid">Solid</Button>
      </div>
    `,
  }),
};

// Shape variations
export const Shapes: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    const buttons = canvas.getAllByRole('button');

    await expect(buttons).toHaveLength(3);
    await expect(buttons[0]).toHaveClass('is-normal');
    await expect(buttons[1]).toHaveClass('is-circle');
    await expect(buttons[2]).toHaveClass('is-square');
  },
  render: () => ({
    components: { Button },
    template: `
      <div style="display: flex; gap: 1rem; align-items: center;">
        <Button shape="normal">Normal</Button>
        <Button shape="circle">O</Button>
        <Button shape="square">X</Button>
      </div>
    `,
  }),
};

// Loading state
export const Loading: Story = {
  args: {
    isLoading: true,
    default: 'Loading...',
  },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    const button = canvas.getByRole('button');

    await expect(button).toHaveClass('is-loading');
  },
  render: (args) => ({
    components: { Button },
    setup() {
      return { args };
    },
    template: '<Button v-bind="args">{{ args.default }}</Button>',
  }),
};

// Disabled state
export const Disabled: Story = {
  args: {
    default: 'Disabled',
  },
  play: async ({ canvasElement, args }) => {
    const canvas = within(canvasElement);
    const button = canvas.getByRole('button');

    await expect(button).toBeDisabled();

    // Click should not trigger handler
    await userEvent.click(button);
    await expect(args.onClick).not.toHaveBeenCalled();
  },
  render: (args) => ({
    components: { Button },
    setup() {
      return { args };
    },
    template: '<Button v-bind="args" disabled @click="args.onClick">{{ args.default }}</Button>',
  }),
};

// As anchor tag
export const AsAnchor: Story = {
  args: {
    tagName: 'a',
    default: 'Link Button',
  },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    const anchor = canvas.getByRole('link');

    await expect(anchor).toBeInTheDocument();
    await expect(anchor.tagName.toLowerCase()).toBe('a');
  },
  render: (args) => ({
    components: { Button },
    setup() {
      return { args };
    },
    template: '<Button v-bind="args" href="#">{{ args.default }}</Button>',
  }),
};

// Keyboard navigation
export const KeyboardNavigation: Story = {
  args: {
    default: 'Press Enter',
  },
  play: async ({ canvasElement, args }) => {
    const canvas = within(canvasElement);
    const button = canvas.getByRole('button');

    // Focus the button
    button.focus();
    await expect(button).toHaveFocus();

    // Press Enter
    await userEvent.keyboard('{Enter}');
    await expect(args.onClick).toHaveBeenCalled();

    // Reset and test Space
    (args.onClick as ReturnType<typeof fn>).mockClear();
    await userEvent.keyboard(' ');
    await expect(args.onClick).toHaveBeenCalled();
  },
  render: (args) => ({
    components: { Button },
    setup() {
      return { args };
    },
    template: '<Button v-bind="args" @click="args.onClick">{{ args.default }}</Button>',
  }),
};

// Color matrix
export const ColorMatrix: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    const buttons = canvas.getAllByRole('button');

    await expect(buttons).toHaveLength(6);
  },
  render: () => ({
    components: { Button },
    template: `
      <div style="display: grid; grid-template-columns: repeat(3, 1fr); gap: 1rem;">
        <Button color="neutral" variant="ghost">Neutral Ghost</Button>
        <Button color="neutral" variant="outlined">Neutral Outlined</Button>
        <Button color="neutral" variant="solid">Neutral Solid</Button>
        <Button color="primary" variant="ghost">Primary Ghost</Button>
        <Button color="primary" variant="outlined">Primary Outlined</Button>
        <Button color="primary" variant="solid">Primary Solid</Button>
      </div>
    `,
  }),
};
