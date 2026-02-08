/**
 * TUS-based chunked file upload composable.
 *
 * Features:
 * - Resumable uploads via TUS protocol
 * - 5 MB chunk size for fast parallel transfer
 * - Automatic retry on network failures
 * - Multiple concurrent uploads
 * - Real-time progress, speed, and ETA tracking
 * - Pause/resume support
 */
import { ref, computed } from 'vue'
import * as tus from 'tus-js-client'
import type {
  UploadFile,
  UploadStatus,
  UploadBatchState,
  UploadOptions,
} from '~/types/upload'

const CHUNK_SIZE = 5 * 1024 * 1024 // 5 MB
const MAX_PARALLEL_UPLOADS = 3
const RETRY_DELAYS = [0, 1000, 3000, 5000, 10000]

export function useFileUpload() {
  const config = useRuntimeConfig()

  const files = ref<UploadFile[]>([])
  const activeUploads = ref(0)
  const tusInstances = new Map<string, tus.Upload>()
  let speedSamples: { time: number; bytes: number }[] = []

  const isUploading = computed(() => activeUploads.value > 0)

  const batchState = computed<UploadBatchState>(() => {
    const total = files.value.reduce((sum, f) => sum + f.bytesTotal, 0)
    const uploaded = files.value.reduce((sum, f) => sum + f.bytesUploaded, 0)
    const speed = calculateSpeed()
    const remaining = total - uploaded
    const eta = speed > 0 ? remaining / speed : 0

    return {
      files: files.value,
      totalSize: total,
      uploadedSize: uploaded,
      activeUploads: activeUploads.value,
      isUploading: isUploading.value,
      overallProgress: total > 0 ? (uploaded / total) * 100 : 0,
      speed,
      eta,
    }
  })

  function calculateSpeed(): number {
    const now = Date.now()
    // Keep only last 5 seconds of samples
    speedSamples = speedSamples.filter((s) => now - s.time < 5000)
    if (speedSamples.length < 2) return 0

    const oldest = speedSamples[0]
    const newest = speedSamples[speedSamples.length - 1]
    const timeDiff = (newest.time - oldest.time) / 1000
    if (timeDiff <= 0) return 0

    return (newest.bytes - oldest.bytes) / timeDiff
  }

  function generateId(): string {
    return crypto.randomUUID()
  }

  function getFileExt(name: string): string {
    const parts = name.split('.')
    return parts.length > 1 ? parts[parts.length - 1].toLowerCase() : ''
  }

  function addFiles(rawFiles: File[], folderId: string): UploadFile[] {
    const newFiles: UploadFile[] = rawFiles.map((file) => ({
      id: generateId(),
      file,
      name: file.name,
      size: file.size,
      ext: getFileExt(file.name),
      folderId,
      status: 'queued' as UploadStatus,
      progress: 0,
      bytesUploaded: 0,
      bytesTotal: file.size,
    }))

    files.value.push(...newFiles)
    processQueue()
    return newFiles
  }

  function processQueue() {
    while (activeUploads.value < MAX_PARALLEL_UPLOADS) {
      const next = files.value.find((f) => f.status === 'queued')
      if (!next) break
      startUpload(next)
    }
  }

  function startUpload(uploadFile: UploadFile) {
    const endpoint = `${config.public.apiBaseUrl}/api/uploads`

    uploadFile.status = 'uploading'
    uploadFile.startedAt = Date.now()
    activeUploads.value++

    const tusUpload = new tus.Upload(uploadFile.file, {
      endpoint,
      retryDelays: RETRY_DELAYS,
      chunkSize: CHUNK_SIZE,
      metadata: {
        filename: uploadFile.name,
        filetype: uploadFile.file.type || 'application/octet-stream',
        folderId: uploadFile.folderId,
        ext: uploadFile.ext,
      },
      onProgress: (bytesUploaded: number, bytesTotal: number) => {
        uploadFile.bytesUploaded = bytesUploaded
        uploadFile.bytesTotal = bytesTotal
        uploadFile.progress = (bytesUploaded / bytesTotal) * 100

        speedSamples.push({
          time: Date.now(),
          bytes: files.value.reduce((sum, f) => sum + f.bytesUploaded, 0),
        })
      },
      onSuccess: () => {
        uploadFile.status = 'complete'
        uploadFile.progress = 100
        uploadFile.completedAt = Date.now()
        uploadFile.tusUrl = tusUpload.url || undefined
        activeUploads.value--
        tusInstances.delete(uploadFile.id)
        processQueue()
      },
      onError: (err: Error) => {
        uploadFile.status = 'error'
        uploadFile.error = err.message
        activeUploads.value--
        tusInstances.delete(uploadFile.id)
        processQueue()
      },
    })

    tusInstances.set(uploadFile.id, tusUpload)
    tusUpload.start()
  }

  function pauseUpload(fileId: string) {
    const tusUpload = tusInstances.get(fileId)
    const file = files.value.find((f) => f.id === fileId)
    if (tusUpload && file) {
      tusUpload.abort()
      file.status = 'paused'
      activeUploads.value--
      processQueue()
    }
  }

  function resumeUpload(fileId: string) {
    const tusUpload = tusInstances.get(fileId)
    const file = files.value.find((f) => f.id === fileId)
    if (tusUpload && file) {
      file.status = 'uploading'
      activeUploads.value++
      tusUpload.start()
    }
  }

  function retryUpload(fileId: string) {
    const file = files.value.find((f) => f.id === fileId)
    if (file && file.status === 'error') {
      file.status = 'queued'
      file.error = undefined
      file.progress = 0
      file.bytesUploaded = 0
      tusInstances.delete(fileId)
      processQueue()
    }
  }

  function removeUpload(fileId: string) {
    const tusUpload = tusInstances.get(fileId)
    if (tusUpload) {
      tusUpload.abort()
      tusInstances.delete(fileId)
    }

    const file = files.value.find((f) => f.id === fileId)
    if (file?.status === 'uploading') {
      activeUploads.value--
    }

    files.value = files.value.filter((f) => f.id !== fileId)
    processQueue()
  }

  function clearCompleted() {
    files.value = files.value.filter((f) => f.status !== 'complete')
  }

  function cancelAll() {
    for (const [id, upload] of tusInstances) {
      upload.abort()
    }
    tusInstances.clear()
    activeUploads.value = 0
    files.value.forEach((f) => {
      if (f.status === 'uploading' || f.status === 'queued') {
        f.status = 'paused'
      }
    })
  }

  return {
    files,
    batchState,
    isUploading,
    addFiles,
    pauseUpload,
    resumeUpload,
    retryUpload,
    removeUpload,
    clearCompleted,
    cancelAll,
  }
}
