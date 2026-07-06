# Cycle Counting

Cycle counting is the process of physically counting stock at specific locators and reconciling discrepancies against the system record — without stopping all warehouse operations.

## Who Can Cycle Count

**System Admin** and **Admin WMS** (roles with `manage_system` permission).

## Lifecycle

```
CREATED → IN_PROGRESS → RECONCILED
```

During `IN_PROGRESS`, the locator is **frozen**: no outbound movements or kitting can consume stock from it until reconciliation is complete.

## Step-by-Step

### 1. Create a Cycle Count

Go to **Cycle Counting → New**.

Select:
- **Locator** to count
- The system auto-populates **expected quantities** from current batch balances

Save → status is `CREATED`.

### 2. Start the Count

Click **Start Count** → status moves to `IN_PROGRESS`.

The locator is now frozen. Physically count the stock and enter **actual quantities** for each product line.

> You can save partial progress and return to the document later — the locator remains frozen until you reconcile.

### 3. Reconcile

Click **Reconcile** → status moves to `RECONCILED`.

For each line where `actual ≠ expected`:
- **Positive variance** (actual > expected): adjustment batch is added
- **Negative variance** (actual < expected): excess batch is consumed
- **Zero variance**: no change

Adjustments are recorded in the Inventory Ledger as inventory adjustment entries.

The locator freeze is lifted after reconciliation.

## Best Practices

- Count high-value or high-velocity locators weekly
- Count all locators at least once per quarter (full stock opname)
- Do not start a cycle count on a locator with pending inbound movements — complete or cancel those first
- If a discrepancy is large, investigate before reconciling (check recent ledger entries for that locator)

## Full Stock Opname

A full opname is simply cycle counting all locators. Run them in batches:
1. Create counts for all locators in one warehouse
2. Freeze and count sequentially or in parallel
3. Reconcile each one

## Viewing Results

After reconciliation:
- **Inventory → Stock Overview** → locator balances reflect the reconciled quantities
- **Inventory → Inventory Ledger** → adjustment entries show the delta with `cycle_count` as the source document type
