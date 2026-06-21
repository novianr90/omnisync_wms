# WMS Overview

## What is Omnisync WMS?

A self-hosted warehouse management system for tracking inventory movements, managing master data, and maintaining an audit-grade stock ledger — built for small-to-medium warehouses that need reliable control without enterprise overhead.

## Core Concepts

### Stock = Batches at Locators

Every unit of inventory is stored as a **batch** at a **locator**. A batch has:
- Product, quantity, UoM
- An inbound source document (the receiving movement that created it)

When stock moves out, it is consumed FIFO from the oldest batch first.

### Documents Drive Movement

All stock changes happen through **documents**:

| Document Type | Direction | What it does |
|---------------|-----------|--------------|
| Inbound Movement | → stock in | Receives goods into a locator |
| Outbound Movement | ← stock out | Issues goods from a locator |
| Transfer | locator → locator | Moves stock within the warehouse |
| Kitting | multi-in → single-out | Assembles components into a finished product |
| RTV (Return to Vendor) | ← stock out | Returns defective/excess to supplier |

Each document follows a **lifecycle**:
```
CREATED → IN_PROGRESS → COMPLETED
```
Stock is not affected until a document reaches `COMPLETED`.

### Ledger = Immutable Audit Trail

Every completed movement writes an entry to the **Inventory Ledger**, mapping:
- Physical stock change (product, qty, locator)
- Financial entry (debit/credit against Chart of Accounts)

The ledger cannot be edited. It can be exported to Excel or PDF.

## User Roles

| Role | Can do |
|------|--------|
| System Admin | Everything — users, roles, master data, movements, ledger |
| Admin WMS | Master data + movements + system settings, no user management |
| Procurement | Create/complete inbound movements |
| POS | Create/complete outbound movements |

## Navigation Map

```
Dashboard
├── Inventory
│   ├── Stock Overview       ← current locator balances
│   ├── Inventory Ledger     ← audit trail (export to Excel/PDF)
│   └── Valuation Report     ← cost-based stock value
├── Movements
│   ├── Inbound              ← receiving
│   ├── Outbound             ← issuing
│   ├── Transfers            ← locator-to-locator
│   ├── Kitting              ← assembly
│   ├── RTV                  ← return to vendor
│   └── Cross-Docking        ← direct inbound-to-outbound
├── Quality Control
│   └── QC Hold              ← freeze stock under investigation
├── Cycle Counting           ← physical count reconciliation
├── Master Data
│   ├── Products
│   ├── Warehouses
│   ├── Locators
│   └── Units of Measure
└── Settings
    ├── Users
    └── Roles
```

## Typical Daily Workflow

1. **Morning receiving** → Inbound Movement → complete → stock appears in locators
2. **Fulfillment** → Outbound Movement → complete → stock decreases, ledger updated
3. **Internal moves** → Transfer → relocate stock between locators
4. **Periodic counts** → Cycle Count → freeze locator → count → reconcile discrepancies
