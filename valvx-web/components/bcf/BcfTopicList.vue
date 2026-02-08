<script setup lang="ts">
/**
 * BcfTopicList — Lists BCF topics for a project with filtering.
 * Provides the main interface for browsing BIM collaboration issues.
 */
import { ref, onMounted, watch } from 'vue'
import { useBcf } from '~/composables/useBcf'
import type { BcfTopic } from '~/types/bcf'

const props = defineProps<{
  projectId: string
}>()

const emit = defineEmits<{
  selectTopic: [topic: BcfTopic]
  createTopic: []
  restoreViewpoint: [viewpoint: any]
}>()

const { topics, openTopics, closedTopics, isLoading, error, fetchTopics, downloadBcfExport } = useBcf(props.projectId)

const filter = ref<'all' | 'open' | 'closed'>('all')
const searchQuery = ref('')

const filteredTopics = computed(() => {
  let list = filter.value === 'open' ? openTopics.value
    : filter.value === 'closed' ? closedTopics.value
    : topics.value

  if (searchQuery.value) {
    const q = searchQuery.value.toLowerCase()
    list = list.filter(
      (t) =>
        t.title.toLowerCase().includes(q) ||
        t.description?.toLowerCase().includes(q)
    )
  }
  return list
})

function priorityClass(p?: string) {
  switch (p) {
    case 'Critical': return 'badge-critical'
    case 'Major': return 'badge-major'
    case 'Normal': return 'badge-normal'
    case 'Minor': return 'badge-minor'
    default: return 'badge-normal'
  }
}

function statusClass(s?: string) {
  switch (s) {
    case 'Closed': return 'badge-closed'
    case 'InProgress': return 'badge-inprogress'
    default: return 'badge-open'
  }
}

function formatDate(d: string) {
  return new Date(d).toLocaleDateString('sv-SE')
}

function handleQuickViewpoint(topic: BcfTopic) {
  if (topic.viewpoints?.length) {
    emit('restoreViewpoint', topic.viewpoints[0])
  }
}

onMounted(() => fetchTopics())
watch(() => props.projectId, () => fetchTopics())
</script>

