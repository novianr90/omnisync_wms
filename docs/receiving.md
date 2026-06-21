# Receiving (Inbound Movements)

Receiving adds stock into a locator. Every received batch is traceable back to its inbound document.

## Who Can Receive

Roles with `manage_movements` permission: **System Admin**, **Admin WMS**, **Procurement**.

## Step-by-Step

### 1. Create an Inbound Movement

Go to **Movements → Inbound → New**.

Fill in:
- **Reference** (auto-generated or manual — e.g. PO number)
- **Warehouse** and **Locator** (where stock will be placed)
- **Lines**: Product, Quantity, UoM

Save → document is `CREATED`.

### 2. Start Processing

Click **Start** → status moves to `IN_PROGRESS`.

At this point the document is locked for editing. No stock has moved yet.

### 3. Complete

Click **Complete** → status moves to `COMPLETED`.

Stock is now added to the locator as a new batch. The Inventory Ledger records the entry automatically.

## Cross-Docking

If goods arrive and need to go out immediately (no put-away):

Go to **Movements → Cross-Docking**.

This links an inbound line directly to an outbound movement — no intermediate locator storage required.

## Viewing Received Stock

After completion:
- **Inventory → Stock Overview** → filter by locator to see the new batch
- **Inventory → Inventory Ledger** → see the inbound ledger entry with financial mapping

## Common Issues

| Problem | Fix |
|---------|-----|
| Locator not listed | Create it under Master Data → Locators |
| Product not found | Add it under Master Data → Products |
| Cannot complete | Check that all lines have qty > 0 |
| Stock shows 0 after completion | Document may still be IN_PROGRESS — click Complete |
