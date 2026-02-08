/**
 * Speckle Viewer composable — replaces the iframe integration with a
 * deeply embedded Three.js-based viewer using @speckle/viewer SDK.
 *
 * Manages viewer lifecycle, model loading/unloading, camera control,
 * object selection, section planes, and BCF viewpoint capture/restore.
 */
import { ref, shallowRef, onMounted, onBeforeUnmount, type Ref } from 'vue'
import type {
  LoadedModel,
  SectionPlane,
  SpeckleFileMapping,
} from '~/types/speckle'
import type { BcfViewpoint, BcfComponents, BcfClippingPlane } from '~/types/bcf'

let ViewerModule: any = null
let viewerInitPromise: Promise<void> | null = null

async function ensureViewerModule() {
  if (ViewerModule) return
  if (viewerInitPromise) {
    await viewerInitPromise
    return
  }
  viewerInitPromise = (async () => {
    ViewerModule = await import('@speckle/viewer')
  })()
  await viewerInitPromise
}

export function useSpeckleViewer(containerRef: Ref<HTMLElement | null>) {
  const config = useRuntimeConfig()
  const speckleBaseUrl = config.public.speckleBaseUrl as string
  const apiBaseUrl = config.public.apiBaseUrl as string

  const viewer = shallowRef<any>(null)
  const cameraController = shallowRef<any>(null)
  const selectionExtension = shallowRef<any>(null)
  const filteringExtension = shallowRef<any>(null)
  const sectionTool = shallowRef<any>(null)
  const measurementsTool = shallowRef<any>(null)

  const isInitialized = ref(false)
  const isLoading = ref(false)
  const loadingProgress = ref(0)
  const loadedModels = ref<LoadedModel[]>([])
  const selectedObjectIds = ref<string[]>([])
  const selectedObjectProperties = ref<Record<string, unknown> | null>(null)
  const sectionPlanes = ref<SectionPlane[]>([])
  const error = ref<string | null>(null)

  async function init() {
    if (!containerRef.value || isInitialized.value) return

    try {
      await ensureViewerModule()

      const {
        Viewer,
        DefaultViewerParams,
        SpeckleLoader,
        CameraController,
        SelectionExtension,
        FilteringExtension,
        MeasurementsExtension,
        SectionTool,
        SectionOutlines,
      } = ViewerModule

      const params = DefaultViewerParams
      params.showStats = false
      params.verbose = false

      const v = new Viewer(containerRef.value, params)
      await v.init()

      const cam = v.createExtension(CameraController)
      const sel = v.createExtension(SelectionExtension)
      const filt = v.createExtension(FilteringExtension)
      const sec = v.createExtension(SectionTool)
      v.createExtension(SectionOutlines)

      let meas: any = null
      try {
        meas = v.createExtension(MeasurementsExtension)
      } catch {
        // Measurements may not be available in all versions
      }

      // Selection event handler
      sel.on('select', (event: any) => {
        if (event?.hits?.length > 0) {
          const hit = event.hits[0]
          selectedObjectIds.value = [hit.node?.model?.raw?.id || hit.id]
          selectedObjectProperties.value = hit.node?.model?.raw || null
        } else {
          selectedObjectIds.value = []
          selectedObjectProperties.value = null
        }
      })

      viewer.value = v
      cameraController.value = cam
      selectionExtension.value = sel
      filteringExtension.value = filt
      sectionTool.value = sec
      measurementsTool.value = meas
      isInitialized.value = true
    } catch (err: any) {
      error.value = `Failed to initialize viewer: ${err.message}`
      console.error('Viewer init error:', err)
    }
  }

  async function loadModel(model: LoadedModel, token?: string) {
    if (!viewer.value) return

    const existing = loadedModels.value.find(
      (m) => m.fileVersionId === model.fileVersionId
    )
    if (existing && !existing.loading) return

    try {
      isLoading.value = true
      loadingProgress.value = 0

      const modelEntry: LoadedModel = {
        ...model,
        loading: true,
        visible: true,
      }

      const idx = loadedModels.value.findIndex(
        (m) => m.fileVersionId === model.fileVersionId
      )
      if (idx >= 0) {
        loadedModels.value[idx] = modelEntry
      } else {
        loadedModels.value.push(modelEntry)
      }

      const { SpeckleLoader, UrlHelper } = ViewerModule

      const urls = await UrlHelper.getResourceUrls(model.url, token)
      for (const url of urls) {
        const loader = new SpeckleLoader(viewer.value.getWorldTree(), url, token)
        await viewer.value.loadObject(loader, true)
      }

      const mi = loadedModels.value.findIndex(
        (m) => m.fileVersionId === model.fileVersionId
      )
      if (mi >= 0) {
        loadedModels.value[mi] = { ...loadedModels.value[mi], loading: false }
      }
    } catch (err: any) {
      error.value = `Failed to load model ${model.fileName}: ${err.message}`
      console.error('Model load error:', err)

      const mi = loadedModels.value.findIndex(
        (m) => m.fileVersionId === model.fileVersionId
      )
      if (mi >= 0) {
        loadedModels.value[mi] = { ...loadedModels.value[mi], loading: false }
      }
    } finally {
      isLoading.value = loadedModels.value.some((m) => m.loading)
      loadingProgress.value = 100
    }
  }

  async function unloadModel(fileVersionId: string) {
    if (!viewer.value) return

    const model = loadedModels.value.find(
      (m) => m.fileVersionId === fileVersionId
    )
    if (!model) return

    try {
      await viewer.value.unloadObject(model.url)
      loadedModels.value = loadedModels.value.filter(
        (m) => m.fileVersionId !== fileVersionId
      )
    } catch (err: any) {
      console.error('Unload error:', err)
    }
  }

  function toggleModelVisibility(fileVersionId: string) {
    const model = loadedModels.value.find(
      (m) => m.fileVersionId === fileVersionId
    )
    if (!model || !filteringExtension.value) return

    model.visible = !model.visible
    // Use filtering extension to hide/show objects from this model
    // This is a simplified approach - actual filtering depends on tree structure
  }

  function fitAll() {
    cameraController.value?.setCameraView({ type: 'fit' }, true)
  }

  function setTopView() {
    cameraController.value?.setCameraView({ type: 'top' }, true)
  }

  function setFrontView() {
    cameraController.value?.setCameraView({ type: 'front' }, true)
  }

  function togglePerspective() {
    if (!cameraController.value) return
    const current = cameraController.value.controls
    if (current?.isPerspective !== undefined) {
      cameraController.value.toggleProjection()
    }
  }

  function toggleSectionTool() {
    if (!sectionTool.value) return
    sectionTool.value.toggle()
  }

  function toggleMeasurements() {
    if (!measurementsTool.value) return
    measurementsTool.value.toggle()
  }

  /**
   * Capture the current viewer state as a BCF viewpoint.
   * This is used when creating BCF topics from the 3D view.
   */
  function captureViewpoint(): Omit<BcfViewpoint, 'id' | 'guid' | 'topicId' | 'createdAt'> {
    if (!viewer.value || !cameraController.value) {
      throw new Error('Viewer not initialized')
    }

    const camera = viewer.value.getRenderer().renderingCamera
    const isPerspective = camera.isPerspectiveCamera ?? true

    const position = camera.position
    const target = cameraController.value.controls?.getTarget?.() || { x: 0, y: 0, z: 0 }
    const direction = {
      x: target.x - position.x,
      y: target.y - position.y,
      z: target.z - position.z,
    }
    // Normalize direction
    const len = Math.sqrt(direction.x ** 2 + direction.y ** 2 + direction.z ** 2)
    if (len > 0) {
      direction.x /= len
      direction.y /= len
      direction.z /= len
    }

    const up = camera.up

    // Capture components state
    const components: BcfComponents = {}

    if (selectedObjectIds.value.length > 0) {
      components.selection = selectedObjectIds.value.map((id) => ({
        ifcGuid: id,
      }))
    }

    // Capture section planes
    const clippingPlanes: BcfClippingPlane[] = sectionPlanes.value
      .filter((p) => p.enabled)
      .map((p) => ({
        location: p.origin,
        direction: p.normal,
      }))

    // Capture screenshot
    let snapshotBase64: string | undefined
    try {
      const canvas = containerRef.value?.querySelector('canvas')
      if (canvas) {
        snapshotBase64 = canvas.toDataURL('image/png')
      }
    } catch {
      // Canvas capture may fail due to tainted canvas
    }

    return {
      cameraType: isPerspective ? 'perspective' : 'orthogonal',
      cameraPosition: { x: position.x, y: position.y, z: position.z },
      cameraDirection: direction,
      cameraUp: { x: up.x, y: up.y, z: up.z },
      fieldOfView: isPerspective ? camera.fov : undefined,
      viewWorldScale: !isPerspective ? camera.zoom : undefined,
      snapshotBase64,
      components: Object.keys(components).length > 0 ? components : undefined,
      clippingPlanes: clippingPlanes.length > 0 ? clippingPlanes : undefined,
    }
  }

  /**
   * Restore a BCF viewpoint — sets camera, visibility, selection, and section planes.
   */
  function restoreViewpoint(viewpoint: BcfViewpoint) {
    if (!viewer.value || !cameraController.value) return

    // Restore camera position
    const { cameraPosition: pos, cameraDirection: dir, cameraUp: up } = viewpoint

    const target = {
      x: pos.x + dir.x * 10,
      y: pos.y + dir.y * 10,
      z: pos.z + dir.z * 10,
    }

    cameraController.value.setCameraView(
      {
        position: pos,
        target,
      },
      true
    )

    // Restore selection
    if (viewpoint.components?.selection?.length) {
      const guids = viewpoint.components.selection.map((c) => c.ifcGuid)
      selectByIfcGuid(guids)
    }

    // Restore clipping planes
    if (viewpoint.clippingPlanes?.length && sectionTool.value) {
      viewpoint.clippingPlanes.forEach((plane) => {
        sectionTool.value.addPlane({
          origin: plane.location,
          normal: plane.direction,
        })
      })
    }
  }

  function selectByIfcGuid(guids: string[]) {
    if (!filteringExtension.value) return
    filteringExtension.value.selectObjects(guids)
  }

  function isolateByIfcGuid(guids: string[]) {
    if (!filteringExtension.value) return
    filteringExtension.value.isolateObjects(guids)
  }

  function resetFilters() {
    if (!filteringExtension.value) return
    filteringExtension.value.resetFilters()
  }

  function captureScreenshot(): string | null {
    try {
      const canvas = containerRef.value?.querySelector('canvas')
      return canvas?.toDataURL('image/png') || null
    } catch {
      return null
    }
  }

  function dispose() {
    if (viewer.value) {
      viewer.value.dispose()
      viewer.value = null
    }
    isInitialized.value = false
    loadedModels.value = []
    selectedObjectIds.value = []
  }

  onMounted(() => {
    init()
  })

  onBeforeUnmount(() => {
    dispose()
  })

  return {
    // State
    viewer,
    isInitialized,
    isLoading,
    loadingProgress,
    loadedModels,
    selectedObjectIds,
    selectedObjectProperties,
    sectionPlanes,
    error,

    // Model operations
    loadModel,
    unloadModel,
    toggleModelVisibility,

    // Camera
    fitAll,
    setTopView,
    setFrontView,
    togglePerspective,

    // Tools
    toggleSectionTool,
    toggleMeasurements,

    // BCF integration
    captureViewpoint,
    restoreViewpoint,
    captureScreenshot,

    // Selection/filtering
    selectByIfcGuid,
    isolateByIfcGuid,
    resetFilters,

    // Lifecycle
    init,
    dispose,
  }
}
