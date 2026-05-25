# WMS Dashboard — Omnisync WMS

The main operational dashboard for Omnisync WMS. Provides real-time inventory visibility, inventory movement workflows, and a full Master Data Registry — all rendered server-side using Go HTML templates with HTMX for zero-reload interactivity.

---

## Features

| Feature | Description |
|---|---|
| **Live Inventory View** | Stock levels per product/locator with FIFO batch tracking |
| **Inventory Movements** | Create inbound, outbound, or transfer orders with claim → journal → complete lifecycle |
| **Master Data: Products** | SKU catalog with base Unit of Measure assignment |
| **Master Data: Warehouses** | Physical facility registry with active/inactive status |
| **Master Data: Locators** | Shelf-level coordinates (Zone / Aisle / Shelf / Level) |
| **Master Data: UoM** | Standard units (kg, pcs, box…) with user-defined conversion rules |
| **UoM Conversions** | Dynamic formula rules (e.g. `1 pack = 1.0 kg`) for inbound processing |
| **RBAC** | `admin` can mutate master data; `operator` has read-only access |

---

## Setup

### 1. Install dependencies

```bash
go mod download
```

For CSS (Tailwind):

```bash
npm install
```

### 2. Configure environment

```bash
cp .env.example .env
```

Edit `.env`:

```env
JWT_SECRET_KEY=your-strong-random-secret   # Must match auth_services
PORT=9901
AUTH_API_URL=http://localhost:8000
DB_TYPE=sqlite                             # or postgres
WMS_DATABASE_URL=...                       # Only needed for postgres
```

> ⚠️ `JWT_SECRET_KEY` must be identical in both `auth_services/.env` and `wms_dashboard/.env`.

### 3. Run

```bash
go run cmd/main.go
```

Dashboard available at **http://localhost:9901**.

---

## API Routes

### Public

| Method | Path | Description |
|---|---|---|
| `GET` | `/login` | Login page |
| `POST` | `/auth/login-submit` | Submit credentials (proxied to auth_services) |
| `GET` | `/logout` | Clear session and redirect to login |

### Authenticated (any role)

| Method | Path | Description |
|---|---|---|
| `GET` | `/` | Main dashboard |
| `GET` | `/wms/inventory` | Live inventory fragment (HTMX) |
| `GET` | `/wms/masters/products` | Products master list |
| `GET` | `/wms/masters/warehouses` | Warehouses master list |
| `GET` | `/wms/masters/locators` | Locators master list |
| `GET` | `/wms/masters/uoms` | Units of Measure + Conversions panel |

### Admin Only

| Method | Path | Description |
|---|---|---|
| `POST/PUT/DELETE` | `/wms/masters/products/:id?` | Create / update / delete products |
| `POST/PUT/DELETE` | `/wms/masters/warehouses/:id?` | Create / update / delete warehouses |
| `POST/PUT/DELETE` | `/wms/masters/locators/:id?` | Create / update / delete locators |
| `POST/PUT/DELETE` | `/wms/masters/uoms/:id?` | Create / update / delete UoMs |
| `POST/DELETE` | `/wms/masters/conversions/:id?` | Create / delete conversion rules |
| `POST` | `/wms/movements/new` | Create an inventory movement |
| `POST` | `/wms/movements/:id/claim` | Claim a movement |
| `POST` | `/wms/movements/:id/journal` | Journal a movement |
| `POST` | `/wms/movements/:id/complete` | Complete a movement |
| `POST` | `/wms/movements/:id/reject` | Reject a movement |

---

## Database

- **Development**: SQLite (`wms.db`) — auto-created on first run.
- **Production**: PostgreSQL — set `DB_TYPE=postgres` and provide `WMS_DATABASE_URL`.

Schema is automatically migrated via GORM `AutoMigrate` on startup.

### Deletion Safeguards

The repository layer prevents destructive operations that would break historical integrity:

- **Products**: Cannot delete if stock exists or is referenced in movement history
- **Warehouses**: Cannot delete if locators are defined within it
- **Locators**: Cannot delete if physical stock currently occupies the shelf
- **UoMs**: Cannot delete if used as a product's base UoM or in any conversion rule

All deletes are **soft-deletes** (`gorm.DeletedAt`) to preserve the audit ledger.

---

## Project Structure

```
wms_dashboard/
├── cmd/
│   └── main.go                  # Entry point, routes, seed data
├── internal/
│   ├── database/
│   │   └── database.go          # DB init & AutoMigrate
│   ├── handlers/
│   │   ├── wms_handler.go       # Dashboard, inventory, movement handlers
│   │   └── master_handler.go    # Master data CRUD handlers
│   ├── middleware/
│   │   ├── jwt_auth.go          # JWT validation middleware
│   │   └── admin_auth.go        # Admin role guard middleware
│   ├── models/                  # GORM models (Product, Warehouse, Locator, UoM…)
│   └── repository/              # DB query logic with safeguard checks
└── web/
    ├── static/
    │   └── css/style.css        # Compiled Tailwind CSS + custom theme
    └── templates/
        ├── layouts/base.html    # Base shell (sidebar, nav, HTMX scripts)
        ├── pages/               # Full-page templates
        └── partials/            # HTMX fragment templates (rows, modals, notifications)
```

---

## Tech Stack

| | |
|---|---|
| Framework | Go Fiber v2 |
| ORM | GORM v2 |
| Database (dev) | SQLite (`glebarez/sqlite`, pure Go — no CGo) |
| Frontend | HTMX v1.9 |
| Styling | Tailwind CSS v4 + custom glassmorphism dark theme |
| Fonts | Outfit + Inter (Google Fonts) |
| Icons | Lucide Icons |
| Env | `godotenv` |

---

## License

[MIT](../LICENSE) © 2026 Novian Iskandar
