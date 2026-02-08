/** Types for IFC model integration (replaces Speckle types) */

export interface LoadedModel {
  fileVersionId: string
  fileName: string
  url: string
  visible: boolean
  loading: boolean
  modelId?: string
}

export interface SectionPlane {
  id: string
  origin: { x: number; y: number; z: number }
  normal: { x: number; y: number; z: number }
  enabled: boolean
}

export interface SpeckleFileMapping {
  fileVersionId: string
  status: 'pending' | 'processing' | 'ready' | 'error'
}
