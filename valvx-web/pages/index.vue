<script setup lang="ts">
const config = useRuntimeConfig()

const { data: projects } = await useFetch<any[]>(`${config.public.apiBaseUrl}/projects`)
</script>

<template>
  <div class="home-page">
    <div class="home-header">
      <h1>Projects</h1>
    </div>
    <div class="projects-grid">
      <NuxtLink
        v-for="project in (projects || [])"
        :key="project.id"
        :to="`/projects/${project.id}`"
        class="project-card"
      >
        <div class="project-icon">
          {{ project.name.charAt(0) }}
        </div>
        <div class="project-info">
          <div class="project-name">{{ project.name }}</div>
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
