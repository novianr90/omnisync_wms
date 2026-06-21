# Kitting (Assembly)

Kitting consumes component stock and produces a finished/assembled product. FIFO batch consumption applies to all components.

## Who Can Kit

Roles with `manage_movements` permission: **System Admin**, **Admin WMS**.

## Concepts

- **Components** (inputs): raw materials or sub-assemblies consumed
- **Output** (finished product): the assembled product added to stock
- All component stock is consumed before the output batch is created

## Step-by-Step

### 1. Create a Kitting Order

Go to **Movements → Kitting → New**.

Fill in:
- **Output Product**: the finished item being produced
- **Output Quantity + UoM**
- **Output Locator**: where finished goods will be placed
- **Components**: add each component with quantity and source locator

> The system checks available batch stock per locator when you add components. If insufficient stock exists, you will see a validation error.

### 2. Start

Click **Start** → status moves to `IN_PROGRESS`.

Component batches are reserved (not yet consumed).

### 3. Complete

Click **Complete** → status moves to `COMPLETED`.

- Component batches are consumed FIFO
- Output batch is created at the output locator
- Ledger entries are written: COGS debit for components, inventory credit for output

## Partial Kitting

Kitting does not support partial completion. Either complete the full order or cancel it and create a new one for the revised quantity.

## Viewing Kitting Results

- **Inventory → Stock Overview** → output locator now shows the new finished product batch
- **Inventory → Inventory Ledger** → component consumption and output creation entries appear as a paired transaction

## Common Issues

| Problem | Fix |
|---------|-----|
| "Insufficient stock" on component | Check Stock Overview for available batches at that locator |
| Output locator not available | Create it under Master Data → Locators |
| Components consumed but output missing | Check if output product exists in Master Data → Products |
