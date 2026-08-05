# Omnisync WMS - Agent Workspace & Architecture Guide

Welcome to the Omnisync WMS developer and agent context registry. This file documents the structural details, active configurations, database architecture, security layers, and test suites for the Omnisync WMS Suite.

---

## 🗺️ Project Architecture Overview

The WMS suite is organized into two primary microservices routed through **Kong API Gateway** running in DB-less mode (`KONG_DATABASE=off`), connecting to separate SQLite/PostgreSQL data backends:

```mermaid
graph TD
    Client[HTMX Browser / Mobile Client] -->|Port 80| Kong[Kong API Gateway - DB-less]
    Kong -->|/auth, /users, /roles| AuthService[Auth Service - Port 8000]
    Kong -->|/api/v1| WMSDashboard[WMS Dashboard Service - Port 9901]
    Kong -->|/*| WMSDashboard
    WMSDashboard -->|Authenticate Claims| AuthService
    WMSDashboard -->|Read/Write Operations| WMSDB[(SQLite dev / PostgreSQL prod+CI)]
    AuthService -->|Read/Write Operations| AuthDB[(SQLite dev / PostgreSQL prod+CI)]
```

> **Database modes**: Set `DB_TYPE=postgres` with a `WMS_DATABASE_URL` / `AUTH_DATABASE_URL` connection string to switch to PostgreSQL. For Supabase, append `?sslmode=require`. Omit `DB_TYPE` (or set `sqlite`) for local development with zero infrastructure. All SQL migrations are compatible with both engines.

### 0. API Gateway (`kong/`)
- **Port**: `80` (proxy)
- **Mode**: DB-less (`KONG_DATABASE=off`) with declarative config in `kong/kong.yml`
- **Responsibility**: Central entry point, path routing (`/auth/*`, `/api/v1/*`, `/*`), rate limiting (auth & mobile endpoints), and HTMX SSE/long-polling timeout management.

### 1. Auth Service (`auth_services/`)
- **Port**: `8000`
- **Database**: `auth.db`
- **Responsibility**: Authenticates credentials, generates cryptographically signed JWT tokens, and manages the `User` and `Role` definitions. The default initial roles are `System Admin`, `Admin WMS`, `Procurement`, and `POS`.

### 2. WMS Dashboard Service (`wms_dashboard/`)
- **Port**: `9901` (dynamically configurable)
- **Database**: `wms.db` (uses pure-Go `github.com/glebarez/sqlite` to prevent CGo runtime requirements on Windows/Linux host environments)
- **Responsibility**: Orchestrates the main WMS client dashboard, physical stock tracking with unit labels, inventory movements with dynamic packaging conversions, mobile REST APIs (`/api/v1/*`), and the **Master Data Registry**.

---

## 🔒 Master Data & Role Access Policies

The **Master Data Maintenance Registry** manages the physical layouts, product catalog items, units of measure, and dynamic packaging conversions.

### Dynamic Permissions System

RBAC has been refactored from hardcoded role-name checks to a **dynamic, permission-based system** stored in the `role_permissions` database table and propagated via JWT claims.

- **Permission granularity**: `view_ledger`, `modify_masters`, `manage_system`, `manage_movements`
- **JWT propagation**: Permissions are serialized into the `permissions` claim at login and verified by middleware on every request
- **Sidebar visibility**: Navigation menus use the `hasPermission` template function — users only see items their token grants
- **Legacy fallback**: Empty/null permissions fall back to role-name checks (System Admin → all, Admin WMS → modify_masters + manage_system)

### Default Seeded Permissions

| Role | view_ledger | modify_masters | manage_system | manage_movements |
| :--- | :---: | :---: | :---: | :---: |
| System Admin | Yes | Yes | Yes | Yes |
| Admin WMS | No | Yes | Yes | Yes |
| Procurement | No | No | No | No |
| POS | No | No | No | No |

