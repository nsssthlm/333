<script setup lang="ts">
/**
 * FileUploader — Fast chunked upload component using TUS protocol.
 *
 * Features:
 * - Drag & drop + click to browse
 * - Parallel uploads (3 concurrent)
 * - 5 MB chunks for fast transfer
 * - Automatic resume on failure
 * - Real-time speed and ETA display
 * - Pause/resume/retry per file
 */
import { ref, computed } from 'vue'
import { useFileUpload } from '~/composables/useFileUpload'

const props = defineProps<{
  folderId: string
  accept?: string
}>()

const emit = defineEmits<{
  uploaded: [fileId: string, tusUrl: string]
  allComplete: []
}>()

const { files, batchState, isUploading, addFiles, pauseUpload, resumeUpload, retryUpload, removeUpload, clearCompleted, cancelAll } = useFileUpload()

const isDragOver = ref(false)
const fileInput = ref<HTMLInputElement | null>(null)

function handleDrop(e: DragEvent) {
  isDragOver.value = false
  const droppedFiles = Array.from(e.dataTransfer?.files || [])
  if (droppedFiles.length > 0) {
    addFiles(droppedFiles, props.folderId)
  }
}

function handleFileSelect(e: Event) {
  const input = e.target as HTMLInputElement
  const selectedFiles = Array.from(input.files || [])
  if (selectedFiles.length > 0) {
    addFiles(selectedFiles, props.folderId)
  }
  input.value = ''
}

function openFilePicker() {
  fileInput.value?.click()
}

