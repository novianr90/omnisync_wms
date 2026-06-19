# Omnisync WMS

> A lightweight, self-hosted **Warehouse Management System** built with Go, HTMX, and Tailwind CSS — designed for teams that need fast, real-time inventory control without the enterprise overhead.

![License](https://img.shields.io/badge/license-MIT-blue.svg)
![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go)
![HTMX](https://img.shields.io/badge/HTMX-1.9-3D72D7)

---

## Architecture

Omnisync WMS is a two-service monorepo. Each service is independently deployable and communicates over HTTP.

```
omnisync_wms/
├── auth_services/       # JWT Authentication Service  (port 8000)
└── wms_dashboard/       # WMS Dashboard & CRUD Service (port 9901)
```

```
┌─────────────────────┐        ┌──────────────────────────┐
│   Browser (HTMX)    │──9901──▶   WMS Dashboard Service   │
└─────────────────────┘        │  Go Fiber + GORM + SQLite │
                                │  Tailwind CSS v4 + HTMX   │
                                └────────────┬─────────────┘
                                             │ POST /auth/login
                                             ▼
                                ┌──────────────────────────┐
                                │   Auth Service  :8000     │
                                │  Go Fiber + bcrypt + JWT  │
                                └──────────────────────────┘
```

### Services at a Glance

| Service | Directory | Port | Database |
|---|---|---|---|
| Auth Service | `auth_services/` | `8000` | `auth.db` (SQLite) |
| WMS Dashboard | `wms_dashboard/` | `9901` | `wms.db` (SQLite) |

---

## Features

- 📦 **Inventory Tracking** — Real-time stock levels with FIFO batch management
- 📖 **Inventory Ledger** — Immutable audit trail of all physical stock movements mapped to Chart of Accounts (COGS/Valuation)
- 🏭 **Master Data Registry** — Products, Warehouses, Locators, and Units of Measure
- ⚖️ **Dynamic UoM Conversions** — User-defined formulas (e.g. convert `kg` ↔ `packs`)
- 📝 **Inventory Transactions** — Direct stock adjustments and Product Kitting/Assembly
- 🚚 **Inventory Movements** — Inbound, outbound, and transfer workflows with claim/journal/complete lifecycle
- 📱 **Mobile REST APIs** — Dedicated JSON endpoints for mobile operators to execute warehouse workflows
- 🔢 **Dynamic Numbering Sequence Engine** — Traceable document/batch numbers using transactional row locks (SELECT FOR UPDATE) and auto fiscal-year rollover
- 🛡️ **QC Hold (Stock Freeze)** — Quarantine specific stock quantities under QC investigation; frozen stock is excluded from all outbound movements and kitting
- 🔒 **Dynamic Role-Based Access Control** — Granular permissions (`view_ledger`, `modify_masters`, `manage_system`, `manage_movements`) stored in DB, propagated via JWT, with an editable Role Registry UI
- ⚡ **HTMX-Powered UI** — No page reloads, instant feedback via server-side HTML fragments
- 🌑 **Glassmorphism Dark Theme** — Tailwind CSS v4 with custom design tokens
- 🗺️ **Collapsible & Responsive Navigation** — Clean collapsible sidebar on desktop (state persistent via `localStorage` with FOUC prevention) and fluid drawer slide-over navigation on mobile views

---

## Prerequisites

- [Go 1.21+](https://go.dev/dl/)
- [Node.js](https://nodejs.org/) (for Tailwind CSS CLI)
- Git

---

## Quick Start

### 1. Clone the repository

```bash
git clone https://github.com/novianr90/omnisync_wms.git
cd omnisync_wms
```

### 2. Configure environment variables

Both services require a `.env` file. Copy the examples and fill in values:

```bash
# Auth Service
cp auth_services/.env.example auth_services/.env

# WMS Dashboard
cp wms_dashboard/.env.example wms_dashboard/.env
```

> **Important:** Both services must share the same `JWT_SECRET_KEY` value.

### 3. Start the Auth Service

```bash
cd auth_services
go run cmd/main.go
```

### 4. Start the WMS Dashboard

```bash
cd wms_dashboard
go run cmd/main.go
```

Then open **http://localhost:9901** in your browser.

### Default Seed Credentials

On first boot, the Auth Service seeds two demo users (configurable via `.env`):

| Role | Email | Password |
|---|---|---|
| Admin | `admin@omnisync.com` | `admin123` |
| Operator | `operator@omnisync.com` | `operator123` |

> Change these in `auth_services/.env` before deploying.

---

## Running Both Services at Once

A convenience script is included in the root:

```powershell
# Windows PowerShell
.\run_all.ps1
```

```bash
# Linux / macOS
chmod +x run_all.sh && ./run_all.sh
```

---

## Running Tests

Omnisync WMS comes equipped with robust unit and end-to-end testing suites.

### 1. Backend Unit Tests
Run backend unit tests offline using local, isolated SQLite test databases:
```bash
cd wms_dashboard
go test -v ./...
```

### 2. Playwright E2E Tests
To run full end-to-end browser workflows (encompassing Authentication, Side-menu HTMX navigation, Inbound, FIFO Outbound, QC Hold quarantine, Stock Adjustments, and Kitting Light Assembly):

1. **Clear DB locks & clean states**:
   ```powershell
   # Windows PowerShell
   Stop-Process -Name "main" -Force -ErrorAction SilentlyContinue
   Stop-Process -Name "wms_dashboard" -Force -ErrorAction SilentlyContinue
   Remove-Item -Path "wms_dashboard\wms.db*", "auth_services\auth.db*" -Force -ErrorAction SilentlyContinue
   ```
2. **Execute Playwright runner**:
   ```bash
   cd wms_dashboard
   npx playwright test
   ```

---

## Tech Stack

| Layer | Technology |
|---|---|
| Backend Framework | [Go Fiber v2](https://gofiber.io/) |
| ORM | [GORM v2](https://gorm.io/) |
| Database (dev) | SQLite via [`glebarez/sqlite`](https://github.com/glebarez/sqlite) (pure Go, no CGo) |
| Database (prod) | PostgreSQL (set `DB_TYPE=postgres`, append `?sslmode=require` for Supabase) |
| Frontend | [HTMX v1.9](https://htmx.org/) |
| Styling | [Tailwind CSS v4](https://tailwindcss.com/) |
| Icons | [Lucide Icons](https://lucide.dev/) |
| Auth | JWT via [`golang-jwt/jwt`](https://github.com/golang-jwt/jwt) + bcrypt |
| Env Loading | [`godotenv`](https://github.com/joho/godotenv) |

---

## Repository Structure

```
omnisync_wms/
├── auth_services/
│   ├── cmd/main.go              # Entry point
│   ├── internal/
│   │   ├── database/            # DB init & migrations
│   │   ├── handlers/            # Auth endpoints (login, register, verify)
│   │   └── models/              # User model
│   ├── .env.example
│   └── README.md
├── wms_dashboard/
│   ├── cmd/main.go              # Entry point + seed data
│   ├── internal/
│   │   ├── database/            # DB init & migrations
│   │   ├── handlers/            # Page & HTMX fragment handlers
│   │   ├── middleware/          # JWT auth + role guard
│   │   ├── models/              # GORM models
│   │   └── repository/          # DB query logic
│   ├── web/
│   │   ├── static/              # CSS, JS assets
│   │   └── templates/           # Go HTML templates (layouts, pages, partials)
│   ├── .env.example
│   └── README.md
├── AGENT.md                     # Architecture guide for AI agents
├── LICENSE
└── README.md
```

---

## License

This project is licensed under the [MIT License](LICENSE).