### Seeded System Roles
- **System Admin**: Full cross-system access, role/user administration, and all permissions.
- **Admin WMS**: Administrator for warehouse operations — has `modify_masters` and `manage_system`.
- **Procurement**: Operator for incoming logistics and stock adjustments (no elevated permissions).
- **POS**: Point of Sale operator for outbound picks and stock inquiries (no elevated permissions).

Administrators can create custom roles with any combination of permissions via **System → System Roles** in the dashboard UI.

### Endpoint Matrix & Permissions

| Component | Path Prefix | HTTP Methods | Required Permission | Description |
| :--- | :--- | :--- | :--- | :--- |
| **Products** | `/wms/masters/products` | `GET` (List & Forms) | Any authenticated | View catalog list with Base UoM / open forms |
| **Products** | `/wms/masters/products` | `POST`, `PUT`, `DELETE` | `modify_masters` | Create, update, or soft-delete products |
| **Warehouses** | `/wms/masters/warehouses` | `GET` (List & Forms) | Any authenticated | View facilities list / open forms |
| **Warehouses** | `/wms/masters/warehouses` | `POST`, `PUT`, `DELETE` | `modify_masters` | Create, update, or soft-delete facilities |
| **Locators** | `/wms/masters/locators` | `GET` (List & Forms) | Any authenticated | View locator grid / open forms |
| **Locators** | `/wms/masters/locators` | `POST`, `PUT`, `DELETE` | `modify_masters` | Create, update, or soft-delete shelf locators |
| **Units of Measure** | `/wms/masters/uoms` | `GET` (List & Forms) | Any authenticated | View UoM list and Conversions split panel / open forms |
| **Units of Measure** | `/wms/masters/uoms` | `POST`, `PUT`, `DELETE` | `modify_masters` | Create, update, or soft-delete standard units |
| **Conversions** | `/wms/masters/conversions` | `GET` (Form) | Any authenticated | Open conversion modal with dynamic formula preview |
| **Conversions** | `/wms/masters/conversions` | `POST`, `DELETE` | `modify_masters` | Create or delete conversion rules |
| **System Roles** | `/wms/system/roles` | `GET`, `POST`, `PUT`, `DELETE` | `manage_system` | Manage operational roles and their permissions |
| **System Users** | `/wms/system/users` | `GET`, `POST`, `PUT` | `manage_system` | Manage users and their role assignments |
| **Movements** | `/wms/movements` | `GET` (List & Forms), `POST` (Progress steps) | `manage_movements` | View, claim, progress, journal, or complete Inbound/Outbound/Cross-Dock/Transfer movements |
| **Mobile API (Movements)** | `/api/v1/movements` | `GET`, `POST` | Valid JWT Token | Dedicated REST JSON endpoints for mobile clients (list, claim, scan-verify, submit) |
| **Inventory Ledger** | `/wms/ledger` | `GET` | `view_ledger` | View immutable audit trail of stock movements |
| **Adjustments** | `/wms/adjustments` | `GET`, `POST` | Any authenticated | View and create direct stock adjustments |
| **Cycle Counts** | `/wms/cycle-counts` | `GET`, `POST` | Any authenticated | Manage document flow (CREATED/IN_PROGRESS/RECONCILED) and physical counts |
| **Kitting** | `/wms/kitting` | `GET`, `POST` | Any authenticated | Perform product assembly and kitting |
| **QC Holds** | `/wms/qc-holds` | `GET`, `POST` | Any authenticated | Freeze stock quantities under QC investigation |
| **QC Holds** | `/wms/qc-holds/:id/release` | `POST` | Any authenticated | Release frozen stock back to available inventory |

---

## 🗃️ Database Migrations System

To prevent SQLite lockups and crash-prone schema synchronization in production, **GORM `AutoMigrate()` has been deprecated** for core schemas (except for tracking migrations). Instead, the system uses a custom pure-Go transactional SQL migration runner.

