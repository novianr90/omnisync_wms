const { test, expect } = require('./fixtures');
const { login } = require('./helpers/auth');

test.describe('Authentication & Navigation E2E Flows', () => {

  test('Navigate to login -> enter wrong credentials -> verify failure toast', async ({ page }) => {
    await page.goto('/login');

    // Attempt login with wrong password
    await page.fill('#email', 'admin@omnisync.com');
    await page.fill('#password', 'wrongpassword123');
    await page.click('button[type="submit"]');

    // Wait for the alert toast to appear
    const toast = page.locator('.notyf__toast');
    await expect(toast).toBeVisible();
    await expect(page.locator('.notyf__message')).toContainText('Invalid email or password');
    
    // URL should still be /login
    await expect(page).toHaveURL(/.*\/login$/);
  });

  test('Enter valid credentials -> verify redirect to dashboard', async ({ page }) => {
    await login(page, 'operator@omnisync.com', 'operator123');
    
    // Should be redirected to the main dashboard
    await expect(page).toHaveURL(/.*\/$/);
    
    // Check that we see user profile footer showing the operator info
    const footerUser = page.locator('#sidebar h4');
    await expect(footerUser).toContainText('Alex Mercer');
  });

  test('Verify sidebar navigation links work (HTMX partial swaps) and push history URLs', async ({ page }) => {
    // Start by logging in as administrator
    await login(page, 'admin@omnisync.com', 'admin123');

    // 1. Click on Products Master
    await page.click('a[href="/wms/masters/products"]');
    await expect(page).toHaveURL(/.*\/wms\/masters\/products$/);
    await expect(page.locator('#main-workspace h3')).toContainText('Product Master Maintenance');

    // 2. Click on QC Holds
    await page.click('a[href="/wms/qc-holds"]');
    await expect(page).toHaveURL(/.*\/wms\/qc-holds$/);
    await expect(page.locator('#main-workspace h2')).toContainText('Quality Control (QC) Holds');

    // 3. Click on Dashboard
    await page.click('a[href="/"]');
    await expect(page).toHaveURL(/.*\/$/);
    await expect(page.locator('header h2')).toContainText('Operational Center');
  });

});
