<script setup lang="ts">
/**
 * IfcModelViewer — Main 3D viewer component.
 * Uses That Open Engine (@thatopen/components) with client-side
 * IFC parsing via web-ifc WASM. No server-side conversion needed.
 *
 * Features:
 * - Direct IFC file loading from MinIO/S3
 * - Three.js rendering with smooth orbit controls
 * - Object selection with IFC property display
 * - Section planes (double-click to create, right-click to delete)
 * - BCF viewpoint capture for issue creation
 */
import { ref, watch, onMounted } from 'vue'
import { useIfcViewer, type IfcModel } from '~/composables/useIfcViewer'

const props = defineProps<{
  projectId: string
  models?: IfcModel[]
}>()

const emit = defineEmits<{
  objectSelected: [objectId: string, properties: Record<string, unknown> | null]
  viewpointCaptured: [viewpoint: any]
  modelLoaded: [model: IfcModel]
}>()

const viewerContainer = ref<HTMLElement | null>(null)

const {
  isInitialized,
  isLoading,
  loadingProgress,
  loadedModels,
  selectedObjectIds,
  selectedObjectProperties,
  error,
  init,
  loadModel,
  unloadModel,
  toggleModelVisibility,
  fitAll,
  setTopView,
  setFrontView,
  togglePerspective,
  toggleSectionTool,
  toggleMeasurements,
  captureViewpoint,
  restoreViewpoint,
  resetFilters,
} = useIfcViewer(viewerContainer)

// Initialize viewer when container is ready
onMounted(() => {
  init()
})

// Watch for model list changes
watch(
  () => props.models,
  async (newModels) => {
    if (!newModels || !isInitialized.value) return
    for (const model of newModels) {
      await loadModel(model)
      emit('modelLoaded', model)
    }
  },
  { deep: true }
)

// Forward selection events
watch(selectedObjectIds, (ids) => {
  if (ids.length > 0) {
    emit('objectSelected', ids[0], selectedObjectProperties.value)
  }
})

// Toolbar state
const showLayers = ref(true)
const showProperties = ref(false)
const sectionActive = ref(false)
const measureActive = ref(false)

function handleToggleSection() {
  sectionActive.value = !sectionActive.value
  toggleSectionTool()
}

function handleToggleMeasure() {
  measureActive.value = !measureActive.value
  toggleMeasurements()
}

function handleCaptureBcf() {
  try {
    const viewpoint = captureViewpoint()
    emit('viewpointCaptured', viewpoint)
  } catch (err) {
    console.error('Failed to capture viewpoint:', err)
  }
}

defineExpose({
  captureViewpoint,
  restoreViewpoint,
  fitAll,
  loadModel,
  unloadModel,
  resetFilters,
  loadedModels,
})
</script>

