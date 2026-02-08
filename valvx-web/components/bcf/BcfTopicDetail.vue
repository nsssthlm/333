<script setup lang="ts">
/**
 * BcfTopicDetail — Shows a single BCF topic with comments and viewpoints.
 * Allows adding comments and navigating to viewpoints in the 3D viewer.
 */
import { ref, onMounted, watch } from 'vue'
import { useBcf } from '~/composables/useBcf'
import type { BcfTopic, BcfViewpoint } from '~/types/bcf'

const props = defineProps<{
  projectId: string
  topicId: string
}>()

const emit = defineEmits<{
  back: []
  restoreViewpoint: [viewpoint: BcfViewpoint]
  topicUpdated: [topic: BcfTopic]
}>()

const { currentTopic, isLoading, error, fetchTopic, addComment, updateTopic, deleteTopic } = useBcf(props.projectId)

const newComment = ref('')
const isSubmittingComment = ref(false)

async function handleAddComment() {
  if (!newComment.value.trim()) return
  isSubmittingComment.value = true
  await addComment(props.topicId, { body: newComment.value.trim() })
  newComment.value = ''
  isSubmittingComment.value = false
}

async function handleStatusChange(status: string) {
  const updated = await updateTopic(props.topicId, { topicType: undefined })
  if (updated) {
    // Workaround: update status through dedicated endpoint
    emit('topicUpdated', updated)
  }
}

async function handleDelete() {
  if (confirm('Delete this issue? This cannot be undone.')) {
    await deleteTopic(props.topicId)
    emit('back')
  }
}

function formatDate(d: string) {
  return new Date(d).toLocaleString('sv-SE')
}

function handleViewpointClick(vp: BcfViewpoint) {
  emit('restoreViewpoint', vp)
}

onMounted(() => fetchTopic(props.topicId))
watch(() => props.topicId, () => fetchTopic(props.topicId))
</script>

<template>
  <div class="topic-detail">
    <!-- Header -->
    <div class="detail-header">
      <button class="btn btn-icon btn-sm" @click="emit('back')">
        <svg width="16" height="16" viewBox="0 0 16 16" fill="none">
          <path d="M10 4L6 8l4 4" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/>
        </svg>
      </button>
      <h3 v-if="currentTopic" class="truncate" style="flex: 1">{{ currentTopic.title }}</h3>
      <button class="btn btn-sm btn-danger" @click="handleDelete">Delete</button>
    </div>

    <div v-if="isLoading" class="flex justify-center p-4">
      <div class="spinner" />
    </div>

    <div v-else-if="currentTopic" class="detail-content overflow-auto">
      <!-- Meta -->
      <div class="detail-meta p-4">
        <div class="flex items-center gap-2 mb-2">
          <span
            class="badge"
            :class="currentTopic.topicStatus === 'Closed' ? 'badge-closed' : 'badge-open'"
          >
            {{ currentTopic.topicStatus || 'Open' }}
          </span>
          <span v-if="currentTopic.priority" class="badge" :class="`badge-${currentTopic.priority?.toLowerCase()}`">
            {{ currentTopic.priority }}
          </span>
          <span v-if="currentTopic.topicType" class="text-xs text-muted">
            {{ currentTopic.topicType }}
          </span>
        </div>
        <p v-if="currentTopic.description" class="text-sm" style="color: var(--color-text-secondary); line-height: 1.6">
          {{ currentTopic.description }}
        </p>
        <div class="text-xs text-muted mt-2">
          Created {{ formatDate(currentTopic.createdAt) }}
          <span v-if="currentTopic.creatorName"> by {{ currentTopic.creatorName }}</span>
        </div>
        <div v-if="currentTopic.labels?.length" class="flex gap-1 mt-2">
          <span
            v-for="label in currentTopic.labels"
            :key="label"
            class="label-tag"
          >{{ label }}</span>
        </div>
      </div>

      <!-- Viewpoints -->
      <div v-if="currentTopic.viewpoints?.length" class="detail-section">
        <div class="section-title">Viewpoints ({{ currentTopic.viewpoints.length }})</div>
        <div class="viewpoint-grid">
          <div
            v-for="vp in currentTopic.viewpoints"
            :key="vp.id"
            class="viewpoint-card"
            @click="handleViewpointClick(vp)"
          >
            <img
              v-if="vp.snapshotBase64"
              :src="vp.snapshotBase64"
              alt="Viewpoint"
            />
            <div v-else class="viewpoint-placeholder">
              <svg width="24" height="24" viewBox="0 0 24 24" fill="none">
                <circle cx="12" cy="12" r="10" stroke="currentColor" stroke-width="1.5"/>
                <path d="M12 8v4l3 3" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/>
              </svg>
            </div>
            <div class="viewpoint-label text-xs">
              {{ vp.cameraType === 'perspective' ? 'Perspective' : 'Ortho' }}
            </div>
          </div>
        </div>
      </div>

      <!-- Comments -->
      <div class="detail-section">
        <div class="section-title">
          Comments ({{ currentTopic.comments?.length || 0 }})
        </div>
        <div class="comments-list">
          <div
            v-for="comment in currentTopic.comments"
            :key="comment.id"
            class="comment-item"
          >
            <div class="comment-header flex items-center justify-between">
              <span class="text-xs font-medium">{{ comment.authorName || 'User' }}</span>
              <span class="text-xs text-muted">{{ formatDate(comment.createdAt) }}</span>
            </div>
            <p class="comment-body text-sm">{{ comment.body }}</p>
          </div>
          <div v-if="!currentTopic.comments?.length" class="text-xs text-muted p-3">
            No comments yet
          </div>
        </div>

        <!-- Add comment -->
        <div class="add-comment">
          <textarea
            v-model="newComment"
            class="input"
            placeholder="Add a comment..."
            rows="2"
            @keydown.meta.enter="handleAddComment"
            @keydown.ctrl.enter="handleAddComment"
          />
          <div class="flex justify-between items-center mt-2">
            <span class="text-xs text-muted">Cmd+Enter to send</span>
            <button
              class="btn btn-sm btn-primary"
              :disabled="!newComment.trim() || isSubmittingComment"
              @click="handleAddComment"
            >
              {{ isSubmittingComment ? 'Sending...' : 'Comment' }}
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.topic-detail {
  display: flex;
  flex-direction: column;
  height: 100%;
  background: var(--color-bg-surface);
}

