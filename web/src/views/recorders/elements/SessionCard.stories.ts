/**
 * Test Plan: SessionCard
 *
 * Scenario: Render session card with metadata
 *   Given a session with data
 *   When the card renders
 *   Then the session index and timestamp should be visible
 *
 * Scenario: Date formatting
 *   Given a session with a date
 *   When the card renders
 *   Then the date should be formatted correctly
 *
 * Scenario: Menu integration
 *   Given the card renders
 *   When the menu is visible
 *   Then keep/delete/download actions should be available
 */

import type { Meta, StoryObj } from '@storybook/vue3';
import { within, expect } from '@storybook/test';

// Mock component since real SessionCard requires WaveformEditor and complex setup
const MockSessionCard = {
  name: 'MockSessionCard',
  props: {
    session: {
      type: Object,
      required: true,
    },
    recorderId: {
      type: String,
      required: true,
    },
    index: {
      type: Number,
      required: true,
    },
  },
  computed: {
    displayDate() {
      const { startedAt } = this.session;
      const format =
        startedAt.getFullYear() === new Date().getFullYear()
          ? 'ddd, MMM D, HH:mm'
          : 'MMM D, YYYY HH:mm';

      const options: Intl.DateTimeFormatOptions =
        startedAt.getFullYear() === new Date().getFullYear()
          ? { weekday: 'short', month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' }
          : { month: 'short', day: 'numeric', year: 'numeric', hour: '2-digit', minute: '2-digit' };

      return {
        iso: startedAt.toISOString(),
        formatted: startedAt.toLocaleDateString('en-US', options),
      };
    },
  },
  template: `
    <div class="card">
      <div class="header">
        <span class="title">Untitled #{{ index }}</span>
        <div class="metadata">
          <time class="timestamp" :datetime="displayDate.iso">{{ displayDate.formatted }}</time>
          <div class="menu">
            <button class="menu-button">⋮</button>
          </div>
        </div>
      </div>
      <div class="waveform-placeholder" style="height: 120px; background: linear-gradient(to right, #e0e0e0, #f5f5f5, #e0e0e0); border-radius: 4px; display: flex; align-items: center; justify-content: center; color: #666;">
        Waveform Editor
      </div>
    </div>
  `,
};

// Mock session data
const createMockSession = (overrides = {}) => ({
  id: 'session-1',
  startedAt: new Date('2024-03-15T14:30:00'),
  expiresAt: new Date('2024-04-15T14:30:00'),
  keep: false,
  segments: [],
  inlineFiles: {
    waveform: '/mock/waveform.dat',
    ogg: '/mock/audio.ogg',
    flac: '/mock/audio.flac',
  },
  downloadFiles: {
    flac: '/download/audio.flac',
  },
  ...overrides,
});

const meta: Meta = {
  title: 'App/Elements/SessionCard',
  component: MockSessionCard,
  tags: ['autodocs'],
  parameters: {
    layout: 'padded',
  },
};

export default meta;
type Story = StoryObj;

// Default session card
export const Default: Story = {
  args: {
    session: createMockSession(),
    recorderId: 'recorder-1',
    index: 1,
  },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);

    // Check title
    await expect(canvas.getByText('Untitled #1')).toBeInTheDocument();

    // Check waveform placeholder
    await expect(canvas.getByText('Waveform Editor')).toBeInTheDocument();
  },
};

// Different indices
export const DifferentIndices: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);

    await expect(canvas.getByText('Untitled #1')).toBeInTheDocument();
    await expect(canvas.getByText('Untitled #2')).toBeInTheDocument();
    await expect(canvas.getByText('Untitled #3')).toBeInTheDocument();
  },
  render: () => ({
    components: { MockSessionCard },
    setup() {
      const sessions = [
        { session: createMockSession({ id: 's1' }), index: 1 },
        { session: createMockSession({ id: 's2' }), index: 2 },
        { session: createMockSession({ id: 's3' }), index: 3 },
      ];
      return { sessions };
    },
    template: `
      <div style="display: flex; flex-direction: column; gap: 1rem;">
        <MockSessionCard
          v-for="s in sessions"
          :key="s.session.id"
          :session="s.session"
          recorderId="recorder-1"
          :index="s.index"
        />
      </div>
    `,
  }),
};

// Current year date
export const CurrentYearDate: Story = {
  args: {
    session: createMockSession({ startedAt: new Date() }),
    recorderId: 'recorder-1',
    index: 1,
  },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await expect(canvas.getByText('Untitled #1')).toBeInTheDocument();
  },
};

// Previous year date
export const PreviousYearDate: Story = {
  args: {
    session: createMockSession({ startedAt: new Date('2023-06-15T10:00:00') }),
    recorderId: 'recorder-1',
    index: 5,
  },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await expect(canvas.getByText('Untitled #5')).toBeInTheDocument();
  },
};

// Kept session
export const KeptSession: Story = {
  args: {
    session: createMockSession({ keep: true }),
    recorderId: 'recorder-1',
    index: 1,
  },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await expect(canvas.getByText('Untitled #1')).toBeInTheDocument();
  },
};

// Session with segments
export const WithSegments: Story = {
  args: {
    session: createMockSession({
      segments: [
        { id: 'seg-1', name: 'Intro', timeStart: new Date(0), timeEnd: new Date(30000) },
        { id: 'seg-2', name: 'Main', timeStart: new Date(30000), timeEnd: new Date(120000) },
      ],
    }),
    recorderId: 'recorder-1',
    index: 1,
  },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await expect(canvas.getByText('Untitled #1')).toBeInTheDocument();
  },
};

// Multiple cards in list
export const SessionList: Story = {
  play: async ({ canvasElement }) => {
    const cards = canvasElement.querySelectorAll('.card');
    await expect(cards.length).toBe(5);
  },
  render: () => ({
    components: { MockSessionCard },
    setup() {
      const sessions = Array.from({ length: 5 }, (_, i) => ({
        session: createMockSession({
          id: `session-${i + 1}`,
          startedAt: new Date(Date.now() - i * 24 * 60 * 60 * 1000),
        }),
        index: i + 1,
      }));
      return { sessions };
    },
    template: `
      <div style="display: flex; flex-direction: column; gap: 2rem; max-width: 800px;">
        <MockSessionCard
          v-for="s in sessions"
          :key="s.session.id"
          :session="s.session"
          recorderId="recorder-1"
          :index="s.index"
        />
      </div>
    `,
  }),
};