<template>
  <div class="viewer-wrapper">
    <!-- Loading overlay -->
    <div v-if="isLoading" class="viewer-loading">
      <div class="viewer-loading-content">
        <div class="spinner" />
        <span>Loading IFC model...</span>
        <div class="progress-bar" style="width: 200px">
          <div
            class="progress-bar-fill"
            :style="{ width: `${loadingProgress}%` }"
          />
        </div>
      </div>
    </div>

    <!-- Error banner -->
    <div v-if="error" class="viewer-error">
      {{ error }}
      <button class="btn btn-sm" @click="error = null">Dismiss</button>
    </div>

    <!-- Toolbar -->
    <div class="viewer-toolbar">
      <div class="toolbar-group">
        <button
          class="btn btn-icon btn-sm"
          title="Fit all"
          @click="fitAll"
        >
          <svg width="16" height="16" viewBox="0 0 16 16" fill="none">
            <rect x="2" y="2" width="12" height="12" rx="1" stroke="currentColor" stroke-width="1.5"/>
            <path d="M5 8h6M8 5v6" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/>
          </svg>
        </button>
        <button class="btn btn-icon btn-sm" title="Top view" @click="setTopView">T</button>
        <button class="btn btn-icon btn-sm" title="Front view" @click="setFrontView">F</button>
        <button class="btn btn-icon btn-sm" title="Toggle perspective" @click="togglePerspective">P</button>
      </div>

      <div class="toolbar-divider" />

      <div class="toolbar-group">
        <button
          class="btn btn-icon btn-sm"
          :class="{ active: sectionActive }"
          title="Section plane (double-click to place)"
          @click="handleToggleSection"
        >
          <svg width="16" height="16" viewBox="0 0 16 16" fill="none">
            <path d="M2 8h12" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-dasharray="2 2"/>
            <rect x="3" y="3" width="10" height="10" rx="1" stroke="currentColor" stroke-width="1.5" opacity="0.4"/>
          </svg>
        </button>
        <button
          class="btn btn-icon btn-sm"
          :class="{ active: measureActive }"
          title="Measure"
          @click="handleToggleMeasure"
        >
          <svg width="16" height="16" viewBox="0 0 16 16" fill="none">
            <path d="M3 13L13 3" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/>
            <circle cx="3" cy="13" r="1.5" fill="currentColor"/>
            <circle cx="13" cy="3" r="1.5" fill="currentColor"/>
          </svg>
        </button>
        <button class="btn btn-icon btn-sm" title="Reset selection" @click="resetFilters">
          <svg width="16" height="16" viewBox="0 0 16 16" fill="none">
            <path d="M2 4h12M4 8h8M6 12h4" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/>
          </svg>
        </button>
      </div>

      <div class="toolbar-divider" />

      <div class="toolbar-group">
        <button class="btn btn-sm btn-primary" title="Create BCF issue" @click="handleCaptureBcf">
          <svg width="14" height="14" viewBox="0 0 14 14" fill="none">
            <circle cx="7" cy="7" r="6" stroke="currentColor" stroke-width="1.5"/>
            <path d="M7 4v6M4 7h6" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/>
          </svg>
          Issue
        </button>
      </div>

      <div style="flex: 1" />

      <div class="toolbar-group">
        <button
          class="btn btn-icon btn-sm"
          :class="{ active: showLayers }"
          title="Toggle layers panel"
          @click="showLayers = !showLayers"
        >
          <svg width="16" height="16" viewBox="0 0 16 16" fill="none">
            <path d="M8 2L2 5.5L8 9L14 5.5L8 2Z" stroke="currentColor" stroke-width="1.5" stroke-linejoin="round"/>
            <path d="M2 8l6 3.5L14 8" stroke="currentColor" stroke-width="1.5" stroke-linejoin="round"/>
            <path d="M2 10.5L8 14l6-3.5" stroke="currentColor" stroke-width="1.5" stroke-linejoin="round"/>
          </svg>
        </button>
        <button
          class="btn btn-icon btn-sm"
          :class="{ active: showProperties }"
          title="Toggle properties"
          @click="showProperties = !showProperties"
        >
          <svg width="16" height="16" viewBox="0 0 16 16" fill="none">
            <rect x="2" y="3" width="12" height="10" rx="1" stroke="currentColor" stroke-width="1.5"/>
            <path d="M5 6h6M5 8.5h4" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/>
          </svg>
        </button>
      </div>
    </div>

    <!-- 3D Canvas -->
    <div ref="viewerContainer" class="viewer-canvas" />

    <!-- Layers panel -->
    <div v-if="showLayers && loadedModels.length > 0" class="viewer-panel layers-panel">
      <div class="panel-header">
        Models
        <span class="text-muted text-xs">({{ loadedModels.length }})</span>
      </div>
      <div class="panel-body" style="padding: 8px">
        <div
          v-for="model in loadedModels"
          :key="model.fileVersionId"
          class="layer-item"
          @click="toggleModelVisibility(model.fileVersionId)"
        >
          <div
            class="layer-visibility"
            :class="{ hidden: !model.visible }"
          >
            <svg v-if="model.visible" width="14" height="14" viewBox="0 0 14 14" fill="none">
              <path d="M1 7s2.5-4.5 6-4.5S13 7 13 7s-2.5 4.5-6 4.5S1 7 1 7z" stroke="currentColor" stroke-width="1.2"/>
              <circle cx="7" cy="7" r="2" stroke="currentColor" stroke-width="1.2"/>
            </svg>
            <svg v-else width="14" height="14" viewBox="0 0 14 14" fill="none">
              <path d="M2 2l10 10M1 7s2.5-4.5 6-4.5c1 0 1.8.3 2.5.7M13 7s-1 1.8-2.8 3" stroke="currentColor" stroke-width="1.2" stroke-linecap="round"/>
            </svg>
          </div>
          <span class="layer-name truncate">{{ model.fileName }}</span>
          <div v-if="model.loading" class="spinner-sm" />
        </div>
      </div>
    </div>

    <!-- Properties panel -->
    <div v-if="showProperties && selectedObjectProperties" class="viewer-panel properties-panel">
      <div class="panel-header">
        Properties
        <button class="btn btn-icon btn-sm" @click="showProperties = false">&times;</button>
      </div>
      <div class="panel-body overflow-auto" style="max-height: 400px; padding: 8px">
        <div
          v-for="(value, key) in selectedObjectProperties"
          :key="String(key)"
          class="prop-row"
        >
          <span class="prop-key">{{ key }}</span>
          <span class="prop-value">{{ value }}</span>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.viewer-wrapper {
  position: relative;
  width: 100%;
  height: 100%;
  background: var(--color-bg);
  overflow: hidden;
}

