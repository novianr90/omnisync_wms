const { test, expect } = require('./fixtures');
const { login } = require('./helpers/auth');

test.describe('In-Progress Documents page', () => {
    test.beforeEach(async ({ page }) => {
        // Authenticate as System Admin
        await login(page, 'admin@omnisync.com', 'admin123');
    });

    test('should allow user to navigate to and interact with in-progress documents page', async ({ page }) => {
        // Navigate to the Dashboard first
        await page.goto('/');
        
        // Wait for the sidebar to load and click on In Progress Docs
        const inProgressLink = page.locator('a[href="/wms/in-progress"]');
        await expect(inProgressLink).toBeVisible();
        await inProgressLink.click();

        // Wait for the HTMX swap to complete and verify the header
        await expect(page.locator('h3:has-text("Active Transaction Queue")')).toBeVisible();

        // Verify that search input and select dropdown exist
        const searchInput = page.locator('#search-input');
        const typeSelect = page.locator('#type-select');
        await expect(searchInput).toBeVisible();
        await expect(typeSelect).toBeVisible();

        // Type a search query to trigger HTMX filter
        await searchInput.fill('MOV-999');
        await page.waitForTimeout(500); // Debounce delay

        // Change select filter
        await typeSelect.selectOption('Movement');
        await page.waitForTimeout(500);

        // Verify queue size badge is visible
        await expect(page.locator('#queue-size-badge')).toBeVisible();

        // Verify table and headers are visible
        const table = page.locator('#in-progress-tbody');
        await expect(table).toBeAttached();
    });
});