### Structural Details
- **Migration Location**: `/migrations/` (found in both `auth_services` and `wms_dashboard` roots).
- **Execution Mechanism**: Under `database.InitDB()`, the runner:
  1. Spawns/maintains a lightweight GORM-managed tracking table called `schema_migrations`.
  2. Reads all `.sql` files in the service's `migrations/` directory.
  3. Sorts them alphabetically and cross-checks them against already applied records.
  4. Executes any pending migrations inside a **single ACID transaction** using raw SQL.
- **Current Seed Scripts**:
  - `auth_services/migrations/0002_seed_roles.sql`: Seeds the operational system roles.
  - `auth_services/migrations/0003_add_role_permissions.sql`: Creates the `role_permissions` table and seeds default permission assignments.
  - `wms_dashboard/migrations/0002_seed_wms_master.sql`: Seeds default physical layouts, base units, and conversions.
  - `wms_dashboard/migrations/0017_seed_capacity_data.sql`: Seeds realistic `unit_weight`/`unit_volume` on demo products and `max_weight`/`max_volume` on all seeded locators (200 kg / 0.5 m³ per shelf).

---

## 🛡️ Relational Database Safeguards

To protect historical movement transactions and inventory records, the repository layer enforces strict transaction checks:

1. **Product Deletion Block**:
   - Deletion is prevented if physical stock remains (`qty_on_hand > 0`) in storage lots.
   - Deletion is prevented if the product is referenced in any historic `InventoryMovementLine` documents.
2. **Warehouse Deletion Block**:
   - Deletion is prevented if storage locators are defined within this warehouse.
3. **Locator Deletion Block**:
   - Deletion is prevented if physical stock currently resides on the shelf coordinates.
4. **UoM Deletion Block**:
   - Deletion is prevented if any product uses it as its Base UoM.
   - Deletion is prevented if any active conversion rule references it.

5. **QC Hold Stock Freeze**:
   - The `Storage` model now carries a `qty_on_hold` column. This quantity is excluded from **all** available-stock calculations across Outbound Movements, Kitting, and Adjustments.
   - Only the `ReleaseQCHold` function can decrement `qty_on_hold` back to available stock.

6. **Cycle Count Locator Freeze**:
   - Locators that are associated with an `IN_PROGRESS` cycle count document are frozen.
   - Any manual inventory adjustments or outbound movements involving these locators are blocked until the count is reconciled or canceled.

*Note: All deleted master records are **soft-deleted** (`gorm.DeletedAt` GORM schema attribute) to preserve ledger audits.*

---

## 🔢 Centralized Sequence & Numbering Engine

Omnisync WMS uses a transaction-locked dynamic numbering sequence engine to generate standardized, traceable document and batch numbers.

### 1. Database Schema (`sequence_generators`)
Tracks prefixes, lengths, current offset, and fiscal year per table context:
- `usage_table` (Unique Index, e.g. `inventory_movements`, `qc_holds`, `storages`)
- `prefix` (e.g. `MOV`, `ADJ`, `KIT`, `QCH`, `BAT`)
- `fiscal_year` (e.g. `2026`)
- `current_number` (Atomic increment offset)
- `number_length` (Padded length)

### 2. ACID Transaction Safety
Sequences are requested entirely within GORM transaction blocks using GORM `.Clauses(clause.Locking{Strength: "UPDATE"})` to trigger standard row-level transaction locks (`SELECT FOR UPDATE`), preventing collisions or duplicate numbers during high-volume operations.

### 3. Dynamic Fiscal Year Rollover
If the calendar system year is greater than the stored `fiscal_year`, the engine:
1. Resets `current_number` back to `1`.
2. Updates `fiscal_year` to the current year.
3. Automatically transitions numbering patterns seamlessly (e.g. `MOV-2026-00001` -> `MOV-2027-00001`).

---

## 📓 Inventory Ledger Integration (Stock Ledger)

Omnisync WMS now features an integrated `InventoryLedger` that acts as an immutable, append-only audit trail (rekening koran) for all physical stock movements, alongside their financial account mappings.

