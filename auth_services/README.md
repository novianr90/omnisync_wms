# Auth Service — Omnisync WMS

The authentication microservice for Omnisync WMS. Handles user registration, login, and JWT token issuance. All other services validate identity by consuming the tokens this service produces.

---

## Responsibilities

- User registration and credential storage (bcrypt-hashed passwords)
- Login with email + password → returns a signed JWT
- Token verification endpoint for other services
- Role assignment: `admin` or `operator`

---

## Endpoints

| Method | Path | Auth Required | Description |
|---|---|---|---|
| `POST` | `/auth/register` | No | Register a new user |
| `POST` | `/auth/login` | No | Login and receive a JWT cookie |
| `POST` | `/auth/logout` | No | Clear the JWT cookie |
| `GET` | `/auth/verify` | JWT cookie or Bearer | Verify token validity and return claims |

---

## Setup

### 1. Install dependencies

```bash
go mod download
```

### 2. Configure environment

```bash
cp .env.example .env
```

Edit `.env` and set your values:

```env
JWT_SECRET_KEY=your-strong-random-secret   # Must match wms_dashboard
PORT=8000
DB_TYPE=sqlite                             # or postgres
AUTH_DATABASE_URL=...                      # Only needed for postgres
SEED_ADMIN_EMAIL=admin@omnisync.com
SEED_ADMIN_PASSWORD=CHANGE_ME
SEED_OPERATOR_EMAIL=operator@omnisync.com
SEED_OPERATOR_PASSWORD=CHANGE_ME
```

> ⚠️ `JWT_SECRET_KEY` must be identical in both `auth_services/.env` and `wms_dashboard/.env`.

### 3. Run

```bash
go run cmd/main.go
```

Service starts on `http://localhost:8000`.

---

## Database

- **Development**: SQLite (`auth.db`) — auto-created on first run, no setup needed.
- **Production**: PostgreSQL — set `DB_TYPE=postgres` and provide `AUTH_DATABASE_URL` (append `?sslmode=require` if using Supabase).

Schema is automatically migrated via GORM `AutoMigrate` on startup.

---

## Seed Users

On first boot (when the database is empty), two users are seeded using credentials from `.env`:

| Role | Default Email | Default Password |
|---|---|---|
| `admin` | `admin@omnisync.com` | `admin123` |
| `operator` | `operator@omnisync.com` | `operator123` |

Change these in `.env` before first run in any shared environment.

---

## Tech Stack

| | |
|---|---|
| Framework | Go Fiber v2 |
| ORM | GORM v2 |
| Database (dev) | SQLite (`glebarez/sqlite`) |
| Password Hashing | bcrypt (cost 14) |
| JWT | `golang-jwt/jwt` v5 (HS256) |
| Env | `godotenv` |

---

## License

[MIT](../LICENSE) © 2026 Novian Iskandar
