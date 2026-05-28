const { expect } = require('@playwright/test');

/**
 * Log in as a specified user and wait for redirect to dashboard
 * @param {import('@playwright/test').Page} page 
 * @param {string} email 
 * @param {string} password 
 */
async function login(page, email, password) {
  await page.goto('/login');
  await page.fill('#email', email);
  await page.fill('#password', password);
  await page.click('button[type="submit"]');
  // Wait for redirect to dashboard
  await page.waitForURL('**/');
  await expect(page).toHaveURL(/.*\/$/);
}

module.exports = {
  login,
};
