import { test, expect, type Page } from '@playwright/test';

const USER = { id: 1, username: 'forsen', email: 'forsen@ggtalentgroup.com' };

type Project = { id: string; name: string; created_at: string };

const json = (status: number, body: unknown) => ({
  status,
  contentType: 'application/json',
  body: JSON.stringify(body),
});

/** Installs a logged-in session and an in-memory projects collection. */
async function mockProjects(page: Page, initial: Project[] = []) {
  const state = { projects: [...initial] };
  let nextId = 1000;

  await page.route('**/api/auth/me', (route) => route.fulfill(json(200, USER)));
  await page.route('**/api/auth/logout', (route) => route.fulfill({ status: 204, body: '' }));

  await page.route('**/api/projects', (route) => {
    const request = route.request();
    if (request.method() === 'POST') {
      const { name } = JSON.parse(request.postData() ?? '{}');
      const project: Project = { id: `id-${nextId++}`, name, created_at: new Date().toISOString() };
      state.projects.unshift(project);
      return route.fulfill(json(200, project));
    }
    return route.fulfill(json(200, state.projects));
  });

  // DELETE /projects/{id}
  await page.route('**/api/projects/*', (route) => {
    const id = route.request().url().split('/').pop() ?? '';
    state.projects = state.projects.filter((p) => p.id !== id);
    return route.fulfill({ status: 204, body: '' });
  });

  return state;
}

const seeded: Project[] = [
  { id: 'id-1', name: 'forsenCD design doc', created_at: '2026-06-15T12:00:00Z' },
  { id: 'id-2', name: 'WeskerU notes', created_at: '2026-06-14T12:00:00Z' },
];

test('shows the empty state when there are no projects', async ({ page }) => {
  await mockProjects(page, []);
  await page.goto('/projects');

  await expect(page.getByText('No projects yet. Create your first one.')).toBeVisible();
});

test('lists existing projects with an editor link', async ({ page }) => {
  await mockProjects(page, seeded);
  await page.goto('/projects');

  await expect(page.getByText('forsenCD design doc')).toBeVisible();
  await expect(page.getByText('WeskerU notes')).toBeVisible();
  await expect(page.getByRole('link', { name: /forsenCD design doc/ })).toHaveAttribute(
    'href',
    '/editor/id-1',
  );
});

test('creates a project through the modal', async ({ page }) => {
  await mockProjects(page, []);
  await page.goto('/projects');

  await page.getByRole('button', { name: 'New project' }).click();

  const dialog = page.getByRole('dialog');
  await expect(dialog).toBeVisible();
  await dialog.getByRole('textbox', { name: 'Name' }).fill('TREEHUNTER doc');
  await dialog.getByRole('button', { name: 'Create' }).click();

  await expect(dialog).toBeHidden();
  await expect(page.getByText('TREEHUNTER doc')).toBeVisible();
});

test('deletes a project after confirmation', async ({ page }) => {
  await mockProjects(page, seeded);
  await page.goto('/projects');

  page.on('dialog', (d) => d.accept());

  const card = page.getByRole('listitem').filter({ hasText: 'forsenCD design doc' });
  await card.getByRole('button', { name: 'Delete' }).click();

  await expect(page.getByText('forsenCD design doc')).toBeHidden();
  await expect(page.getByText('WeskerU notes')).toBeVisible();
});

test('keeps the project when the delete confirm is dismissed', async ({ page }) => {
  await mockProjects(page, seeded);
  await page.goto('/projects');

  page.on('dialog', (d) => d.dismiss());

  const card = page.getByRole('listitem').filter({ hasText: 'forsenCD design doc' });
  await card.getByRole('button', { name: 'Delete' }).click();

  await expect(page.getByText('forsenCD design doc')).toBeVisible();
});
