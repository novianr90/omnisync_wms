INSERT INTO accounts (account_no, account_name, account_type) VALUES
    ('11000', 'Raw Materials Inventory', 'ASSET'),
    ('11010', 'Finished Goods Inventory', 'ASSET'),
    ('11020', 'Work In Progress (WIP) Inventory', 'ASSET'),
    ('21000', 'Accounts Payable', 'LIABILITY'),
    ('40000', 'Sales Revenue', 'REVENUE'),
    ('51000', 'Cost of Goods Sold (COGS)', 'EXPENSE'),
    ('51010', 'Inventory Adjustment Expense', 'EXPENSE'),
    ('51020', 'QC Scrap / Write-Off Expense', 'EXPENSE')
ON CONFLICT (account_no) DO NOTHING;
