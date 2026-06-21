const { test, expect } = require('./fixtures');
const { login } = require('./helpers/auth');

test.describe('Inventory Ledger Export', () => {
    test.beforeEach(async ({ page }) => {
        // Authenticate as System Admin
        await login(page, 'admin@omnisync.com', 'admin123');
    });

    test('should render export buttons and download PDF/Excel reports', async ({ page }) => {
        // Navigate to Dashboard
        await page.goto('/');
        
        // Navigate to Ledger
        const ledgerLink = page.locator('a[href="/wms/ledger"]');
        await expect(ledgerLink).toBeVisible();
        await ledgerLink.click();

        // Verify page header
        await expect(page.locator('h1:has-text("Inventory Ledger")')).toBeVisible();

        // Verify export buttons exist
        const pdfBtn = page.locator('button:has-text("Export PDF")');
        const excelBtn = page.locator('button:has-text("Export Excel")');
        await expect(pdfBtn).toBeVisible();
        await expect(excelBtn).toBeVisible();

        // Test PDF Download
        const [pdfDownload] = await Promise.all([
            page.waitForEvent('download'),
            pdfBtn.click(),
        ]);
        expect(pdfDownload.suggestedFilename()).toContain('.pdf');

        // Test Excel Download
        const [excelDownload] = await Promise.all([
            page.waitForEvent('download'),
            excelBtn.click(),
        ]);
        expect(excelDownload.suggestedFilename()).toContain('.xlsx');
    });
});
