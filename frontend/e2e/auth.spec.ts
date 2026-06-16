import { test, expect, type Page } from '@playwright/test';

// These tests mock the backend at the network boundary (`page.route`), so they
// exercise the frontend's auth behaviour — guard redirects, the login/register
// toggle, the `?redirect` round-trip, session hydration and logout — without a
// real server. The actual API + cookie handling is covered by the backend's own
// e2e tests, so there is deliberately no full-stack test here.

type MockUser = { id: number; username: string; email: string };

const GOOD_PASSWORD = 'treehunter';

/**
 * Installs fake `/api/auth/*` routes backed by an in-memory session. Returns the
 * mutable state so a test can assert on it or pre-seed a logged-in user.
 */
async function mockAuth(page: Page, initialUser: MockUser | null = null) {
  const state: { user: MockUser | null } = { user: initialUser };

  const json = (status: number, body: unknown) => ({
    status,
    contentType: 'application/json',
    body: JSON.stringify(body),
  });
  const unauthorized = json(401, {
    title: 'Unauthorized',
    detail: 'Invalid username or password',
    status: 401,
  });

  await page.route('**/api/auth/me', (route) =>
    route.fulfill(state.user ? json(200, state.user) : unauthorized),
  );

  await page.route('**/api/auth/login', (route) => {
    const { username, password } = JSON.parse(route.request().postData() ?? '{}');
    if (password !== GOOD_PASSWORD) {
      return route.fulfill(unauthorized);
    }
    state.user = { id: 1, username, email: `${username}@example.com` };
    return route.fulfill(json(200, state.user));
  });

  await page.route('**/api/auth/register', (route) => {
    const { username, email } = JSON.parse(route.request().postData() ?? '{}');
    state.user = { id: 1, username, email };
    return route.fulfill(json(200, state.user));
  });

  await page.route('**/api/auth/logout', (route) => {
    state.user = null;
    return route.fulfill({ status: 204, body: '' });
  });

  return state;
}

test('redirects an unauthenticated visitor to /login with a redirect param', async ({ page }) => {
  await mockAuth(page);
  await page.goto('/');
  await expect(page).toHaveURL('/login?redirect=/projects');
  await expect(page.getByRole('heading', { name: 'fossenCD' })).toBeVisible();
});

test('toggles between login and register, revealing the email field', async ({ page }) => {
  await mockAuth(page);
  await page.goto('/login');

  await expect(page.getByRole('textbox', { name: 'Email' })).toBeHidden();
  await expect(page.getByRole('button', { name: 'Sign in' })).toBeVisible();

  await page.getByRole('link', { name: 'Create one' }).click();

  await expect(page.getByRole('textbox', { name: 'Email' })).toBeVisible();
  await expect(page.getByRole('button', { name: 'Create account' })).toBeVisible();
});

test('logs in and lands on the redirect target', async ({ page }) => {
  await mockAuth(page);
  await page.goto('/login?redirect=/projects');

  await page.getByRole('textbox', { name: 'Username' }).fill('forsen');
  await page.getByRole('textbox', { name: 'Password' }).fill(GOOD_PASSWORD);
  await page.getByRole('button', { name: 'Sign in' }).click();

  await expect(page).toHaveURL('/projects');
  await expect(page.getByText('forsen')).toBeVisible();
});

test('registers a new account and lands on /projects', async ({ page }) => {
  await mockAuth(page);
  await page.goto('/login');
  await page.getByRole('link', { name: 'Create one' }).click();

  await page.getByRole('textbox', { name: 'Username' }).fill('WeskerU');
  await page.getByRole('textbox', { name: 'Email' }).fill('weskeru@ugandagaming.net');
  await page.getByRole('textbox', { name: 'Password' }).fill(GOOD_PASSWORD);
  await page.getByRole('button', { name: 'Create account' }).click();

  await expect(page).toHaveURL('/projects');
  await expect(page.getByText('WeskerU')).toBeVisible();
});

test('shows an error and stays on /login for bad credentials', async ({ page }) => {
  await mockAuth(page);
  await page.goto('/login');

  await page.getByRole('textbox', { name: 'Username' }).fill('forsen');
  await page.getByRole('textbox', { name: 'Password' }).fill('wrong-password');
  await page.getByRole('button', { name: 'Sign in' }).click();

  await expect(page.getByText('Invalid username or password')).toBeVisible();
  await expect(page).toHaveURL('/login');
});

test('hydrates an existing session on direct navigation', async ({ page }) => {
  await mockAuth(page, { id: 1, username: 'VADIKUS007', email: 'vadikus007@example.com' });
  await page.goto('/projects');

  await expect(page).toHaveURL('/projects');
  await expect(page.getByText('VADIKUS007')).toBeVisible();
});

test('logs out and returns to /login', async ({ page }) => {
  await mockAuth(page, { id: 1, username: 'VADIKUS007', email: 'vadikus007@example.com' });
  await page.goto('/projects');

  await page.getByRole('button', { name: 'Log out' }).click();

  await expect(page).toHaveURL('/login');
});

test('blocks a protected route after logout', async ({ page }) => {
  const state = await mockAuth(page, { id: 1, username: 'VADIKUS007', email: 'vadikus007@example.com' });
  await page.goto('/projects');
  await expect(page.getByText('VADIKUS007')).toBeVisible();

  await page.getByRole('button', { name: 'Log out' }).click();
  await expect(page).toHaveURL('/login');

  // Session is gone; the guard should bounce a direct visit back to /login.
  expect(state.user).toBeNull();
  await page.goto('/projects');
  await expect(page).toHaveURL('/login?redirect=/projects');
});
