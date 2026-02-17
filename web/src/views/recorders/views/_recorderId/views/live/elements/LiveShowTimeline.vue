<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue';
import type { Show, Act } from '@/types';

const props = defineProps<{
  show: Show;
}>();

const emit = defineEmits<{
  advanceAct: [];
}>();

const now = ref(new Date());
let interval: ReturnType<typeof setInterval>;

onMounted(() => {
  interval = setInterval(() => {
    now.value = new Date();
  }, 1000);
});

onUnmounted(() => {
  clearInterval(interval);
});

function actStatus(act: Act): 'upcoming' | 'active' | 'completed' {
  // Prefer actual timestamps set by AdvanceAct
  if (act.actualEnd) return 'completed';
  if (act.actualStart) return 'active';
  // Fall back to planned times for acts not yet touched
  const t = now.value.getTime();
  if (t < act.plannedStart.getTime()) return 'upcoming';
  if (t > act.plannedEnd.getTime()) return 'completed';
  return 'active';
}

const hasActiveAct = computed(() =>
  props.show.acts.some((a) => actStatus(a) === 'active'),
);

const allActsCompleted = computed(() =>
  props.show.acts.every((a) => actStatus(a) === 'completed'),
);

const nextActLabel = computed(() => {
  if (allActsCompleted.value) return null;
  if (!hasActiveAct.value) return 'Start First Act';
  const activeIdx = props.show.acts.findIndex((a) => actStatus(a) === 'active');
  if (activeIdx >= 0 && activeIdx + 1 < props.show.acts.length) {
    return `Next: ${props.show.acts[activeIdx + 1].name}`;
  }
  return 'End Act';
});

function formatTime(d: Date): string {
  return `${String(d.getHours()).padStart(2, '0')}:${String(d.getMinutes()).padStart(2, '0')}`;
}

function formatElapsed(): string {
  const diff = now.value.getTime() - props.show.acts[0]?.plannedStart.getTime();
  if (diff < 0) return '';
  const seconds = Math.floor(diff / 1000);
  const minutes = Math.floor(seconds / 60);
  const hours = Math.floor(minutes / 60);
  const pad = (n: number) => n.toString().padStart(2, '0');
  if (hours > 0) return `${pad(hours)}:${pad(minutes % 60)}:${pad(seconds % 60)}`;
  return `${pad(minutes)}:${pad(seconds % 60)}`;
}
</script>

<template>
  <div class="timeline">
    <div class="timeline-header">
      <h1>{{ show.name }}</h1>
      <div class="timeline-meta">
        <span class="live-badge">
          <span class="live-dot" />
          LIVE
        </span>
        <span v-if="formatElapsed()" class="elapsed">{{ formatElapsed() }}</span>
      </div>
    </div>

    <div class="acts">
      <div
        v-for="act in show.acts"
        :key="act.id"
        class="act"
        :class="actStatus(act)"
      >
        <div class="act-indicator">
          <div class="act-dot" />
          <div class="act-line" />
        </div>
        <div class="act-content">
          <div class="act-header">
            <span class="act-name">{{ act.name }}</span>
            <span v-if="actStatus(act) === 'active'" class="now-badge">NOW</span>
          </div>
          <span class="act-time">
            {{ formatTime(act.actualStart ?? act.plannedStart) }} – {{ formatTime(act.actualEnd ?? act.plannedEnd) }}
          </span>
        </div>
      </div>
    </div>

    <button
      v-if="nextActLabel"
      class="advance-btn"
      @click="emit('advanceAct')"
    >
      {{ nextActLabel }}
    </button>
  </div>
</template>

<style scoped>
.timeline {
  flex: 1;
  display: flex;
  flex-direction: column;
  padding: var(--size-6);
  max-width: 600px;
  margin: 0 auto;
  width: 100%;
}

.timeline-header {
  margin-bottom: var(--size-6);
}

h1 {
  font-size: 2.5rem;
  font-weight: var(--weight-bold);
  color: var(--text-primary);
  margin-bottom: var(--size-2);
}

.timeline-meta {
  display: flex;
  align-items: center;
  gap: var(--size-3);
}

.live-badge {
  display: flex;
  align-items: center;
  gap: var(--size-1);
  font-size: var(--scale-1);
  font-weight: var(--weight-bold);
  color: #dc2626;
}

.live-dot {
  width: 12px;
  height: 12px;
  border-radius: 50%;
  background: #dc2626;
  animation: pulse 1.5s ease-in-out infinite;
}

@keyframes pulse {
  0%,
  100% {
    opacity: 1;
  }
  50% {
    opacity: 0.4;
  }
}

.elapsed {
  font-family: monospace;
  font-size: var(--scale-2);
  color: var(--text-muted);
}

.acts {
  display: flex;
  flex-direction: column;
}

.act {
  display: flex;
  gap: var(--size-4);
  min-height: 80px;
}

.act-indicator {
  display: flex;
  flex-direction: column;
  align-items: center;
  width: 24px;
  flex-shrink: 0;
}

.act-dot {
  width: 16px;
  height: 16px;
  border-radius: 50%;
  border: 3px solid var(--color-grey-400);
  background: var(--bg-primary);
  flex-shrink: 0;
  margin-top: var(--size-1);
}

.act-line {
  width: 2px;
  flex: 1;
  background: var(--color-grey-300);
}

.act:last-child .act-line {
  display: none;
}

.act.active .act-dot {
  border-color: #dc2626;
  background: #dc2626;
  box-shadow: 0 0 0 4px rgba(220, 38, 38, 0.2);
}

.act.completed .act-dot {
  border-color: #16a34a;
  background: #16a34a;
}

.act.completed .act-line {
  background: #16a34a;
}

.act-content {
  flex: 1;
  padding-bottom: var(--size-4);
}

.act-header {
  display: flex;
  align-items: center;
  gap: var(--size-2);
  margin-bottom: var(--size-1);
}

.act-name {
  font-size: 1.5rem;
  font-weight: var(--weight-semibold);
  color: var(--text-primary);
}

.act.upcoming .act-name {
  color: var(--text-muted);
}

.now-badge {
  font-size: var(--scale-00);
  padding: 2px 8px;
  border-radius: var(--radius-xs);
  background: #dc2626;
  color: white;
  font-weight: var(--weight-bold);
  animation: pulse 1.5s ease-in-out infinite;
}

.act-time {
  font-size: var(--scale-1);
  color: var(--text-muted);
  font-family: monospace;
}

.advance-btn {
  margin-top: var(--size-4);
  padding: var(--size-3) var(--size-5);
  font-size: var(--scale-2);
  font-weight: var(--weight-bold);
  color: white;
  background: #dc2626;
  border: none;
  border-radius: var(--radius-sm);
  cursor: pointer;
  transition: background 0.15s ease;
}

.advance-btn:hover {
  background: #b91c1c;
}
</style>
