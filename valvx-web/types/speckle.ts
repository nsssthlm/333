/** Types for Speckle integration */

export interface SpeckleModel {
  id: string
  name: string
  description?: string
  createdAt: string
  updatedAt: string
  versions?: SpeckleModelVersion[]
}

export interface SpeckleModelVersion {
  id: string
  message?: string
  referencedObject: string
  sourceApplication?: string
  createdAt: string
}

export interface SpeckleObjectData {
  id: string
  speckleType: string
  data: Record<string, unknown>
  children?: SpeckleObjectData[]
}

export interface SpeckleViewerState {
  loadedModels: LoadedModel[]
  selectedObjects: string[]
  hiddenObjects: string[]
  isolatedObjects: string[]
  sectionPlanes: SectionPlane[]
}

export interface LoadedModel {
  fileVersionId: string
  fileName: string
  speckleModelId: string
  speckleVersionId?: string
  speckleObjectId: string
  url: string
  visible: boolean
  loading: boolean
}

export interface SectionPlane {
  id: string
  origin: { x: number; y: number; z: number }
  normal: { x: number; y: number; z: number }
  enabled: boolean
}

export interface SpeckleFileMapping {
  fileVersionId: string
  speckleModelId: string
  speckleVersionId?: string
  speckleObjectId?: string
  status: 'pending' | 'processing' | 'ready' | 'error'
}