.detail-header {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px 16px;
  border-bottom: 1px solid var(--color-border);
}
.detail-header h3 {
  font-size: 15px;
  font-weight: 600;
}

.detail-content {
  flex: 1;
  overflow-y: auto;
}

.detail-meta {
  border-bottom: 1px solid var(--color-border-light);
}

.detail-section {
  border-bottom: 1px solid var(--color-border-light);
}
.section-title {
  padding: 10px 16px;
  font-size: 12px;
  font-weight: 600;
  color: var(--color-text-secondary);
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.label-tag {
  padding: 2px 8px;
  background: var(--color-bg);
  border-radius: 10px;
  font-size: 11px;
  color: var(--color-text-secondary);
}

/* Viewpoints grid */
.viewpoint-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(100px, 1fr));
  gap: 8px;
  padding: 0 16px 12px;
}
.viewpoint-card {
  position: relative;
  border-radius: var(--radius-sm);
  overflow: hidden;
  cursor: pointer;
  border: 1px solid var(--color-border);
  transition: border-color var(--transition);
  aspect-ratio: 4/3;
}
.viewpoint-card:hover {
  border-color: var(--color-primary);
}
.viewpoint-card img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}
.viewpoint-placeholder {
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--color-bg);
  color: var(--color-text-muted);
}
.viewpoint-label {
  position: absolute;
  bottom: 4px;
  left: 4px;
  padding: 1px 6px;
  background: rgba(0, 0, 0, 0.7);
  border-radius: var(--radius-sm);
  color: white;
}

/* Comments */
.comments-list {
  padding: 0 16px;
}
.comment-item {
  padding: 10px 0;
  border-bottom: 1px solid var(--color-border-light);
}
.comment-item:last-child {
  border-bottom: none;
}
.comment-body {
  margin-top: 4px;
  color: var(--color-text);
  line-height: 1.5;
}

.add-comment {
  padding: 12px 16px;
}
</style>
