const { test: base, expect } = require('@playwright/test');

const WMS_BASE_PORT = 9901;

const test = base.extend({
  baseURL: async ({}, use, testInfo) => {
    await use(`http://localhost:${WMS_BASE_PORT + testInfo.workerIndex}`);
  },
});

module.exports = { test, expect };
