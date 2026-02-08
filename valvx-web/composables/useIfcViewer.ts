/**
 * IFC Viewer composable — client-side IFC loading via That Open Engine.
 *
 * Replaces the Speckle iframe/SDK integration. No server-side conversion
 * needed — IFC files are parsed directly in the browser via web-ifc WASM.
 *
 * Features:
 * - Direct IFC loading from MinIO/S3 URLs
 * - Three.js rendering with orbit controls
 * - Object selection with IFC property display
 * - Section planes and measurements
 * - BCF viewpoint capture/restore
 */
import { ref, shallowRef, onBeforeUnmount, type Ref } from 'vue'
import * as THREE from 'three'
import * as OBC from '@thatopen/components'
import * as OBCF from '@thatopen/components-front'
import type { BcfViewpoint, BcfComponents, BcfClippingPlane } from '~/types/bcf'

export interface IfcModel {
  fileVersionId: string
  fileName: string
  url: string
  visible: boolean
  loading: boolean
  modelId?: string
}

export function useIfcViewer(containerRef: Ref<HTMLElement | null>) {
  const config = useRuntimeConfig()
  const apiBaseUrl = config.public.apiBaseUrl as string

  // Core engine refs
  const components = shallowRef<OBC.Components | null>(null)
  const world = shallowRef<any>(null)

  // State
  const isInitialized = ref(false)
  const isLoading = ref(false)
  const loadingProgress = ref(0)
  const loadedModels = ref<IfcModel[]>([])
  const selectedObjectIds = ref<string[]>([])
  const selectedObjectProperties = ref<Record<string, unknown> | null>(null)
  const error = ref<string | null>(null)

  // Internal refs for extensions
  let ifcLoader: any = null
  let highlighter: any = null
  let clipper: any = null
  let measurements: any = null
  let fragmentsManager: any = null

  async function init() {
    if (!containerRef.value || isInitialized.value) return

    try {
      const c = new OBC.Components()
      const worlds = c.get(OBC.Worlds)

      const w = worlds.create<
        OBC.SimpleScene,
        OBC.SimpleCamera,
        OBCF.PostproductionRenderer
      >()

      // Scene
      w.scene = new OBC.SimpleScene(c)
      w.scene.setup()
      const scene = w.scene.three
      scene.background = new THREE.Color(0x1a1d23)

      // Renderer
      w.renderer = new OBCF.PostproductionRenderer(c, containerRef.value)
      w.renderer.postproduction.enabled = true

      // Camera with orbit controls
      w.camera = new OBC.SimpleCamera(c)
      w.camera.controls.setLookAt(20, 20, 20, 0, 0, 0)

      // Grids
      const grids = c.get(OBC.Grids)
      const grid = grids.create(w)
      if (w.renderer.postproduction) {
        w.renderer.postproduction.customEffects.excludedMeshes.push(grid.three)
      }

      // IFC Loader
      fragmentsManager = c.get(OBC.FragmentsManager)
      ifcLoader = c.get(OBC.IfcLoader)
      await ifcLoader.setup()

      // Configure web-ifc WASM path
      ifcLoader.settings.wasm = {
        path: 'https://unpkg.com/web-ifc@0.0.68/',
        absolute: true,
      }

      // Highlighter for selection
      highlighter = c.get(OBCF.Highlighter)
      highlighter.setup({ world: w })

      highlighter.events.select.onHighlight.add((fragmentIdMap: any) => {
        handleSelection(fragmentIdMap)
      })

      highlighter.events.select.onClear.add(() => {
        selectedObjectIds.value = []
        selectedObjectProperties.value = null
      })

      // Clipper (section planes)
      clipper = c.get(OBC.Clipper)
      clipper.enabled = false

      // Length measurement
      try {
        measurements = c.get(OBCF.LengthMeasurement)
        measurements.world = w
        measurements.enabled = false
        measurements.snapDistance = 1
      } catch {
        // Measurements may not be available
      }

      // Double-click to create clipping planes
      containerRef.value.addEventListener('dblclick', () => {
        if (clipper.enabled) {
          clipper.create(w)
        }
      })

      // Right-click to delete clipping planes
      containerRef.value.addEventListener('contextmenu', (e: MouseEvent) => {
        if (clipper.enabled) {
          e.preventDefault()
          clipper.delete(w)
        }
      })

      components.value = c
      world.value = w
      isInitialized.value = true

      // Start render loop
      w.renderer.postproduction.enabled = true
    } catch (err: any) {
      error.value = `Failed to initialize viewer: ${err.message}`
      console.error('Viewer init error:', err)
    }
  }

  function handleSelection(fragmentIdMap: any) {
    if (!fragmentsManager) return

    const ids: string[] = []
    const props: Record<string, unknown> = {}

    for (const [fragmentId, expressIds] of fragmentIdMap) {
      const fragment = fragmentsManager.list.get(fragmentId)
      if (!fragment) continue

      for (const expressId of expressIds) {
        ids.push(`${fragmentId}:${expressId}`)

        // Try to get IFC properties
        try {
          const model = fragment.mesh?.parent?.userData?.model
          if (model) {
            const allProps = model.getProperties(expressId)
            if (allProps) {
              Object.assign(props, allProps)
            }
          }
        } catch {
          // Property access may fail for some elements
        }
      }
    }

    selectedObjectIds.value = ids
    selectedObjectProperties.value = Object.keys(props).length > 0 ? props : null
  }

  /**
   * Load an IFC file from a URL (typically MinIO/S3 presigned URL).
   */
  async function loadModel(model: IfcModel) {
    if (!components.value || !ifcLoader || !world.value) return

    const existing = loadedModels.value.find(
      (m) => m.fileVersionId === model.fileVersionId
    )
    if (existing && !existing.loading) return

    try {
      isLoading.value = true
      loadingProgress.value = 0

      const modelEntry: IfcModel = {
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

      // Fetch the IFC file as ArrayBuffer
      const response = await fetch(model.url, { credentials: 'include' })
      if (!response.ok) throw new Error(`Failed to fetch ${model.fileName}: ${response.status}`)

      const data = await response.arrayBuffer()
      const buffer = new Uint8Array(data)

      loadingProgress.value = 50

      // Parse IFC with web-ifc WASM
      const fragmentGroup = await ifcLoader.load(buffer)
      fragmentGroup.name = model.fileName

      // Add to scene
      world.value.scene.three.add(fragmentGroup)

      // Fit camera to model
      world.value.camera.controls.fitToSphere(fragmentGroup, true)

      loadingProgress.value = 100

      const mi = loadedModels.value.findIndex(
        (m) => m.fileVersionId === model.fileVersionId
      )
      if (mi >= 0) {
        loadedModels.value[mi] = {
          ...loadedModels.value[mi],
          loading: false,
          modelId: fragmentGroup.uuid,
        }
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
    }
  }

  function unloadModel(fileVersionId: string) {
    const model = loadedModels.value.find((m) => m.fileVersionId === fileVersionId)
    if (!model?.modelId || !world.value) return

    const obj = world.value.scene.three.getObjectByProperty('uuid', model.modelId)
    if (obj) {
      world.value.scene.three.remove(obj)
      obj.traverse((child: any) => {
        if (child.geometry) child.geometry.dispose()
        if (child.material) {
          if (Array.isArray(child.material)) {
            child.material.forEach((m: any) => m.dispose())
          } else {
            child.material.dispose()
          }
        }
      })
    }

    loadedModels.value = loadedModels.value.filter(
      (m) => m.fileVersionId !== fileVersionId
    )
  }

  function toggleModelVisibility(fileVersionId: string) {
    const model = loadedModels.value.find((m) => m.fileVersionId === fileVersionId)
    if (!model?.modelId || !world.value) return

    const obj = world.value.scene.three.getObjectByProperty('uuid', model.modelId)
    if (obj) {
      obj.visible = !obj.visible
      model.visible = obj.visible
    }
  }

  // Camera controls
  function fitAll() {
    if (!world.value) return
    const box = new THREE.Box3()
    world.value.scene.three.traverse((child: any) => {
      if (child.isMesh) box.expandByObject(child)
    })
    if (!box.isEmpty()) {
      const sphere = new THREE.Sphere()
      box.getBoundingSphere(sphere)
      world.value.camera.controls.fitToSphere(sphere, true)
    }
  }

  function setTopView() {
    if (!world.value) return
    world.value.camera.controls.setLookAt(0, 50, 0, 0, 0, 0, true)
  }

  function setFrontView() {
    if (!world.value) return
    world.value.camera.controls.setLookAt(0, 10, 50, 0, 10, 0, true)
  }

  function togglePerspective() {
    // camera-controls doesn't easily toggle projection type,
    // so we set a very wide or narrow FOV as an approximation
    // In practice, @thatopen/components uses perspective by default
  }

  function toggleSectionTool() {
    if (!clipper) return
    clipper.enabled = !clipper.enabled
  }

  function toggleMeasurements() {
    if (!measurements) return
    measurements.enabled = !measurements.enabled
  }

  /**
   * Capture current viewer state as a BCF viewpoint.
   */
  function captureViewpoint(): Omit<BcfViewpoint, 'id' | 'guid' | 'topicId' | 'createdAt'> {
    if (!world.value) throw new Error('Viewer not initialized')

    const camera = world.value.camera.three as THREE.PerspectiveCamera
    const target = new THREE.Vector3()
    world.value.camera.controls.getTarget(target)

    const position = camera.position
    const direction = new THREE.Vector3().subVectors(target, position).normalize()
    const up = camera.up

    const isPerspective = camera instanceof THREE.PerspectiveCamera

    // Capture clipping planes
    const clippingPlanes: BcfClippingPlane[] = []
    if (clipper) {
      for (const plane of clipper.list) {
        if (plane.enabled !== false) {
          const normal = plane.normal || { x: 0, y: 0, z: 1 }
          const origin = plane.origin || { x: 0, y: 0, z: 0 }
          clippingPlanes.push({ location: origin, direction: normal })
        }
      }
    }

    // Capture selection
    const components: BcfComponents = {}
    if (selectedObjectIds.value.length > 0) {
      components.selection = selectedObjectIds.value.map((id) => ({
        ifcGuid: id,
      }))
    }

    // Capture screenshot
    let snapshotBase64: string | undefined
    try {
      const canvas = containerRef.value?.querySelector('canvas')
      if (canvas) {
        snapshotBase64 = canvas.toDataURL('image/png')
      }
    } catch {
      // Canvas capture may fail
    }

    return {
      cameraType: isPerspective ? 'perspective' : 'orthogonal',
      cameraPosition: { x: position.x, y: position.y, z: position.z },
      cameraDirection: { x: direction.x, y: direction.y, z: direction.z },
      cameraUp: { x: up.x, y: up.y, z: up.z },
      fieldOfView: isPerspective ? camera.fov : undefined,
      viewWorldScale: !isPerspective ? (camera as any).zoom : undefined,
      snapshotBase64,
      components: Object.keys(components).length > 0 ? components : undefined,
      clippingPlanes: clippingPlanes.length > 0 ? clippingPlanes : undefined,
    }
  }

  /**
   * Restore a BCF viewpoint — sets camera and clipping planes.
   */
  function restoreViewpoint(viewpoint: BcfViewpoint) {
    if (!world.value) return

    const { cameraPosition: pos, cameraDirection: dir } = viewpoint
    const target = {
      x: pos.x + dir.x * 10,
      y: pos.y + dir.y * 10,
      z: pos.z + dir.z * 10,
    }

    world.value.camera.controls.setLookAt(
      pos.x, pos.y, pos.z,
      target.x, target.y, target.z,
      true
    )

    // Restore clipping planes
    if (viewpoint.clippingPlanes?.length && clipper) {
      clipper.deleteAll()
      // Re-create planes from viewpoint
      for (const plane of viewpoint.clippingPlanes) {
        const normal = new THREE.Vector3(plane.direction.x, plane.direction.y, plane.direction.z)
        const point = new THREE.Vector3(plane.location.x, plane.location.y, plane.location.z)
        clipper.createFromNormalAndCoplanarPoint(world.value, normal, point)
      }
    }
  }

  function resetFilters() {
    if (highlighter) {
      highlighter.clear()
    }
    selectedObjectIds.value = []
    selectedObjectProperties.value = null
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
    if (components.value) {
      components.value.dispose()
      components.value = null
    }
    world.value = null
    isInitialized.value = false
    loadedModels.value = []
    selectedObjectIds.value = []
  }

  onBeforeUnmount(() => {
    dispose()
  })

  return {
    // State
    isInitialized,
    isLoading,
    loadingProgress,
    loadedModels,
    selectedObjectIds,
    selectedObjectProperties,
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
    resetFilters,

    // Lifecycle
    init,
    dispose,
  }
}
