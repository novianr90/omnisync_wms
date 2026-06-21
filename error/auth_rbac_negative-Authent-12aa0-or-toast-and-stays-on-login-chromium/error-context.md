# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: auth_rbac_negative.spec.js >> Authentication & RBAC Negative Flows >> Invalid login: wrong password shows error toast and stays on /login
- Location: e2e/auth_rbac_negative.spec.js:6:3

# Error details

```
Error: page.goto: net::ERR_CONNECTION_REFUSED at http://localhost:9903/login
Call log:
  - navigating to "http://localhost:9903/login", waiting until "load"

```

# Test source

```ts
  1  | const { test, expect } = require('./fixtures');
  2  | const { login } = require('./helpers/auth');
  3  | 
  4  | test.describe('Authentication & RBAC Negative Flows', () => {
  5  | 
  6  |   test('Invalid login: wrong password shows error toast and stays on /login', async ({ page }) => {
> 7  |     await page.goto('/login');
     |                ^ Error: page.goto: net::ERR_CONNECTION_REFUSED at http://localhost:9903/login
  8  |     await page.fill('#email', 'admin@omnisync.com');
  9  |     await page.fill('#password', 'wrongpassword123');
  10 |     await page.click('button[type="submit"]');
  11 | 
  12 |     await expect(page.locator('.notyf__toast').last()).toBeVisible();
  13 |     await expect(page.locator('.notyf__message').last()).toContainText('Invalid email or password');
  14 |     await expect(page).toHaveURL(/.*\/login$/);
  15 |   });
  16 | 
  17 |   test('Invalid login: non-existent email shows error toast and stays on /login', async ({ page }) => {
  18 |     await page.goto('/login');
  19 |     await page.fill('#email', 'nobody@omnisync.com');
  20 |     await page.fill('#password', 'somepassword');
  21 |     await page.click('button[type="submit"]');
  22 | 
  23 |     await expect(page.locator('.notyf__toast').last()).toBeVisible();
  24 |     await expect(page.locator('.notyf__message').last()).toContainText('Invalid email or password');
  25 |     await expect(page).toHaveURL(/.*\/login$/);
  26 |   });
  27 | 
  28 |   // operator@omnisync.com is seeded with POS role (no modify_masters permission)
  29 |   test('POS role blocked from modifying masters — Add Product button is hidden', async ({ page }) => {
  30 |     await login(page, 'operator@omnisync.com', 'operator123');
  31 | 
  32 |     // Operator CAN view the list
  33 |     await page.goto('/wms/masters/products');
  34 |     
  35 |     // Operator should NOT see the Add Product button
  36 |     const addBtn = page.locator('button[hx-get="/wms/masters/products/new"]');
  37 |     await expect(addBtn).toBeHidden();
  38 |   });
  39 | 
  40 |   test('Non-system-admin blocked from /wms/system/roles — shows Access Denied', async ({ page }) => {
  41 |     await login(page, 'operator@omnisync.com', 'operator123');
  42 | 
  43 |     await page.goto('/wms/system/roles');
  44 |     await expect(page.locator('body')).toContainText(/Access Denied|403/);
  45 |   });
  46 | 
  47 | });
  48 | 
```