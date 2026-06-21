# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: auth_nav.spec.js >> Authentication & Navigation E2E Flows >> Navigate to login -> enter wrong credentials -> verify failure toast
- Location: e2e/auth_nav.spec.js:6:3

# Error details

```
Error: page.goto: net::ERR_CONNECTION_REFUSED at http://localhost:9902/login
Call log:
  - navigating to "http://localhost:9902/login", waiting until "load"

```

# Test source

```ts
  1  | const { test, expect } = require('./fixtures');
  2  | const { login } = require('./helpers/auth');
  3  | 
  4  | test.describe('Authentication & Navigation E2E Flows', () => {
  5  | 
  6  |   test('Navigate to login -> enter wrong credentials -> verify failure toast', async ({ page }) => {
> 7  |     await page.goto('/login');
     |                ^ Error: page.goto: net::ERR_CONNECTION_REFUSED at http://localhost:9902/login
  8  | 
  9  |     // Attempt login with wrong password
  10 |     await page.fill('#email', 'admin@omnisync.com');
  11 |     await page.fill('#password', 'wrongpassword123');
  12 |     await page.click('button[type="submit"]');
  13 | 
  14 |     // Wait for the alert toast to appear
  15 |     const toast = page.locator('.notyf__toast');
  16 |     await expect(toast).toBeVisible();
  17 |     await expect(page.locator('.notyf__message')).toContainText('Invalid email or password');
  18 |     
  19 |     // URL should still be /login
  20 |     await expect(page).toHaveURL(/.*\/login$/);
  21 |   });
  22 | 
  23 |   test('Enter valid credentials -> verify redirect to dashboard', async ({ page }) => {
  24 |     await login(page, 'operator@omnisync.com', 'operator123');
  25 |     
  26 |     // Should be redirected to the main dashboard
  27 |     await expect(page).toHaveURL(/.*\/$/);
  28 |     
  29 |     // Check that we see user profile footer showing the operator info
  30 |     const footerUser = page.locator('#sidebar h4');
  31 |     await expect(footerUser).toContainText('Alex Mercer');
  32 |   });
  33 | 
  34 |   test('Verify sidebar navigation links work (HTMX partial swaps) and push history URLs', async ({ page }) => {
  35 |     // Start by logging in as administrator
  36 |     await login(page, 'admin@omnisync.com', 'admin123');
  37 | 
  38 |     // 1. Click on Products Master
  39 |     await page.click('a[href="/wms/masters/products"]');
  40 |     await expect(page).toHaveURL(/.*\/wms\/masters\/products$/);
  41 |     await expect(page.locator('#main-workspace h3')).toContainText('Product Master Maintenance');
  42 | 
  43 |     // 2. Click on QC Holds
  44 |     await page.click('a[href="/wms/qc-holds"]');
  45 |     await expect(page).toHaveURL(/.*\/wms\/qc-holds$/);
  46 |     await expect(page.locator('#main-workspace h2')).toContainText('Quality Control (QC) Holds');
  47 | 
  48 |     // 3. Click on Dashboard
  49 |     await page.click('a[href="/"]');
  50 |     await expect(page).toHaveURL(/.*\/$/);
  51 |     await expect(page.locator('header h2')).toContainText('Operational Center');
  52 |   });
  53 | 
  54 | });
  55 | 
```