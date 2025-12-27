/**
 * Test Plan: Checkbox
 *
 * Scenario: Render unchecked checkbox
 *   Given the Checkbox component is rendered
 *   When modelValue is false
 *   Then the checkbox should appear unchecked
 *   And no check icon should be visible
 *
 * Scenario: Render checked checkbox
 *   Given the Checkbox component is rendered
 *   When modelValue is true
 *   Then the checkbox should appear checked
 *   And a check icon should be visible
 *
 * Scenario: Toggle checkbox on click
 *   Given an unchecked checkbox
 *   When the user clicks the checkbox
 *   Then update:modelValue should emit with true
 *
 * Scenario: Render indeterminate state
 *   Given the Checkbox component with indeterminate=true
 *   When the component renders
 *   Then a minus icon should be visible instead of check
 *
 * Scenario: Disabled checkbox
 *   Given a disabled checkbox
 *   When the user tries to click
 *   Then the checkbox state should not change
 *
 * Scenario: Checkbox with label
 *   Given a checkbox with slot content
 *   When the component renders
 *   Then the label should be visible next to the checkbox
 *
 * Scenario: Size variants
 *   Given different size props
 *   When the component renders
 *   Then the checkbox should have appropriate sizing
 */

import type { Meta, StoryObj } from '@storybook/vue3';
import { userEvent, within, expect, fn } from '@storybook/test';
import { ref } from 'vue';
import Checkbox from './Checkbox.vue';

const meta: Meta<typeof Checkbox> = {
  title: 'Lib/Controls/Checkbox',
  component: Checkbox,
  tags: ['autodocs'],
  argTypes: {
    modelValue: {
      control: 'boolean',
      description: 'Whether the checkbox is checked',
    },
    disabled: {
      control: 'boolean',
      description: 'Whether the checkbox is disabled',
    },
    indeterminate: {
      control: 'boolean',
      description: 'Whether the checkbox shows indeterminate state',
    },
    size: {
      control: 'select',
      options: ['sm', 'md'],
      description: 'Size of the checkbox',
    },
    default: {
      control: 'text',
      description: 'Label content via slot',
    },
  },
  args: {
    'onUpdate:modelValue': fn(),
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

// Default unchecked
export const Default: Story = {
  args: {
    modelValue: false,
  },
  play: async ({ canvasElement }) => {
    const checkbox = canvasElement.querySelector(
      'input[type="checkbox"]'
    ) as HTMLInputElement;

    await expect(checkbox).toBeInTheDocument();
    await expect(checkbox.checked).toBe(false);

    // Should not have checked class
    const label = canvasElement.querySelector('.checkbox');
    await expect(label).not.toHaveClass('is-checked');
  },
  render: (args) => ({
    components: { Checkbox },
    setup() {
      const checked = ref(args.modelValue);
      return { checked, args };
    },
    template: '<Checkbox v-model="checked" />',
  }),
};

// Checked state
export const Checked: Story = {
  args: {
    modelValue: true,
  },
  play: async ({ canvasElement }) => {
    const checkbox = canvasElement.querySelector(
      'input[type="checkbox"]'
    ) as HTMLInputElement;

    await expect(checkbox.checked).toBe(true);

    // Should have checked class
    const label = canvasElement.querySelector('.checkbox');
    await expect(label).toHaveClass('is-checked');

    // Check icon should be visible
    const icon = canvasElement.querySelector('.checkbox__icon');
    await expect(icon).toBeInTheDocument();
  },
  render: (args) => ({
    components: { Checkbox },
    setup() {
      const checked = ref(args.modelValue);
      return { checked };
    },
    template: '<Checkbox v-model="checked" />',
  }),
};

// Toggle interaction
export const ToggleInteraction: Story = {
  args: {
    modelValue: false,
  },
  play: async ({ canvasElement }) => {
    const checkbox = canvasElement.querySelector(
      'input[type="checkbox"]'
    ) as HTMLInputElement;
    const label = canvasElement.querySelector('.checkbox');

    // Initially unchecked
    await expect(checkbox.checked).toBe(false);
    await expect(label).not.toHaveClass('is-checked');

    // Click to check
    await userEvent.click(checkbox);
    await expect(checkbox.checked).toBe(true);
    await expect(label).toHaveClass('is-checked');

    // Click to uncheck
    await userEvent.click(checkbox);
    await expect(checkbox.checked).toBe(false);
    await expect(label).not.toHaveClass('is-checked');
  },
  render: (args) => ({
    components: { Checkbox },
    setup() {
      const checked = ref(args.modelValue);
      return { checked };
    },
    template: '<Checkbox v-model="checked" />',
  }),
};

// With label
export const WithLabel: Story = {
  args: {
    modelValue: false,
    default: 'Accept terms and conditions',
  },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);

    await expect(
      canvas.getByText('Accept terms and conditions')
    ).toBeInTheDocument();

    // Clicking label should toggle checkbox
    const label = canvas.getByText('Accept terms and conditions');
    const checkbox = canvasElement.querySelector(
      'input[type="checkbox"]'
    ) as HTMLInputElement;

    await userEvent.click(label);
    await expect(checkbox.checked).toBe(true);
  },
  render: (args) => ({
    components: { Checkbox },
    setup() {
      const checked = ref(args.modelValue);
      return { checked, args };
    },
    template: '<Checkbox v-model="checked">{{ args.default }}</Checkbox>',
  }),
};

