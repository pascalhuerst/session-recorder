<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { storeToRefs } from 'pinia';
import { useShowsStore } from '@/store/useShowsStore';
import { useRecordersStore } from '@/store/useRecordersStore';
import { toastService } from '@/services/Toaster/ToastService';
import type { Act } from '@/types';

const route = useRoute();
const router = useRouter();
const showsStore = useShowsStore();
const { selectedRecorderId } = storeToRefs(useRecordersStore());
const { shows } = storeToRefs(showsStore);

const showId = computed(() => route.params.showId as string);
const isNew = computed(() => showId.value === 'new');
const basePath = computed(
  () => `/recorders/${selectedRecorderId.value}/shows`,
);

const show = computed(() =>
  isNew.value ? null : shows.value.find((s) => s.id === showId.value),
);

// Form state for DRAFT / new show
const formName = ref('');
const formDate = ref('');
const formActs = ref<
  { name: string; plannedStart: string; plannedEnd: string; emails: string }[]
>([]);

// Initialize form when show changes
watch(
  show,
  (s) => {
    if (s && s.state === 'draft') {
      formName.value = s.name;
      formDate.value = s.date.toISOString().split('T')[0];
      formActs.value = s.acts.map((a) => ({
        name: a.name,
        plannedStart: toTimeString(a.plannedStart),
        plannedEnd: toTimeString(a.plannedEnd),
        emails: a.emails.join(', '),
      }));
    }
  },
  { immediate: true },
);

// Ended state: editable acts with actual times
const editedActs = ref<Act[]>([]);
watch(
  show,
  (s) => {
    if (s && s.state === 'ended') {
      editedActs.value = s.acts.map((a) => ({ ...a }));
    }
  },
  { immediate: true },
);

function toTimeString(d: Date): string {
  return `${String(d.getHours()).padStart(2, '0')}:${String(d.getMinutes()).padStart(2, '0')}`;
}

function addAct() {
  formActs.value.push({
    name: '',
    plannedStart: '',
    plannedEnd: '',
    emails: '',
  });
}

function removeAct(index: number) {
  formActs.value.splice(index, 1);
}

function parseActTime(dateStr: string, timeStr: string): Date {
  const d = new Date(dateStr);
  const [h, m] = timeStr.split(':').map(Number);
  d.setHours(h, m, 0, 0);
  return d;
}

async function saveShow() {
  try {
    const acts = formActs.value.map((a) => ({
      name: a.name,
      plannedStart: parseActTime(formDate.value, a.plannedStart),
      plannedEnd: parseActTime(formDate.value, a.plannedEnd),
      emails: a.emails
        .split(/[,;]/)
        .map((e) => e.trim())
        .filter(Boolean),
    }));

    if (isNew.value) {
      await showsStore.createShow({
        name: formName.value,
        date: new Date(formDate.value),
        recorderId: selectedRecorderId.value,
        acts,
      });
      toastService.success('Show created');
      router.push(basePath.value);
    } else {
      await showsStore.updateShow({
        id: showId.value,
        name: formName.value,
        date: new Date(formDate.value),
        recorderId: selectedRecorderId.value,
        acts: acts.map((a, i) => ({
          ...a,
          id: show.value?.acts[i]?.id ?? '',
          segmentId: null,
          actualStart: null,
          actualEnd: null,
        })),
      });
      toastService.success('Show updated');
    }
  } catch (e) {
    toastService.error(`Failed to save: ${e}`);
  }
}

async function handleStartShow() {
  try {
    await showsStore.startShow(showId.value);
    toastService.success('Show started — recording in progress');
  } catch (e) {
    toastService.error(`Failed to start: ${e}`);
  }
}

async function handleDelete() {
  try {
    await showsStore.deleteShow(showId.value);
    toastService.success('Show deleted');
    router.push(basePath.value);
  } catch (e) {
    toastService.error(`Failed to delete: ${e}`);
  }
}

async function handleUpdateEnded() {
  try {
    await showsStore.updateShow({
      id: showId.value,
      acts: editedActs.value,
    });
    toastService.success('Acts updated');
  } catch (e) {
    toastService.error(`Failed to update: ${e}`);
  }
}

async function handleRenderAll() {
  try {
    await showsStore.renderAllActs(showId.value);
    toastService.success('Rendering started for all acts');
  } catch (e) {
    toastService.error(`Failed to render: ${e}`);
  }
}

async function handleDistributeAll() {
  try {
    await showsStore.distributeAllActs(showId.value);
    toastService.success('Recordings distributed');
  } catch (e) {
    toastService.error(`Failed to distribute: ${e}`);
  }
}
</script>

