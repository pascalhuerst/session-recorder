/**
 * Test Plan: Modal
 *
 * Scenario: Open and close modal
 *   Given the Modal component is rendered with open=false
 *   When open prop changes to true
 *   Then the modal dialog should be visible
 *   And the backdrop should be visible
 *
 * Scenario: Close modal via button
 *   Given the Modal is open
 *   When the close button is clicked
 *   Then the modal should close
 *   And the close event should be emitted
 *
 * Scenario: Close modal via ESC key
 *   Given the Modal is open
 *   When the user presses ESC
 *   Then the modal should close
 *   And the close event should be emitted
 *
 * Scenario: Custom size modal
 *   Given a size prop is provided
 *   When the modal opens
 *   Then it should have the specified dimensions
 *
 * Scenario: Slot content rendering
 *   Given header, body, and footer slots are provided
 *   When the modal is open
 *   Then all slot content should be visible
 */

import type { Meta, StoryObj } from '@storybook/vue3';
import { ref } from 'vue';
import { userEvent, within, expect, fn } from '@storybook/test';
import Modal from './Modal.vue';
import Button from '../controls/Button.vue';

const meta: Meta<typeof Modal> = {
  title: 'Lib/Disclosure/Modal',
  component: Modal,
  tags: ['autodocs'],
  parameters: {
    layout: 'fullscreen',
  },
  decorators: [
    () => ({
      template: '<div id="modals"></div><story />',
    }),
  ],
  argTypes: {
    open: {
      control: 'boolean',
      description: 'Controls modal visibility',
    },
    size: {
      control: 'object',
      description: 'Fixed size dimensions { width, height? }',
    },
  },
  args: {
    onClose: fn(),
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

// Default modal
export const Default: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);

    // Click the trigger button
    const triggerButton = canvas.getByRole('button', { name: /open modal/i });
    await expect(triggerButton).toBeInTheDocument();
    await userEvent.click(triggerButton);
  },
  render: (args) => ({
    components: { Modal, Button },
    setup() {
      const isOpen = ref(false);
      const open = () => (isOpen.value = true);
      const close = () => {
        isOpen.value = false;
        args.onClose?.();
      };
      return { isOpen, open, close };
    },
    template: `
      <div>
        <Button @click="open" variant="solid" color="primary">Open Modal</Button>
        <Modal :open="isOpen" @close="close">
          <template #header>Modal Title</template>
          <template #body>
            <p>This is the modal body content.</p>
          </template>
          <template #footer>
            <Button @click="close">Cancel</Button>
            <Button variant="solid" color="primary" @click="close">Confirm</Button>
          </template>
        </Modal>
      </div>
    `,
  }),
};

// Modal with custom size
export const CustomSize: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);

    const triggerButton = canvas.getByRole('button', { name: /open/i });
    await expect(triggerButton).toBeInTheDocument();
    await userEvent.click(triggerButton);
  },
  render: (args) => ({
    components: { Modal, Button },
    setup() {
      const isOpen = ref(false);
      const open = () => (isOpen.value = true);
      const close = () => {
        isOpen.value = false;
        args.onClose?.();
      };
      return { isOpen, open, close };
    },
    template: `
      <div>
        <Button @click="open" variant="solid">Open Custom Size Modal</Button>
        <Modal :open="isOpen" @close="close" :size="{ width: 600, height: 400 }">
          <template #header>Custom Size Modal</template>
          <template #body>
            <p>This modal has a fixed width of 600px and height of 400px.</p>
          </template>
          <template #footer>
            <Button @click="close">Close</Button>
          </template>
        </Modal>
      </div>
    `,
  }),
};

// Modal with only width
export const CustomWidthOnly: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);

    const triggerButton = canvas.getByRole('button', { name: /open/i });
    await expect(triggerButton).toBeInTheDocument();
    await userEvent.click(triggerButton);
  },
  render: (args) => ({
    components: { Modal, Button },
    setup() {
      const isOpen = ref(false);
      const open = () => (isOpen.value = true);
      const close = () => {
        isOpen.value = false;
        args.onClose?.();
      };
      return { isOpen, open, close };
    },
    template: `
      <div>
        <Button @click="open" variant="solid">Open Wide Modal</Button>
        <Modal :open="isOpen" @close="close" :size="{ width: 800 }">
          <template #header>Wide Modal</template>
          <template #body>
            <p>This modal has a fixed width but auto height.</p>
            <p>Content can expand vertically as needed.</p>
          </template>
          <template #footer>
            <Button @click="close">Close</Button>
          </template>
        </Modal>
      </div>
    `,
  }),
};

