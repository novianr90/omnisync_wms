const { test, expect } = require('./fixtures');
const { login } = require('./helpers/auth');

test.describe('QC Holds & Stock Adjustments E2E Flows', () => {

  test.beforeEach(async ({ page }) => {
    // Standard login as administrator
    await login(page, 'admin@omnisync.com', 'admin123');
  });

  test('QC Hold Cycle: Freeze Stock -> Verify reduction -> Release Hold -> Verify restore', async ({ page }) => {
    // 1. Go to QC Holds page
    await page.click('a[href="/wms/qc-holds"]');
    await expect(page).toHaveURL(/.*\/wms\/qc-holds$/);

    // 2. Open Freeze Stock Modal
    await page.click('button:has-text("Freeze Stock")');
    await expect(page.locator('#modal-create-hold')).toBeVisible();

    // 3. Fill the Form
    // Select an eligible storage lot with plenty of stock
    await page.selectOption('#modal-create-hold select[name="storage_id"]', 'stor-004');
    await page.fill('#modal-create-hold input[name="qty"]', '5');
    await page.selectOption('#modal-create-hold select[name="reason"]', 'DAMAGED');
    await page.fill('#modal-create-hold textarea[name="notes"]', 'E2E stock quarantine hold test');

    // 4. Submit
    await page.click('#modal-create-hold button[type="submit"]');

    // Verify Notyf notification toast showing success
    await expect(page.locator('.notyf__toast')).toBeVisible();
    await expect(page.locator('.notyf__message')).toContainText('Hold executed successfully');

    // 5. Verify the Hold row appears in table with status LOCKED
    const holdRow = page.locator('tr:has-text("LOCKED")').first();
    await expect(holdRow).toBeVisible();
    await expect(holdRow.locator('td:nth-child(5)')).toContainText('5'); // Qty held
    await expect(holdRow.locator('td:nth-child(6)')).toContainText('DAMAGED'); // Reason

    // Extract unique Hold Number to uniquely track the row after reload/refresh
    const holdNo = await holdRow.locator('td:first-child').innerText();
    const activeRow = page.locator(`tr:has-text("${holdNo.trim()}")`);

    // 6. Release the Hold
    await activeRow.locator('button:has-text("Release")').click();
    
    // Sleek inline confirmation modal opens
    const confirmModal = page.locator('div[id^="confirm-modal-"]').filter({ visible: true });
    await expect(confirmModal).toBeVisible();
    await confirmModal.locator('button:has-text("Confirm")').click();

    // Verify success toast
    await expect(page.locator('.notyf__toast')).toBeVisible();
    await expect(page.locator('.notyf__message')).toContainText('released successfully');

    // 7. Verify status changed to RELEASED
    const releasedRow = page.locator(`tr:has-text("${holdNo.trim()}")`);
    await expect(releasedRow.locator('td:nth-child(7)')).toContainText('RELEASED');
  });

  test('Stock Adjustments Cycle: Create Positive Adjustment -> Journalize -> Verify physical update', async ({ page }) => {
    // 1. Go to Adjustments page
    await page.click('a[href="/wms/adjustments"]');
    await expect(page).toHaveURL(/.*\/wms\/adjustments$/);

    // 2. Open New Adjustment Modal
    await page.click('button:has-text("New Adjustment")');
    await expect(page.locator('#newAdjustmentModal')).toBeVisible();

    // 3. Fill the Form
    await page.selectOption('#newAdjustmentModal select[name="product_id"]', { index: 1 });
    await page.selectOption('#newAdjustmentModal select[name="locator_id"]', { index: 1 });
    await page.fill('#newAdjustmentModal input[name="qty_delta"]', '15'); // Positive adjustment
    await page.selectOption('#newAdjustmentModal select[name="reason_code"]', 'FOUND');
    await page.fill('#newAdjustmentModal input[name="remarks"]', 'E2E positive stock adjustments test');

    // 4. Submit
    await page.click('#newAdjustmentModal button[type="submit"]');

    // Verify Notyf notification toast showing success
    await expect(page.locator('.notyf__toast')).toBeVisible();
    await expect(page.locator('.notyf__message')).toContainText('Adjustment ticket registered');

    // 5. Verify the Adjustment row appears in table with status OPEN
    const adjRow = page.locator('tr:has-text("OPEN")').first();
    await expect(adjRow).toBeVisible();
    await expect(adjRow.locator('td:nth-child(3)')).toContainText('FOUND'); // Reason
    await expect(adjRow.locator('td:nth-child(6)')).toContainText('+15'); // Qty delta

    // Extract unique Document Number to uniquely track the row after reload/refresh
    const docNo = await adjRow.locator('td:first-child').innerText();
    const activeAdjRow = page.locator(`tr:has-text("${docNo.trim()}")`);

    // 6. Journalize the Adjustment
    await activeAdjRow.locator('button:has-text("Journal")').click();

    // Verify success toast
    await expect(page.locator('.notyf__toast')).toBeVisible();
    await expect(page.locator('.notyf__message')).toContainText('successfully journaled');

    // 7. Verify status changed to JOURNALED
    const journaledRow = page.locator(`tr:has-text("${docNo.trim()}")`);
    await expect(journaledRow.locator('td:nth-child(2)')).toContainText('JOURNALED');
  });

});
