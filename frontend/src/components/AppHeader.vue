<script setup lang="ts">
import UserMenu from '@/components/UserMenu.vue';
</script>

<template>
  <header class="app-header">
    <div class="left">
      <RouterLink :to="{ name: 'projects' }" class="brand">fossenCD</RouterLink>
      <nav class="nav">
        <RouterLink :to="{ name: 'projects' }" class="label-sm link">Projects</RouterLink>
      </nav>
    </div>
    <div class="spacer" />
    <UserMenu />
  </header>
</template>

<style scoped>
.app-header {
  display: flex;
  align-items: center;
  gap: var(--space-5);
  padding: var(--space-3) var(--space-6);
  background: var(--color-bg-card);
  border-bottom: var(--border-thin) solid var(--color-border);
}

.left {
  display: flex;
  align-items: center;
  gap: var(--space-5);
}

.brand {
  font: 600 18px var(--font-serif);
  line-height: 1;
  letter-spacing: -0.01em;
  color: var(--color-text);
  text-decoration: none;
}

.brand:hover {
  text-decoration: none;
}

.nav {
  display: flex;
  gap: var(--space-4);
}

.link {
  line-height: 1;
  color: var(--color-text-secondary);
  text-decoration: none;
}

.link:hover,
.link.router-link-active {
  color: var(--color-text);
  text-decoration: none;
}

.spacer {
  flex: 1;
}
</style>