// Indeterminate state
export const Indeterminate: Story = {
  args: {
    modelValue: false,
    indeterminate: true,
  },
  play: async ({ canvasElement }) => {
    const label = canvasElement.querySelector('.checkbox');
    await expect(label).toHaveClass('is-indeterminate');

    // Minus icon should be visible
    const icon = canvasElement.querySelector('.checkbox__icon');
    await expect(icon).toBeInTheDocument();
  },
  render: (args) => ({
    components: { Checkbox },
    setup() {
      const checked = ref(args.modelValue);
      return { checked, args };
    },
    template: '<Checkbox v-model="checked" :indeterminate="args.indeterminate" />',
  }),
};

// Disabled unchecked
export const DisabledUnchecked: Story = {
  args: {
    modelValue: false,
    disabled: true,
  },
  play: async ({ canvasElement }) => {
    const checkbox = canvasElement.querySelector(
      'input[type="checkbox"]'
    ) as HTMLInputElement;
    const label = canvasElement.querySelector('.checkbox');

    await expect(checkbox.disabled).toBe(true);
    await expect(label).toHaveClass('is-disabled');

    // Try to click - should not change
    await userEvent.click(checkbox);
    await expect(checkbox.checked).toBe(false);
  },
  render: (args) => ({
    components: { Checkbox },
    setup() {
      const checked = ref(args.modelValue);
      return { checked, args };
    },
    template: '<Checkbox v-model="checked" :disabled="args.disabled" />',
  }),
};

// Disabled checked
export const DisabledChecked: Story = {
  args: {
    modelValue: true,
    disabled: true,
  },
  play: async ({ canvasElement }) => {
    const checkbox = canvasElement.querySelector(
      'input[type="checkbox"]'
    ) as HTMLInputElement;

    await expect(checkbox.disabled).toBe(true);
    await expect(checkbox.checked).toBe(true);
  },
  render: (args) => ({
    components: { Checkbox },
    setup() {
      const checked = ref(args.modelValue);
      return { checked, args };
    },
    template: '<Checkbox v-model="checked" :disabled="args.disabled" />',
  }),
};

// Small size
export const SmallSize: Story = {
  args: {
    modelValue: true,
    size: 'sm',
  },
  play: async ({ canvasElement }) => {
    const label = canvasElement.querySelector('.checkbox');
    await expect(label).toHaveClass('is-sm');
  },
  render: (args) => ({
    components: { Checkbox },
    setup() {
      const checked = ref(args.modelValue);
      return { checked, args };
    },
    template: '<Checkbox v-model="checked" :size="args.size">Small checkbox</Checkbox>',
  }),
};

// Medium size (default)
export const MediumSize: Story = {
  args: {
    modelValue: true,
    size: 'md',
  },
  play: async ({ canvasElement }) => {
    const label = canvasElement.querySelector('.checkbox');
    await expect(label).toHaveClass('is-md');
  },
  render: (args) => ({
    components: { Checkbox },
    setup() {
      const checked = ref(args.modelValue);
      return { checked, args };
    },
    template: '<Checkbox v-model="checked" :size="args.size">Medium checkbox</Checkbox>',
  }),
};

// All states
export const AllStates: Story = {
  play: async ({ canvasElement }) => {
    const checkboxes = canvasElement.querySelectorAll('input[type="checkbox"]');
    await expect(checkboxes.length).toBe(6);
  },
  render: () => ({
    components: { Checkbox },
    setup() {
      const states = {
        unchecked: ref(false),
        checked: ref(true),
        indeterminate: ref(false),
        disabledUnchecked: ref(false),
        disabledChecked: ref(true),
        disabledIndeterminate: ref(false),
      };
      return states;
    },
    template: `
      <div style="display: flex; flex-direction: column; gap: 1rem;">
        <Checkbox v-model="unchecked">Unchecked</Checkbox>
        <Checkbox v-model="checked">Checked</Checkbox>
        <Checkbox v-model="indeterminate" indeterminate>Indeterminate</Checkbox>
        <Checkbox v-model="disabledUnchecked" disabled>Disabled unchecked</Checkbox>
        <Checkbox v-model="disabledChecked" disabled>Disabled checked</Checkbox>
        <Checkbox v-model="disabledIndeterminate" disabled indeterminate>Disabled indeterminate</Checkbox>
      </div>
    `,
  }),
};

// Size comparison
export const SizeComparison: Story = {
  play: async ({ canvasElement }) => {
    const labels = canvasElement.querySelectorAll('.checkbox');
    await expect(labels[0]).toHaveClass('is-sm');
    await expect(labels[1]).toHaveClass('is-md');
  },
  render: () => ({
    components: { Checkbox },
    setup() {
      const sm = ref(true);
      const md = ref(true);
      return { sm, md };
    },
    template: `
      <div style="display: flex; flex-direction: column; gap: 1rem;">
        <Checkbox v-model="sm" size="sm">Small (sm)</Checkbox>
        <Checkbox v-model="md" size="md">Medium (md)</Checkbox>
      </div>
    `,
  }),
};

// Keyboard navigation
export const KeyboardNavigation: Story = {
  args: {
    modelValue: false,
  },
  play: async ({ canvasElement }) => {
    const checkbox = canvasElement.querySelector(
      'input[type="checkbox"]'
    ) as HTMLInputElement;

    // Focus the checkbox
    checkbox.focus();
    await expect(checkbox).toHaveFocus();

    // Press Space to toggle
    await userEvent.keyboard(' ');
    await expect(checkbox.checked).toBe(true);

    // Press Space again to untoggle
    await userEvent.keyboard(' ');
    await expect(checkbox.checked).toBe(false);
  },
  render: (args) => ({
    components: { Checkbox },
    setup() {
      const checked = ref(args.modelValue);
      return { checked };
    },
    template: '<Checkbox v-model="checked">Press Space to toggle</Checkbox>',
  }),
};
