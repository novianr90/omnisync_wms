const { test: base, expect } = require('@playwright/test');

const WMS_BASE_PORT = 9901;

const test = base.extend({
  baseURL: [async ({ workerIndex }, use) => {
    await use(`http://localhost:${WMS_BASE_PORT + workerIndex}`);
  }, { scope: 'worker' }],
});

module.exports = { test, expect };
