const { test, expect } = require('@playwright/test');
const { login } = require('./helpers/auth');

test.describe('Dynamic RBAC access control', () => {
    test('should allow custom role with modify_masters to view products but restrict ledger access', async ({ page }) => {
        test.setTimeout(90000);
        // 1. Log in as System Admin
        await login(page, 'admin@omnisync.com', 'admin123');

        // Generate unique names
        const uniqueId = Date.now();
        const roleName = `Specialist-${uniqueId}`;
        const email = `specialist-${uniqueId}@omnisync.com`;

        // 2. Navigate to Roles Registry and create a custom Specialist role
        await page.goto('http://localhost:9901/wms/system/roles');
        await page.locator('button:has-text("Create New Role")').click();
        await page.fill('#new-role-modal input[name="name"]', roleName);
        await page.fill('#new-role-modal textarea[name="description"]', 'Custom specialist role');
        
        // Check only modify_masters permission
        const modifyMastersCb = page.locator('#new-role-modal input[name="permissions"][value="modify_masters"]');
        await modifyMastersCb.check();
        
        await page.locator('#new-role-modal button[type="submit"]').click();
        
        // Wait for system role success toast to ensure DB write completed
        await expect(page.locator('.notyf__message').last()).toContainText('Role created successfully');

        // 3. Navigate to User Account Registry and assign the custom role to a new user
        await page.goto('http://localhost:9901/wms/system/users');
        await page.locator('button:has-text("Add New User")').click();
        await page.fill('#new-user-modal input[name="first_name"]', 'Jane');
        await page.fill('#new-user-modal input[name="last_name"]', 'Specialist');
        await page.fill('#new-user-modal input[name="email"]', email);
        await page.fill('#new-user-modal input[name="password"]', 'password123');
        await page.selectOption('#new-user-modal select[name="role"]', roleName);
        await page.locator('#new-user-modal button[type="submit"]').click();

        // Wait for user creation success toast to ensure DB write completed
        await expect(page.locator('.notyf__message').last()).toContainText('User created successfully');

        // 4. Log out System Admin
        await page.goto('http://localhost:9901/logout');
        await page.waitForURL('**/login');

        // 5. Log in as the new user
        await login(page, email, 'password123');

        // 6. Verify Jane can view products master (modify_masters)
        await page.goto('http://localhost:9901/wms/masters/products');
        await expect(page.locator('h3:has-text("Product Master Maintenance")')).toBeVisible();

        // 7. Verify Jane is denied access to Ledger (view_ledger)
        await page.goto('http://localhost:9901/wms/ledger');
        await expect(page.locator('body')).toContainText('Access Denied');
    });
});