<template>
  <!-- NEW SHOW -->
  <div v-if="isNew" class="show-detail">
    <h1>New Show</h1>
    <form @submit.prevent="saveShow" class="show-form">
      <label>
        <span>Name</span>
        <input v-model="formName" type="text" placeholder="Day 1 — Main Stage" required />
      </label>
      <label>
        <span>Date</span>
        <input v-model="formDate" type="date" required />
      </label>

      <div class="acts-section">
        <h2>Acts</h2>
        <div v-for="(act, i) in formActs" :key="i" class="act-row">
          <input v-model="act.name" placeholder="Act name" required />
          <input v-model="act.plannedStart" type="time" required />
          <input v-model="act.plannedEnd" type="time" required />
          <input v-model="act.emails" placeholder="email1, email2" />
          <button type="button" class="remove-btn" @click="removeAct(i)">×</button>
        </div>
        <button type="button" class="add-act-btn" @click="addAct">+ Add Act</button>
      </div>

      <div class="actions">
        <button type="submit" class="primary-btn">Save</button>
      </div>
    </form>
  </div>

  <!-- DRAFT -->
  <div v-else-if="show?.state === 'draft'" class="show-detail">
    <h1>{{ show.name }}</h1>
    <span class="badge draft">Draft</span>

    <form @submit.prevent="saveShow" class="show-form">
      <label>
        <span>Name</span>
        <input v-model="formName" type="text" required />
      </label>
      <label>
        <span>Date</span>
        <input v-model="formDate" type="date" required />
      </label>

      <div class="acts-section">
        <h2>Acts</h2>
        <div v-for="(act, i) in formActs" :key="i" class="act-row">
          <input v-model="act.name" placeholder="Act name" required />
          <input v-model="act.plannedStart" type="time" required />
          <input v-model="act.plannedEnd" type="time" required />
          <input v-model="act.emails" placeholder="email1, email2" />
          <button type="button" class="remove-btn" @click="removeAct(i)">×</button>
        </div>
        <button type="button" class="add-act-btn" @click="addAct">+ Add Act</button>
      </div>

      <div class="actions">
        <button type="submit" class="primary-btn">Save</button>
        <button type="button" class="start-btn" @click="handleStartShow">Start Show</button>
        <button type="button" class="danger-btn" @click="handleDelete">Delete</button>
      </div>
    </form>
  </div>

  <!-- LIVE -->
  <div v-else-if="show?.state === 'live'" class="show-detail">
    <h1>{{ show.name }}</h1>
    <span class="badge live">Live</span>
    <p class="live-info">Recording in progress. The show will end when the recording stops.</p>

    <div class="acts-list">
      <div v-for="act in show.acts" :key="act.id" class="act-card">
        <span class="act-name">{{ act.name }}</span>
        <span class="act-time">{{ toTimeString(act.plannedStart) }} – {{ toTimeString(act.plannedEnd) }}</span>
      </div>
    </div>
  </div>

  <!-- ENDED -->
  <div v-else-if="show?.state === 'ended'" class="show-detail">
    <h1>{{ show.name }}</h1>
    <span class="badge ended">Ended</span>

    <div class="acts-section">
      <h2>Adjust Act Times</h2>
      <div v-for="(act, i) in editedActs" :key="act.id" class="act-edit-row">
        <span class="act-name">{{ act.name }}</span>
        <label>
          <span>Start</span>
          <input
            type="time"
            :value="toTimeString(act.actualStart ?? act.plannedStart)"
            @input="editedActs[i].actualStart = parseActTime(show!.date.toISOString().split('T')[0], ($event.target as HTMLInputElement).value)"
          />
        </label>
        <label>
          <span>End</span>
          <input
            type="time"
            :value="toTimeString(act.actualEnd ?? act.plannedEnd)"
            @input="editedActs[i].actualEnd = parseActTime(show!.date.toISOString().split('T')[0], ($event.target as HTMLInputElement).value)"
          />
        </label>
        <span class="act-emails">{{ act.emails.join(', ') || 'No emails' }}</span>
        <span v-if="act.segmentId" class="act-segment-badge">Segment linked</span>
      </div>
    </div>

    <div class="actions">
      <button class="primary-btn" @click="handleUpdateEnded">Save Changes</button>
      <button class="start-btn" @click="handleRenderAll">Render All</button>
      <button class="primary-btn" @click="handleDistributeAll">Distribute All</button>
    </div>
  </div>

  <!-- ARCHIVED -->
  <div v-else-if="show?.state === 'archived'" class="show-detail">
    <h1>{{ show.name }}</h1>
    <span class="badge archived">Archived</span>

    <div class="acts-list">
      <div v-for="act in show.acts" :key="act.id" class="act-card">
        <span class="act-name">{{ act.name }}</span>
        <span class="act-time">
          {{ toTimeString(act.actualStart ?? act.plannedStart) }} –
          {{ toTimeString(act.actualEnd ?? act.plannedEnd) }}
        </span>
        <span class="act-emails">{{ act.emails.join(', ') }}</span>
      </div>
    </div>
  </div>

  <!-- NOT FOUND -->
  <div v-else class="show-detail">
    <p>Show not found.</p>
  </div>
