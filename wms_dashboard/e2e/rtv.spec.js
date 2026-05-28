const { test, expect } = require('@playwright/test');
const { login } = require('./helpers/auth');

test.describe('Return to Vendor (RTV) E2E Operational Flows', () => {

  test.beforeEach(async ({ page }) => {
    // Standard login as administrator
    await login(page, 'admin@omnisync.com', 'admin123');
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(500);
  });

  test('RTV Regular Stock Cycle: Create -> Claim -> Journalize -> Complete', async ({ page }) => {
    // 1. Navigate to Return to Vendor page
    await page.click('a[href="/wms/rtv"]');
    await expect(page).toHaveURL(/.*\/wms\/rtv$/);

    // 2. Open Create Return Ticket Modal
    await page.click('button:has-text("Create Return Ticket")');
    await expect(page.locator('#modal-create-rtv')).toBeVisible();

    // 3. Fill the Form for Regular Stock Return
    // Select Keychron Keyboard product
    await page.selectOption('#rtv-product-select', 'prod-001');

    // Wait for HTMX dynamic lots to load
    await page.waitForTimeout(500);

    // Select the first eligible storage lot
    await page.selectOption('#rtv-lot-select', { index: 1 });

    // Verify info box displays with stocks
    await expect(page.locator('#lot-info-box')).toBeVisible();

    // Select Regular Stock, input quantity, write remarks
    await page.check('input[name="is_from_hold"][value="false"]');
    await page.fill('input[name="quantity"]', '10');
    await page.fill('textarea[name="remarks"]', 'E2E regular vendor return operation');

    // 4. Submit
    await page.click('button:has-text("Submit Return")');

    // Verify success toast notification
    await expect(page.locator('.notyf__toast')).toBeVisible();
    await expect(page.locator('.notyf__message')).toContainText('created successfully');

    // 5. Verify the created RTV row in table is OPEN
    const rtvRow = page.locator('tr:has-text("OPEN")').first();
    await expect(rtvRow).toBeVisible();
    await expect(rtvRow.locator('td:nth-child(2)')).toContainText('PROD-KYBD-01');
    await expect(rtvRow.locator('td:nth-child(5)')).toContainText('10');

    // Extract unique Document Number
    const docNo = await rtvRow.locator('td:first-child').innerText();
    const activeRow = page.locator(`tr:has-text("${docNo.trim()}")`);

    // 6. Claim the RTV task
    await activeRow.locator('button:has-text("Claim")').click();

    // Verify success toast
    await expect(page.locator('.notyf__toast')).toBeVisible();
    await expect(page.locator('.notyf__message')).toContainText('claimed successfully');

    // Verify status changes to IN_PROGRESS
    const inProgressRow = page.locator(`tr:has-text("${docNo.trim()}")`);
    await expect(inProgressRow.locator('td:nth-child(7)')).toContainText('IN_PROGRESS');

    // 7. Journalize the RTV task
    await inProgressRow.locator('button:has-text("Journalize")').click();

    // Verify success toast
    await expect(page.locator('.notyf__toast')).toBeVisible();
    await expect(page.locator('.notyf__message')).toContainText('successfully journaled');

    // Verify status changes to JOURNALED
    const journaledRow = page.locator(`tr:has-text("${docNo.trim()}")`);
    await expect(journaledRow.locator('td:nth-child(7)')).toContainText('JOURNALED');

    // 8. Complete the RTV task
    await journaledRow.locator('button:has-text("Complete")').click();

    // Verify success toast
    await expect(page.locator('.notyf__toast')).toBeVisible();
    await expect(page.locator('.notyf__message')).toContainText('completed successfully');

    // Verify status changes to COMPLETED
    const completedRow = page.locator(`tr:has-text("${docNo.trim()}")`);
    await expect(completedRow.locator('td:nth-child(7)')).toContainText('COMPLETED');
  });

});
