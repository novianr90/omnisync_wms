const { defineConfig, devices } = require('@playwright/test');
const fs = require('fs');
const path = require('path');

const isCI = !!process.env.CI;

// Auto-detect pre-built binaries to run them instead of slow 'go run'
const authBinaryDir = path.join(__dirname, '../auth_services');
const authBinaryName = process.platform === 'win32' ? 'auth_services.exe' : 'auth_services';
const useAuthBinary = fs.existsSync(path.join(authBinaryDir, authBinaryName));

const wmsBinaryDir = __dirname;
const wmsBinaryName = process.platform === 'win32' ? 'wms_dashboard.exe' : 'wms_dashboard';
const useWmsBinary = fs.existsSync(path.join(wmsBinaryDir, wmsBinaryName));

const authCmd = useAuthBinary
  ? (process.platform === 'win32' ? 'auth_services.exe' : './auth_services')
  : 'go run cmd/main.go';

const wmsCmd = useWmsBinary
  ? (process.platform === 'win32' ? 'wms_dashboard.exe' : './wms_dashboard')
  : 'go run cmd/main.go';

module.exports = defineConfig({
  testDir: './e2e',
  // Timeout per test (CI gets 90s to account for slow runners/DB setup, local gets 45s)
  timeout: isCI ? 90 * 1000 : 45 * 1000,
  expect: {
    timeout: isCI ? 10000 : 5000,
  },
  fullyParallel: false,
  workers: 1,
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
      command: authCmd,
      cwd: '../auth_services',
      url: 'http://localhost:8000/health', // Standardized health check returns 200 OK
      reuseExistingServer: !isCI,
      stdout: 'inherit', // Stream logs in real-time to terminal
      stderr: 'inherit',
      timeout: 60 * 1000, // Pre-built binary boots instantly, 60s is extremely safe
      env: {
        PORT: '8000',
        JWT_SECRET_KEY: process.env.JWT_SECRET_KEY || 'test-signing-key-for-auth-services-unit-tests-12345',
        DB_TYPE: process.env.DB_TYPE || 'sqlite',
        ...(process.env.AUTH_DATABASE_URL ? { AUTH_DATABASE_URL: process.env.AUTH_DATABASE_URL } : {}),
      },
    },
    {
      command: wmsCmd,
      cwd: '.',
      url: 'http://localhost:9901/health', // Standardized health check returns 200 OK
      reuseExistingServer: !isCI,
      stdout: 'inherit', // Stream logs in real-time to terminal
      stderr: 'inherit',
      timeout: 60 * 1000, // Pre-built binary boots instantly, 60s is extremely safe
      env: {
        PORT: '9901',
        JWT_SECRET_KEY: process.env.JWT_SECRET_KEY || 'test-signing-key-for-auth-services-unit-tests-12345',
        AUTH_API_URL: 'http://localhost:8000',
        DB_TYPE: process.env.DB_TYPE || 'sqlite',
        ...(process.env.WMS_DATABASE_URL ? { WMS_DATABASE_URL: process.env.WMS_DATABASE_URL } : {}),
        ...(process.env.GOMODCACHE ? { GOMODCACHE: process.env.GOMODCACHE } : {}),
        ...(process.env.GOCACHE ? { GOCACHE: process.env.GOCACHE } : {}),
      },
    },
  ],
});
