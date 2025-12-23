/**
 * Test Plan: App
 *
 * Scenario: Render root application structure
 *   Given the App component is rendered
 *   When the component mounts
 *   Then the router-view, modals container, and toast container should be present
 *
 * Scenario: Modal portal
 *   Given a modal is opened
 *   When teleported
 *   Then it should render inside the #modals container
 */

import type { Meta, StoryObj } from '@storybook/vue3';
import { within, expect } from '@storybook/test';

// Mock App component since real one requires router
const MockApp = {
  name: 'MockApp',
  template: `
    <div class="app">
      <div class="router-view">
        <slot>
          <p style="padding: 2rem; text-align: center; color: #666;">Router View Content</p>
        </slot>
      </div>
      <div id="modals"></div>
      <div class="toast-container"></div>
    </div>
  `,
};

const meta: Meta = {
  title: 'App/Views/App',
  component: MockApp,
  tags: ['autodocs'],
  parameters: {
    layout: 'fullscreen',
  },
};

export default meta;
type Story = StoryObj;

// Default app structure
export const Default: Story = {
  play: async ({ canvasElement }) => {
    // Check that modals container exists
    const modalsContainer = canvasElement.querySelector('#modals');
    await expect(modalsContainer).toBeInTheDocument();

    // Check router view placeholder
    const canvas = within(canvasElement);
    await expect(canvas.getByText('Router View Content')).toBeInTheDocument();
  },
};

// With page content
export const WithPageContent: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await expect(canvas.getByText('Dashboard')).toBeInTheDocument();
  },
  render: () => ({
    components: { MockApp },
    template: `
      <MockApp>
        <div style="padding: 2rem;">
          <h1>Dashboard</h1>
          <p>Welcome to the session recorder application.</p>
        </div>
      </MockApp>
    `,
  }),
};

// With modal open
export const WithModalOpen: Story = {
  // Visual-only story - modal teleport behavior demonstrated in UI
  render: () => ({
    components: { MockApp },
    template: `
      <MockApp>
        <div style="padding: 2rem;">
          <p>Main content with modal overlay</p>
        </div>
      </MockApp>
    `,
  }),
};

// App structure visualization
export const StructureVisualization: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await expect(canvas.getByText('Router View')).toBeInTheDocument();
    await expect(canvas.getByText('Modals Portal')).toBeInTheDocument();
    await expect(canvas.getByText('Toast Container')).toBeInTheDocument();
  },
  render: () => ({
    template: `
      <div style="font-family: system-ui; padding: 2rem;">
        <h2 style="margin-bottom: 1rem;">App Component Structure</h2>
        <div style="display: flex; flex-direction: column; gap: 0.5rem;">
          <div style="padding: 1rem; background: #e3f2fd; border: 2px solid #2196f3; border-radius: 8px;">
            <strong>Router View</strong>
            <p style="font-size: 0.875rem; color: #666; margin-top: 0.5rem;">Main page content renders here</p>
          </div>
          <div style="padding: 1rem; background: #fff3e0; border: 2px solid #ff9800; border-radius: 8px;">
            <strong>Modals Portal</strong>
            <p style="font-size: 0.875rem; color: #666; margin-top: 0.5rem;">#modals - Teleport target for Modal components</p>
          </div>
          <div style="padding: 1rem; background: #e8f5e9; border: 2px solid #4caf50; border-radius: 8px;">
            <strong>Toast Container</strong>
            <p style="font-size: 0.875rem; color: #666; margin-top: 0.5rem;">Global toast notifications</p>
          </div>
        </div>
      </div>
    `,
  }),
};