<template>
  <div class="bcf-list">
    <!-- Header -->
    <div class="bcf-header">
      <h3>Issues</h3>
      <div class="flex gap-2">
        <button class="btn btn-sm" @click="downloadBcfExport" title="Export BCF">
          <svg width="14" height="14" viewBox="0 0 14 14" fill="none">
            <path d="M7 2v8M4 7l3 3 3-3" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/>
            <path d="M2 11h10" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/>
          </svg>
          Export
        </button>
        <button class="btn btn-sm btn-primary" @click="emit('createTopic')">
          + New Issue
        </button>
      </div>
    </div>

    <!-- Filters -->
    <div class="bcf-filters flex gap-2 items-center">
      <input
        v-model="searchQuery"
        class="input"
        placeholder="Search issues..."
        style="flex: 1"
      />
      <div class="filter-tabs">
        <button
          class="filter-tab"
          :class="{ active: filter === 'all' }"
          @click="filter = 'all'"
        >All ({{ topics.length }})</button>
        <button
          class="filter-tab"
          :class="{ active: filter === 'open' }"
          @click="filter = 'open'"
        >Open ({{ openTopics.length }})</button>
        <button
          class="filter-tab"
          :class="{ active: filter === 'closed' }"
          @click="filter = 'closed'"
        >Closed ({{ closedTopics.length }})</button>
      </div>
    </div>

    <!-- Loading -->
    <div v-if="isLoading" class="bcf-loading">
      <div class="spinner" />
    </div>

    <!-- Error -->
    <div v-else-if="error" class="bcf-error text-sm" style="color: var(--color-danger); padding: 16px">
      {{ error }}
    </div>

    <!-- Empty state -->
    <div v-else-if="filteredTopics.length === 0" class="bcf-empty">
      <p class="text-muted">No issues found</p>
      <button class="btn btn-sm btn-primary mt-2" @click="emit('createTopic')">
        Create first issue
      </button>
    </div>

    <!-- Topic list -->
    <div v-else class="topic-list">
      <div
        v-for="topic in filteredTopics"
        :key="topic.id"
        class="topic-item"
        @click="emit('selectTopic', topic)"
      >
        <div class="topic-snapshot" @click.stop="handleQuickViewpoint(topic)">
          <img
            v-if="topic.viewpoints?.[0]?.snapshotBase64"
            :src="topic.viewpoints[0].snapshotBase64"
            alt="Viewpoint"
          />
          <div v-else class="snapshot-placeholder">
            <svg width="20" height="20" viewBox="0 0 20 20" fill="none">
              <rect x="2" y="4" width="16" height="12" rx="2" stroke="currentColor" stroke-width="1.5"/>
              <circle cx="7" cy="9" r="2" stroke="currentColor" stroke-width="1.5"/>
              <path d="M18 14l-4-4-3 3-2-2-5 5" stroke="currentColor" stroke-width="1.5"/>
            </svg>
          </div>
        </div>
        <div class="topic-content">
          <div class="topic-title">{{ topic.title }}</div>
          <div class="topic-meta flex items-center gap-2 mt-2">
            <span class="badge" :class="statusClass(topic.topicStatus)">
              {{ topic.topicStatus || 'Open' }}
            </span>
            <span v-if="topic.priority" class="badge" :class="priorityClass(topic.priority)">
              {{ topic.priority }}
            </span>
            <span class="text-xs text-muted">{{ formatDate(topic.createdAt) }}</span>
          </div>
          <div v-if="topic.description" class="topic-desc text-xs text-secondary mt-2 truncate">
            {{ topic.description }}
          </div>
        </div>
        <div class="topic-stats">
          <span v-if="topic.comments?.length" class="text-xs text-muted">
            {{ topic.comments.length }} comments
          </span>
          <span v-if="topic.viewpoints?.length" class="text-xs text-muted">
            {{ topic.viewpoints.length }} viewpoints
          </span>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.bcf-list {
  display: flex;
  flex-direction: column;
  height: 100%;
  background: var(--color-bg-surface);
}

.bcf-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 16px;
  border-bottom: 1px solid var(--color-border);
}
.bcf-header h3 {
  font-size: 15px;
  font-weight: 600;
}

.bcf-filters {
  padding: 8px 16px;
  border-bottom: 1px solid var(--color-border);
}

.filter-tabs {
  display: flex;
  background: var(--color-bg);
  border-radius: var(--radius-sm);
  overflow: hidden;
}
.filter-tab {
  padding: 4px 10px;
  font-size: 11px;
  font-weight: 500;
  color: var(--color-text-muted);
  background: transparent;
  border: none;
  cursor: pointer;
  white-space: nowrap;
  transition: all var(--transition);
}
.filter-tab:hover {
  color: var(--color-text);
}
.filter-tab.active {
  background: var(--color-primary);
  color: white;
}

.bcf-loading {
  display: flex;
  justify-content: center;
  padding: 32px;
}

.bcf-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 32px;
}

.topic-list {
  flex: 1;
  overflow-y: auto;
}

.topic-item {
  display: flex;
  gap: 12px;
  padding: 12px 16px;
  border-bottom: 1px solid var(--color-border-light);
  cursor: pointer;
  transition: background var(--transition);
}
.topic-item:hover {
  background: var(--color-bg-hover);
}

.topic-snapshot {
  width: 64px;
  height: 48px;
  border-radius: var(--radius-sm);
  overflow: hidden;
  flex-shrink: 0;
  background: var(--color-bg);
  cursor: pointer;
}
.topic-snapshot img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}
.snapshot-placeholder {
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--color-text-muted);
}

.topic-content {
  flex: 1;
  min-width: 0;
}
.topic-title {
  font-size: 13px;
  font-weight: 600;
  line-height: 1.3;
}
.topic-desc {
  max-width: 300px;
}

.topic-stats {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 4px;
  flex-shrink: 0;
}
</style>
