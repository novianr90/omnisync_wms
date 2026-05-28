const { test, expect } = require('@playwright/test');
const { login } = require('./helpers/auth');

test.describe('Inventory Movements & FIFO Outbound E2E Flows', () => {

  test.beforeEach(async ({ page }) => {
    // Standard login as administrator
    await login(page, 'admin@omnisync.com', 'admin123');
  });

  test('Inbound Receipt Cycle: Register -> Claim -> Journal -> Complete', async ({ page }) => {
    await page.goto('/');

    // 1. Open the Inbound Modal
    await page.click('button:has-text("Register Inbound")');
    await expect(page.locator('#modal-inbound')).toBeVisible();

    // 2. Fill the Form
    await page.selectOption('#inbound-product', { label: 'PROD-MON-03 - Dell UltraSharp 27" 4K Monitor' });
    await page.fill('#inbound-qty', '25');
    await page.selectOption('#inbound-locator', { label: 'WH-MAIN-Zone-A-Aisle-1-Shelf-2-Level-1' });
    await page.fill('#inbound-remarks', 'E2E inbound delivery test');
    
    // 3. Submit Form
    await page.click('#modal-inbound button[type="submit"]');

    // Verify Notyf notification toast showing success
    const toast = page.locator('.notyf__toast');
    await expect(toast).toBeVisible();
    await expect(page.locator('.notyf__message')).toContainText('registered successfully');

    // 4. Verify Movement Card in Right Panel
    // Search for the newly created INBOUND card with status OPEN
    const inboundCard = page.locator('.glass-panel .glass-panel:has-text("INBOUND"):has-text("OPEN")').first();
    await expect(inboundCard).toBeVisible();
    await expect(inboundCard).toContainText('Dell UltraSharp 27" 4K Monitor');

    // Extract the unique Document Number to avoid stale selector during status changes
    const cardText = await inboundCard.innerText();
    const docNoMatch = cardText.match(/MOV-\d+-\d+|MOV-\d+/);
    const docNo = docNoMatch ? docNoMatch[0] : "";
    const card = page.locator(`.glass-panel .glass-panel:has-text("${docNo}")`).first();

    // 5. Advance through lifecycle: Claim Task
    await card.locator('button:has-text("Claim Task")').click();
    await expect(card).toContainText('IN PROGRESS');

    // 6. Advance through lifecycle: Journal Receipt
    await card.locator('button:has-text("Journalize Stock")').click();
    await expect(card).toContainText('JOURNALED');

    // 7. Advance through lifecycle: Complete Task
    await card.locator('button:has-text("Complete Ticket")').click();

    // After completion, the card should either have completed status or disappear from the active board
    await expect(card).toContainText('COMPLETED');

    // 8. Verify catalog stock quantities updated dynamically
    // Dell monitor should now show 25 on hand and available in the table row
    const monitorRow = page.locator('tr:has-text("Dell UltraSharp 27\\" 4K Monitor")');
    await expect(monitorRow.locator('td:nth-child(3)')).toContainText('25'); // On hand
    await expect(monitorRow.locator('td:nth-child(5)')).toContainText('25'); // Available
  });

  test('Outbound FIFO Validation: Fail over-limit -> Succeed within limit -> Deplete stock', async ({ page }) => {
    await page.goto('/');

    // 1. Try to dispatch more than available (Logitech mouse has 80 on hand)
    await page.click('button:has-text("Dispatch Outbound")');
    await expect(page.locator('#modal-outbound')).toBeVisible();

    await page.selectOption('#outbound-product', { label: 'PROD-MOUS-02 - Logitech MX Master 3S Mouse' });
    await page.fill('#outbound-qty', '95'); // Exceeds available 80
    await page.click('#modal-outbound button[type="submit"]');

    // Verify Notyf notification toast showing failure
    await expect(page.locator('.notyf__toast')).toBeVisible();
    await expect(page.locator('.notyf__message')).toContainText('insufficient stock');

    // Close modal if still open
    if (await page.locator('#modal-outbound').isVisible()) {
      await page.click('#modal-outbound button[onclick*="modal-outbound"]');
    }

    // 2. Dispatch a valid amount (30 mice)
    await page.click('button:has-text("Dispatch Outbound")');
    await page.selectOption('#outbound-product', { label: 'PROD-MOUS-02 - Logitech MX Master 3S Mouse' });
    await page.fill('#outbound-qty', '30'); // Valid amount
    await page.click('#modal-outbound button[type="submit"]');

    // Verify success toast
    await expect(page.locator('.notyf__toast')).toBeVisible();
    await expect(page.locator('.notyf__message')).toContainText('registered successfully');

    // 3. Process the Outbound card
    const outboundCard = page.locator('.glass-panel .glass-panel:has-text("OUTBOUND"):has-text("OPEN")').first();
    await expect(outboundCard).toBeVisible();
    await expect(outboundCard).toContainText('Logitech MX Master 3S Mouse');

    // Extract the unique Document Number to avoid stale selector during status changes
    const cardText = await outboundCard.innerText();
    const docNoMatch = cardText.match(/MOV-\d+-\d+|MOV-\d+/);
    const docNo = docNoMatch ? docNoMatch[0] : "";
    const card = page.locator(`.glass-panel .glass-panel:has-text("${docNo}")`).first();

    // Verify that the catalog shows 30 reserved (allocated) immediately
    const mouseRow = page.locator('tr:has-text("Logitech MX Master 3S Mouse")');
    await expect(mouseRow.locator('td:nth-child(4)')).toContainText('30'); // Reserved
    await expect(mouseRow.locator('td:nth-child(5)')).toContainText('50'); // Available = 80 - 30

    // Claim, Journal and Complete the outbound task
    await card.locator('button:has-text("Claim Task")').click();
    await expect(card).toContainText('IN PROGRESS');

    await card.locator('button:has-text("Journalize Stock")').click();
    await expect(card).toContainText('JOURNALED');

    await card.locator('button:has-text("Complete Ticket")').click();
    await expect(card).toContainText('COMPLETED');

    // Verify final stock levels in catalog decreased appropriately
    // On hand: 80 - 30 = 50. Reserved: 0. Available: 50.
    await expect(mouseRow.locator('td:nth-child(3)')).toContainText('50'); // On hand
    await expect(mouseRow.locator('td:nth-child(4)')).toContainText('0');  // Reserved
    await expect(mouseRow.locator('td:nth-child(5)')).toContainText('50'); // Available
  });

});
