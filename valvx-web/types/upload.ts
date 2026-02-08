/** Types for file upload engine */

export interface UploadFile {
  id: string
  file: File
  name: string
  size: number
  ext: string
  folderId: string
  status: UploadStatus
  progress: number
  bytesUploaded: number
  bytesTotal: number
  tusUrl?: string
  error?: string
  fileVersionId?: string
  startedAt?: number
  completedAt?: number
}

export type UploadStatus =
  | 'queued'
  | 'uploading'
  | 'paused'
  | 'processing'
  | 'complete'
  | 'error'

export interface UploadOptions {
  endpoint: string
  chunkSize?: number
  parallelUploads?: number
  retryDelays?: number[]
  folderId: string
  onProgress?: (file: UploadFile) => void
  onComplete?: (file: UploadFile) => void
  onError?: (file: UploadFile, error: Error) => void
}

export interface UploadBatchState {
  files: UploadFile[]
  totalSize: number
  uploadedSize: number
  activeUploads: number
  isUploading: boolean
  overallProgress: number
  speed: number
  eta: number
}