</template>

<style scoped>
.show-detail {
  max-width: 800px;
}

h1 {
  font-size: var(--scale-3);
  font-weight: var(--weight-bold);
  margin-bottom: var(--size-2);
  color: var(--text-primary);
}

h2 {
  font-size: var(--scale-1);
  font-weight: var(--weight-semibold);
  margin-bottom: var(--size-3);
  color: var(--text-primary);
}

.badge {
  display: inline-block;
  font-size: var(--scale-00);
  padding: 2px 8px;
  border-radius: var(--radius-xs);
  text-transform: uppercase;
  font-weight: var(--weight-semibold);
  margin-bottom: var(--size-4);
}

.badge.draft {
  background: var(--color-grey-200);
  color: var(--color-grey-700);
}

.badge.live {
  background: #dc2626;
  color: white;
}

.badge.ended {
  background: #2563eb;
  color: white;
}

.badge.archived {
  background: var(--color-grey-300);
  color: var(--color-grey-600);
}

.show-form {
  display: flex;
  flex-direction: column;
  gap: var(--size-3);
}

.show-form label {
  display: flex;
  flex-direction: column;
  gap: var(--size-1);
}

.show-form label > span {
  font-size: var(--scale-0);
  font-weight: var(--weight-medium);
  color: var(--text-muted);
}

input,
select {
  padding: var(--size-2);
  border: 1px solid var(--border-primary);
  border-radius: var(--radius-sm);
  font-size: var(--scale-0);
  font-family: inherit;
  background: var(--bg-primary);
  color: var(--text-primary);
}

input:focus,
select:focus {
  outline: 2px solid #2563eb;
  outline-offset: -1px;
}

.acts-section {
  margin-top: var(--size-3);
}

.act-row {
  display: flex;
  gap: var(--size-2);
  align-items: center;
  margin-bottom: var(--size-2);
}

.act-row input {
  flex: 1;
}

.act-row input:first-child {
  flex: 2;
}

.act-edit-row {
  display: flex;
  gap: var(--size-3);
  align-items: center;
  padding: var(--size-2) 0;
  border-bottom: 1px solid var(--border-primary);
}

.act-edit-row label {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.act-edit-row label > span {
  font-size: var(--scale-00);
  color: var(--text-muted);
}

.act-name {
  font-weight: var(--weight-medium);
  min-width: 150px;
}

.act-time,
.act-emails {
  font-size: var(--scale-0);
  color: var(--text-muted);
}

.act-segment-badge {
  font-size: var(--scale-00);
  padding: 2px 6px;
  border-radius: var(--radius-xs);
  background: #16a34a;
  color: white;
}

.remove-btn {
  width: 28px;
  height: 28px;
  border: none;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--text-muted);
  font-size: var(--scale-2);
  cursor: pointer;
  flex-shrink: 0;
}

.remove-btn:hover {
  background: #dc2626;
  color: white;
}

.add-act-btn {
  padding: var(--size-2);
  border: 1px dashed var(--border-primary);
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--text-muted);
  font-family: inherit;
  cursor: pointer;
}

.add-act-btn:hover {
  border-color: var(--text-primary);
  color: var(--text-primary);
}

.actions {
  display: flex;
  gap: var(--size-2);
  margin-top: var(--size-4);
}

.primary-btn,
.start-btn,
.danger-btn {
  padding: var(--size-2) var(--size-4);
  border: none;
  border-radius: var(--radius-sm);
  font-family: inherit;
  font-size: var(--scale-0);
  font-weight: var(--weight-medium);
  cursor: pointer;
  transition: opacity 0.15s ease;
}

.primary-btn {
  background: #2563eb;
  color: white;
}

.start-btn {
  background: #16a34a;
  color: white;
}

.danger-btn {
  background: #dc2626;
  color: white;
}

.primary-btn:hover,
.start-btn:hover,
.danger-btn:hover {
  opacity: 0.9;
}

.live-info {
  color: var(--text-muted);
  margin-bottom: var(--size-4);
}

.acts-list {
  display: flex;
  flex-direction: column;
  gap: var(--size-2);
}

.act-card {
  display: flex;
  gap: var(--size-3);
  align-items: center;
  padding: var(--size-3);
  border: 1px solid var(--border-primary);
  border-radius: var(--radius-sm);
}
</style>
