const { defineConfig, devices } = require('@playwright/test');

const isCI = !!process.env.CI;

module.exports = defineConfig({
  testDir: './e2e',
  globalSetup: './e2e/global-setup.js',
  globalTeardown: './e2e/global-teardown.js',
  timeout: isCI ? 90 * 1000 : 45 * 1000,
  expect: {
    timeout: isCI ? 10000 : 5000,
  },
  fullyParallel: false,
  workers: isCI ? 4 : 1,
  reporter: 'line',
  use: {
    baseURL: 'http://localhost:9901',
    trace: 'on-first-retry',
    screenshot: 'only-on-failure',
    headless: true,
  },
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
  ],
});