### 1. Model Structure
- Tracks granular details: `TransactionDate`, `ProductID`, `LocatorID`, `BatchNumber`, `TransactionType` (INBOUND, OUTBOUND, TRANSFER, RTV, KITTING, ADJUSTMENT, HOLD, RELEASE), and `DocumentNo`.
- **Financial Mapping**: Links movements to a Chart of Accounts (`account_no` and `contra_account_no`) for downstream COGS and valuation analysis.

### 2. Acid Compliance & Triggers
Every core WMS physical modification automatically inserts a ledger row within the same ACID `gorm.DB.Transaction`:
- **Inbound Receipts**: Debits Raw Materials/Inventory (`11000`), Credits AP (`21000`).
- **Outbound Dispatch**: Debits COGS (`51000`), Credits Inventory (`11000`).
- **RTV (Return to Vendor)**: Debits AP (`21000`), Credits Inventory (`11000`).
- **Adjustments (Found)**: Debits Inventory (`11000`), Credits Adjustment Exp (`51010`).
- **Adjustments (Lost)**: Debits Adjustment Exp (`51010`), Credits Inventory (`11000`).
- **Kitting (Components)**: Debits WIP (`11020`), Credits Inventory (`11000`).
- **Kitting (Finished)**: Debits Finished Goods (`11010`), Credits WIP (`11020`).
- **QC Hold & Release**: Logged purely for physical tracking with `QtyChange = 0`.

---

## 📦 Locator Occupancy & Space Utilization

The `/wms/masters/locators` page includes a real-time occupancy heat map computed via a read-only SQL query — no writes or locks.

### How It Works
- **Confirmed weight/volume**: `SUM(storages.qty_on_hand × products.unit_weight/unit_volume)` grouped by locator.
- **Pending inbound**: correlated subquery over `inventory_movement_lines` where `movement_type = 'INBOUND'` and `status NOT IN ('JOURNALED', 'COMPLETED', 'REJECTED')`.
- **Utilization %**: `MAX(confirmed+pending weight ÷ max_weight, confirmed+pending volume ÷ max_volume) × 100`. Falls back to `0` (displays `—`) when no capacity limit is set on a locator.
- **Colour bands**: Green < 50%, Amber 50–89%, Red ≥ 90%. Locators with pending-only inbound show a `↑` indicator.

### Key Fields Added (migration `0016`)
| Table | Column | Default |
|---|---|---|
| `locators` | `max_weight DECIMAL(10,2)` | `0` (unlimited) |
| `locators` | `max_volume DECIMAL(10,4)` | `0` (unlimited) |
| `products` | `unit_weight DECIMAL(10,4)` | `0` |
| `products` | `unit_volume DECIMAL(10,6)` | `0` |

> **Important**: Utilization will always show `0%` (even with stock or in-progress movements) if `unit_weight` and `unit_volume` are both `0` on all products, since `SUM(qty × 0) = 0`. Set these values via the Products master UI or via migration `0017`.

---

## 🛠️ Operations & Development Playbook

### Running the Services Locally
Start both backend services using PowerShell in the root directory:

```powershell
# Start the Auth Service
cd auth_services
go run cmd/main.go

# Start the WMS Dashboard Service (Set GOCACHE paths to projects local directory)
cd ../wms_dashboard
$env:GOMODCACHE="D:\Code\projects\omnisync_wms\go_cache\pkg\mod"
$env:GOCACHE="D:\Code\projects\omnisync_wms\go_cache\build"
go run cmd/main.go
```

### Running Automated Test Suites

#### 1. Backend Unit Tests
Run the Go unit test suites (which include repository safeguards and Fiber template handlers) offline using isolated SQLite environments:

```powershell
cd wms_dashboard
$env:GOMODCACHE="D:\Code\projects\omnisync_wms\go_cache\pkg\mod"
$env:GOCACHE="D:\Code\projects\omnisync_wms\go_cache\build"
go test -v ./...
```

