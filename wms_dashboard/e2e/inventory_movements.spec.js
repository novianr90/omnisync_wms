const { test, expect } = require('@playwright/test');
const { login } = require('./helpers/auth');

test.describe('Inventory Movements & FIFO Outbound E2E Flows', () => {

  test.beforeEach(async ({ page }) => {
    // Standard login as administrator
    await login(page, 'admin@omnisync.com', 'admin123');
  });

  test('Inbound Receipt Cycle: Register -> Claim -> Journal -> Complete', async ({ page }) => {
    test.setTimeout(90000);
    // 1. Navigate to Movements Page
    await page.goto('/wms/movements');
    await expect(page.locator('h3:has-text("Inventory Movements")')).toBeVisible();

    // 2. Open Register Movement Form
    await page.click('a:has-text("Register Movement")');
    await expect(page.locator('h3:has-text("Create Movement Order")')).toBeVisible();

    // 3. Fill Form Lines
    // Selecting the product and filling quantity
    await page.selectOption('.product-select', { label: 'PROD-MON-03 - Dell UltraSharp 27" 4K Monitor' });
    await page.fill('.quantity-input', '25');
    await page.selectOption('.locator-select', { label: 'WH-MAIN-Zone-A-Aisle-1-Shelf-2-Level-1' });
    await page.fill('#movement-remarks', 'E2E multi-item inbound delivery test');

    // 4. Submit Form
    await page.click('button[type="submit"]');

    // Verify redirect back to list page and success toast
    const toast = page.locator('.notyf__toast').last();
    await expect(toast).toBeVisible();
    await expect(page.locator('.notyf__message').last()).toContainText('registered successfully');

    await expect(page.locator('h3:has-text("Inventory Movements")')).toBeVisible();

    // 5. Navigate to Detail View of the new movement
    const firstRow = page.locator('#movements-tbody tr').first();
    await expect(firstRow).toBeVisible();
    const docLink = firstRow.locator('td:first-child a');
    const docNo = await docLink.innerText();

    await docLink.click();
    await expect(page.locator('h3.font-bold')).toContainText(docNo);

    // 6. Advance through lifecycle: Claim Task
    await page.click('button:has-text("Claim Task")');
    await expect(page.locator('.notyf__toast').last()).toBeVisible();
    await expect(page.locator('.notyf__message').last()).toContainText('claimed successfully');
    await expect(page.locator('span:has-text("Claimed & In Progress")')).toBeVisible();

    // 7. Advance through lifecycle: Journal Receipt
    await page.click('button:has-text("Journalize Stock")');
    await expect(page.locator('.notyf__toast').last()).toBeVisible();
    await expect(page.locator('.notyf__message').last()).toContainText('successfully journaled');
    await expect(page.locator('span:has-text("Stock Journaled")')).toBeVisible();

    // 8. Advance through lifecycle: Complete Task
    await page.click('button:has-text("Complete Document")');
    await expect(page.locator('.notyf__toast').last()).toBeVisible();
    await expect(page.locator('.notyf__message').last()).toContainText('completed successfully');

    // After completion, verify status badge says Completed
    await expect(page.locator('.w-full:has-text("Document Locked & Closed")')).toBeVisible();

    // 9. Verify catalog stock quantities updated dynamically on Dashboard
    await page.goto('/');
    const monitorRow = page.locator('tr:has-text("Dell UltraSharp 27\\" 4K Monitor")');
    await expect(monitorRow.locator('td:nth-child(3)')).toContainText('25'); // On hand
    await expect(monitorRow.locator('td:nth-child(5)')).toContainText('25'); // Available
  });

  test('Outbound FIFO Validation: Fail over-limit -> Succeed within limit -> Deplete stock', async ({ page }) => {
    test.setTimeout(90000);
    // 1. Navigate to register movements form
    await page.goto('/wms/movements/new');
    await expect(page.locator('h3:has-text("Create Movement Order")')).toBeVisible();

    // Change type to OUTBOUND
    await page.selectOption('#movement-type-select', 'OUTBOUND');

    // Try to dispatch more than available (Logitech mouse has 80 on hand)
    await page.selectOption('.product-select', { label: 'PROD-MOUS-02 - Logitech MX Master 3S Mouse' });
    await page.fill('.quantity-input', '95'); // Exceeds available 80
    await page.click('button[type="submit"]');

    // Verify fail toast
    await expect(page.locator('.notyf__toast').last()).toBeVisible();
    await expect(page.locator('.notyf__message').last()).toContainText('insufficient stock');

    // 2. Fill a valid amount (30 mice)
    await page.fill('.quantity-input', '30'); // Valid amount
    await page.click('button[type="submit"]');

    // Verify success toast
    await expect(page.locator('.notyf__toast').last()).toBeVisible();
    await expect(page.locator('.notyf__message').last()).toContainText('registered successfully');

    // 3. Navigate to detail view
    await expect(page.locator('h3:has-text("Inventory Movements")')).toBeVisible();
    const firstRow = page.locator('#movements-tbody tr').first();
    const docLink = firstRow.locator('td:first-child a');
    const docNo = await docLink.innerText();

    // Verify that the catalog shows 30 reserved (allocated) immediately on Dashboard
    await page.goto('/');
    const mouseRow = page.locator('tr:has-text("Logitech MX Master 3S Mouse")');
    await expect(mouseRow.locator('td:nth-child(4)')).toContainText('30'); // Reserved
    await expect(mouseRow.locator('td:nth-child(5)')).toContainText('50'); // Available = 80 - 30

    // Go back to the movement page detail to process
    await page.goto('/wms/movements');
    await page.locator(`#movements-tbody tr:has-text("${docNo}") td:first-child a`).click();

    // Claim, Journal and Complete the outbound task
    await page.click('button:has-text("Claim Task")');
    await expect(page.locator('.notyf__toast').last()).toBeVisible();
    await expect(page.locator('.notyf__message').last()).toContainText('claimed successfully');

    await page.click('button:has-text("Journalize Stock")');
    await expect(page.locator('.notyf__toast').last()).toBeVisible();
    await expect(page.locator('.notyf__message').last()).toContainText('successfully journaled');

    await page.click('button:has-text("Complete Document")');
    await expect(page.locator('.notyf__toast').last()).toBeVisible();
    await expect(page.locator('.notyf__message').last()).toContainText('completed successfully');

    // Verify final stock levels in catalog decreased appropriately on Dashboard
    await page.goto('/');
    const updatedMouseRow = page.locator('tr:has-text("Logitech MX Master 3S Mouse")');
    await expect(updatedMouseRow.locator('td:nth-child(3)')).toContainText('50'); // On hand
    await expect(updatedMouseRow.locator('td:nth-child(4)')).toContainText('0');  // Reserved
    await expect(updatedMouseRow.locator('td:nth-child(5)')).toContainText('50'); // Available
  });

});
