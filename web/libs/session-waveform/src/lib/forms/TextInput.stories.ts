/**
 * Test Plan: TextInput
 *
 * Scenario: Render text input with v-model binding
 *   Given the TextInput component is rendered
 *   When the user types in the input
 *   Then the model value should update
 *
 * Scenario: Size variants
 *   Given different size props are passed
 *   When the component renders
 *   Then the input should have the appropriate size class
 *
 * Scenario: Variant styles
 *   Given different variant props are passed
 *   When the component renders
 *   Then the input should have the appropriate variant class
 *
 * Scenario: Slot composition
 *   Given slots are provided (prepend, actions, append)
 *   When the component renders
 *   Then the slot content should be visible in correct positions
 *
 * Scenario: Focus and blur events
 *   Given the TextInput is rendered
 *   When the user focuses and blurs the input
 *   Then the appropriate visual states should change
 */

import type { Meta, StoryObj } from '@storybook/vue3';
import { ref } from 'vue';
import { userEvent, within, expect } from '@storybook/test';
import TextInput from './TextInput.vue';

const meta: Meta<typeof TextInput> = {
  title: 'Lib/Forms/TextInput',
  component: TextInput,
  tags: ['autodocs'],
  argTypes: {
    size: {
      control: 'select',
      options: ['xs', 'sm', 'md', 'lg'],
      description: 'Input size',
    },
    variant: {
      control: 'select',
      options: ['ghost', 'outlined'],
      description: 'Input variant style',
    },
    disableReset: {
      control: 'boolean',
      description: 'Disable reset functionality',
    },
    placeholder: {
      control: 'text',
      description: 'Input placeholder text',
    },
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

// Default text input
export const Default: Story = {
  args: {
    placeholder: 'Enter text...',
  },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    const input = canvas.getByRole('textbox');

    await expect(input).toBeInTheDocument();
    await expect(input).toHaveAttribute('placeholder', 'Enter text...');

    // Type in the input
    await userEvent.type(input, 'Hello World');
    await expect(input).toHaveValue('Hello World');
  },
  render: (args) => ({
    components: { TextInput },
    setup() {
      const model = ref('');
      return { args, model };
    },
    template: '<TextInput v-model="model" v-bind="args" />',
  }),
};

// All sizes
export const Sizes: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    const inputs = canvas.getAllByRole('textbox');

    await expect(inputs).toHaveLength(4);

    // Verify all inputs are rendered
    for (const input of inputs) {
      await expect(input).toBeInTheDocument();
    }
  },
  render: () => ({
    components: { TextInput },
    setup() {
      const models = {
        xs: ref(''),
        sm: ref(''),
        md: ref(''),
        lg: ref(''),
      };
      return { models };
    },
    template: `
      <div style="display: flex; flex-direction: column; gap: 1rem;">
        <TextInput v-model="models.xs" size="xs" placeholder="Extra Small" />
        <TextInput v-model="models.sm" size="sm" placeholder="Small" />
        <TextInput v-model="models.md" size="md" placeholder="Medium" />
        <TextInput v-model="models.lg" size="lg" placeholder="Large" />
      </div>
    `,
  }),
};

// Variants
export const Variants: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    const inputs = canvas.getAllByRole('textbox');

    await expect(inputs).toHaveLength(2);
  },
  render: () => ({
    components: { TextInput },
    setup() {
      const models = {
        ghost: ref(''),
        outlined: ref(''),
      };
      return { models };
    },
    template: `
      <div style="display: flex; flex-direction: column; gap: 1rem;">
        <TextInput v-model="models.ghost" variant="ghost" placeholder="Ghost variant" />
        <TextInput v-model="models.outlined" variant="outlined" placeholder="Outlined variant" />
      </div>
    `,
  }),
};

// With prepend slot
export const WithPrependSlot: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    const input = canvas.getByRole('textbox');
    const prependText = canvas.getByText('$');

    await expect(prependText).toBeInTheDocument();
    await expect(input).toBeInTheDocument();

    await userEvent.type(input, '100');
    await expect(input).toHaveValue('100');
  },
  render: () => ({
    components: { TextInput },
    setup() {
      const model = ref('');
      return { model };
    },
    template: `
      <TextInput v-model="model" placeholder="Amount">
        <template #prepend>$</template>
      </TextInput>
    `,
  }),
};

