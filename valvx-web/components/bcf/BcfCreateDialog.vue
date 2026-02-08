<script setup lang="ts">
/**
 * BcfCreateDialog — Dialog for creating new BCF topics with viewpoint.
 * Captures the current 3D viewer state as a BCF viewpoint.
 */
import { ref, reactive } from 'vue'
import type { BcfCreateTopicRequest } from '~/types/bcf'

const props = defineProps<{
  visible: boolean
  viewpointData?: any
  snapshotBase64?: string | null
}>()

const emit = defineEmits<{
  close: []
  submit: [data: BcfCreateTopicRequest]
}>()

const form = reactive({
  title: '',
  description: '',
  priority: 'Normal' as BcfCreateTopicRequest['priority'],
  topicType: 'Issue' as BcfCreateTopicRequest['topicType'],
  assignedTo: '',
  labels: '',
})

const isSubmitting = ref(false)

async function handleSubmit() {
  if (!form.title.trim()) return

  isSubmitting.value = true

  const data: BcfCreateTopicRequest = {
    title: form.title.trim(),
    description: form.description.trim() || undefined,
    priority: form.priority,
    topicType: form.topicType,
    assignedTo: form.assignedTo || undefined,
    labels: form.labels ? form.labels.split(',').map((l) => l.trim()).filter(Boolean) : undefined,
    viewpoint: props.viewpointData
      ? {
          ...props.viewpointData,
          snapshotBase64: props.snapshotBase64 || undefined,
        }
      : undefined,
  }

  emit('submit', data)
  isSubmitting.value = false
  resetForm()
}

function resetForm() {
  form.title = ''
  form.description = ''
  form.priority = 'Normal'
  form.topicType = 'Issue'
  form.assignedTo = ''
  form.labels = ''
}

function handleClose() {
  resetForm()
  emit('close')
}
</script>

<template>
  <Teleport to="body">
    <div v-if="visible" class="dialog-overlay" @click.self="handleClose">
      <div class="dialog">
        <div class="dialog-header">
          <h3>Create Issue</h3>
          <button class="btn btn-icon btn-sm" @click="handleClose">&times;</button>
        </div>

        <form class="dialog-body" @submit.prevent="handleSubmit">
          <!-- Snapshot preview -->
          <div v-if="snapshotBase64" class="snapshot-preview">
            <img :src="snapshotBase64" alt="Viewpoint snapshot" />
            <div class="snapshot-badge">Viewpoint captured</div>
          </div>

          <!-- Title -->
          <div class="field">
            <label class="label">Title *</label>
            <input
              v-model="form.title"
              class="input"
              placeholder="Describe the issue..."
              required
              autofocus
            />
          </div>

          <!-- Description -->
          <div class="field">
            <label class="label">Description</label>
            <textarea
              v-model="form.description"
              class="input"
              placeholder="Detailed description of the issue..."
              rows="3"
            />
          </div>

          <!-- Priority + Type row -->
          <div class="field-row">
            <div class="field" style="flex: 1">
              <label class="label">Priority</label>
              <select v-model="form.priority" class="input">
                <option value="Critical">Critical</option>
                <option value="Major">Major</option>
                <option value="Normal">Normal</option>
                <option value="Minor">Minor</option>
              </select>
            </div>
            <div class="field" style="flex: 1">
              <label class="label">Type</label>
              <select v-model="form.topicType" class="input">
                <option value="Issue">Issue</option>
                <option value="Request">Request</option>
                <option value="Comment">Comment</option>
                <option value="Clash">Clash</option>
              </select>
            </div>
          </div>

          <!-- Labels -->
          <div class="field">
            <label class="label">Labels (comma separated)</label>
            <input
              v-model="form.labels"
              class="input"
              placeholder="architecture, structural, review..."
            />
          </div>

          <!-- Actions -->
          <div class="dialog-actions">
            <button type="button" class="btn" @click="handleClose">Cancel</button>
            <button
              type="submit"
              class="btn btn-primary"
              :disabled="!form.title.trim() || isSubmitting"
            >
              {{ isSubmitting ? 'Creating...' : 'Create Issue' }}
            </button>
          </div>
        </form>
      </div>
    </div>
  </Teleport>
</template>

<style scoped>
.dialog-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.6);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 100;
  backdrop-filter: blur(4px);
}

.dialog {
  background: var(--color-bg-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-lg);
  width: 520px;
  max-width: 90vw;
  max-height: 85vh;
  overflow-y: auto;
}

.dialog-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 20px;
  border-bottom: 1px solid var(--color-border);
}
.dialog-header h3 {
  font-size: 16px;
  font-weight: 600;
}

.dialog-body {
  padding: 20px;
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.snapshot-preview {
  position: relative;
  border-radius: var(--radius-md);
  overflow: hidden;
  border: 1px solid var(--color-border);
}
.snapshot-preview img {
  width: 100%;
  height: 160px;
  object-fit: cover;
}
.snapshot-badge {
  position: absolute;
  bottom: 8px;
  left: 8px;
  padding: 2px 8px;
  background: rgba(66, 190, 101, 0.9);
  border-radius: var(--radius-sm);
  font-size: 11px;
  font-weight: 600;
  color: white;
}

.field {
  display: flex;
  flex-direction: column;
}

.field-row {
  display: flex;
  gap: 12px;
}

.dialog-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  padding-top: 8px;
  border-top: 1px solid var(--color-border);
}
</style>
