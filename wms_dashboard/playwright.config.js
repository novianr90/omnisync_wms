const { defineConfig, devices } = require('@playwright/test');

const isCI = !!process.env.CI;

module.exports = defineConfig({
  testDir: './e2e',
  timeout: 30 * 1000,
  expect: {
    timeout: 5000,
  },
  fullyParallel: false,
  workers: 1, // Run sequentially to avoid DB locks and race conditions
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
  webServer: [
    {
      // In CI, run the pre-built binary; locally, use go run
      command: isCI
        ? 'cd ../auth_services && ./auth_services'
        : 'cd ../auth_services && go run cmd/main.go',
      url: 'http://localhost:8000/auth/verify', // Verify it's ready by polling the verify endpoint (returns 401 but indicates live server)
      reuseExistingServer: !isCI,
      stdout: 'pipe',
      stderr: 'pipe',
      timeout: 300 * 1000,
      env: {
        JWT_SECRET_KEY: 'test-signing-key-for-auth-services-unit-tests-12345',
      },
    },
    {
      // In CI, run the pre-built binary; locally, use go run
      command: isCI
        ? './wms_dashboard'
        : 'go run cmd/main.go',
      url: 'http://localhost:9901/login', // Verify dashboard is ready by loading the login page
      reuseExistingServer: !isCI,
      stdout: 'pipe',
      stderr: 'pipe',
      timeout: 300 * 1000,
      env: {
        JWT_SECRET_KEY: 'test-signing-key-for-auth-services-unit-tests-12345',
        ...(process.env.GOMODCACHE ? { GOMODCACHE: process.env.GOMODCACHE } : { GOMODCACHE: 'D:\\Code\\projects\\omnisync_wms\\go_cache\\pkg\\mod' }),
        ...(process.env.GOCACHE ? { GOCACHE: process.env.GOCACHE } : { GOCACHE: 'D:\\Code\\projects\\omnisync_wms\\go_cache\\build' }),
      },
    },
  ],
});
