const { test, expect } = require('./fixtures');
const { login } = require('./helpers/auth');

test.describe('Kitting / Light Assembly E2E Flows', () => {

  test.beforeEach(async ({ page }) => {
    // Standard login as administrator
    await login(page, 'admin@omnisync.com', 'admin123');
  });

  test('Kitting Order Cycle: Create Assembly Order -> Verify dropdowns -> Journalize -> Verify output', async ({ page }) => {
    // 1. Go to Kitting page
    await page.click('a[href="/wms/kitting"]');
    await expect(page).toHaveURL(/.*\/wms\/kitting$/);

    // 2. Open New Kitting Order Modal
    await page.click('button:has-text("New Kitting Order")');
    await expect(page.locator('#newKittingModal')).toBeVisible();

    // Select the seeded bundle product to build (prod-005)
    await page.selectOption('#newKittingModal select[name="finished_product_id"]', 'prod-005');
    await page.selectOption('#newKittingModal select[name="finished_locator_id"]', { index: 1 });
    await page.fill('#newKittingModal input[name="finished_qty"]', '2');

    // 4. Add Component Row & Fill details
    await page.click('button:has-text("Add Component")');
    
    // Fill the dynamically generated row
    const row = page.locator('#componentsContainer > div').first();
    await expect(row).toBeVisible();

    await row.locator('select[name="comp_product_id[]"]').selectOption('prod-004');
    
    // Wait for locator dropdown to populate via async fetch
    const compOption = row.locator('select[name="comp_locator_id[]"] option[value="loc-001"]');
    await expect(compOption).toBeAttached({ timeout: 10000 });
    await page.waitForTimeout(250); // Give HTMX/JS a brief moment to settle the DOM before selecting
    await row.locator('select[name="comp_locator_id[]"]').selectOption('loc-001');
    await row.locator('input[name="comp_qty[]"]').fill('4');

    await page.fill('#newKittingModal input[name="remarks"]', 'E2E light assembly test');

    // 5. Submit Order
    await page.click('#newKittingModal button[type="submit"]');

    // Verify Notyf success toast
    await expect(page.locator('.notyf__toast')).toBeVisible();
    await expect(page.locator('.notyf__message')).toContainText('Kitting order created');

    // 6. Verify row in kitting list table with status OPEN
    const kitRow = page.locator('tr:has-text("OPEN")').first();
    await expect(kitRow).toBeVisible();
    await expect(kitRow.locator('td:nth-child(5)')).toContainText('+2'); // Finished Qty

    // Extract unique Document Number to uniquely track the row after reload/refresh
    const docNo = await kitRow.locator('td:first-child').innerText();
    const activeKitRow = page.locator(`tr:has-text("${docNo.trim()}")`);

    // 7. Advance through lifecycle: Journal
    await activeKitRow.locator('button:has-text("Journal")').click();

    // Verify success toast
    await expect(page.locator('.notyf__toast')).toBeVisible();
    await expect(page.locator('.notyf__message')).toContainText('successfully journaled');

    // 8. Verify status changed to JOURNALED
    const journaledRow = page.locator(`tr:has-text("${docNo.trim()}")`);
    await expect(journaledRow.locator('td:nth-child(2)')).toContainText('JOURNALED');
  });

});
