const { test, expect } = require('@playwright/test');
const { login } = require('./helpers/auth');

test.describe('Cycle Counting Feature (Positive & Negative Flows)', () => {

  test('Cycle Counting Positive & Negative Flows', async ({ page }) => {
    test.setTimeout(120000);

    // Login as admin
    await login(page, 'admin@omnisync.com', 'admin123');

    // 1. Navigate to Cycle Counts and Start New
    await page.goto('/wms/cycle-counts');
    await expect(page.locator('h3:has-text("Cycle Counting")')).toBeVisible();

    await page.click('a:has-text("Start New Count")');
    await expect(page.locator('h3:has-text("Start New Cycle Count")')).toBeVisible();

    // Select the first locator in the list
    const locatorSelect = page.locator('#locators');
    await expect(locatorSelect.locator('option').first()).toBeVisible();
    
    // Select the first option (assuming it has stock, since we filter by qty > 0)
    const firstOption = locatorSelect.locator('option').first();
    const locatorId = await firstOption.getAttribute('value');
    const locatorLabel = await firstOption.innerText();
    const locatorCode = locatorLabel.split(' - ')[0];

    // Select it
    // --- NEGATIVE FLOW: Generate without selecting locator ---
    await page.click('button[type="submit"]:has-text("Generate Count Sheet")');
    await expect(page.locator('.notyf__toast.notyf__toast--error')).toBeVisible();
    await expect(page.locator('.notyf__message').last()).toContainText('Please select at least one locator');

    await locatorSelect.selectOption(locatorId);
    
    // Generate Count Sheet
    await page.click('button[type="submit"]:has-text("Generate Count Sheet")');

    // Wait for redirect to detail page
    await expect(page.locator('h3:has-text("Count Sheet:")')).toBeVisible();
    
    // Verify it is CREATED initially
    await expect(page.locator('span:has-text("CREATED")')).toBeVisible();

    const docNo = (await page.locator('h3:has-text("Count Sheet:")').innerText()).replace('Count Sheet: ', '').trim();
    
    // Extract the exact SKU that is in this locator from the count sheet
    const productSku = await page.locator('tbody.divide-y tr').first().locator('td:nth-child(2) div.font-medium').innerText();

    // --- POSITIVE FLOW: Start Count to transition to IN_PROGRESS ---
    await page.click('button#btnStart');
    await expect(page.locator('.notyf__toast')).toBeVisible(); // Notyf success toast
    await expect(page.locator('span:has-text("IN PROGRESS (Frozen)")').last()).toBeVisible();

    // --- NEGATIVE FLOW 1: Outbound movement from frozen locator should fail ---
    await page.goto('/wms/movements/new');
    await page.selectOption('#movement-type-select', 'OUTBOUND');
    
    // Select the exact product that we know is in this frozen locator using RegExp to match SKU
    await page.selectOption('.product-select', { label: new RegExp(productSku) });
    await page.waitForTimeout(500); // Wait for UI update
    await page.fill('.quantity-input', '1');
    await page.selectOption('.locator-select', locatorId);
    await page.click('button[type="submit"]');

    // Should see an error toast about frozen locator
    await expect(page.locator('.notyf__toast').last()).toBeVisible();
    await expect(page.locator('.notyf__message').last()).toContainText('frozen for cycle counting');

    // --- NEGATIVE FLOW 2: Adjustment on frozen locator should fail ---
    await page.goto('/wms/adjustments');
    await page.click('button:has-text("New Adjustment")');
    await expect(page.locator('select[name="product_id"]')).toBeVisible();
    await page.selectOption('select[name="product_id"]', { label: new RegExp(productSku) });
    await page.waitForTimeout(500); // Wait for HTMX to load locators
    await page.selectOption('select[name="locator_id"]', locatorId);
    await page.fill('input[name="qty_delta"]', '1');
    await page.fill('input[name="remarks"]', 'Test frozen adjustment');
    await page.click('button[type="submit"]:has-text("Create Adjustment")');

    // Should see an error toast about frozen locator
    await expect(page.locator('.notyf__toast').last()).toBeVisible();
    await expect(page.locator('.notyf__message').last()).toContainText('frozen');

    // --- RETURN TO CYCLE COUNT ---
    await page.goto('/wms/cycle-counts');
    await page.click(`tr:has-text("${docNo}") a:has-text("View")`);
    await expect(page.locator('h3:has-text("Count Sheet:")')).toBeVisible();

    // --- NEGATIVE FLOW 3: Reconcile without counting should fail ---
    // Handle JS alert dialog for confirm
    page.on('dialog', async dialog => {
      await dialog.accept(); // Accept the "Are you sure you want to reconcile" confirm
    });

    await page.click('button#btnReconcile');
    
    // Wait for the Notyf error toast indicating not all lines are counted
    await expect(page.locator('.notyf__toast.notyf__toast--error')).toBeVisible();
    await expect(page.locator('.notyf__message').last()).toContainText('All lines must be counted');
    
    // Check that we are still IN_PROGRESS
    await expect(page.locator('span:has-text("IN PROGRESS (Frozen)")').last()).toBeVisible();

    // --- POSITIVE FLOW: Count all lines and Reconcile ---
    // Fill all count inputs
    const inputs = page.locator('.count-input');
    const count = await inputs.count();
    
    for(let i=0; i<count; i++) {
        const expectedStr = await inputs.nth(i).getAttribute('data-expected');
        // Let's count EXACTLY the expected amount
        await inputs.nth(i).fill(expectedStr);
        // Dispatch change event to save
        await inputs.nth(i).evaluate(node => {
            node.dispatchEvent(new Event('change', { bubbles: true }));
        });
        await page.waitForTimeout(500); // Wait for optimistic UI and API
    }

    // Now reconcile
    await page.click('button#btnReconcile');
    
    // Wait for the page to reload with new status
    await expect(page.locator('span:has-text("RECONCILED")')).toBeVisible();

    // --- POSITIVE FLOW: Complete the Count ---
    await page.click('button#btnComplete');
    await expect(page.locator('span:has-text("COMPLETED")')).toBeVisible();
  });

});
