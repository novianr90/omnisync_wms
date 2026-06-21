const { test, expect } = require('./fixtures');
const { login } = require('./helpers/auth');

test.describe('Inventory Transfers E2E Operational Flows', () => {

  test.beforeEach(async ({ page }) => {
    // Standard login as administrator
    await login(page, 'admin@omnisync.com', 'admin123');
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(500);
  });

  test('Transfer Cycle: Create -> Claim -> Journalize -> Complete', async ({ page }) => {
    // 1. Navigate to Transfers page
    await page.click('a[href="/wms/transfers"]');
    await expect(page).toHaveURL(/.*\/wms\/transfers$/);

    // 2. Open Create Transfer Modal
    await page.click('button:has-text("Create Transfer Ticket")');
    await expect(page.locator('#modal-create-transfer')).toBeVisible();

    // 3. Fill the Form
    // Select Keychron Keyboard product
    await page.selectOption('#trf-product-select', 'prod-001');

    // Wait for HTMX dynamic locators to load
    await page.waitForTimeout(500);

    // Select the first eligible source locator
    await page.selectOption('#trf-source-select', { index: 1 });

    // Wait for HTMX dynamic destination locators to load
    await page.waitForTimeout(500);

    // Select the first eligible destination locator
    await page.selectOption('#trf-dest-select', { index: 1 });

    // Verify info box displays with stocks
    await expect(page.locator('#trf-info-box')).toBeVisible();

    // Input quantity, write remarks
    await page.fill('input[name="quantity"]', '5');
    await page.fill('textarea[name="remarks"]', 'E2E internal transfer operation');

    // 4. Submit
    await page.click('button:has-text("Submit Transfer")');

    // Verify success toast notification
    await expect(page.locator('.notyf__toast')).toBeVisible();
    await expect(page.locator('.notyf__message')).toContainText('created successfully');

    // 5. Verify the created transfer row in table is OPEN
    const trfRow = page.locator('tr:has-text("OPEN")').first();
    await expect(trfRow).toBeVisible();
    await expect(trfRow.locator('td:nth-child(2)')).toContainText('PROD-KYBD-01');
    await expect(trfRow.locator('td:nth-child(6)')).toContainText('5');

    // Extract unique Document Number
    const docNo = await trfRow.locator('td:first-child').innerText();
    const activeRow = page.locator(`tr:has-text("${docNo.trim()}")`);

    // 6. Claim the transfer task
    await activeRow.locator('button:has-text("Claim")').click();

    // Verify success toast
    await expect(page.locator('.notyf__toast')).toBeVisible();
    await expect(page.locator('.notyf__message')).toContainText('claimed successfully');

    // Verify status changes to IN_PROGRESS
    const inProgressRow = page.locator(`tr:has-text("${docNo.trim()}")`);
    await expect(inProgressRow.locator('td:nth-child(7)')).toContainText('IN_PROGRESS');

    // 7. Journalize the transfer task
    await inProgressRow.locator('button:has-text("Journalize")').click();

    // Verify success toast
    await expect(page.locator('.notyf__toast')).toBeVisible();
    await expect(page.locator('.notyf__message')).toContainText('successfully journaled');

    // Verify status changes to JOURNALED
    const journaledRow = page.locator(`tr:has-text("${docNo.trim()}")`);
    await expect(journaledRow.locator('td:nth-child(7)')).toContainText('JOURNALED');

    // 8. Complete the transfer task
    await journaledRow.locator('button:has-text("Complete")').click();

    // Verify success toast
    await expect(page.locator('.notyf__toast')).toBeVisible();
    await expect(page.locator('.notyf__message')).toContainText('completed successfully');

    // Verify status changes to COMPLETED
    const completedRow = page.locator(`tr:has-text("${docNo.trim()}")`);
    await expect(completedRow.locator('td:nth-child(7)')).toContainText('COMPLETED');
  });

});
