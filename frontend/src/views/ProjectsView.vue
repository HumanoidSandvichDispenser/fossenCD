<script setup lang="ts">
import UserMenu from '@/components/UserMenu.vue';
</script>

<template>
  <main class="projects">
    <header>
      <h1 class="h3">Projects</h1>
      <div class="spacer" />
      <UserMenu />
    </header>

    <p class="placeholder text-sm">No projects yet.</p>
  </main>
</template>

<style scoped>
.projects {
  max-width: var(--width-content);
  margin: 0 auto;
  padding: var(--space-8) var(--space-4);
}

header {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  padding-bottom: var(--space-4);
  border-bottom: var(--border-thin) solid var(--color-border);
}

.spacer {
  flex: 1;
}

.placeholder {
  margin-top: var(--space-8);
  color: var(--color-text-tertiary);
  text-align: center;
}
</style>
