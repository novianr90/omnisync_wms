const { test, expect } = require('@playwright/test');
const { login } = require('./helpers/auth');

test.describe('Master Data Negative Flows', () => {

  test.beforeEach(async ({ page }) => {
    await login(page, 'admin@omnisync.com', 'admin123');
  });

  test('Duplicate SKU: creating product with existing SKU shows constraint error', async ({ page }) => {
    test.setTimeout(60000);
    await page.goto('http://localhost:9901/wms/masters/products');
    await expect(page.locator('h3:has-text("Product Master Maintenance")')).toBeVisible();

    await page.click('button:has-text("Add Product")');
    await expect(page.locator('[id*="modal"], [id*="Modal"]').filter({ visible: true })).toBeVisible();

    // PROD-KYBD-01 is seeded — use existing SKU to trigger duplicate error
    const modal = page.locator('[id*="modal"], [id*="Modal"]').filter({ visible: true }).first();
    await modal.locator('input[name="sku"]').fill('PROD-KYBD-01');
    await modal.locator('input[name="name"]').fill('Duplicate Keyboard');
    await modal.locator('button[type="submit"]').click();

    await expect(page.locator('.notyf__toast')).toBeVisible();
    await expect(page.locator('.notyf__message')).toContainText(/already exists|duplicate|unique/i);
  });

  test('Duplicate warehouse code: creating warehouse with existing code shows constraint error', async ({ page }) => {
    test.setTimeout(60000);
    await page.goto('http://localhost:9901/wms/masters/warehouses');
    await expect(page.locator('h3, h2').filter({ hasText: /Warehouse/i })).toBeVisible();

    await page.click('button:has-text("Add Warehouse")');
    await expect(page.locator('[id*="modal"], [id*="Modal"]').filter({ visible: true })).toBeVisible();

    const modal = page.locator('[id*="modal"], [id*="Modal"]').filter({ visible: true }).first();
    await modal.locator('input[name="code"]').fill('WH-MAIN');
    await modal.locator('input[name="name"]').fill('Duplicate Main Warehouse');
    await modal.locator('button[type="submit"]').click();

    await expect(page.locator('.notyf__toast')).toBeVisible();
    await expect(page.locator('.notyf__message')).toContainText(/already exists|duplicate|unique/i);
  });

  test('Delete locator with active stock is blocked', async ({ page }) => {
    test.setTimeout(60000);
    await page.goto('http://localhost:9901/wms/masters/locators');
    await expect(page.locator('h3, h2').filter({ hasText: /Locator/i })).toBeVisible();

    // Attempt to delete the first locator (seeded with stock: loc-001)
    const firstRow = page.locator('tbody tr').first();
    await firstRow.locator('button:has-text("Delete")').click();

    // Confirm deletion in modal if present
    const confirmModal = page.locator('div[id^="confirm-modal-"]').filter({ visible: true });
    if (await confirmModal.isVisible()) {
      await confirmModal.locator('button:has-text("Confirm")').click();
    }

    await expect(page.locator('.notyf__toast')).toBeVisible();
    await expect(page.locator('.notyf__message')).toContainText(/cannot delete|in use|stock/i);
  });

  test('Delete base UoM referenced by active products is blocked', async ({ page }) => {
    test.setTimeout(60000);
    await page.goto('http://localhost:9901/wms/masters/uom');
    await expect(page.locator('h3, h2').filter({ hasText: /UoM|Unit/i })).toBeVisible();

    // Attempt to delete first UoM row (expected to be referenced by products)
    const firstRow = page.locator('tbody tr').first();
    await firstRow.locator('button:has-text("Delete")').click();

    const confirmModal = page.locator('div[id^="confirm-modal-"]').filter({ visible: true });
    if (await confirmModal.isVisible()) {
      await confirmModal.locator('button:has-text("Confirm")').click();
    }

    await expect(page.locator('.notyf__toast')).toBeVisible();
    await expect(page.locator('.notyf__message')).toContainText(/cannot delete|in use|referenced/i);
  });

});
