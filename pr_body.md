Resolves #15.

**Features Implemented:**
- Introduces `InventoryLedger` and `Account` models.
- Integrates ledger insertions (double-entry bookkeeping) inside all WMS transactional queries.
- Adds `System Administrator` restricted Dashboard UI for viewing the Inventory Ledger via HTMX (with search and filters).
- Refactors Ledger logic to use singleton constants for `models.Acc*` account numbers instead of hardcoded strings.
- Adds Unit tests for Ledger Repo logic.
- Adds Playwright E2E tests for the new Ledger UI.
- Updates documentation (`AGENT.md`, `README.md`).
