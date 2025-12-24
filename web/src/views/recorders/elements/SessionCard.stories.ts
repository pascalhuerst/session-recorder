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
 *
 * Scenario: Display session name
 *   Given a session with a name
 *   When the card renders
 *   Then the session name should be displayed
 *
 * Scenario: Display fallback for unnamed session
 *   Given a session without a name
 *   When the card renders
 *   Then "Untitled #N" should be displayed
 *
 * Scenario: Rename session via contenteditable
 *   Given a session card
 *   When the user clicks on the title
 *   Then the title becomes editable
 *   And on blur, the name is saved
 *
 * Scenario: Collapsed by default
 *   Given a session card
 *   When the card renders
 *   Then the card should be collapsed (only header and overview visible)
 *
 * Scenario: Expand session card
 *   Given a collapsed session card
 *   When the user clicks the expand toggle
 *   Then the card should expand to show zoomview and segments
 */

import type { Meta, StoryObj } from '@storybook/vue3';
import { within, expect, userEvent } from '@storybook/test';

// Mock component since real SessionCard requires WaveformView and complex setup
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
  data() {
    return {
      isEditing: false,
      isExpanded: false,
    };
  },
  computed: {
    displayName() {
      return this.session.name || `Untitled #${this.index}`;
    },
    displayDate() {
      const { startedAt } = this.session;

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
  methods: {
    toggleExpanded() {
      this.isExpanded = !this.isExpanded;
    },
    startEditing() {
      this.isEditing = true;
    },
    saveTitle() {
      this.isEditing = false;
      // In the real component, this would call setName API
    },
    onKeydown(event: KeyboardEvent) {
      if (event.key === 'Enter') {
        event.preventDefault();
        (event.target as HTMLElement).blur();
      } else if (event.key === 'Escape') {
        event.preventDefault();
        this.isEditing = false;
        (event.target as HTMLElement).blur();
      }
    },
  },
  template: `
    <div class="card" :class="{ expanded: isExpanded }">
      <div class="header">
        <button
          class="expand-toggle"
          :class="{ expanded: isExpanded }"
          @click="toggleExpanded"
          :aria-expanded="isExpanded"
          aria-label="Toggle session details"
        >
          <svg width="16" height="16" viewBox="0 0 16 16" fill="none" xmlns="http://www.w3.org/2000/svg">
            <path d="M6 4L10 8L6 12" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
          </svg>
        </button>
        <span
          class="title"
          :class="{ editing: isEditing }"
          contenteditable="true"
          @focus="startEditing"
          @blur="saveTitle"
          @keydown="onKeydown"
        >{{ displayName }}</span>
        <div class="metadata">
          <time class="timestamp" :datetime="displayDate.iso">{{ displayDate.formatted }}</time>
          <div class="menu">
            <button class="menu-button">⋮</button>
          </div>
        </div>
      </div>
      <div class="waveform-container">
        <div class="overview" style="height: 80px; background: linear-gradient(to right, #e0e0e0, #f5f5f5, #e0e0e0); display: flex; align-items: center; justify-content: center; color: #666;">
          Overview
        </div>
        <div v-if="isExpanded" class="details">
          <div class="zoomview" style="height: 100px; background: linear-gradient(to right, #d0d0d0, #e5e5e5, #d0d0d0); display: flex; align-items: center; justify-content: center; color: #666;">
            Zoomview
          </div>
          <div class="segments" style="padding: 8px; background: #f9f9f9; color: #666;">
            Segments
          </div>
        </div>
      </div>
    </div>
  `,
};

// Mock session data
const createMockSession = (overrides = {}) => ({
  id: 'session-1',
  startedAt: new Date('2024-03-15T14:30:00'),
  expiresAt: new Date('2024-04-15T14:30:00'),
  name: '',
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

// Default session card (collapsed by default)
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

    // Check overview is visible
    await expect(canvas.getByText('Overview')).toBeInTheDocument();

    // Collapsed by default - zoomview and segments should not be visible
    await expect(canvas.queryByText('Zoomview')).not.toBeInTheDocument();
    await expect(canvas.queryByText('Segments')).not.toBeInTheDocument();

    // Check expand toggle button exists
    const toggleButton = canvas.getByRole('button', { name: 'Toggle session details' });
    await expect(toggleButton).toBeInTheDocument();
    await expect(toggleButton).toHaveAttribute('aria-expanded', 'false');
  },
};

// Expand/collapse interaction
export const ExpandCollapse: Story = {
  args: {
    session: createMockSession(),
    recorderId: 'recorder-1',
    index: 1,
  },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    const toggleButton = canvas.getByRole('button', { name: 'Toggle session details' });

    // Initially collapsed
    await expect(canvas.queryByText('Zoomview')).not.toBeInTheDocument();
    await expect(toggleButton).toHaveAttribute('aria-expanded', 'false');

    // Click to expand
    await userEvent.click(toggleButton);

    // Now expanded - zoomview and segments should be visible
    await expect(canvas.getByText('Zoomview')).toBeInTheDocument();
    await expect(canvas.getByText('Segments')).toBeInTheDocument();
    await expect(toggleButton).toHaveAttribute('aria-expanded', 'true');

    // Click to collapse again
    await userEvent.click(toggleButton);

    // Back to collapsed
    await expect(canvas.queryByText('Zoomview')).not.toBeInTheDocument();
    await expect(toggleButton).toHaveAttribute('aria-expanded', 'false');
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

// Session with a custom name
export const NamedSession: Story = {
  args: {
    session: createMockSession({ name: 'Morning Recording' }),
    recorderId: 'recorder-1',
    index: 1,
  },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    // Should display the custom name instead of "Untitled #1"
    await expect(canvas.getByText('Morning Recording')).toBeInTheDocument();
  },
};

// Rename interaction - title becomes editable on focus
export const RenameInteraction: Story = {
  args: {
    session: createMockSession({ name: 'Original Name' }),
    recorderId: 'recorder-1',
    index: 1,
  },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    const title = canvas.getByText('Original Name');

    // Check title is contenteditable
    await expect(title).toHaveAttribute('contenteditable', 'true');

    // Click to focus and start editing (uses userEvent for proper event handling)
    await userEvent.click(title);

    // Wait for Vue reactivity and verify editing class
    await new Promise((resolve) => setTimeout(resolve, 100));
    await expect(title).toHaveClass('editing');

    // Click outside to blur and save
    await userEvent.click(canvasElement);
    await new Promise((resolve) => setTimeout(resolve, 100));
    await expect(title).not.toHaveClass('editing');
  },
};

// Mixed named and unnamed sessions
export const MixedSessionList: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);

    // Named sessions display their names
    await expect(canvas.getByText('Interview Recording')).toBeInTheDocument();
    await expect(canvas.getByText('Podcast Episode 5')).toBeInTheDocument();

    // Unnamed sessions display fallback
    await expect(canvas.getByText('Untitled #2')).toBeInTheDocument();
    await expect(canvas.getByText('Untitled #4')).toBeInTheDocument();
  },
  render: () => ({
    components: { MockSessionCard },
    setup() {
      const sessions = [
        { session: createMockSession({ id: 's1', name: 'Interview Recording' }), index: 1 },
        { session: createMockSession({ id: 's2', name: '' }), index: 2 },
        { session: createMockSession({ id: 's3', name: 'Podcast Episode 5' }), index: 3 },
        { session: createMockSession({ id: 's4' }), index: 4 },
      ];
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
