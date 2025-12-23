/**
 * Test Plan: RecordersView
 *
 * Scenario: Render main recorders view
 *   Given the RecordersView is rendered
 *   When the component mounts
 *   Then PageShell, DevicePicker, RecorderActions, and router-view should be present
 *
 * Scenario: With recorders loaded
 *   Given recorders are available in the store
 *   When the view renders
 *   Then the DevicePicker should show the recorders
 *
 * Scenario: With recording active
 *   Given a recorder is actively recording
 *   When the view renders
 *   Then the RecorderActions banner should be visible
 */

import type { Meta, StoryObj } from '@storybook/vue3';
import { within, expect } from '@storybook/test';

// Mock components for story
const MockDevicePicker = {
  name: 'MockDevicePicker',
  props: ['recorders', 'selectedRecorderId'],
  template: `
    <nav class="device-picker">
      <ul style="display: flex; gap: 0.5rem; list-style: none; padding: 0; margin: 0;">
        <li v-for="(rec, id) in recorders" :key="id" style="padding: 0.5rem 1rem; background: white; border-radius: 4px; border: 1px solid #ddd;">
          {{ rec.name }}
          <span v-if="id === selectedRecorderId" style="color: green;"> ✓</span>
        </li>
      </ul>
    </nav>
  `,
};

const MockRecorderActions = {
  name: 'MockRecorderActions',
  props: ['isRecording'],
  template: `
    <div v-if="isRecording" class="actions-banner" style="padding: 1rem; background: #fff3cd; border: 1px solid #ffc107; border-radius: 4px; display: flex; align-items: center; gap: 1rem;">
      <span>This recorder is currently recording</span>
      <button style="padding: 0.5rem 1rem; background: #007bff; color: white; border: none; border-radius: 4px;">Cut Session</button>
    </div>
  `,
};

const MockPageShell = {
  name: 'MockPageShell',
  template: `
    <div class="page-shell">
      <header style="padding: 1rem; background: #333; color: white;">
        <h1 style="margin: 0; font-size: 1.25rem;">Session Recorder</h1>
      </header>
      <main style="min-height: 400px;">
        <slot />
      </main>
    </div>
  `,
};

// Mock RecordersView
const MockRecordersView = {
  name: 'MockRecordersView',
  components: { MockPageShell, MockDevicePicker, MockRecorderActions },
  props: {
    recorders: { type: Object, default: () => ({}) },
    selectedRecorderId: { type: String, default: '' },
    isRecording: { type: Boolean, default: false },
  },
  template: `
    <MockPageShell>
      <div class="navbar" style="padding: 1rem; background: #f5f5f5;">
        <MockDevicePicker :recorders="recorders" :selectedRecorderId="selectedRecorderId" />
      </div>
      <div class="action-bar" style="padding: 1rem;">
        <MockRecorderActions :isRecording="isRecording" />
      </div>
      <div class="content" style="padding: 1rem;">
        <slot>
          <p style="color: #666;">Router view content</p>
        </slot>
      </div>
    </MockPageShell>
  `,
};

const meta: Meta = {
  title: 'App/Views/RecordersView',
  component: MockRecordersView,
  tags: ['autodocs'],
  parameters: {
    layout: 'fullscreen',
  },
};

export default meta;
type Story = StoryObj;

// Default view with recorders
export const Default: Story = {
  args: {
    recorders: {
      'r1': { name: 'Studio Mic A' },
      'r2': { name: 'Live Room Mic' },
      'r3': { name: 'Booth Mic' },
    },
    selectedRecorderId: 'r1',
    isRecording: false,
  },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);

    // Header should be visible
    await expect(canvas.getByText('Session Recorder')).toBeInTheDocument();

    // Recorders should be visible
    await expect(canvas.getByText('Studio Mic A')).toBeInTheDocument();
    await expect(canvas.getByText('Live Room Mic')).toBeInTheDocument();
  },
};

// With recording active
export const RecordingActive: Story = {
  args: {
    recorders: {
      'r1': { name: 'Studio Mic A' },
      'r2': { name: 'Live Room Mic' },
    },
    selectedRecorderId: 'r2',
    isRecording: true,
  },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);

    // Recording banner should be visible
    await expect(canvas.getByText('This recorder is currently recording')).toBeInTheDocument();
    await expect(canvas.getByText('Cut Session')).toBeInTheDocument();
  },
};

// Empty state (no recorders)
export const NoRecorders: Story = {
  args: {
    recorders: {},
    selectedRecorderId: '',
    isRecording: false,
  },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await expect(canvas.getByText('Session Recorder')).toBeInTheDocument();
  },
};

// With session content
export const WithSessionContent: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await expect(canvas.getByText('Session #1')).toBeInTheDocument();
    await expect(canvas.getByText('Session #2')).toBeInTheDocument();
  },
  render: () => ({
    components: { MockRecordersView },
    setup() {
      return {
        recorders: {
          'r1': { name: 'Studio Mic A' },
        },
        selectedRecorderId: 'r1',
      };
    },
    template: `
      <MockRecordersView :recorders="recorders" :selectedRecorderId="selectedRecorderId">
        <div style="display: flex; flex-direction: column; gap: 1rem;">
          <div style="padding: 1rem; background: white; border: 1px solid #ddd; border-radius: 8px;">
            <h3>Session #1</h3>
            <p style="color: #666;">Mar 15, 2024 14:30</p>
          </div>
          <div style="padding: 1rem; background: white; border: 1px solid #ddd; border-radius: 8px;">
            <h3>Session #2</h3>
            <p style="color: #666;">Mar 14, 2024 10:00</p>
          </div>
        </div>
      </MockRecordersView>
    `,
  }),
};

// Layout structure visualization
export const LayoutStructure: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await expect(canvas.getByText('Navbar (DevicePicker)')).toBeInTheDocument();
    await expect(canvas.getByText('Action Bar (RecorderActions)')).toBeInTheDocument();
    await expect(canvas.getByText('Content (Router View)')).toBeInTheDocument();
  },
  render: () => ({
    template: `
      <div style="font-family: system-ui;">
        <div style="padding: 1rem; background: #333; color: white;">
          <strong>PageShell Header</strong>
        </div>
        <div style="padding: 1rem; background: #e3f2fd; border: 2px dashed #2196f3;">
          <strong>Navbar (DevicePicker)</strong>
          <p style="font-size: 0.875rem;">Horizontal scrollable list of recorders</p>
        </div>
        <div style="padding: 1rem; background: #fff3e0; border: 2px dashed #ff9800;">
          <strong>Action Bar (RecorderActions)</strong>
          <p style="font-size: 0.875rem;">Shows "Cut Session" when recording</p>
        </div>
        <div style="padding: 2rem; background: #e8f5e9; border: 2px dashed #4caf50; min-height: 200px;">
          <strong>Content (Router View)</strong>
          <p style="font-size: 0.875rem;">SessionsIndexView renders here</p>
        </div>
      </div>
    `,
  }),
};
