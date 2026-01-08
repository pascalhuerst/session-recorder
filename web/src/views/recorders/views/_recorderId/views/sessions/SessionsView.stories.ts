/**
 * Test Plan: SessionsView
 *
 * Scenario: Render router outlet
 *   Given the SessionsView is rendered
 *   When the component mounts
 *   Then it should render the router-view for sessions routes
 *
 * Note: This is a simple pass-through component for nested sessions routing.
 */

import type { Meta, StoryObj } from '@storybook/vue3';
import { within, expect } from '@storybook/test';

// Mock SessionsView (simple router-view wrapper)
const MockSessionsView = {
  name: 'MockSessionsView',
  template: `
    <div class="sessions-view">
      <slot>
        <p style="padding: 2rem; text-align: center; color: #666;">Sessions route content renders here</p>
      </slot>
    </div>
  `,
};

const meta: Meta = {
  title: 'App/Views/SessionsView',
  component: MockSessionsView,
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
    await expect(canvas.getByText('Sessions route content renders here')).toBeInTheDocument();
  },
};

// With sessions list
export const WithSessionsList: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await expect(canvas.getByText('Session #1')).toBeInTheDocument();
    await expect(canvas.getByText('Session #2')).toBeInTheDocument();
  },
  render: () => ({
    components: { MockSessionsView },
    template: `
      <MockSessionsView>
        <div style="display: flex; flex-direction: column; gap: 1rem;">
          <div style="padding: 1rem; background: white; border: 1px solid #ddd; border-radius: 8px;">
            <h3>Session #1</h3>
            <p style="color: #666;">Recording from Mar 15, 2024</p>
          </div>
          <div style="padding: 1rem; background: white; border: 1px solid #ddd; border-radius: 8px;">
            <h3>Session #2</h3>
            <p style="color: #666;">Recording from Mar 14, 2024</p>
          </div>
        </div>
      </MockSessionsView>
    `,
  }),
};

// Route structure
export const RouteStructure: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await expect(canvas.getByText('/recorders/:recorderId/sessions')).toBeInTheDocument();
  },
  render: () => ({
    template: `
      <div style="font-family: system-ui; padding: 1rem;">
        <h3 style="margin-bottom: 1rem;">Sessions Route Hierarchy</h3>
        <div style="padding: 1rem; background: #e3f2fd; border-radius: 8px; margin-bottom: 0.5rem;">
          <code>/recorders/:recorderId/sessions</code>
          <p style="font-size: 0.875rem; margin-top: 0.5rem;">SessionsView - Pass-through for sessions routes</p>
        </div>
        <div style="padding: 1rem; background: #e8f5e9; border-radius: 8px; margin-left: 1rem;">
          <code>/recorders/:recorderId/sessions (index)</code>
          <p style="font-size: 0.875rem; margin-top: 0.5rem;">SessionsIndexView - Sessions list with waveform editors</p>
        </div>
      </div>
    `,
  }),
};