#### 2. Playwright E2E Tests
Run the automated end-to-end user journeys inside a headless Chrome environment.

**SQLite mode (default):**
```powershell
Stop-Process -Name "wms_dashboard" -Force -ErrorAction SilentlyContinue
Remove-Item -Path "wms_dashboard\wms.db*", "auth_services\auth.db*" -Force -ErrorAction SilentlyContinue

cd wms_dashboard
npx playwright test
```

**PostgreSQL mode (mirrors CI):**
```bash
DB_TYPE=postgres PGHOST=localhost PGPORT=5432 PGUSER=omnisync PGPASSWORD=test1234 \
AUTH_DATABASE_URL=postgres://omnisync:test1234@localhost:5432/omnisync_test?sslmode=disable \
npx playwright test
```

**Parallel worker infrastructure (CI):**
- `e2e/global-setup.js` boots 1 shared Auth Service (port 8000) + 4 WMS instances (ports 9901–9904), each backed by its own isolated Postgres DB (`wms_test_0..3`). Migrations run automatically on WMS startup via `schema_migrations` tracker.
- `e2e/fixtures/index.js` resolves per-worker `baseURL` using `testInfo.workerIndex % NUM_WORKERS` to guard against worker restart index overflow.
- `e2e/global-teardown.js` sends SIGTERM to all spawned PIDs recorded in `.e2e-pids.json`.

---

## ⚠️ AI Agent Development Gotchas & Best Practices

1. **Environment Variables & Secrets:** 
   - There are **no hardcoded credentials** in the source code. Both services strictly require `.env` files.
   - If writing custom run scripts (like `run_all.ps1`), remember that background child processes or new PowerShell windows **do not** automatically inherit `.env` context via `godotenv`. You must explicitly parse the `.env` file and export the variables into the parent shell environment before launching child processes.
2. **GORM Pluralization Quirk:**
   - GORM's automatic snake_case pluralization breaks on acronyms with mixed casing. For example, `UoM` becomes `uo_ms` and `UoMConversion` becomes `uo_m_conversions`. 
   - **Rule:** Always use explicit `TableName()` receiver methods on models involving acronyms to enforce clean database table names (e.g., `func (UoM) TableName() string { return "uoms" }`).
3. **Number Inputs with Absolute Text:**
   - Avoid absolutely positioning text (like `Factor`) inside a `<input type="number">` on the right side. Browser default spinner arrows (up/down) will render over the text. Prefer putting labels outside the input or using browser-native styling hacks.
4. **Dynamic Lucide Re-rendering & UI Collapses:**
   - Lucide converts standard `<i>` tags into `<svg>` elements on load. Calling `.setAttribute("data-lucide", ...)` directly on the generated `<svg>` tag will **not** trigger Lucide to re-draw.
   - **Rule:** To dynamically change an icon, always recreate a fresh `<i>` element inside the container (e.g. `container.innerHTML = '<i data-lucide="..."></i>'`) and trigger `lucide.createIcons()` again.
   - **Clipping/Opacity on Collapsible Containers:** When animating a container to `width: 0` and `opacity: 0` (like a collapsed sidebar), do **not** place toggle triggers inside that container. They will inherit transparency and clipping, rendering them invisible. Keep floating controllers outside the collapsed aside in the DOM tree.
5. **No Manual Closing of GitHub Issues:**
   - **Rule:** Do NOT manually close GitHub issues. Always leave issues open and let the Pull Request automatically close the linked issue upon merging (e.g. by using the `Closes #XX` or `Resolves #XX` pattern in the PR body description). Ensure you commit all changes, push them to a feature branch, and create a PR instead of directly closing the issue.