.viewer-canvas {
  width: 100%;
  height: 100%;
}
.viewer-canvas :deep(canvas) {
  width: 100% !important;
  height: 100% !important;
  outline: none;
}

.viewer-loading {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(15, 17, 23, 0.8);
  z-index: 50;
}
.viewer-loading-content {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
  color: var(--color-text-secondary);
  font-size: 13px;
}

.viewer-error {
  position: absolute;
  top: 56px;
  left: 50%;
  transform: translateX(-50%);
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 8px 16px;
  background: rgba(255, 77, 79, 0.15);
  border: 1px solid rgba(255, 77, 79, 0.3);
  border-radius: var(--radius-md);
  color: var(--color-danger);
  font-size: 13px;
  z-index: 40;
}

/* Toolbar */
.viewer-toolbar {
  position: absolute;
  top: 12px;
  left: 12px;
  right: 12px;
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 4px;
  background: var(--color-bg-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  z-index: 30;
  box-shadow: var(--shadow-md);
}
.toolbar-group {
  display: flex;
  align-items: center;
  gap: 2px;
}
.toolbar-divider {
  width: 1px;
  height: 24px;
  background: var(--color-border);
  margin: 0 4px;
}
.viewer-toolbar .btn-icon.active {
  background: var(--color-primary);
  border-color: var(--color-primary);
  color: white;
}

/* Side panels */
.viewer-panel {
  position: absolute;
  background: var(--color-bg-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  box-shadow: var(--shadow-lg);
  z-index: 20;
  min-width: 220px;
  max-width: 300px;
}
.layers-panel {
  top: 60px;
  right: 12px;
}
.properties-panel {
  bottom: 12px;
  right: 12px;
}

.layer-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 8px;
  border-radius: var(--radius-sm);
  cursor: pointer;
  font-size: 12px;
  transition: background var(--transition);
}
.layer-item:hover {
  background: var(--color-bg-hover);
}
.layer-visibility {
  color: var(--color-primary);
  flex-shrink: 0;
}
.layer-visibility.hidden {
  color: var(--color-text-muted);
}
.layer-name {
  flex: 1;
}

.prop-row {
  display: flex;
  justify-content: space-between;
  gap: 8px;
  padding: 4px 0;
  border-bottom: 1px solid var(--color-border-light);
  font-size: 11px;
}
.prop-key {
  color: var(--color-text-secondary);
  flex-shrink: 0;
}
.prop-value {
  color: var(--color-text);
  text-align: right;
  word-break: break-all;
}

/* Spinners */
.spinner {
  width: 24px;
  height: 24px;
  border: 2px solid var(--color-border);
  border-top-color: var(--color-primary);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}
.spinner-sm {
  width: 14px;
  height: 14px;
  border: 1.5px solid var(--color-border);
  border-top-color: var(--color-primary);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}
@keyframes spin {
  to { transform: rotate(360deg); }
}
</style>
