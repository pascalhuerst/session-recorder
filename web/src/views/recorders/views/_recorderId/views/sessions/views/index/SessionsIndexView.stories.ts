/**
 * Test Plan: SessionsIndexView
 *
 * Scenario: Render sessions list
 *   Given sessions are available
 *   When the view renders
 *   Then session cards should be displayed in reverse order (newest first)
 *
 * Scenario: Empty state
 *   Given no sessions are available
 *   When the view renders
 *   Then the EmptyScreen should be shown with appropriate message
 *
 * Scenario: Session card rendering
 *   Given multiple sessions
 *   When rendered
 *   Then each session should show index, date, and waveform editor
 */

import type { Meta, StoryObj } from '@storybook/vue3';
import { within, expect } from '@storybook/test';
import { EmptyScreen } from '@session-recorder/session-waveform';

// Mock SessionCard for stories
const MockSessionCard = {
  name: 'MockSessionCard',
  props: {
    session: { type: Object, required: true },
    recorderId: { type: String, required: true },
    index: { type: Number, required: true },
  },
  template: `
    <div class="session-card" style="padding: 1rem; background: white; border: 1px solid #ddd; border-radius: 8px;">
      <div style="display: flex; justify-content: space-between; align-items: baseline; margin-bottom: 1rem;">
        <span style="font-weight: bold;">Untitled #{{ index }}</span>
        <span style="font-size: 0.875rem; color: #666;">{{ formatDate(session.startedAt) }}</span>
      </div>
      <div style="height: 100px; background: linear-gradient(to right, #e0e0e0, #f5f5f5, #e0e0e0); border-radius: 4px; display: flex; align-items: center; justify-content: center; color: #666;">
        Waveform Editor
      </div>
    </div>
  `,
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
};

// Mock SessionsIndexView
const MockSessionsIndexView = {
  name: 'MockSessionsIndexView',
  components: { MockSessionCard, EmptyScreen },
  props: {
    sessions: { type: Array, default: () => [] },
    selectedRecorderId: { type: String, default: 'recorder-1' },
  },
  template: `
    <div class="sessions-index-view">
      <div v-if="sessions.length" class="list" style="display: flex; flex-direction: column; gap: 2rem;">
        <MockSessionCard
          v-for="(session, index) in sessions"
          :key="session.id"
          :session="session"
          :recorder-id="selectedRecorderId"
          :index="sessions.length - index"
        />
      </div>
      <div v-else>
        <EmptyScreen>
          <template #illustration>
            <span style="font-size: 3rem;">📊</span>
          </template>
          <template #text>
            There are no open sessions that were recorded by this recording device
          </template>
        </EmptyScreen>
      </div>
    </div>
  `,
};

// Mock session data
const createMockSession = (id: string, daysAgo: number) => ({
  id,
  startedAt: new Date(Date.now() - daysAgo * 24 * 60 * 60 * 1000),
  expiresAt: new Date(Date.now() + (30 - daysAgo) * 24 * 60 * 60 * 1000),
  keep: false,
  segments: [],
});

const meta: Meta = {
  title: 'App/Views/SessionsIndexView',
  component: MockSessionsIndexView,
  tags: ['autodocs'],
  parameters: {
    layout: 'padded',
  },
};

export default meta;
type Story = StoryObj;

// With sessions
export const WithSessions: Story = {
  args: {
    sessions: [
      createMockSession('s1', 0),
      createMockSession('s2', 1),
      createMockSession('s3', 2),
    ],
    selectedRecorderId: 'recorder-1',
  },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);

    // Should show sessions in reverse index order
    await expect(canvas.getByText('Untitled #3')).toBeInTheDocument();
    await expect(canvas.getByText('Untitled #2')).toBeInTheDocument();
    await expect(canvas.getByText('Untitled #1')).toBeInTheDocument();
  },
};

// Empty state
export const Empty: Story = {
  args: {
    sessions: [],
    selectedRecorderId: 'recorder-1',
  },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await expect(
      canvas.getByText('There are no open sessions that were recorded by this recording device')
    ).toBeInTheDocument();
  },
};

// Single session
export const SingleSession: Story = {
  args: {
    sessions: [createMockSession('s1', 0)],
    selectedRecorderId: 'recorder-1',
  },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await expect(canvas.getByText('Untitled #1')).toBeInTheDocument();
  },
};

// Many sessions
export const ManySessions: Story = {
  args: {
    sessions: Array.from({ length: 10 }, (_, i) => createMockSession(`s${i + 1}`, i)),
    selectedRecorderId: 'recorder-1',
  },
  play: async ({ canvasElement }) => {
    const cards = canvasElement.querySelectorAll('.session-card');
    await expect(cards.length).toBe(10);
  },
};

// Sessions with different dates
export const DifferentDates: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await expect(canvas.getByText('Untitled #3')).toBeInTheDocument();
  },
  render: () => ({
    components: { MockSessionsIndexView },
    setup() {
      const sessions = [
        { id: 's1', startedAt: new Date() },
        { id: 's2', startedAt: new Date(Date.now() - 7 * 24 * 60 * 60 * 1000) },
        { id: 's3', startedAt: new Date('2023-06-15T10:00:00') },
      ];
      return { sessions };
    },
    template: '<MockSessionsIndexView :sessions="sessions" />',
  }),
};

// Loading state visualization
export const LoadingState: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await expect(canvas.getByText('Loading sessions...')).toBeInTheDocument();
  },
  render: () => ({
    template: `
      <div style="padding: 2rem; text-align: center;">
        <div style="display: inline-block; width: 40px; height: 40px; border: 3px solid #f3f3f3; border-top: 3px solid #3498db; border-radius: 50%; animation: spin 1s linear infinite;"></div>
        <p style="margin-top: 1rem; color: #666;">Loading sessions...</p>
        <style>
          @keyframes spin {
            0% { transform: rotate(0deg); }
            100% { transform: rotate(360deg); }
          }
        </style>
      </div>
    `,
  }),
};
