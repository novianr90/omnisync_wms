# Setup Guide

## Docker Deployment (Recommended)

### Prerequisites

- Docker + Docker Compose v2
- A PostgreSQL database (local or Supabase)

### 1. Configure Environment

```bash
cp .env.compose.example .env
```

Edit `.env` — minimum required values:

```env
JWT_SECRET_KEY=<strong-random-secret>
APP_ENV=production
```

**Supabase (external Postgres):**
```env
AUTH_DATABASE_URL=postgres://user:pass@db.supabase.co:5432/postgres?sslmode=require
WMS_DATABASE_URL=postgres://user:pass@db.supabase.co:5432/postgres?sslmode=require
```

**Local Postgres (no external DB):**
```env
POSTGRES_USER=omnisync
POSTGRES_PASSWORD=your_password
POSTGRES_DB=omnisync_wms
AUTH_DATABASE_URL=postgres://omnisync:your_password@postgres:5432/omnisync_wms?sslmode=disable
WMS_DATABASE_URL=postgres://omnisync:your_password@postgres:5432/omnisync_wms?sslmode=disable
```

### 2. Start the Stack

```bash
# Supabase / external Postgres
make up

# Local Postgres
make up-local

# Local Postgres + Portainer (monitoring UI on :9000) + Cloudflare Tunnel
make up-full
```

All services start in the background. Check status:

```bash
docker compose ps
docker compose logs -f
```

### 3. Create the First Admin User

In `APP_ENV=production`, no default users are seeded. Create the first admin manually:

```bash
make create-admin EMAIL=admin@yourcompany.com PASSWORD=StrongPassword123
```

Or directly:
```bash
docker compose exec auth ./auth_services create-admin admin@yourcompany.com StrongPassword123
```

Login at **http://localhost** (nginx on port 80).

### 4. Define Master Data

After first login as System Admin, set up master data in this order:

1. **Units of Measure** → `Master Data > Units of Measure`
   - Create base units: `pcs`, `kg`, `box`, etc.
   - Add conversions if needed (e.g. 1 `box` = 12 `pcs`)

2. **Products** → `Master Data > Products`
   - Add SKU, name, default UoM, and costing method

3. **Warehouses** → `Master Data > Warehouses`
   - Create at least one warehouse before adding locators

4. **Locators** → `Master Data > Locators`
   - Assign each locator to a warehouse (e.g. `A-01-01`, `A-01-02`)
   - Set capacity if using the space utilization dashboard

### 5. Assign Roles

Go to `Settings > Roles` to create or edit roles. Default roles:

| Role | Permissions |
|------|-------------|
| System Admin | All permissions |
| Admin WMS | modify_masters, manage_system, manage_movements |
| Procurement | manage_movements |
| POS | manage_movements |

Register additional users via `Settings > Users`.

---

## Local Development (without Docker)

```bash
cp auth_services/.env.example auth_services/.env
cp wms_dashboard/.env.example wms_dashboard/.env
# Set the same JWT_SECRET_KEY in both files

# Start both services (PowerShell)
.\run_all.ps1

# Or individually
cd auth_services && go run cmd/main.go
cd wms_dashboard && go run cmd/main.go
```

Open **http://localhost:9901**. Default credentials: `admin@omnisync.com` / `admin123`.
