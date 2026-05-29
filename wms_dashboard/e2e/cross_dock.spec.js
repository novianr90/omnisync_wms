const { test, expect } = require('@playwright/test');
const { login } = require('./helpers/auth');

test.describe('Cross-Docking Operations E2E Flows', () => {

  test.beforeEach(async ({ page }) => {
    // Standard login as administrator
    await login(page, 'admin@omnisync.com', 'admin123');
  });

  test('Cross-Dock Cycle: Register -> Claim -> Confirm Inbound -> Initiate Loading -> Confirm Dispatch -> Complete', async ({ page }) => {
    await page.goto('/');

    // 1. Open the Inbound Modal
    await page.click('button:has-text("Register Inbound")');
    await expect(page.locator('#modal-inbound')).toBeVisible();

    // 2. Select product and check Cross-Docking
    await page.selectOption('#inbound-product', { label: 'PROD-MON-03 - Dell UltraSharp 27" 4K Monitor' });
    
    // Toggle the Cross-Docking checkbox
    await page.check('#inbound-crossdock');

    // Verify target shelf locator container is disabled/faded
    const locatorContainer = page.locator('#locator-select-container');
    await expect(locatorContainer).toHaveClass(/pointer-events-none/);

    // 3. Fill quantity and remarks
    await page.fill('#inbound-qty', '15');
    await page.fill('#inbound-remarks', 'E2E Cross-Docking lifecycle validation');

    // Submit Form
    await page.click('#modal-inbound button[type="submit"]');

    // Verify success toast
    await expect(page.locator('.notyf__toast')).toBeVisible();
    await expect(page.locator('.notyf__message')).toContainText('registered successfully');

    // 4. Verify CROSS-DOCK Card in Right Panel
    const crossdockCard = page.locator('.glass-panel .glass-panel:has-text("CROSS-DOCK"):has-text("OPEN")').first();
    await expect(crossdockCard).toBeVisible();
    await expect(crossdockCard).toContainText('Dell UltraSharp 27" 4K Monitor');

    // Extract unique Document Number
    const cardText = await crossdockCard.innerText();
    const docNoMatch = cardText.match(/MOV-\d+-\d+|MOV-\d+/);
    const docNo = docNoMatch ? docNoMatch[0] : "";
    const card = page.locator(`.glass-panel .glass-panel:has-text("${docNo}")`).first();

    // 5. Claim Task -> IN PROGRESS
    await card.locator('button:has-text("Claim Task")').click();
    await expect(card).toContainText('IN PROGRESS');

    // 6. Confirm Inbound -> INBOUND
    await card.locator('button:has-text("Confirm Inbound")').click();
    await expect(card).toContainText('INBOUND');

    // 7. Initiate Loading -> SHIPPING
    await card.locator('button:has-text("Initiate Loading")').click();
    await expect(card).toContainText('SHIPPING');

    // 8. Confirm Dispatch -> OUTBOUND
    await card.locator('button:has-text("Confirm Dispatch")').click();
    await expect(card).toContainText('OUTBOUND');

    // 9. Complete Ticket -> COMPLETED
    await card.locator('button:has-text("Complete Ticket")').click();
    await expect(card).toContainText('COMPLETED');

    // 10. Check Cross Docking dashboard visibility
    await page.click('a[href="/wms/crossdock"]');
    await expect(page).toHaveURL(/\/wms\/crossdock/);
    
    // Cross Docking page should render the page contents with p-6 container
    const crossdockTitle = page.locator('h1:has-text("Cross Docking Dashboard")');
    await expect(crossdockTitle).toBeVisible();
  });
});
