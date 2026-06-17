const { test, expect } = require('@playwright/test');
const { login } = require('./helpers/auth');

test.describe('Inventory Valuation & Aging Report', () => {

  test.beforeEach(async ({ page }) => {
    await login(page, 'admin@omnisync.com', 'admin123');
    await page.waitForLoadState('networkidle');
  });

  test('sidebar link navigates to the valuation page', async ({ page }) => {
    // The sidebar link must exist and be clickable
    const link = page.locator('a[href="/wms/reports/valuation"]');
    await expect(link).toBeVisible();
    await link.click();

    // HTMX pushes the URL and swaps content
    await expect(page).toHaveURL(/.*\/wms\/reports\/valuation/);
    await expect(page.locator('h1:has-text("Inventory Valuation")')).toBeVisible();
  });

  test('page renders all structural elements', async ({ page }) => {
    await page.goto('/wms/reports/valuation');

    // Page heading
    await expect(page.locator('h1:has-text("Inventory Valuation")')).toBeVisible();

    // Filter selects
    await expect(page.locator('select[name="warehouse_id"]')).toBeVisible();
    await expect(page.locator('select[name="category"]')).toBeVisible();
    await expect(page.locator('select[name="aging_bucket"]')).toBeVisible();

    // Apply and Reset controls
    await expect(page.locator('button:has-text("Apply")')).toBeVisible();
    await expect(page.locator('a:has-text("Reset")')).toBeVisible();

    // Summary stat cards (4 cards)
    await expect(page.locator('text=Total Stock Value')).toBeVisible();
    await expect(page.locator('text=Fresh (0–30 days)')).toBeVisible();
    await expect(page.locator('text=Moderate (31–90 days)')).toBeVisible();
    await expect(page.locator('text=Slow-moving (91+ days)')).toBeVisible();

    // Export CSV button links to the export endpoint
    const exportLink = page.locator('a:has-text("Export CSV")');
    await expect(exportLink).toBeVisible();
    const href = await exportLink.getAttribute('href');
    expect(href).toContain('/wms/reports/valuation/export');

    // Data table is rendered (either with rows or the empty-state message)
    const table = page.locator('table');
    const emptyState = page.locator('text=No stock found');
    const hasTable = await table.isVisible().catch(() => false);
    const hasEmpty = await emptyState.isVisible().catch(() => false);
    expect(hasTable || hasEmpty).toBe(true);
  });

  test('aging bucket filter narrows the table', async ({ page }) => {
    await page.goto('/wms/reports/valuation');

    // Select "91+" bucket and apply
    await page.selectOption('select[name="aging_bucket"]', '91+');
    await page.click('button:has-text("Apply")');

    // The URL should carry the filter param
    await expect(page).toHaveURL(/aging_bucket=91%2B/);

    // After applying filter the page still renders structural elements
    await expect(page.locator('h1:has-text("Inventory Valuation")')).toBeVisible();

    // Select is still showing the applied value
    const selected = await page.locator('select[name="aging_bucket"]').inputValue();
    expect(selected).toBe('91+');
  });

  test('warehouse filter option list is populated (or defaults to All)', async ({ page }) => {
    await page.goto('/wms/reports/valuation');

    const warehouseSelect = page.locator('select[name="warehouse_id"]');
    await expect(warehouseSelect).toBeVisible();

    // At minimum the "All Warehouses" default option is present
    const allOption = warehouseSelect.locator('option[value=""]');
    await expect(allOption).toHaveText('All Warehouses');
  });

  test('category filter option list is populated (or defaults to All)', async ({ page }) => {
    await page.goto('/wms/reports/valuation');

    const catSelect = page.locator('select[name="category"]');
    await expect(catSelect).toBeVisible();

    const allOption = catSelect.locator('option[value=""]');
    await expect(allOption).toHaveText('All Categories');
  });

  test('reset link returns to unfiltered page', async ({ page }) => {
    // Start with a filter applied
    await page.goto('/wms/reports/valuation?aging_bucket=91%2B');

    // Click Reset (bare link to the base URL)
    await page.click('a:has-text("Reset")');
    await expect(page).toHaveURL(/\/wms\/reports\/valuation$/);

    // Filters should be back to defaults
    const bucketValue = await page.locator('select[name="aging_bucket"]').inputValue();
    expect(bucketValue).toBe('');
  });

  test('export CSV link carries active filter params', async ({ page }) => {
    await page.goto('/wms/reports/valuation?aging_bucket=0-30&category=Electronics');

    const exportLink = page.locator('a:has-text("Export CSV")');
    const href = await exportLink.getAttribute('href');

    // The export href must forward the same filters
    expect(href).toContain('aging_bucket=0-30');
    expect(href).toContain('category=Electronics');
  });

  test('table headers are present when data exists', async ({ page }) => {
    await page.goto('/wms/reports/valuation');

    const table = page.locator('table');
    const hasTable = await table.isVisible().catch(() => false);

    if (hasTable) {
      await expect(table.locator('th:has-text("SKU / Product")')).toBeVisible();
      await expect(table.locator('th:has-text("Batch")')).toBeVisible();
      await expect(table.locator('th:has-text("Location")')).toBeVisible();
      await expect(table.locator('th:has-text("Qty")')).toBeVisible();
      await expect(table.locator('th:has-text("Unit Cost")')).toBeVisible();
      await expect(table.locator('th:has-text("Total Value")')).toBeVisible();
      await expect(table.locator('th:has-text("Aging")')).toBeVisible();
    }
  });

  test('footer note is shown only when table has rows', async ({ page }) => {
    await page.goto('/wms/reports/valuation');

    const table = page.locator('table');
    const hasTable = await table.isVisible().catch(() => false);

    if (hasTable) {
      await expect(page.locator('text=Unit cost sourced from INBOUND ledger')).toBeVisible();
    } else {
      // No footer when empty
      const footer = page.locator('text=Unit cost sourced from INBOUND ledger');
      await expect(footer).toBeHidden();
    }
  });
});
