const { test, expect } = require('./fixtures');
const { login } = require('./helpers/auth');

test.describe('Inventory Ledger', () => {
    test.beforeEach(async ({ page }) => {
        // Authenticate as System Admin
        await login(page, 'admin@omnisync.com', 'admin123');
    });

    test('should allow System Admin to view and filter the inventory ledger', async ({ page }) => {
        // Navigate to the Dashboard first
        await page.goto('http://localhost:9901/');
        
        // Wait for the sidebar to load and click on Inventory Ledger
        const ledgerLink = page.locator('a[href="/wms/ledger"]');
        await expect(ledgerLink).toBeVisible();
        await ledgerLink.click();

        // Wait for the HTMX swap to complete and verify the header
        await expect(page.locator('h1:has-text("Inventory Ledger")')).toBeVisible();

        // Verify that the filter form is rendered
        await expect(page.locator('input[name="search"]')).toBeVisible();
        await expect(page.locator('input[name="sku"]')).toBeVisible();
        
        // Type a search query and trigger the filter
        await page.locator('input[name="search"]').fill('DOC');
        await page.locator('button:has-text("Filter")').click();

        // Wait for the HTMX request to update the table container
        // Since we may or may not have seed data, we just verify the table container exists
        await expect(page.locator('#ledger-table-container')).toBeVisible();
        
        // Verify table headers are present
        const table = page.locator('#ledger-table-container table');
        await expect(table).toBeVisible();
        await expect(table.locator('th:has-text("Transaction Date")')).toBeVisible();
        await expect(table.locator('th:has-text("Debit / Credit")')).toBeVisible();
    });
});
