const { test, expect } = require('./fixtures');
const { login } = require('./helpers/auth');

test.describe('Master Data Negative Flows', () => {

  test.beforeEach(async ({ page }) => {
    await login(page, 'admin@omnisync.com', 'admin123');
  });

  test('Duplicate SKU: creating product with existing SKU shows constraint error', async ({ page }) => {
    test.setTimeout(60000);
    await page.goto('http://localhost:9901/wms/masters/products');
    await expect(page.locator('h3:has-text("Product Master Maintenance")')).toBeVisible();

    // Button text is "Add Catalog Product"
    await page.click('button:has-text("Add Catalog Product")');
    await expect(page.locator('#modal-container [id*="modal"], #modal-container [id*="Modal"]')).toBeVisible();

    // PROD-KYBD-01 is seeded — duplicate SKU triggers unique constraint error
    const modal = page.locator('#modal-container').first();
    await modal.locator('input[name="sku"]').fill('PROD-KYBD-01');
    await modal.locator('input[name="name"]').fill('Duplicate Keyboard');
    
    // Select base UoM to pass HTML5 validation
    await modal.locator('select[name="uom_id"]').selectOption({ index: 1 });

    await modal.locator('button[type="submit"]').click();

    await expect(page.locator('.notyf__toast').last()).toBeVisible();
    await expect(page.locator('.notyf__message').last()).toContainText(/already exists|duplicate|unique/i);
  });

  test('Duplicate warehouse code: creating warehouse with existing code shows constraint error', async ({ page }) => {
    test.setTimeout(60000);
    await page.goto('http://localhost:9901/wms/masters/warehouses');
    await expect(page.locator('h3:has-text("Warehouse Facilities Registry")')).toBeVisible();

    // Button text is "Add Warehouse Node"
    await page.click('button:has-text("Add Warehouse Node")');
    await expect(page.locator('#modal-container [id*="modal"]')).toBeVisible();

    const modal = page.locator('#modal-container').first();
    await modal.locator('input[name="code"]').fill('WH-MAIN');
    await modal.locator('input[name="name"]').fill('Duplicate Main Warehouse');
    await modal.locator('button[type="submit"]').click();

    await expect(page.locator('.notyf__toast').last()).toBeVisible();
    await expect(page.locator('.notyf__message').last()).toContainText(/already exists|duplicate|unique/i);
  });

  test('Delete locator with active stock is blocked', async ({ page }) => {
    test.setTimeout(60000);
    await page.goto('http://localhost:9901/wms/masters/locators');
    await expect(page.locator('h3, h2').filter({ hasText: /Locator/i })).toBeVisible();

    // Accept native browser confirm dialog triggered by HTMX hx-confirm
    page.once('dialog', dialog => dialog.accept());

    // Attempt to delete the locator with stock (loc-001 is WH-MAIN-Zone-A-Aisle-1-Shelf-1-Level-1)
    const locatorRow = page.locator('tr:has-text("WH-MAIN-Zone-A-Aisle-1-Shelf-1-Level-1")');
    await locatorRow.locator('button[title*="Delete"]').click();

    await expect(page.locator('.notyf__toast').last()).toBeVisible();
    await expect(page.locator('.notyf__message').last()).toContainText(/cannot delete|in use|stock/i);
  });

  test('Delete base UoM referenced by active products is blocked', async ({ page }) => {
    test.setTimeout(60000);
    await page.goto('http://localhost:9901/wms/masters/uoms');
    await expect(page.locator('h3, h2').filter({ hasText: /UoM|Unit/i })).toBeVisible();

    // Accept native browser confirm dialog triggered by HTMX hx-confirm
    page.once('dialog', dialog => dialog.accept());

    // Attempt to delete UoM row referenced by products (PCS)
    const uomRow = page.locator('tr:has-text("PCS")');
    await uomRow.locator('button[title*="Delete"]').click();

    await expect(page.locator('.notyf__toast').last()).toBeVisible();
    await expect(page.locator('.notyf__message').last()).toContainText(/cannot delete|in use|referenced/i);
  });

});
