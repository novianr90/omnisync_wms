# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: qc_hold_adjustments.spec.js >> QC Holds & Stock Adjustments E2E Flows >> QC Hold Cycle: Freeze Stock -> Verify reduction -> Release Hold -> Verify restore
- Location: e2e/qc_hold_adjustments.spec.js:11:3

# Error details

```
Error: page.goto: net::ERR_CONNECTION_REFUSED at http://localhost:9903/login
Call log:
  - navigating to "http://localhost:9903/login", waiting until "load"

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
     |              ^ Error: page.goto: net::ERR_CONNECTION_REFUSED at http://localhost:9903/login
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