<script setup lang="ts">
/**
 * Home page — project selection
 */
const config = useRuntimeConfig()

// In production, this would come from the API
const projects = ref([
  {
    id: 'aa03cc85-fe34-4ffd-b25f-e2edbef08b59',
    name: 'ValvX Demo Project',
    code: 'valvx-demo-project',
  },
  {
    id: 'af96c893-cbb8-4190-ab1b-8d394a3b6c7e',
    name: 'Gökbo',
    code: 'cavada-gokbo',
  },
])
</script>

<template>
  <div class="home-page">
    <div class="home-header">
      <h1>Projects</h1>
    </div>
    <div class="projects-grid">
      <NuxtLink
        v-for="project in projects"
        :key="project.id"
        :to="`/projects/${project.id}/viewer`"
        class="project-card"
      >
        <div class="project-icon">
          {{ project.name.charAt(0) }}
        </div>
        <div class="project-info">
          <div class="project-name">{{ project.name }}</div>
          <div class="project-code text-xs text-muted">{{ project.code }}</div>
        </div>
      </NuxtLink>
    </div>
  </div>
</template>

<style scoped>
.home-page {
  padding: 32px;
  height: 100%;
  overflow-y: auto;
}

.home-header {
  margin-bottom: 24px;
}
.home-header h1 {
  font-size: 24px;
  font-weight: 700;
}

.projects-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 16px;
}

.project-card {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 20px;
  background: var(--color-bg-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  text-decoration: none;
  color: var(--color-text);
  transition: all var(--transition);
}
.project-card:hover {
  border-color: var(--color-primary);
  box-shadow: var(--shadow-md);
  transform: translateY(-1px);
}

.project-icon {
  width: 48px;
  height: 48px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--color-primary);
  color: white;
  font-weight: 700;
  font-size: 20px;
  border-radius: var(--radius-md);
  flex-shrink: 0;
}

.project-name {
  font-size: 15px;
  font-weight: 600;
}
</style>
