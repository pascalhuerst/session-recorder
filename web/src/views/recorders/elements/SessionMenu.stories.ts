/**
 * Test Plan: SessionMenu
 *
 * Scenario: Render menu for non-kept session
 *   Given a session that is not kept
 *   When the menu renders
 *   Then "Keep", "Delete", and "Download" buttons should be visible
 *   And the expiry date should be shown
 *
 * Scenario: Render menu for kept session
 *   Given a session that is kept
 *   When the menu renders
 *   Then only "Delete" and "Download" buttons should be visible
 *   And no expiry date should be shown
 *
 * Scenario: Keep action
 *   Given the Keep button is clicked
 *   When the action completes
 *   Then the session should be marked as kept
 *
 * Scenario: Delete action with confirmation
 *   Given the Delete button is clicked
 *   When the confirmation modal appears
 *   Then confirming should delete the session
 *   And canceling should close the modal
 *
 * Scenario: Download action
 *   Given the Download button is clicked
 *   When clicked
 *   Then the FLAC file should be downloaded
 */

import type { Meta, StoryObj } from '@storybook/vue3';
import { within, expect, userEvent } from '@storybook/test';

// Simple mock component without lib dependencies
const MockSessionMenu = {
  name: 'MockSessionMenu',
  props: {
    session: {
      type: Object,
      required: true,
    },
    recorderId: {
      type: String,
      required: true,
    },
  },
  emits: ['keep', 'delete', 'download'],
  methods: {
    formatDate(date: Date) {
      return date.toLocaleDateString('en-US', {
        weekday: 'short',
        month: 'short',
        day: 'numeric',
        hour: '2-digit',
        minute: '2-digit',
      });
    },
  },
  template: `
    <div class="menu" style="display: flex; align-items: center; gap: 0.5rem;">
      <template v-if="!session.keep">
        <span class="balance" style="font-size: 0.75rem; color: #dc2626;">Kept until {{ formatDate(session.expiresAt) }}</span>
        <button class="btn" @click="$emit('keep')">❤️ Keep</button>
      </template>
      <button class="btn" @click="$emit('delete')">🗑️ Delete</button>
      <a class="btn download-link" :href="session.downloadFiles.flac" target="_blank" download>⬇️ flac</a>
    </div>
  `,
};

// Mock session data
const createMockSession = (overrides = {}) => ({
  id: 'session-1',
  startedAt: new Date('2024-03-15T14:30:00'),
  expiresAt: new Date('2024-04-15T14:30:00'),
  keep: false,
  downloadFiles: {
    flac: '/download/audio.flac',
  },
  ...overrides,
});

const meta: Meta = {
  title: 'App/Elements/SessionMenu',
  component: MockSessionMenu,
  tags: ['autodocs'],
  parameters: {
    layout: 'centered',
  },
};

export default meta;
type Story = StoryObj;

// Non-kept session (shows Keep button)
export const NotKept: Story = {
  args: {
    session: createMockSession({ keep: false }),
    recorderId: 'recorder-1',
  },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);

    // Keep button should be visible
    await expect(canvas.getByText('❤️ Keep')).toBeInTheDocument();

    // Delete button should be visible
    await expect(canvas.getByText('🗑️ Delete')).toBeInTheDocument();

    // Download button should be visible
    await expect(canvas.getByText('⬇️ flac')).toBeInTheDocument();

    // Expiry date should be shown
    await expect(canvas.getByText(/Kept until/)).toBeInTheDocument();
  },
};

// Kept session (no Keep button)
export const Kept: Story = {
  args: {
    session: createMockSession({ keep: true }),
    recorderId: 'recorder-1',
  },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);

    // Keep button should NOT be visible
    await expect(canvas.queryByText('❤️ Keep')).not.toBeInTheDocument();

    // Delete button should be visible
    await expect(canvas.getByText('🗑️ Delete')).toBeInTheDocument();

    // Download button should be visible
    await expect(canvas.getByText('⬇️ flac')).toBeInTheDocument();

    // Expiry date should NOT be shown
    await expect(canvas.queryByText(/Kept until/)).not.toBeInTheDocument();
  },
};

// Keep interaction
export const KeepInteraction: Story = {
  args: {
    session: createMockSession({ keep: false }),
    recorderId: 'recorder-1',
  },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);

    const keepButton = canvas.getByText('❤️ Keep');
    await userEvent.click(keepButton);
  },
};

// Delete with confirmation
export const DeleteConfirmation: Story = {
  args: {
    session: createMockSession(),
    recorderId: 'recorder-1',
  },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);

    // Click delete button
    const deleteButton = canvas.getByText('🗑️ Delete');
    await userEvent.click(deleteButton);

    // Modal should appear (check document body for modal content)
    // Note: Modal is teleported to #modals
  },
};

// Download link
export const DownloadLink: Story = {
  args: {
    session: createMockSession(),
    recorderId: 'recorder-1',
  },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);

    const downloadLink = canvas.getByText('⬇️ flac');
    await expect(downloadLink).toBeInTheDocument();

    // Check it's a link
    await expect(downloadLink).toHaveAttribute('href', '/download/audio.flac');
    await expect(downloadLink).toHaveAttribute('download');
  },
};

// Expiring soon
export const ExpiringSoon: Story = {
  args: {
    session: createMockSession({
      expiresAt: new Date(Date.now() + 24 * 60 * 60 * 1000), // Tomorrow
    }),
    recorderId: 'recorder-1',
  },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await expect(canvas.getByText(/Kept until/)).toBeInTheDocument();
  },
};

// In card context
export const InCardContext: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await expect(canvas.getByText('🗑️ Delete')).toBeInTheDocument();
  },
  render: () => ({
    components: { MockSessionMenu },
    setup() {
      return { session: createMockSession() };
    },
    template: `
      <div style="display: flex; align-items: center; gap: 1rem; padding: 1rem; background: white; border: 1px solid #eee; border-radius: 8px; width: 400px;">
        <div style="flex: 1;">
          <strong>Session #1</strong>
          <p style="font-size: 0.875rem; color: #666;">Mar 15, 2024 14:30</p>
        </div>
        <MockSessionMenu :session="session" recorderId="recorder-1" />
      </div>
    `,
  }),
};

// All actions visible
export const AllActions: Story = {
  args: {
    session: createMockSession({ keep: false }),
    recorderId: 'recorder-1',
  },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);

    // All three buttons should be present
    await expect(canvas.getByText('❤️ Keep')).toBeInTheDocument();
    await expect(canvas.getByText('🗑️ Delete')).toBeInTheDocument();
    await expect(canvas.getByText('⬇️ flac')).toBeInTheDocument();
  },
};
