<script setup lang="ts">
const config = useRuntimeConfig()
const route = useRoute()
const projectId = route.params.projectId as string

interface Folder {
  id: string
  name: string
  parentId: string | null
}
interface FileItem {
  id: string
  name: string
  ext: string
  size: number
  createdAt: string
}

const { data: project } = await useFetch<any>(`${config.public.apiBaseUrl}/projects/${projectId}`)
const { data: allFolders } = await useFetch<Folder[]>(`${config.public.apiBaseUrl}/projects/${projectId}/folders`)

const currentFolderId = ref<string | null>(null)
const files = ref<FileItem[]>([])
const loading = ref(false)

// Build folder tree
const rootFolders = computed(() => {
  if (!allFolders.value) return []
  return allFolders.value.filter(f => !f.parentId || !allFolders.value!.find(p => p.id === f.parentId))
})

const childFolders = computed(() => {
  if (!allFolders.value || !currentFolderId.value) return []
  return allFolders.value.filter(f => f.parentId === currentFolderId.value)
})

const currentFolder = computed(() => {
  if (!currentFolderId.value || !allFolders.value) return null
  return allFolders.value.find(f => f.id === currentFolderId.value)
})

const parentFolder = computed(() => {
  if (!currentFolder.value?.parentId || !allFolders.value) return null
  return allFolders.value.find(f => f.id === currentFolder.value!.parentId)
})

const breadcrumbs = computed(() => {
  const crumbs: Folder[] = []
  if (!allFolders.value || !currentFolderId.value) return crumbs
  let folder = allFolders.value.find(f => f.id === currentFolderId.value)
  while (folder) {
    crumbs.unshift(folder)
    folder = folder.parentId ? allFolders.value.find(f => f.id === folder!.parentId) : undefined
  }
  return crumbs
})

const displayFolders = computed(() => {
  return currentFolderId.value ? childFolders.value : rootFolders.value
})

async function openFolder(folderId: string) {
  currentFolderId.value = folderId
  loading.value = true
  try {
    const data = await $fetch<FileItem[]>(`${config.public.apiBaseUrl}/projects/${projectId}/folders/${folderId}/files`)
    files.value = data || []
  } catch {
    files.value = []
  }
  loading.value = false
}

function goUp() {
  if (currentFolder.value?.parentId) {
    openFolder(currentFolder.value.parentId)
  } else {
    currentFolderId.value = null
    files.value = []
  }
}

function formatSize(bytes: number): string {
  if (bytes === 0) return '-'
  if (bytes < 1024) return bytes + ' B'
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB'
  return (bytes / (1024 * 1024)).toFixed(1) + ' MB'
}

function fileIcon(ext: string): string {
  const icons: Record<string, string> = {
    ifc: '🏗',
    pdf: '📄',
    dwg: '📐',
    xlsx: '📊',
    docx: '📝',
    png: '🖼',
    jpg: '🖼',
  }
  return icons[ext?.toLowerCase()] || '📎'
}
</script>

<template>
  <div class="file-browser">
    <div class="browser-header">
      <div class="header-top">
        <NuxtLink to="/" class="back-link">← Projects</NuxtLink>
        <h1>{{ project?.name || 'Project' }}</h1>
      </div>
      <div class="header-nav">
        <NuxtLink :to="`/projects/${projectId}/viewer`" class="nav-tab">3D Viewer</NuxtLink>
      </div>
    </div>

    <div class="browser-toolbar">
      <button v-if="currentFolderId" class="btn-back" @click="goUp">← Back</button>
      <div class="breadcrumbs">
        <span class="crumb root" @click="currentFolderId = null; files = []">Files</span>
        <template v-for="crumb in breadcrumbs" :key="crumb.id">
          <span class="crumb-sep">/</span>
          <span class="crumb" @click="openFolder(crumb.id)">{{ crumb.name }}</span>
        </template>
      </div>
    </div>

    <div class="browser-content">
      <!-- Folders -->
      <div v-for="folder in displayFolders" :key="folder.id" class="item folder" @click="openFolder(folder.id)">
        <span class="item-icon">📁</span>
        <span class="item-name">{{ folder.name }}</span>
      </div>

      <!-- Files -->
      <div v-if="currentFolderId && !loading">
        <div v-for="file in files" :key="file.id" class="item file">
          <span class="item-icon">{{ fileIcon(file.ext) }}</span>
          <span class="item-name">{{ file.name }}<span v-if="file.ext" class="file-ext">.{{ file.ext }}</span></span>
          <span class="item-size">{{ formatSize(file.size) }}</span>
        </div>
        <div v-if="files.length === 0 && displayFolders.length === 0" class="empty-state">
          No files in this folder
        </div>
      </div>

      <div v-if="loading" class="loading">Loading...</div>
    </div>
  </div>
</template>

<style scoped>
.file-browser {
  height: 100%;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}
.browser-header {
  padding: 16px 24px 0;
  border-bottom: 1px solid var(--color-border);
}
.header-top {
  display: flex;
  align-items: center;
  gap: 16px;
  margin-bottom: 12px;
}
.back-link {
  font-size: 13px;
  color: var(--color-text-secondary);
  text-decoration: none;
}
.back-link:hover { color: var(--color-primary); }
.header-top h1 {
  font-size: 20px;
  font-weight: 700;
}
.header-nav {
  display: flex;
  gap: 4px;
  margin-bottom: -1px;
}
.nav-tab {
  padding: 8px 16px;
  font-size: 13px;
  color: var(--color-text-secondary);
  text-decoration: none;
  border-bottom: 2px solid transparent;
}
.nav-tab:hover { color: var(--color-text); }

.browser-toolbar {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 24px;
  background: var(--color-bg-surface);
  border-bottom: 1px solid var(--color-border);
}
.btn-back {
  padding: 4px 10px;
  font-size: 12px;
  background: var(--color-bg-hover);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  cursor: pointer;
  color: var(--color-text);
}
.btn-back:hover { background: var(--color-border); }
.breadcrumbs {
  font-size: 13px;
  color: var(--color-text-secondary);
}
.crumb {
  cursor: pointer;
  padding: 2px 4px;
  border-radius: 3px;
}
.crumb:hover { background: var(--color-bg-hover); color: var(--color-text); }
.crumb-sep { margin: 0 2px; }

.browser-content {
  flex: 1;
  overflow-y: auto;
  padding: 4px 0;
}
.item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 24px;
  font-size: 14px;
  cursor: pointer;
  transition: background 0.1s;
}
.item:hover {
  background: var(--color-bg-hover);
}
.item-icon {
  font-size: 18px;
  width: 24px;
  text-align: center;
  flex-shrink: 0;
}
.item-name {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.file-ext {
  color: var(--color-text-secondary);
}
.item-size {
  font-size: 12px;
  color: var(--color-text-secondary);
  flex-shrink: 0;
}
.empty-state {
  padding: 40px 24px;
  text-align: center;
  color: var(--color-text-secondary);
  font-size: 14px;
}
.loading {
  padding: 40px 24px;
  text-align: center;
  color: var(--color-text-secondary);
}
</style>
