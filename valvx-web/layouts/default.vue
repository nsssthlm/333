<script setup lang="ts">
/**
 * Default layout with sidebar navigation.
 */
const route = useRoute()
</script>

<template>
  <div class="app-layout">
    <nav class="app-sidebar">
      <div class="sidebar-brand">
        <span class="brand-mark">V</span>
        <span class="brand-text">ValvX</span>
      </div>
      <div class="sidebar-nav">
        <NuxtLink to="/" class="nav-item">
          <svg width="18" height="18" viewBox="0 0 18 18" fill="none">
            <path d="M2 9l7-7 7 7M4 7.5V15a1 1 0 001 1h3v-4h2v4h3a1 1 0 001-1V7.5" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/>
          </svg>
          Home
        </NuxtLink>
      </div>
    </nav>
    <main class="app-content">
      <slot />
    </main>
  </div>
</template>

<style scoped>
.app-layout {
  display: flex;
  height: 100vh;
  overflow: hidden;
}

.app-sidebar {
  width: 200px;
  background: var(--color-bg-surface);
  border-right: 1px solid var(--color-border);
  display: flex;
  flex-direction: column;
  flex-shrink: 0;
}

.sidebar-brand {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 16px;
  border-bottom: 1px solid var(--color-border);
}
.brand-mark {
  width: 28px;
  height: 28px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--color-primary);
  color: white;
  font-weight: 800;
  font-size: 14px;
  border-radius: var(--radius-sm);
}
.brand-text {
  font-weight: 700;
  font-size: 16px;
  letter-spacing: -0.5px;
}

.sidebar-nav {
  padding: 8px;
  flex: 1;
}

.nav-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 12px;
  border-radius: var(--radius-sm);
  color: var(--color-text-secondary);
  text-decoration: none;
  font-size: 13px;
  font-weight: 500;
  transition: all var(--transition);
}
.nav-item:hover {
  background: var(--color-bg-hover);
  color: var(--color-text);
}
.nav-item.router-link-active {
  background: rgba(69, 137, 255, 0.1);
  color: var(--color-primary);
}

.app-content {
  flex: 1;
  overflow: hidden;
}
</style>
