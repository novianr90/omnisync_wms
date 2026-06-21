# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: rbac.spec.js >> Dynamic RBAC access control >> should allow custom role with modify_masters to view products but restrict ledger access
- Location: e2e/rbac.spec.js:5:5

# Error details

```
Error: page.goto: net::ERR_CONNECTION_REFUSED at http://localhost:9904/login
Call log:
  - navigating to "http://localhost:9904/login", waiting until "load"

```

# Test source

```ts
  1  | const { expect } = require('@playwright/test');
  2  | 
  3  | /**
  4  |  * Log in as a specified user and wait for redirect to dashboard
  5  |  * @param {import('@playwright/test').Page} page 
  6  |  * @param {string} email 
  7  |  * @param {string} password 
  8  |  */
  9  | async function login(page, email, password) {
> 10 |   await page.goto('/login');
     |              ^ Error: page.goto: net::ERR_CONNECTION_REFUSED at http://localhost:9904/login
  11 |   await page.fill('#email', email);
  12 |   await page.fill('#password', password);
  13 |   await page.click('button[type="submit"]');
  14 |   // Wait for redirect to dashboard
  15 |   await page.waitForURL('**/');
  16 |   await expect(page).toHaveURL(/.*\/$/);
  17 | }
  18 | 
  19 | module.exports = {
  20 |   login,
  21 | };
  22 | 
```