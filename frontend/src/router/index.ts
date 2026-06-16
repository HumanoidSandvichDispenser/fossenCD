import { createRouter, createWebHistory } from 'vue-router';

import EditorView from '@/views/EditorView.vue';
import LoginView from '@/views/LoginView.vue';
import ProjectsView from '@/views/ProjectsView.vue';
import { useAuthStore } from '@/stores/auth';

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: '/',
      redirect: '/projects',
    },
    {
      path: '/login',
      name: 'login',
      component: LoginView,
      meta: { guestOnly: true },
    },
    {
      path: '/projects',
      name: 'projects',
      component: ProjectsView,
      meta: { requiresAuth: true },
    },
    {
      path: '/editor/:id',
      name: 'editor',
      component: EditorView,
      meta: { requiresAuth: true },
    },
  ],
});

router.beforeEach(async (to) => {
  const auth = useAuthStore();
  // Hydrate the session once (deduped + cached after the first check).
  await auth.getUser();

  if (to.meta.requiresAuth && !auth.isLoggedIn) {
    return { name: 'login', query: { redirect: to.fullPath } };
  }
  if (to.meta.guestOnly && auth.isLoggedIn) {
    return { name: 'projects' };
  }
  return true;
});

export default router;
