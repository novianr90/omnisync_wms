const { test, expect } = require('@playwright/test');
const { login } = require('./helpers/auth');

test.describe('Inventory Movements Negative Flows', () => {

  test.beforeEach(async ({ page }) => {
    await login(page, 'admin@omnisync.com', 'admin123');
  });

  test('Out-of-stock dispatch: outbound qty exceeding available stock is rejected', async ({ page }) => {
    test.setTimeout(90000);
    await page.goto('/wms/movements/new');
    await expect(page.locator('h3:has-text("Create Movement Order")')).toBeVisible();

    await page.selectOption('#movement-type-select', 'OUTBOUND');

    // Logitech MX Master 3S Mouse (PROD-MOUS-02) has 80 on hand — request 200
    await page.selectOption('.product-select', { label: 'PROD-MOUS-02 - Logitech MX Master 3S Mouse' });
    await page.fill('.quantity-input', '200');
    await page.click('button[type="submit"]');

    await expect(page.locator('.notyf__toast')).toBeVisible();
    await expect(page.locator('.notyf__message')).toContainText('insufficient stock');

    // URL should remain on /new (not redirected to list)
    await expect(page).toHaveURL(/.*\/movements\/new$/);
  });

  test('QC quarantine exclusion: outbound on QC-held product reduces available qty and blocks over-pick', async ({ page }) => {
    test.setTimeout(90000);
    // 1. Create a QC Hold on stor-004 (freeze 5 units)
    await page.goto('http://localhost:9901/wms/qc-holds');
    await page.click('button:has-text("Freeze Stock")');
    await expect(page.locator('#modal-create-hold')).toBeVisible();

    await page.selectOption('#modal-create-hold select[name="storage_id"]', 'stor-004');
    await page.fill('#modal-create-hold input[name="qty"]', '5');
    await page.selectOption('#modal-create-hold select[name="reason"]', 'DAMAGED');
    await page.fill('#modal-create-hold textarea[name="notes"]', 'E2E QC quarantine negative test');
    await page.click('#modal-create-hold button[type="submit"]');

    await expect(page.locator('.notyf__message')).toContainText('Hold executed successfully');

    // 2. Attempt outbound for PROD-MOUS-02: 80 on-hand, 5 now on-hold → 75 available max
    //    Request 80 (exceeds available 75) — system must reject
    await page.goto('/wms/movements/new');
    await expect(page.locator('h3:has-text("Create Movement Order")')).toBeVisible();

    await page.selectOption('#movement-type-select', 'OUTBOUND');
    await page.selectOption('.product-select', { label: 'PROD-MOUS-02 - Logitech MX Master 3S Mouse' });
    await page.fill('.quantity-input', '80');
    await page.click('button[type="submit"]');

    await expect(page.locator('.notyf__toast')).toBeVisible();
    await expect(page.locator('.notyf__message')).toContainText('insufficient stock');
  });

  test('Cross-claiming: operator cannot claim a movement already IN_PROGRESS by another operator', async ({ page }) => {
    test.setTimeout(120000);
    // 1. Admin creates a new inbound movement
    await page.goto('/wms/movements/new');
    await expect(page.locator('h3:has-text("Create Movement Order")')).toBeVisible();

    await page.selectOption('.product-select', { label: 'PROD-MON-03 - Dell UltraSharp 27" 4K Monitor' });
    await page.fill('.quantity-input', '10');
    await page.selectOption('.locator-select', { index: 1 });
    await page.fill('#movement-remarks', 'E2E cross-claim negative test');
    await page.click('button[type="submit"]');

    await expect(page.locator('.notyf__message')).toContainText('registered successfully');

    // 2. Admin claims the movement
    const firstRow = page.locator('#movements-tbody tr').first();
    const docLink = firstRow.locator('td:first-child a');
    const docNo = await docLink.innerText();
    await docLink.click();

    await page.click('button:has-text("Claim Task")');
    await expect(page.locator('.notyf__message')).toContainText('claimed successfully');
    await expect(page.locator('span:has-text("Claimed & In Progress")')).toBeVisible();

    // 3. Log out admin
    await page.goto('http://localhost:9901/logout');
    await page.waitForURL('**/login');

    // 4. Log in as operator (seeded POS role user)
    await login(page, 'operator@omnisync.com', 'operator123');

    // 5. Navigate to same movement detail
    await page.goto('/wms/movements');
    const targetRow = page.locator(`#movements-tbody tr:has-text("${docNo.trim()}")`);
    await targetRow.locator('td:first-child a').click();

    // 6. Claim Task button absent or clicking it is rejected
    const claimBtn = page.locator('button:has-text("Claim Task")');
    const claimVisible = await claimBtn.isVisible().catch(() => false);
    if (claimVisible) {
      await claimBtn.click();
      await expect(page.locator('.notyf__toast')).toBeVisible();
      await expect(page.locator('.notyf__message')).toContainText(/already claimed|in progress|not available/i);
    } else {
      // Cross-claim blocked at UI level — status badge confirms IN_PROGRESS
      await expect(page.locator('span:has-text("Claimed & In Progress")')).toBeVisible();
    }
  });

});
