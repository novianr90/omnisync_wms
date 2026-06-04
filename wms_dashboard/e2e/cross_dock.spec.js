const { test, expect } = require('@playwright/test');
const { login } = require('./helpers/auth');

test.describe('Cross-Docking Operations E2E Flows', () => {

  test.beforeEach(async ({ page }) => {
    // Standard login as administrator
    await login(page, 'admin@omnisync.com', 'admin123');
  });

  test('Cross-Dock Cycle: Register -> Claim -> Confirm Inbound -> Initiate Loading -> Confirm Dispatch -> Complete', async ({ page }) => {
    test.setTimeout(90000);
    // 1. Navigate to Register Movement Form
    await page.goto('/wms/movements/new');
    await expect(page.locator('h3:has-text("Create Movement Order")')).toBeVisible();

    // 2. Select product and check Cross-Docking
    await page.selectOption('.product-select', { label: 'PROD-MON-03 - Dell UltraSharp 27" 4K Monitor' });
    
    // Toggle the Cross-Docking checkbox
    await page.check('#is-cross-dock-checkbox');

    // Verify warning banner is visible
    await expect(page.locator('#crossdock-warning-banner')).toBeVisible();

    // Verify target shelf locator select cell is disabled/faded
    const cell = page.locator('.locator-select-cell');
    await expect(cell).toHaveClass(/pointer-events-none/);

    // 3. Fill quantity and remarks
    await page.fill('.quantity-input', '15');
    await page.fill('#movement-remarks', 'E2E Cross-Docking lifecycle validation');

    // Submit Form
    await page.click('button[type="submit"]');

    // Verify success toast
    await expect(page.locator('.notyf__toast')).toBeVisible();
    await expect(page.locator('.notyf__message')).toContainText('registered successfully');

    // Verify redirect to movements list page
    await expect(page.locator('h3:has-text("Inventory Movements")')).toBeVisible();

    // 4. Navigate to Detail View of the new crossdock movement
    const firstRow = page.locator('#movements-tbody tr').first();
    await expect(firstRow).toBeVisible();
    const docLink = firstRow.locator('td:first-child a');
    const docNo = await docLink.innerText();

    await docLink.click();
    await expect(page.locator('h3.font-bold')).toContainText(docNo);

    // 5. Claim Task -> IN PROGRESS
    await page.click('button:has-text("Claim Task")');
    await expect(page.locator('.notyf__toast')).toBeVisible();
    await expect(page.locator('.notyf__message')).toContainText('claimed successfully');
    await expect(page.locator('span:has-text("Claimed & In Progress")')).toBeVisible();

    // 6. Confirm Inbound -> INBOUND
    await page.click('button:has-text("Confirm Inbound Receipt")');
    await expect(page.locator('.notyf__toast')).toBeVisible();
    await expect(page.locator('.notyf__message')).toContainText('confirmed');

    // 7. Initiate Loading -> SHIPPING
    await page.click('button:has-text("Initiate Loading")');
    await expect(page.locator('.notyf__toast')).toBeVisible();
    await expect(page.locator('.notyf__message')).toContainText('initiated');

    // 8. Confirm Dispatch -> OUTBOUND
    await page.click('button:has-text("Confirm Dispatch")');
    await expect(page.locator('.notyf__toast')).toBeVisible();
    await expect(page.locator('.notyf__message')).toContainText('confirmed');

    // 9. Complete Ticket -> COMPLETED
    await page.click('button:has-text("Complete Document")');
    await expect(page.locator('.notyf__toast')).toBeVisible();
    await expect(page.locator('.notyf__message')).toContainText('completed successfully');
    await expect(page.locator('.w-full:has-text("Document Locked & Closed")')).toBeVisible();

    // 10. Check Cross Docking dashboard visibility
    await page.click('a[href="/wms/crossdock"]');
    await expect(page).toHaveURL(/\/wms\/crossdock/);
    
    // Cross Docking page should render the page contents with p-6 container
    const crossdockTitle = page.locator('h1:has-text("Cross Docking Dashboard")');
    await expect(crossdockTitle).toBeVisible();
  });
});
