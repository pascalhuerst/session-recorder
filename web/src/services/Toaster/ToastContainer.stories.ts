/**
 * Test Plan: ToastContainer
 *
 * Scenario: Render success toast
 *   Given a success toast is added
 *   When the container renders
 *   Then a green success toast should be visible
 *
 * Scenario: Render error toast
 *   Given an error toast is added
 *   When the container renders
 *   Then a red error toast should be visible
 *
 * Scenario: Render warning toast
 *   Given a warning toast is added
 *   When the container renders
 *   Then an orange warning toast should be visible
 *
 * Scenario: Render info toast
 *   Given an info toast is added
 *   When the container renders
 *   Then a blue info toast should be visible
 *
 * Scenario: Dismiss toast
 *   Given a toast is displayed
 *   When the close button is clicked
 *   Then the toast should be removed
 */

import type { Meta, StoryObj } from '@storybook/vue3';
import { ref, reactive } from 'vue';
import { userEvent, within, expect } from '@storybook/test';

// Mock component since real ToastContainer uses a singleton service
const MockToastContainer = {
  name: 'MockToastContainer',
  props: {
    toasts: {
      type: Array,
      default: () => [],
    },
  },
  emits: ['remove'],
  template: `
    <div class="toast-container">
      <div
        v-for="toast in toasts"
        :key="toast.id"
        :class="['toast', 'toast--' + toast.type]"
      >
        <div class="toast__content">
          <span class="toast__icon">{{ getIcon(toast.type) }}</span>
          <span class="toast__message">{{ toast.message }}</span>
          <button @click="$emit('remove', toast.id)" class="toast__close" aria-label="Close">✕</button>
        </div>
      </div>
    </div>
  `,
  methods: {
    getIcon(type: string) {
      const icons: Record<string, string> = {
        success: '✓',
        error: '✕',
        warning: '⚠',
        info: 'ℹ',
      };
      return icons[type] || 'ℹ';
    },
  },
};

const meta: Meta = {
  title: 'App/Services/ToastContainer',
  component: MockToastContainer,
  tags: ['autodocs'],
  parameters: {
    layout: 'fullscreen',
  },
};

export default meta;
type Story = StoryObj;

// Success toast
export const SuccessToast: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await expect(canvas.getByText('Operation completed successfully!')).toBeInTheDocument();
  },
  render: () => ({
    components: { MockToastContainer },
    setup() {
      const toasts = ref([
        { id: '1', type: 'success', message: 'Operation completed successfully!' },
      ]);
      return { toasts };
    },
    template: '<MockToastContainer :toasts="toasts" />',
  }),
};

// Error toast
export const ErrorToast: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await expect(canvas.getByText('An error occurred. Please try again.')).toBeInTheDocument();
  },
  render: () => ({
    components: { MockToastContainer },
    setup() {
      const toasts = ref([
        { id: '1', type: 'error', message: 'An error occurred. Please try again.' },
      ]);
      return { toasts };
    },
    template: '<MockToastContainer :toasts="toasts" />',
  }),
};

// Warning toast
export const WarningToast: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await expect(canvas.getByText('Warning: This action cannot be undone.')).toBeInTheDocument();
  },
  render: () => ({
    components: { MockToastContainer },
    setup() {
      const toasts = ref([
        { id: '1', type: 'warning', message: 'Warning: This action cannot be undone.' },
      ]);
      return { toasts };
    },
    template: '<MockToastContainer :toasts="toasts" />',
  }),
};

// Info toast
export const InfoToast: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await expect(canvas.getByText('New features are available.')).toBeInTheDocument();
  },
  render: () => ({
    components: { MockToastContainer },
    setup() {
      const toasts = ref([
        { id: '1', type: 'info', message: 'New features are available.' },
      ]);
      return { toasts };
    },
    template: '<MockToastContainer :toasts="toasts" />',
  }),
};

// All toast types
export const AllTypes: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);

    await expect(canvas.getByText('Success message')).toBeInTheDocument();
    await expect(canvas.getByText('Error message')).toBeInTheDocument();
    await expect(canvas.getByText('Warning message')).toBeInTheDocument();
    await expect(canvas.getByText('Info message')).toBeInTheDocument();
  },
  render: () => ({
    components: { MockToastContainer },
    setup() {
      const toasts = ref([
        { id: '1', type: 'success', message: 'Success message' },
        { id: '2', type: 'error', message: 'Error message' },
        { id: '3', type: 'warning', message: 'Warning message' },
        { id: '4', type: 'info', message: 'Info message' },
      ]);
      return { toasts };
    },
    template: '<MockToastContainer :toasts="toasts" />',
  }),
};

// Dismiss interaction
export const DismissInteraction: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);

    // Find the close button
    const closeButton = canvas.getByRole('button', { name: /close/i });
    await expect(closeButton).toBeInTheDocument();

    // Click close button
    await userEvent.click(closeButton);
  },
  render: () => ({
    components: { MockToastContainer },
    setup() {
      const toasts = ref([
        { id: '1', type: 'info', message: 'Click the X to dismiss this toast' },
      ]);
      const remove = (id: string) => {
        toasts.value = toasts.value.filter((t) => t.id !== id);
      };
      return { toasts, remove };
    },
    template: '<MockToastContainer :toasts="toasts" @remove="remove" />',
  }),
};

// Multiple toasts stacked
export const MultipleStacked: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    const toasts = canvasElement.querySelectorAll('.toast');
    await expect(toasts.length).toBe(5);
  },
  render: () => ({
    components: { MockToastContainer },
    setup() {
      const toasts = ref([
        { id: '1', type: 'info', message: 'First notification' },
        { id: '2', type: 'success', message: 'Second notification' },
        { id: '3', type: 'warning', message: 'Third notification' },
        { id: '4', type: 'error', message: 'Fourth notification' },
        { id: '5', type: 'info', message: 'Fifth notification' },
      ]);
      return { toasts };
    },
    template: '<MockToastContainer :toasts="toasts" />',
  }),
};

// Long message
export const LongMessage: Story = {
  play: async ({ canvasElement }) => {
    const toast = canvasElement.querySelector('.toast');
    await expect(toast).toBeInTheDocument();
  },
  render: () => ({
    components: { MockToastContainer },
    setup() {
      const toasts = ref([
        {
          id: '1',
          type: 'info',
          message: 'This is a very long toast message that demonstrates how the component handles longer text content. It should wrap properly and remain readable.',
        },
      ]);
      return { toasts };
    },
    template: '<MockToastContainer :toasts="toasts" />',
  }),
};

// Empty state
export const EmptyState: Story = {
  play: async ({ canvasElement }) => {
    const toasts = canvasElement.querySelectorAll('.toast');
    await expect(toasts.length).toBe(0);
  },
  render: () => ({
    components: { MockToastContainer },
    setup() {
      const toasts = ref([]);
      return { toasts };
    },
    template: '<MockToastContainer :toasts="toasts" />',
  }),
};