6. **Page-Reload Success Toasts (`setReloadToast`):**
   - **Gotcha:** Operational flows that trigger HTMX client-side reloads (via `HX-Refresh: true` header) will destroy any inline toast notifications sent in the direct HTTP response.
   - **Rule:** Use `setReloadToast(c, message, isSuccess)` (defined in `wms_handler.go`) which saves the message into temporary cookie stores (`toast_msg` and `toast_type`). The `base.html` layout parses these cookies globally upon reload and renders a sleek, persistent Notyf pop-up.
7. **Asynchronous Dropdowns (Race Conditions in Playwright):**
   - **Gotcha:** Selecting a product in dynamic dropdowns (like the Kitting components panel) triggers an async AJAX fetch to load locators containing active stock.
   - **Rule:** When writing Playwright E2E tests, always wait for the options to populate before selecting them by asserting `await expect(row.locator('select[name="comp_locator_id[]"] option[value="loc-001"]')).toBeAttached();` to prevent flake.
8. **Docs-Only Changes Skip Testing:**
   - Changes that **only** modify documentation files (e.g. `*.md`, `LICENSE`, `.gitignore`, `*.txt`, `docs/*`, `.github/workflows/*`) do **not** require running unit tests or E2E tests.
   - **Rule:** If a PR or commit exclusively touches documentation, the CI pipeline will automatically skip the Lint, Unit Tests, and E2E Tests jobs. No manual test runs are needed before pushing docs-only changes.
9. **CI Gate Must Pass Before Merge:**
   - The `CI Gate` status check is a **required** check on the `master` branch. PRs **cannot** be merged if any test (lint, unit, or E2E) has failed.
   - **Rule:** Never bypass or force-merge a PR with a failing CI Gate. Fix the failing tests first, push the fix, and wait for the pipeline to go green before merging.
10. **Always Use Relative URLs in Playwright E2E Tests:**
    - **Gotcha:** Hardcoding `http://localhost:9901` in `page.goto()` or `request.get/post()` calls breaks parallel E2E runs. Each Playwright worker is assigned a different port (9901–9904) backed by its own isolated DB. A test running on worker 2 (port 9903) that navigates to port 9901 will hit a different DB — finding no data and timing out.
    - **Rule:** Always use relative paths (e.g. `page.goto('/wms/movements')`, `request.get('/api/v1/movements')`). Playwright resolves them against the per-worker `baseURL` fixture automatically. Never hardcode `localhost:PORT` anywhere in spec files.
11. **PostgreSQL Compatibility Rules:**
    - **GROUP BY strictness:** PostgreSQL requires all non-aggregated `SELECT` columns to appear in `GROUP BY`. SQLite allows selecting unaggregated columns. Always list every projected column explicitly in `GROUP BY` clauses, not just the primary key.
    - **LIKE case sensitivity:** SQLite `LIKE` is case-insensitive; PostgreSQL `LIKE` is case-sensitive. `ILIKE` is Postgres-only. Use `LOWER(col) LIKE LOWER(?)` — it is portable across both engines and safe in unit tests.
    - **GORM `Save()` with nullable FK columns:** `Save()` writes all struct fields, including zero-values. An empty string `""` for a foreign key column is sent as `''` to Postgres which rejects it (FK violation). Use `tx.Omit("FieldName")` conditionally when the FK field is empty before calling `Save()` or `Create()`.
    - **Boolean literals:** SQLite accepts `0`/`1`; PostgreSQL requires `TRUE`/`FALSE`. Never use integer literals for boolean columns in raw SQL.

---

## 💡 Tech Stack Checklist
- **Backend Framework**: Go Fiber v2
- **Database Mapping**: GORM v2 (ORM) + Pure-Go SQLite Driver (`github.com/glebarez/sqlite`)
- **Frontend SPA Layer**: HTMX v1.9.10 (Asynchronous swaps & dynamic forms)
- **Styling Core**: Tailwind CSS v4.0 + Custom Glassmorphism Theme (Outfit + Inter font faces)
- **UI Feedback**: Notyf (Sleek Toast Notifications)
- **Iconsets**: Lucide Icons (asynchronously re-bound on HTMX swaps)