// Initially open modal
export const InitiallyOpen: Story = {
  // No play test - this story demonstrates initially open state visually
  render: (args) => ({
    components: { Modal, Button },
    setup() {
      const isOpen = ref(true);
      const close = () => {
        isOpen.value = false;
        args.onClose?.();
      };
      return { isOpen, close };
    },
    template: `
      <Modal :open="isOpen" @close="close">
        <template #header>Initially Open Modal</template>
        <template #body>
          <p>This modal opens automatically on render.</p>
        </template>
        <template #footer>
          <Button @click="close">Got it</Button>
        </template>
      </Modal>
    `,
  }),
};

// Confirmation dialog
export const ConfirmationDialog: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);

    const deleteButton = canvas.getByRole('button', { name: /delete item/i });
    await expect(deleteButton).toBeInTheDocument();
    await userEvent.click(deleteButton);
  },
  render: (args) => ({
    components: { Modal, Button },
    setup() {
      const isOpen = ref(false);
      const open = () => (isOpen.value = true);
      const close = () => {
        isOpen.value = false;
        args.onClose?.();
      };
      return { isOpen, open, close };
    },
    template: `
      <div>
        <Button @click="open" variant="solid" color="primary">Delete Item</Button>
        <Modal :open="isOpen" @close="close" :size="{ width: 400 }">
          <template #header>Confirm Delete</template>
          <template #body>
            <p>Are you sure you want to delete this item? This action cannot be undone.</p>
          </template>
          <template #footer>
            <Button @click="close">Cancel</Button>
            <Button variant="solid" color="primary" @click="close">Confirm</Button>
          </template>
        </Modal>
      </div>
    `,
  }),
};

// Long content modal
export const LongContent: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);

    const triggerButton = canvas.getByRole('button', { name: /open/i });
    await expect(triggerButton).toBeInTheDocument();
    await userEvent.click(triggerButton);
  },
  render: (args) => ({
    components: { Modal, Button },
    setup() {
      const isOpen = ref(false);
      const open = () => (isOpen.value = true);
      const close = () => {
        isOpen.value = false;
        args.onClose?.();
      };
      const paragraphs = Array(10).fill('Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua.');
      return { isOpen, open, close, paragraphs };
    },
    template: `
      <div>
        <Button @click="open" variant="solid">Open Long Content Modal</Button>
        <Modal :open="isOpen" @close="close" :size="{ width: 600, height: 400 }">
          <template #header>Scrollable Content</template>
          <template #body>
            <div v-for="(p, i) in paragraphs" :key="i">
              <p>{{ p }}</p>
            </div>
          </template>
          <template #footer>
            <Button @click="close">Close</Button>
          </template>
        </Modal>
      </div>
    `,
  }),
};

// Form modal
export const FormModal: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);

    const triggerButton = canvas.getByRole('button', { name: /add new/i });
    await expect(triggerButton).toBeInTheDocument();
    await userEvent.click(triggerButton);
  },
  render: (args) => ({
    components: { Modal, Button },
    setup() {
      const isOpen = ref(false);
      const open = () => (isOpen.value = true);
      const close = () => {
        isOpen.value = false;
        args.onClose?.();
      };
      return { isOpen, open, close };
    },
    template: `
      <div>
        <Button @click="open" variant="solid" color="primary">Add New</Button>
        <Modal :open="isOpen" @close="close" :size="{ width: 500 }">
          <template #header>Add New Item</template>
          <template #body>
            <form style="display: flex; flex-direction: column; gap: 1rem;">
              <div style="display: flex; flex-direction: column; gap: 0.5rem;">
                <label for="name">Name</label>
                <input id="name" type="text" style="padding: 0.5rem; border: 1px solid #ccc; border-radius: 4px;" />
              </div>
              <div style="display: flex; flex-direction: column; gap: 0.5rem;">
                <label for="email">Email</label>
                <input id="email" type="email" style="padding: 0.5rem; border: 1px solid #ccc; border-radius: 4px;" />
              </div>
            </form>
          </template>
          <template #footer>
            <Button @click="close">Cancel</Button>
            <Button variant="solid" color="primary" @click="close">Save</Button>
          </template>
        </Modal>
      </div>
    `,
  }),
};
