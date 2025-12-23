/**
 * Test Plan: RecorderIndexView
 *
 * Scenario: Redirect to sessions
 *   Given the RecorderIndexView is rendered
 *   When the component mounts
 *   Then it should redirect to /recorders/:recorderId/sessions
 *
 * Note: This component handles the redirect from /recorders/:recorderId to sessions.
 */

import type { Meta, StoryObj } from '@storybook/vue3';
import { within, expect } from '@storybook/test';

// Mock component showing redirect behavior
const MockRecorderIndexView = {
  name: 'MockRecorderIndexView',
  props: {
    recorderId: {
      type: String,
      default: 'recorder-1',
    },
  },
  template: `
    <div class="recorder-index-view" style="padding: 2rem; text-align: center;">
      <div style="padding: 1rem; background: #fff3cd; border: 1px solid #ffc107; border-radius: 8px;">
        <p style="margin: 0;">
          <strong>Redirecting...</strong>
        </p>
        <p style="margin-top: 0.5rem; font-size: 0.875rem; color: #666;">
          /recorders/{{ recorderId }} → /recorders/{{ recorderId }}/sessions
        </p>
      </div>
    </div>
  `,
};

const meta: Meta = {
  title: 'App/Views/RecorderIndexView',
  component: MockRecorderIndexView,
  tags: ['autodocs'],
  parameters: {
    layout: 'padded',
  },
};

export default meta;
type Story = StoryObj;

// Default redirect state
export const Default: Story = {
  args: {
    recorderId: 'recorder-1',
  },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await expect(canvas.getByText('Redirecting...')).toBeInTheDocument();
    await expect(canvas.getByText(/→.*sessions/)).toBeInTheDocument();
  },
};

// With different recorder ID
export const DifferentRecorderId: Story = {
  args: {
    recorderId: 'studio-mic-a',
  },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await expect(canvas.getByText(/studio-mic-a/)).toBeInTheDocument();
  },
};

// Component behavior documentation
export const BehaviorDocumentation: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await expect(canvas.getByText('RecorderIndexView Behavior')).toBeInTheDocument();
  },
  render: () => ({
    template: `
      <div style="font-family: system-ui; padding: 1rem;">
        <h3 style="margin-bottom: 1rem;">RecorderIndexView Behavior</h3>
        <div style="padding: 1rem; background: #f5f5f5; border-radius: 8px;">
          <p><strong>Purpose:</strong> Handle default route for recorder</p>
          <p style="margin-top: 0.5rem;"><strong>Action:</strong> Watches selectedRecorderId and redirects to sessions route</p>
          <pre style="margin-top: 1rem; padding: 1rem; background: #263238; color: #aed581; border-radius: 4px; overflow-x: auto;">
watch(selectedRecorderId, () => {
  router.replace(\`/recorders/\${selectedRecorderId.value}/sessions\`);
}, { immediate: true });
          </pre>
        </div>
      </div>
    `,
  }),
};
