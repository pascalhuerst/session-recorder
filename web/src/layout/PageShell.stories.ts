/**
 * Test Plan: PageShell
 *
 * Scenario: Render page shell with header and content
 *   Given the PageShell component is rendered
 *   When slot content is provided
 *   Then the header and main content should be visible
 *
 * Scenario: Layout structure
 *   Given the PageShell is rendered
 *   When examining the DOM
 *   Then header and main elements should be present
 */

import type { Meta, StoryObj } from '@storybook/vue3';
import { within, expect } from '@storybook/test';
import PageShell from './PageShell.vue';

const meta: Meta<typeof PageShell> = {
  title: 'App/Layout/PageShell',
  component: PageShell,
  tags: ['autodocs'],
  parameters: {
    layout: 'fullscreen',
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

// Default page shell
export const Default: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);

    // Check header content
    await expect(canvas.getByText('Session Recorder')).toBeInTheDocument();

    // Check main content
    await expect(canvas.getByText('Page content goes here')).toBeInTheDocument();
  },
  render: () => ({
    components: { PageShell },
    template: `
      <PageShell>
        <p>Page content goes here</p>
      </PageShell>
    `,
  }),
};

// With card content
export const WithCardContent: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await expect(canvas.getByText('Dashboard')).toBeInTheDocument();
  },
  render: () => ({
    components: { PageShell },
    template: `
      <PageShell>
        <div style="max-width: 1200px; margin: auto; padding: 0 2rem;">
          <h1>Dashboard</h1>
          <div style="display: grid; grid-template-columns: repeat(3, 1fr); gap: 1rem; margin-top: 1rem;">
            <div style="padding: 1.5rem; background: white; border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.1);">
              <h3>Active Sessions</h3>
              <p style="font-size: 2rem; font-weight: bold;">12</p>
            </div>
            <div style="padding: 1.5rem; background: white; border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.1);">
              <h3>Total Recordings</h3>
              <p style="font-size: 2rem; font-weight: bold;">48</p>
            </div>
            <div style="padding: 1.5rem; background: white; border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.1);">
              <h3>Storage Used</h3>
              <p style="font-size: 2rem; font-weight: bold;">2.4 GB</p>
            </div>
          </div>
        </div>
      </PageShell>
    `,
  }),
};

// With sidebar layout
export const WithSidebarLayout: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await expect(canvas.getByText('Main Content')).toBeInTheDocument();
    await expect(canvas.getByText('Sidebar')).toBeInTheDocument();
  },
  render: () => ({
    components: { PageShell },
    template: `
      <PageShell>
        <div style="max-width: 1200px; margin: auto; padding: 0 2rem; display: grid; grid-template-columns: 250px 1fr; gap: 2rem;">
          <aside style="background: #f5f5f5; padding: 1rem; border-radius: 8px;">
            <h3 style="font-size: 0.875rem; text-transform: uppercase; color: #666;">Sidebar</h3>
            <nav style="margin-top: 1rem;">
              <a href="#" style="display: block; padding: 0.5rem; color: #333;">Dashboard</a>
              <a href="#" style="display: block; padding: 0.5rem; color: #333;">Sessions</a>
              <a href="#" style="display: block; padding: 0.5rem; color: #333;">Settings</a>
            </nav>
          </aside>
          <main>
            <h1>Main Content</h1>
            <p>This is the main content area.</p>
          </main>
        </div>
      </PageShell>
    `,
  }),
};

// Empty content
export const EmptyContent: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await expect(canvas.getByText('Session Recorder')).toBeInTheDocument();
  },
  render: () => ({
    components: { PageShell },
    template: '<PageShell />',
  }),
};