function formatSize(bytes: number): string {
  if (bytes === 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB']
  const k = 1024
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return `${(bytes / Math.pow(k, i)).toFixed(1)} ${units[i]}`
}

function formatSpeed(bytesPerSec: number): string {
  return `${formatSize(bytesPerSec)}/s`
}

function formatEta(seconds: number): string {
  if (seconds <= 0 || !isFinite(seconds)) return '--'
  if (seconds < 60) return `${Math.ceil(seconds)}s`
  if (seconds < 3600) return `${Math.floor(seconds / 60)}m ${Math.ceil(seconds % 60)}s`
  return `${Math.floor(seconds / 3600)}h ${Math.floor((seconds % 3600) / 60)}m`
}

const statusIcon: Record<string, string> = {
  queued: '...',
  uploading: '',
  paused: '||',
  complete: '',
  error: '!',
  processing: '',
}
</script>

<template>
  <div class="uploader">
    <!-- Drop zone -->
    <div
      class="drop-zone"
      :class="{ 'drag-over': isDragOver, 'has-files': files.length > 0 }"
      @dragover.prevent="isDragOver = true"
      @dragleave="isDragOver = false"
      @drop.prevent="handleDrop"
      @click="openFilePicker"
    >
      <input
        ref="fileInput"
        type="file"
        multiple
        :accept="accept || '.ifc,.pdf,.bcf,.dwg,.rvt,.nwd,.obj,.fbx,.glb,.gltf'"
        style="display: none"
        @change="handleFileSelect"
      />
      <div class="drop-zone-content">
        <svg width="32" height="32" viewBox="0 0 32 32" fill="none">
          <path d="M16 4v18M10 10l6-6 6 6" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
          <path d="M4 22v4a2 2 0 002 2h20a2 2 0 002-2v-4" stroke="currentColor" stroke-width="2" stroke-linecap="round"/>
        </svg>
        <p>Drop files here or <span class="link">browse</span></p>
        <p class="text-xs text-muted">IFC, PDF, BCF, DWG, and more</p>
      </div>
    </div>

    <!-- Overall progress -->
    <div v-if="files.length > 0" class="upload-summary">
      <div class="flex items-center justify-between mb-2">
        <span class="text-sm font-medium">
          {{ files.filter(f => f.status === 'complete').length }}/{{ files.length }} files
        </span>
        <span class="text-xs text-muted">
          {{ formatSize(batchState.uploadedSize) }} / {{ formatSize(batchState.totalSize) }}
        </span>
      </div>
      <div class="progress-bar">
        <div
          class="progress-bar-fill"
          :class="{ complete: batchState.overallProgress >= 100 }"
          :style="{ width: `${batchState.overallProgress}%` }"
        />
      </div>
      <div v-if="isUploading" class="flex items-center justify-between mt-2">
        <span class="text-xs text-muted">{{ formatSpeed(batchState.speed) }}</span>
        <span class="text-xs text-muted">ETA: {{ formatEta(batchState.eta) }}</span>
      </div>
    </div>

    <!-- File list -->
    <div v-if="files.length > 0" class="file-list">
      <div
        v-for="file in files"
        :key="file.id"
        class="file-item"
        :class="file.status"
      >
        <div class="file-icon">
          <span class="file-ext">{{ file.ext || '?' }}</span>
        </div>
        <div class="file-info">
          <div class="file-name truncate">{{ file.name }}</div>
          <div class="file-meta flex items-center gap-2">
            <span class="text-xs text-muted">{{ formatSize(file.size) }}</span>
            <span
              v-if="file.status === 'uploading'"
              class="text-xs"
              style="color: var(--color-primary)"
            >
              {{ Math.round(file.progress) }}%
            </span>
            <span
              v-else-if="file.status === 'complete'"
              class="text-xs"
              style="color: var(--color-success)"
            >
              Done
            </span>
            <span
              v-else-if="file.status === 'error'"
              class="text-xs"
              style="color: var(--color-danger)"
            >
              {{ file.error || 'Failed' }}
            </span>
            <span
              v-else-if="file.status === 'paused'"
              class="text-xs text-muted"
            >
              Paused
            </span>
          </div>
          <div v-if="file.status === 'uploading'" class="progress-bar mt-2">
            <div
              class="progress-bar-fill"
              :style="{ width: `${file.progress}%` }"
            />
          </div>
        </div>
        <div class="file-actions">
          <button
            v-if="file.status === 'uploading'"
            class="btn btn-icon btn-sm"
            title="Pause"
            @click.stop="pauseUpload(file.id)"
          >||</button>
          <button
            v-if="file.status === 'paused'"
            class="btn btn-icon btn-sm"
            title="Resume"
            @click.stop="resumeUpload(file.id)"
          >&#9654;</button>
          <button
            v-if="file.status === 'error'"
            class="btn btn-icon btn-sm"
            title="Retry"
            @click.stop="retryUpload(file.id)"
          >&#8635;</button>
          <button
            class="btn btn-icon btn-sm btn-danger"
            title="Remove"
            @click.stop="removeUpload(file.id)"
          >&times;</button>
        </div>
      </div>
    </div>

    <!-- Batch actions -->
    <div v-if="files.length > 0" class="upload-actions flex gap-2">
      <button
        v-if="files.some(f => f.status === 'complete')"
        class="btn btn-sm"
        @click="clearCompleted"
      >
        Clear completed
      </button>
      <button
        v-if="isUploading"
        class="btn btn-sm btn-danger"
        @click="cancelAll"
      >
        Cancel all
      </button>
    </div>
  </div>
</template>

<style scoped>
.uploader {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.drop-zone {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 32px;
  border: 2px dashed var(--color-border);
  border-radius: var(--radius-md);
  cursor: pointer;
  transition: all var(--transition);
  background: transparent;
}
.drop-zone:hover,
.drop-zone.drag-over {
  border-color: var(--color-primary);
  background: rgba(69, 137, 255, 0.05);
}
.drop-zone.has-files {
  padding: 16px;
}
.drop-zone-content {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  color: var(--color-text-secondary);
}
.drop-zone-content .link {
  color: var(--color-primary);
  text-decoration: underline;
}

.upload-summary {
  padding: 12px;
  background: var(--color-bg-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
}

.file-list {
  display: flex;
  flex-direction: column;
  gap: 4px;
  max-height: 400px;
  overflow-y: auto;
}

.file-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 12px;
  background: var(--color-bg-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  transition: background var(--transition);
}
.file-item:hover {
  background: var(--color-bg-hover);
}
.file-item.complete {
  border-color: rgba(66, 190, 101, 0.3);
}
.file-item.error {
  border-color: rgba(255, 77, 79, 0.3);
}

.file-icon {
  width: 36px;
  height: 36px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--color-bg);
  border-radius: var(--radius-sm);
  flex-shrink: 0;
}
.file-ext {
  font-size: 10px;
  font-weight: 700;
  text-transform: uppercase;
  color: var(--color-primary);
}

.file-info {
  flex: 1;
  min-width: 0;
}
.file-name {
  font-size: 13px;
  font-weight: 500;
}
.file-meta {
  margin-top: 2px;
}

.file-actions {
  display: flex;
  gap: 4px;
  flex-shrink: 0;
}

.upload-actions {
  justify-content: flex-end;
}
</style>
