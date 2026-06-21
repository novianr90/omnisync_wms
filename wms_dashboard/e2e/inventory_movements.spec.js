const { test, expect } = require('./fixtures');
const { login } = require('./helpers/auth');

test.describe('Inventory Movements & FIFO Outbound E2E Flows', () => {

  test.beforeEach(async ({ page }) => {
    // Standard login as administrator
    await login(page, 'admin@omnisync.com', 'admin123');
  });

  test('Inbound Receipt Cycle: Register -> Claim -> Journal -> Complete', async ({ page }) => {
    test.setTimeout(90000);

    // 0. Navigate to Dashboard to read initial stock
    await page.goto('/');
    const monitorRow = page.locator('tr:has-text("Dell UltraSharp 27\\" 4K Monitor")');
    const onHandText = await monitorRow.locator('td:nth-child(3)').innerText().catch(() => "0");
    const availableText = await monitorRow.locator('td:nth-child(5)').innerText().catch(() => "0");
    const initialOnHand = parseInt(onHandText.replace(/[^0-9-]/g, '') || "0", 10);
    const initialAvailable = parseInt(availableText.replace(/[^0-9-]/g, '') || "0", 10);

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

    // 4. Navigate to Detail View of the new crossdock movement
    await page.goto('http://localhost:9901/wms/movements');
    const firstRow = page.locator('#movements-tbody tr').first();
    await expect(firstRow).toBeVisible();
    const docNo = await firstRow.locator('td:first-child').innerText();

    await firstRow.locator('a:has-text("View Details")').click();
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
    // Add 25 to initial values
    await expect(monitorRow.locator('td:nth-child(3)')).toContainText(String(initialOnHand + 25)); // On hand
    await expect(monitorRow.locator('td:nth-child(5)')).toContainText(String(initialAvailable + 25)); // Available
  });

  test('Outbound FIFO Validation: Fail over-limit -> Succeed within limit -> Deplete stock', async ({ page }) => {
    test.setTimeout(90000);

    // 0. Navigate to Dashboard to read initial stock
    await page.goto('/');
    const targetProductRow = page.locator('tr:has-text("Mechanical Keychron K2 Keyboard")');
    const initialOnHandText = await targetProductRow.locator('td:nth-child(3)').innerText();
    const initialReservedText = await targetProductRow.locator('td:nth-child(4)').innerText();
    const initialAvailableText = await targetProductRow.locator('td:nth-child(5)').innerText();
    const initialOnHand = parseInt(initialOnHandText.replace(/[^0-9-]/g, '') || "0", 10);
    const initialReserved = parseInt(initialReservedText.replace(/[^0-9-]/g, '') || "0", 10);
    const initialAvailable = parseInt(initialAvailableText.replace(/[^0-9-]/g, '') || "0", 10);

    // 1. Navigate to register movements form
    await page.goto('/wms/movements/new');
    await expect(page.locator('h3:has-text("Create Movement Order")')).toBeVisible();

    // Change type to OUTBOUND
    await page.selectOption('#movement-type-select', 'OUTBOUND');

    // 1. Select product and check type
    await page.selectOption('.product-select', { label: 'PROD-KYBD-01 - Mechanical Keychron K2 Keyboard' });
    await page.fill('.quantity-input', String(initialAvailable + 15)); // Exceeds available
    await page.click('button[type="submit"]');

    // Verify fail toast
    await expect(page.locator('.notyf__toast').last()).toBeVisible();
    await expect(page.locator('.notyf__message').last()).toContainText('insufficient stock');

    // 2. Fill a valid amount (10 mice)
    await page.fill('.quantity-input', '10'); // Valid amount
    await page.click('button[type="submit"]');

    // Verify success toast
    await expect(page.locator('.notyf__toast').last()).toBeVisible();
    await expect(page.locator('.notyf__message').last()).toContainText('registered successfully');

    // 3. Navigate to detail view
    await expect(page.locator('h3:has-text("Inventory Movements")')).toBeVisible();
    const firstRow = page.locator('#movements-tbody tr').first();
    const docLink = firstRow.locator('td:first-child a');
    const docNo = await docLink.innerText();

    // Verify that the catalog shows 10 reserved (allocated) immediately on Dashboard
    await page.goto('/');
    await expect(targetProductRow.locator('td:nth-child(4)')).toContainText(String(initialReserved + 10)); // Reserved
    
    const expectedAvail1 = initialAvailable - 10;
    const availText1 = expectedAvail1 <= 0 ? "0" : String(expectedAvail1);
    await expect(targetProductRow.locator('td:nth-child(5)')).toContainText(availText1); // Available

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
    await expect(targetProductRow.locator('td:nth-child(3)')).toContainText(String(initialOnHand - 10)); // On hand
    await expect(targetProductRow.locator('td:nth-child(4)')).toContainText(String(initialReserved));  // Reserved
    
    const expectedAvail2 = initialAvailable - 10;
    const availText2 = expectedAvail2 <= 0 ? "OUT OF STOCK" : String(expectedAvail2);
    await expect(targetProductRow.locator('td:nth-child(5)')).toContainText(availText2); // Available
  });

});