// With append slot
export const WithAppendSlot: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    const input = canvas.getByRole('textbox');
    const appendText = canvas.getByText('kg');

    await expect(appendText).toBeInTheDocument();
    await expect(input).toBeInTheDocument();
  },
  render: () => ({
    components: { TextInput },
    setup() {
      const model = ref('');
      return { model };
    },
    template: `
      <TextInput v-model="model" placeholder="Weight">
        <template #append>kg</template>
      </TextInput>
    `,
  }),
};

// With actions slot
export const WithActionsSlot: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    const input = canvas.getByRole('textbox');
    const clearButton = canvas.getByRole('button', { name: /clear/i });

    await userEvent.type(input, 'Some text');
    await expect(input).toHaveValue('Some text');

    await userEvent.click(clearButton);
  },
  render: () => ({
    components: { TextInput },
    setup() {
      const model = ref('');
      const clear = () => {
        model.value = '';
      };
      return { model, clear };
    },
    template: `
      <TextInput v-model="model" placeholder="Type something">
        <template #actions="{ resetProps }">
          <button
            type="button"
            @click="clear"
            style="padding: 0 0.5rem; cursor: pointer; border: none; background: none;"
          >
            Clear
          </button>
        </template>
      </TextInput>
    `,
  }),
};

// Combined slots
export const CombinedSlots: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    const input = canvas.getByRole('textbox');

    await expect(canvas.getByText('@')).toBeInTheDocument();
    await expect(canvas.getByText('.com')).toBeInTheDocument();

    await userEvent.type(input, 'username');
    await expect(input).toHaveValue('username');
  },
  render: () => ({
    components: { TextInput },
    setup() {
      const model = ref('');
      return { model };
    },
    template: `
      <TextInput v-model="model" placeholder="username">
        <template #prepend>@</template>
        <template #append>.com</template>
      </TextInput>
    `,
  }),
};

// Focus and blur
export const FocusAndBlur: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    const input = canvas.getByRole('textbox');

    // Focus the input
    await userEvent.click(input);
    await expect(input).toHaveFocus();

    // Type something
    await userEvent.type(input, 'Test');
    await expect(input).toHaveValue('Test');

    // Tab away (blur)
    await userEvent.tab();
    await expect(input).not.toHaveFocus();
  },
  render: () => ({
    components: { TextInput },
    setup() {
      const model = ref('');
      return { model };
    },
    template: `
      <div>
        <TextInput v-model="model" placeholder="Click me" />
        <button style="margin-top: 1rem;">Another element</button>
      </div>
    `,
  }),
};

// Number input type
export const NumberInput: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    const input = canvas.getByRole('spinbutton');

    await userEvent.type(input, '42');
    await expect(input).toHaveValue(42);
  },
  render: () => ({
    components: { TextInput },
    setup() {
      const model = ref<number>();
      return { model };
    },
    template: '<TextInput v-model="model" type="number" placeholder="Enter number" />',
  }),
};

// Pre-filled value
export const PreFilledValue: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    const input = canvas.getByRole('textbox');

    await expect(input).toHaveValue('Initial value');

    // Clear and type new value
    await userEvent.clear(input);
    await userEvent.type(input, 'New value');
    await expect(input).toHaveValue('New value');
  },
  render: () => ({
    components: { TextInput },
    setup() {
      const model = ref('Initial value');
      return { model };
    },
    template: '<TextInput v-model="model" />',
  }),
};

// Readonly state
export const Readonly: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    const input = canvas.getByRole('textbox');

    await expect(input).toHaveAttribute('readonly');
    await expect(input).toHaveValue('Cannot edit this');
  },
  render: () => ({
    components: { TextInput },
    setup() {
      const model = ref('Cannot edit this');
      return { model };
    },
    template: '<TextInput v-model="model" readonly />',
  }),
};

// Disabled state
export const Disabled: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    const input = canvas.getByRole('textbox');

    await expect(input).toBeDisabled();
  },
  render: () => ({
    components: { TextInput },
    setup() {
      const model = ref('Disabled input');
      return { model };
    },
    template: '<TextInput v-model="model" disabled />',
  }),
};
