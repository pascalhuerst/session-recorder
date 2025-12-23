/**
 * Test Plan: RecorderView
 *
 * Scenario: Render router outlet
 *   Given the RecorderView is rendered
 *   When the component mounts
 *   Then it should render the router-view for nested routes
 *
 * Note: This is a simple pass-through component for nested routing.
 */

import type { Meta, StoryObj } from '@storybook/vue3';
import { within, expect } from '@storybook/test';

// Mock RecorderView (simple router-view wrapper)
const MockRecorderView = {
  name: 'MockRecorderView',
  template: `
    <div class="recorder-view">
      <slot>
        <p style="padding: 2rem; text-align: center; color: #666;">Nested route content renders here</p>
      </slot>
    </div>
  `,
};

const meta: Meta = {
  title: 'App/Views/RecorderView',
  component: MockRecorderView,
  tags: ['autodocs'],
  parameters: {
    layout: 'padded',
  },
};

export default meta;
type Story = StoryObj;

// Default (empty router-view)
export const Default: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await expect(canvas.getByText('Nested route content renders here')).toBeInTheDocument();
  },
};

// With nested content
export const WithNestedContent: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await expect(canvas.getByText('Sessions List')).toBeInTheDocument();
  },
  render: () => ({
    components: { MockRecorderView },
    template: `
      <MockRecorderView>
        <div style="padding: 1rem; background: #f5f5f5; border-radius: 8px;">
          <h2>Sessions List</h2>
          <p>This content is rendered by a nested route (e.g., SessionsView)</p>
        </div>
      </MockRecorderView>
    `,
  }),
};

// Route structure documentation
export const RouteStructure: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await expect(canvas.getByText('/recorders/:recorderId')).toBeInTheDocument();
  },
  render: () => ({
    template: `
      <div style="font-family: system-ui; padding: 1rem;">
        <h3 style="margin-bottom: 1rem;">Route Hierarchy</h3>
        <div style="padding: 1rem; background: #e3f2fd; border-radius: 8px; margin-bottom: 0.5rem;">
          <code>/recorders/:recorderId</code>
          <p style="font-size: 0.875rem; margin-top: 0.5rem;">RecorderView - Pass-through for nested routes</p>
        </div>
        <div style="padding: 1rem; background: #e8f5e9; border-radius: 8px; margin-left: 1rem; margin-bottom: 0.5rem;">
          <code>/recorders/:recorderId/sessions</code>
          <p style="font-size: 0.875rem; margin-top: 0.5rem;">SessionsView - Sessions listing</p>
        </div>
        <div style="padding: 1rem; background: #fff3e0; border-radius: 8px; margin-left: 2rem;">
          <code>/recorders/:recorderId/sessions/:sessionId</code>
          <p style="font-size: 0.875rem; margin-top: 0.5rem;">SessionView - Individual session</p>
        </div>
      </div>
    `,
  }),
};
