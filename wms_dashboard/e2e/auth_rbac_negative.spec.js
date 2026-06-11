const { test, expect } = require('@playwright/test');
const { login } = require('./helpers/auth');

test.describe('Authentication & RBAC Negative Flows', () => {

  test('Invalid login: wrong password shows error toast and stays on /login', async ({ page }) => {
    await page.goto('/login');
    await page.fill('#email', 'admin@omnisync.com');
    await page.fill('#password', 'wrongpassword123');
    await page.click('button[type="submit"]');

    await expect(page.locator('.notyf__toast')).toBeVisible();
    await expect(page.locator('.notyf__message')).toContainText('Invalid email or password');
    await expect(page).toHaveURL(/.*\/login$/);
  });

  test('Invalid login: non-existent email shows error toast and stays on /login', async ({ page }) => {
    await page.goto('/login');
    await page.fill('#email', 'nobody@omnisync.com');
    await page.fill('#password', 'somepassword');
    await page.click('button[type="submit"]');

    await expect(page.locator('.notyf__toast')).toBeVisible();
    await expect(page.locator('.notyf__message')).toContainText('Invalid email or password');
    await expect(page).toHaveURL(/.*\/login$/);
  });

  // operator@omnisync.com is seeded with POS role (no modify_masters permission)
  test('POS role blocked from /wms/masters/* — shown Access Denied', async ({ page }) => {
    await login(page, 'operator@omnisync.com', 'operator123');

    await page.goto('http://localhost:9901/wms/masters/products');
    await expect(page.locator('body')).toContainText(/Access Denied|403/);
  });

  test('Non-system-admin blocked from /wms/system/roles — shows Access Denied', async ({ page }) => {
    await login(page, 'operator@omnisync.com', 'operator123');

    await page.goto('http://localhost:9901/wms/system/roles');
    await expect(page.locator('body')).toContainText(/Access Denied|403/);
  });

});
