/**
 * Test Plan: PageHeader
 *
 * Scenario: Render page header
 *   Given the PageHeader component is rendered
 *   When the component mounts
 *   Then the logo and heading should be visible
 *
 * Scenario: Logo link
 *   Given the header is rendered
 *   When examining the logo
 *   Then it should link to the home page
 */

import type { Meta, StoryObj } from '@storybook/vue3';
import { within, expect } from '@storybook/test';
import PageHeader from './PageHeader.vue';

const meta: Meta<typeof PageHeader> = {
  title: 'App/Layout/PageHeader',
  component: PageHeader,
  tags: ['autodocs'],
  parameters: {
    layout: 'padded',
  },
  decorators: [
    () => ({
      template: '<div style="width: 100%;"><story /></div>',
    }),
  ],
};

export default meta;
type Story = StoryObj<typeof meta>;

// Default header
export const Default: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);

    // Check heading is present
    await expect(canvas.getByText('Session Recorder')).toBeInTheDocument();

    // Check logo link exists
    const logoLink = canvasElement.querySelector('.logo');
    await expect(logoLink).toBeInTheDocument();
  },
  render: () => ({
    components: { PageHeader },
    template: '<PageHeader />',
  }),
};

// In container context
export const InContainer: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await expect(canvas.getByText('Session Recorder')).toBeInTheDocument();
  },
  render: () => ({
    components: { PageHeader },
    template: `
      <div style="max-width: 1200px; margin: auto;">
        <PageHeader />
      </div>
    `,
  }),
};

// Full width background
export const FullWidthBackground: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await expect(canvas.getByText('Session Recorder')).toBeInTheDocument();
  },
  render: () => ({
    components: { PageHeader },
    template: `
      <div style="background: #f5f5f5;">
        <div style="max-width: 1200px; margin: auto;">
          <PageHeader />
        </div>
      </div>
    `,
  }),
};
