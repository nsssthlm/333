/**
 * BCF (BIM Collaboration Format) composable.
 *
 * CRUD operations for BCF topics, comments, and viewpoints.
 * Supports BCF 2.1 export/import via API endpoints.
 */
import { ref, computed } from 'vue'
import type {
  BcfTopic,
  BcfComment,
  BcfViewpoint,
  BcfCreateTopicRequest,
  BcfCreateCommentRequest,
  BcfCreateViewpointRequest,
} from '~/types/bcf'

export function useBcf(projectId: string) {
  const config = useRuntimeConfig()
  const baseUrl = `${config.public.apiBaseUrl}/api/projects/${projectId}/bcf`

  const topics = ref<BcfTopic[]>([])
  const currentTopic = ref<BcfTopic | null>(null)
  const isLoading = ref(false)
  const error = ref<string | null>(null)

  const openTopics = computed(() =>
    topics.value.filter((t) => t.topicStatus !== 'Closed')
  )

  const closedTopics = computed(() =>
    topics.value.filter((t) => t.topicStatus === 'Closed')
  )

  async function apiFetch<T>(
    path: string,
    options: RequestInit = {}
  ): Promise<T> {
    const url = `${baseUrl}${path}`
    const response = await fetch(url, {
      credentials: 'include',
      headers: {
        'Content-Type': 'application/json',
        ...options.headers,
      },
      ...options,
    })

    if (!response.ok) {
      const body = await response.text()
      throw new Error(`API error ${response.status}: ${body}`)
    }

    return response.json()
  }

  // --- Topics ---

  async function fetchTopics(filters?: {
    status?: string
    priority?: string
    assignedTo?: string
  }) {
    isLoading.value = true
    error.value = null
    try {
      const params = new URLSearchParams()
      if (filters?.status) params.set('status', filters.status)
      if (filters?.priority) params.set('priority', filters.priority)
      if (filters?.assignedTo) params.set('assigned_to', filters.assignedTo)
      const qs = params.toString()

      topics.value = await apiFetch<BcfTopic[]>(
        `/topics${qs ? `?${qs}` : ''}`
      )
    } catch (err: any) {
      error.value = err.message
    } finally {
      isLoading.value = false
    }
  }

  async function fetchTopic(topicId: string) {
    isLoading.value = true
    error.value = null
    try {
      currentTopic.value = await apiFetch<BcfTopic>(
        `/topics/${topicId}`
      )
    } catch (err: any) {
      error.value = err.message
    } finally {
      isLoading.value = false
    }
  }

  async function createTopic(data: BcfCreateTopicRequest): Promise<BcfTopic | null> {
    isLoading.value = true
    error.value = null
    try {
      const topic = await apiFetch<BcfTopic>('/topics', {
        method: 'POST',
        body: JSON.stringify(data),
      })
      topics.value.unshift(topic)
      return topic
    } catch (err: any) {
      error.value = err.message
      return null
    } finally {
      isLoading.value = false
    }
  }

  async function updateTopic(
    topicId: string,
    data: Partial<BcfCreateTopicRequest>
  ): Promise<BcfTopic | null> {
    error.value = null
    try {
      const updated = await apiFetch<BcfTopic>(`/topics/${topicId}`, {
        method: 'PUT',
        body: JSON.stringify(data),
      })
      const idx = topics.value.findIndex((t) => t.id === topicId)
      if (idx >= 0) topics.value[idx] = updated
      if (currentTopic.value?.id === topicId) currentTopic.value = updated
      return updated
    } catch (err: any) {
      error.value = err.message
      return null
    }
  }

  async function deleteTopic(topicId: string) {
    error.value = null
    try {
      await apiFetch(`/topics/${topicId}`, { method: 'DELETE' })
      topics.value = topics.value.filter((t) => t.id !== topicId)
      if (currentTopic.value?.id === topicId) currentTopic.value = null
    } catch (err: any) {
      error.value = err.message
    }
  }

  // --- Comments ---

  async function addComment(
    topicId: string,
    data: BcfCreateCommentRequest
  ): Promise<BcfComment | null> {
    error.value = null
    try {
      const comment = await apiFetch<BcfComment>(
        `/topics/${topicId}/comments`,
        {
          method: 'POST',
          body: JSON.stringify(data),
        }
      )
      if (currentTopic.value?.id === topicId) {
        currentTopic.value.comments = [
          ...(currentTopic.value.comments || []),
          comment,
        ]
      }
      return comment
    } catch (err: any) {
      error.value = err.message
      return null
    }
  }

  async function deleteComment(topicId: string, commentId: string) {
    error.value = null
    try {
      await apiFetch(`/topics/${topicId}/comments/${commentId}`, {
        method: 'DELETE',
      })
      if (currentTopic.value?.id === topicId) {
        currentTopic.value.comments = (
          currentTopic.value.comments || []
        ).filter((c) => c.id !== commentId)
      }
    } catch (err: any) {
      error.value = err.message
    }
  }

  // --- Viewpoints ---

  async function addViewpoint(
    topicId: string,
    data: BcfCreateViewpointRequest
  ): Promise<BcfViewpoint | null> {
    error.value = null
    try {
      const viewpoint = await apiFetch<BcfViewpoint>(
        `/topics/${topicId}/viewpoints`,
        {
          method: 'POST',
          body: JSON.stringify(data),
        }
      )
      if (currentTopic.value?.id === topicId) {
        currentTopic.value.viewpoints = [
          ...(currentTopic.value.viewpoints || []),
          viewpoint,
        ]
      }
      return viewpoint
    } catch (err: any) {
      error.value = err.message
      return null
    }
  }

  // --- BCF Export/Import ---

  async function exportBcf(): Promise<Blob | null> {
    error.value = null
    try {
      const response = await fetch(`${baseUrl}/export`, {
        credentials: 'include',
      })
      if (!response.ok) throw new Error(`Export failed: ${response.status}`)
      return await response.blob()
    } catch (err: any) {
      error.value = err.message
      return null
    }
  }

  async function importBcf(file: File): Promise<boolean> {
    isLoading.value = true
    error.value = null
    try {
      const formData = new FormData()
      formData.append('file', file)

      const response = await fetch(`${baseUrl}/import`, {
        method: 'POST',
        credentials: 'include',
        body: formData,
      })

      if (!response.ok) throw new Error(`Import failed: ${response.status}`)
      await fetchTopics()
      return true
    } catch (err: any) {
      error.value = err.message
      return false
    } finally {
      isLoading.value = false
    }
  }

  function downloadBcfExport() {
    exportBcf().then((blob) => {
      if (!blob) return
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = `bcf-export-${projectId}.bcf`
      a.click()
      URL.revokeObjectURL(url)
    })
  }

  return {
    // State
    topics,
    currentTopic,
    openTopics,
    closedTopics,
    isLoading,
    error,

    // Topics
    fetchTopics,
    fetchTopic,
    createTopic,
    updateTopic,
    deleteTopic,

    // Comments
    addComment,
    deleteComment,

    // Viewpoints
    addViewpoint,

    // Export/Import
    exportBcf,
    importBcf,
    downloadBcfExport,
  }
}
