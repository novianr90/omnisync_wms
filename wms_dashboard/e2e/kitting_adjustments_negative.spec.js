const { test, expect } = require('./fixtures');
const { login } = require('./helpers/auth');

test.describe('Kitting & Adjustments Negative Flows', () => {

  test.beforeEach(async ({ page }) => {
    await login(page, 'admin@omnisync.com', 'admin123');
  });

  test('Kitting deficit: journalizing kitting order with insufficient component stock is rejected', async ({ page }) => {
    test.setTimeout(90000);
    await page.click('a[href="/wms/kitting"]');
    await expect(page).toHaveURL(/.*\/wms\/kitting$/);

    await page.click('button:has-text("New Kitting Order")');
    await expect(page.locator('#newKittingModal')).toBeVisible();

    // Use bundle product prod-005
    await page.selectOption('#newKittingModal select[name="finished_product_id"]', 'prod-005');
    await page.selectOption('#newKittingModal select[name="finished_locator_id"]', { index: 1 });
    // Large qty to guarantee component deficit
    await page.fill('#newKittingModal input[name="finished_qty"]', '9999');

    await page.click('button:has-text("Add Component")');
    const row = page.locator('#componentsContainer > div').first();
    await expect(row).toBeVisible();

    await row.locator('select[name="comp_product_id[]"]').selectOption('prod-004');

    const compOption = row.locator('select[name="comp_locator_id[]"] option[value="loc-001"]');
    await expect(compOption).toBeAttached({ timeout: 10000 });
    await page.waitForTimeout(250);
    await row.locator('select[name="comp_locator_id[]"]').selectOption('loc-001');
    await row.locator('input[name="comp_qty[]"]').fill('9999');

    await page.fill('#newKittingModal input[name="remarks"]', 'E2E kitting deficit negative test');
    await page.click('#newKittingModal button[type="submit"]');

    // Order may be created but journalize must fail — or creation itself fails
    const toast = page.locator('.notyf__toast');
    await expect(toast).toBeVisible();

    const toastMsg = page.locator('.notyf__message');
    const msgText = await toastMsg.textContent();
    const isCreated = /created/i.test(msgText);

    if (isCreated) {
      // Order created — attempt to journal it; expect stock error
      const kitRow = page.locator('tr:has-text("OPEN")').first();
      await expect(kitRow).toBeVisible();
      await kitRow.locator('button:has-text("Journal")').click();

      await expect(page.locator('.notyf__toast')).toBeVisible();
      // Actual server message: "cannot consume N items, only M available..."
      await expect(page.locator('.notyf__message')).toContainText(/cannot consume|insufficient/i);
    } else {
      await expect(toastMsg).toContainText(/cannot consume|insufficient/i);
    }
  });

  test('Invalid adjustment: submitting adjustment without reason code is blocked by HTML5 validation', async ({ page }) => {
    test.setTimeout(60000);
    await page.click('a[href="/wms/adjustments"]');
    await expect(page).toHaveURL(/.*\/wms\/adjustments$/);

    await page.click('button:has-text("New Adjustment")');
    await expect(page.locator('#newAdjustmentModal')).toBeVisible();

    // Fill product and locator but force reason_code to empty string to bypass default selection
    await page.selectOption('#newAdjustmentModal select[name="product_id"]', { index: 1 });
    await page.selectOption('#newAdjustmentModal select[name="locator_id"]', { index: 1 });
    await page.fill('#newAdjustmentModal input[name="qty_delta"]', '10');

    // Force reason_code select to empty to trigger required validation
    await page.evaluate(() => {
      const sel = document.querySelector('#newAdjustmentModal select[name="reason_code"]');
      if (sel) {
        const emptyOpt = document.createElement('option');
        emptyOpt.value = '';
        emptyOpt.text = '';
        sel.insertBefore(emptyOpt, sel.firstChild);
        sel.value = '';
      }
    });

    await page.fill('#newAdjustmentModal input[name="remarks"]', 'E2E missing reason code test');
    await page.click('#newAdjustmentModal button[type="submit"]');

    // HTML5 required validation keeps modal open OR server returns error
    const isModalOpen = await page.locator('#newAdjustmentModal').isVisible();
    if (isModalOpen) {
      // Modal still open — HTML5 validation blocked submission
      await expect(page.locator('#newAdjustmentModal select[name="reason_code"]')).toBeVisible();
    } else {
      await expect(page.locator('.notyf__toast')).toBeVisible();
      await expect(page.locator('.notyf__message')).toContainText(/reason|required|mandatory/i);
    }
  });

});
