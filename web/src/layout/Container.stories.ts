/**
 * Test Plan: Container
 *
 * Scenario: Render container with slot content
 *   Given the Container component is rendered
 *   When slot content is provided
 *   Then the content should be displayed within the container
 *
 * Scenario: Container styling
 *   Given the Container is rendered
 *   When examining the DOM
 *   Then it should have max-width and centered margins
 */

import type { Meta, StoryObj } from '@storybook/vue3';
import { within, expect } from '@storybook/test';
import Container from './Container.vue';

const meta: Meta<typeof Container> = {
  title: 'App/Layout/Container',
  component: Container,
  tags: ['autodocs'],
  parameters: {
    layout: 'fullscreen',
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

// Default container
export const Default: Story = {
  play: async ({ canvasElement }) => {
    const container = canvasElement.querySelector('.container');
    await expect(container).toBeInTheDocument();
  },
  render: () => ({
    components: { Container },
    template: `
      <Container>
        <p>Content inside the container</p>
      </Container>
    `,
  }),
};

// With card content
export const WithCardContent: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await expect(canvas.getByText('Card Title')).toBeInTheDocument();
  },
  render: () => ({
    components: { Container },
    template: `
      <Container>
        <div style="background: white; padding: 2rem; border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.1);">
          <h2>Card Title</h2>
          <p>Card content goes here.</p>
        </div>
      </Container>
    `,
  }),
};

// Multiple sections
export const MultipleSections: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await expect(canvas.getByText('Section 1')).toBeInTheDocument();
    await expect(canvas.getByText('Section 2')).toBeInTheDocument();
    await expect(canvas.getByText('Section 3')).toBeInTheDocument();
  },
  render: () => ({
    components: { Container },
    template: `
      <Container>
        <div style="display: flex; flex-direction: column; gap: 1rem;">
          <div style="padding: 1rem; background: #f5f5f5; border-radius: 4px;">Section 1</div>
          <div style="padding: 1rem; background: #f5f5f5; border-radius: 4px;">Section 2</div>
          <div style="padding: 1rem; background: #f5f5f5; border-radius: 4px;">Section 3</div>
        </div>
      </Container>
    `,
  }),
};

// Empty container
export const Empty: Story = {
  play: async ({ canvasElement }) => {
    const container = canvasElement.querySelector('.container');
    await expect(container).toBeInTheDocument();
  },
  render: () => ({
    components: { Container },
    template: '<Container />',
  }),
};

// With background
export const WithBackground: Story = {
  render: () => ({
    components: { Container },
    template: `
      <div style="background: #e9d5ff; padding: 2rem 0;">
        <Container>
          <p style="padding: 1rem; background: white; border-radius: 4px;">
            Content with colored background behind container
          </p>
        </Container>
      </div>
    `,
  }),
};
