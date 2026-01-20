<script setup lang="ts">
import { usePeaksContext } from '../../context/usePeaksContext';
import Segment from './Segment.vue';
import Checkbox from '../../lib/controls/Checkbox.vue';
import Button from '../../lib/controls/Button.vue';
import AddSegmentButton from './controls/AddSegmentButton.vue';
import { computed, ref, watch } from 'vue';

const { state, commandEmitter, eventEmitter } = usePeaksContext();
const segments = computed(() => state.toRef().value.segments);
const permissions = computed(() => state.toRef().value.permissions);

// Selection state
const selectedIds = ref<Set<string>>(new Set());

const isSelected = (id: string) => selectedIds.value.has(id);

const toggleSelection = (id: string) => {
  const newSet = new Set(selectedIds.value);
  if (newSet.has(id)) {
    newSet.delete(id);
  } else {
    newSet.add(id);
  }
  selectedIds.value = newSet;
};

const selectableSegments = computed(() =>
  segments.value.filter((s) => !s.deleted)
);

const allSelected = computed(
  () =>
    selectableSegments.value.length > 0 &&
    selectableSegments.value.every((s) => selectedIds.value.has(s.id))
);

const someSelected = computed(
  () =>
    selectableSegments.value.some((s) => selectedIds.value.has(s.id)) &&
    !allSelected.value
);

const toggleSelectAll = () => {
  if (allSelected.value) {
    selectedIds.value = new Set();
  } else {
    selectedIds.value = new Set(selectableSegments.value.map((s) => s.id));
  }
};

const clearSelection = () => {
  selectedIds.value = new Set();
};

// Computed for bulk actions
const selectedSegments = computed(() =>
  segments.value.filter((s) => selectedIds.value.has(s.id))
);

const selectedCount = computed(() => selectedIds.value.size);

const canBulkRender = computed(() =>
  selectedSegments.value.some(
    (s) => s.state !== 'queued' && s.state !== 'rendering' && s.state !== 'finished' && !s.deleted
  )
);

const canBulkDelete = computed(
  () => permissions.value.delete && selectedCount.value > 0
);

// Bulk actions
const handleBulkRender = () => {
  selectedSegments.value
    .filter((s) => s.state !== 'queued' && s.state !== 'rendering' && s.state !== 'finished')
    .forEach((s) => {
      commandEmitter.emit('renderSegment', s.id);
    });
  clearSelection();
};

const handleBulkDelete = () => {
  const count = selectedSegments.value.length;
  selectedSegments.value.forEach((s) => {
    commandEmitter.emit('removeSegment', s.id);
  });
  clearSelection();
  eventEmitter.emit('segmentsBulkDeleted', count);
};

// Clean up selection when segments are removed
watch(
  segments,
  (newSegments) => {
    const validIds = new Set(newSegments.map((s) => s.id));
    const newSelected = new Set(
      [...selectedIds.value].filter((id) => validIds.has(id))
    );
    if (newSelected.size !== selectedIds.value.size) {
      selectedIds.value = newSelected;
    }
  },
  { deep: true }
);
</script>

<template>
  <div class="segments">
    <table v-if="selectableSegments.length > 0">
      <colgroup>
        <col class="col-checkbox" />
        <col class="col-time" />
        <col class="col-time" />
        <col class="col-label" />
        <col class="col-actions" />
      </colgroup>
      <thead>
        <tr>
          <th class="col-checkbox">
            <div>
              <Checkbox
                :modelValue="allSelected"
                :indeterminate="someSelected"
                size="sm"
                @update:modelValue="toggleSelectAll"
                @click.stop
              />
            </div>
          </th>
          <th colspan="4" class="col-bulk-actions">
            <Transition name="fade">
              <div v-if="selectedCount > 0" class="bulk-actions-bar">
                <span class="bulk-actions-bar__count">
                  {{ selectedCount }} selected
                </span>
                <div class="bulk-actions-bar__actions">
                  <Button
                    size="xs"
                    color="primary"
                    variant="solid"
                    :disabled="!canBulkRender"
                    @click="handleBulkRender"
                  >
                    Render
                  </Button>
                  <Button
                    size="xs"
                    variant="outlined"
                    :disabled="!canBulkDelete"
                    @click="handleBulkDelete"
                  >
                    Delete
                  </Button>
                  <Button size="xs" variant="ghost" @click="clearSelection">
                    Clear
                  </Button>
                </div>
              </div>
            </Transition>
          </th>
        </tr>
      </thead>
      <tbody>
        <Segment
          v-for="segment in selectableSegments"
          :key="segment.id"
          :segment="segment"
          :selected="isSelected(segment.id)"
          @toggle-selection="toggleSelection(segment.id)"
        />
      </tbody>
    </table>
    <div v-if="permissions.create" class="segments__footer">
      <AddSegmentButton />
    </div>
  </div>
</template>

<style scoped>
.segments {
  position: relative;
}

.segments__footer {
  display: flex;
  justify-content: flex-start;
  padding: var(--size-2);
}

table {
  width: 100%;
  border-collapse: collapse;
  table-layout: fixed;
}

thead tr {
  border-bottom: 1px solid var(--color-grey-300);
}

th {
  padding: var(--size-2);
  text-align: left;
  font-weight: var(--weight-medium);
  font-size: var(--scale-0);
  color: var(--color-grey-500);
}

.col-checkbox {
  width: 40px;
  vertical-align: middle;
}

.col-time {
  width: 140px;
}

.col-label {
  /* flexible width */
}

.col-actions {
  width: 160px;
}

.col-bulk-actions {
  padding: 0 !important;
  height: var(--size-12);
  vertical-align: middle;
}

.col-checkbox > div {
  display: flex;
  align-items: center;
  justify-content: center;
}

:deep(td) {
  padding: var(--size-2);
  border-bottom: 1px solid var(--color-grey-300);
  vertical-align: middle;
}

:deep(tr:last-child td) {
  border: none;
}

/* Bulk Actions Bar */
.bulk-actions-bar {
  display: flex;
  align-items: center;
  gap: var(--size-3);
  padding: var(--size-2);
}

.bulk-actions-bar__count {
  font-size: var(--scale-0);
  font-weight: var(--weight-medium);
  color: var(--color-grey-700);
}

.bulk-actions-bar__actions {
  display: flex;
  gap: var(--size-2);
}

/* Fade animation */
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.15s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style>

<i18n locale="en" lang="json"></i18n>
