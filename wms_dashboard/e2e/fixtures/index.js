const { test: base, expect } = require('@playwright/test');

const WMS_BASE_PORT = 9991;
// workerIndex increments on each worker restart; keep it within the pool size
const NUM_WORKERS = process.env.CI ? 4 : 1;

const test = base.extend({
  baseURL: async ({}, use, testInfo) => {
    await use(`http://localhost:${WMS_BASE_PORT + (testInfo.workerIndex % NUM_WORKERS)}`);
  },
});

module.exports = { test, expect };
