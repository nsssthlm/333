<script setup lang="ts">
/**
 * Main 3D Viewer page — The core of ValvX.
 *
 * Integrates:
 * - That Open Engine for 3D IFC model display (client-side WASM parsing)
 * - BCF topic list + creation
 * - File upload panel
 */
import { ref, onMounted } from 'vue'
import { useBcf } from '~/composables/useBcf'
import type { IfcModel } from '~/composables/useIfcViewer'
import type { BcfTopic, BcfViewpoint, BcfCreateTopicRequest } from '~/types/bcf'

const route = useRoute()
const config = useRuntimeConfig()
const projectId = route.params.projectId as string

// BCF composable
const bcf = useBcf(projectId)

// State
const models = ref<IfcModel[]>([])
const showBcfPanel = ref(true)
const showUploadPanel = ref(false)
const showBcfCreate = ref(false)
const selectedTopic = ref<BcfTopic | null>(null)
const pendingViewpoint = ref<any>(null)
const pendingSnapshot = ref<string | null>(null)
const viewerRef = ref<any>(null)

// Current folder for uploads (root folder)
const currentFolderId = ref('root')

// Load project models from API — returns direct MinIO/S3 download URLs
async function loadProjectModels() {
  try {
    const response = await fetch(
      `${config.public.apiBaseUrl}/api/projects/${projectId}/models`,
      { credentials: 'include' }
    )
    if (response.ok) {
      const data = await response.json()
      models.value = data
        .filter((m: any) => m.fileExt === 'ifc')
        .map((m: any) => ({
          fileVersionId: m.fileVersionId,
          fileName: m.fileName,
          url: `${config.public.apiBaseUrl}/api/files/${m.fileVersionId}/download`,
          visible: true,
          loading: false,
        }))
    }
  } catch (err) {
    console.error('Failed to load project models:', err)
  }
}

// BCF handlers
function handleViewpointCaptured(viewpoint: any) {
  pendingViewpoint.value = viewpoint
  pendingSnapshot.value = viewpoint.snapshotBase64 || null
  showBcfCreate.value = true
}

async function handleBcfSubmit(data: BcfCreateTopicRequest) {
  const topic = await bcf.createTopic(data)
  if (topic) {
    showBcfCreate.value = false
    pendingViewpoint.value = null
    pendingSnapshot.value = null
  }
}

function handleSelectTopic(topic: BcfTopic) {
  selectedTopic.value = topic
}

function handleRestoreViewpoint(viewpoint: BcfViewpoint) {
  viewerRef.value?.restoreViewpoint(viewpoint)
}

function handleBackFromTopic() {
  selectedTopic.value = null
}

function handleObjectSelected(objectId: string, properties: Record<string, unknown> | null) {
  // Could display properties panel
}

onMounted(() => {
  loadProjectModels()
  bcf.fetchTopics()
})
</script>

<template>
  <div class="viewer-page">
    <!-- 3D Viewer (main area) -->
    <div class="viewer-area">
      <SpeckleModelViewer
        ref="viewerRef"
        :project-id="projectId"
        :models="models"
        @viewpoint-captured="handleViewpointCaptured"
        @object-selected="handleObjectSelected"
      />
    </div>

    <!-- Right sidebar -->
    <div class="viewer-sidebar">
      <!-- Sidebar tabs -->
      <div class="sidebar-tabs">
        <button
          class="sidebar-tab"
          :class="{ active: showBcfPanel && !showUploadPanel }"
          @click="showBcfPanel = true; showUploadPanel = false"
        >
          Issues
        </button>
        <button
          class="sidebar-tab"
          :class="{ active: showUploadPanel }"
          @click="showUploadPanel = true; showBcfPanel = false"
        >
          Upload
        </button>
      </div>

      <!-- BCF Panel -->
      <div v-if="showBcfPanel && !showUploadPanel" class="sidebar-content">
        <BcfTopicDetail
          v-if="selectedTopic"
          :project-id="projectId"
          :topic-id="selectedTopic.id"
          @back="handleBackFromTopic"
          @restore-viewpoint="handleRestoreViewpoint"
        />
        <BcfTopicList
          v-else
          :project-id="projectId"
          @select-topic="handleSelectTopic"
          @create-topic="showBcfCreate = true"
          @restore-viewpoint="handleRestoreViewpoint"
        />
      </div>

      <!-- Upload Panel -->
      <div v-if="showUploadPanel" class="sidebar-content p-4">
        <h3 style="font-size: 15px; font-weight: 600; margin-bottom: 16px">Upload Files</h3>
        <FileUploader
          :folder-id="currentFolderId"
          @uploaded="loadProjectModels"
        />
      </div>
    </div>

    <!-- BCF Create Dialog -->
    <BcfCreateDialog
      :visible="showBcfCreate"
      :viewpoint-data="pendingViewpoint"
      :snapshot-base64="pendingSnapshot"
      @close="showBcfCreate = false"
      @submit="handleBcfSubmit"
    />
  </div>
</template>

<style scoped>
.viewer-page {
  display: flex;
  height: 100%;
  overflow: hidden;
}

.viewer-area {
  flex: 1;
  position: relative;
  min-width: 0;
}

.viewer-sidebar {
  width: 380px;
  background: var(--color-bg-surface);
  border-left: 1px solid var(--color-border);
  display: flex;
  flex-direction: column;
  flex-shrink: 0;
}

.sidebar-tabs {
  display: flex;
  border-bottom: 1px solid var(--color-border);
}
.sidebar-tab {
  flex: 1;
  padding: 10px;
  font-size: 13px;
  font-weight: 600;
  color: var(--color-text-muted);
  background: transparent;
  border: none;
  cursor: pointer;
  transition: all var(--transition);
  border-bottom: 2px solid transparent;
}
.sidebar-tab:hover {
  color: var(--color-text);
}
.sidebar-tab.active {
  color: var(--color-primary);
  border-bottom-color: var(--color-primary);
}

.sidebar-content {
  flex: 1;
  overflow: hidden;
}
</style>
