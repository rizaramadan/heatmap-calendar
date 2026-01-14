# Heatmap Internal - Load Calendar & Capacity Planning System

A workload visualization tool that tracks tasks against personal/team capacity, displays an interactive heatmap, and triggers alerts when overload is detected.

## Tech Stack

- **Backend:** Go 1.25+ with Echo v4 web framework
- **Database:** PostgreSQL 15
- **Frontend:** HTML Templates + HTMX + Tailwind CSS
- **Email:** Mailgun (OTP authentication)
- **Automation:** n8n webhook integration

## Project Structure

```
heatmap-internal/
├── cmd/server/main.go           # Entry point
├── internal/
│   ├── config/config.go         # Environment configuration
│   ├── database/
│   │   ├── postgres.go          # DB connection pool
│   │   └── migrations.go        # Schema & seed data
│   ├── models/models.go         # Data structures
│   ├── repository/              # Data access layer
│   │   ├── entity.go            # Person & Group CRUD
│   │   ├── load.go              # Load queries
│   │   ├── capacity.go          # Capacity queries
│   │   └── group.go             # Group membership
│   ├── service/                 # Business logic
│   │   ├── heatmap.go           # Heatmap calculations
│   │   ├── load.go              # Load management
│   │   ├── auth.go              # OTP & sessions
│   │   ├── capacity.go          # Capacity updates
│   │   └── webhook.go           # Overload alerts
│   ├── handler/                 # HTTP handlers
│   │   ├── heatmap.go           # Heatmap UI & API
│   │   ├── api.go               # n8n integration
│   │   ├── auth.go              # Login flow
│   │   └── capacity.go          # Capacity form
│   └── middleware/              # HTTP middleware
│       ├── apikey.go            # API key validation
│       └── session.go           # Session auth
├── templates/                   # HTML templates
│   ├── base.html
│   ├── heatmap.html
│   ├── login.html
│   ├── capacity_form.html
│   └── partials/
├── static/css/                  # Stylesheets
├── Makefile                     # Build commands
├── go.mod / go.sum              # Dependencies
└── .env.example                 # Environment template
```

## Quick Start

```bash
# 1. Setup environment and database
make init

# 2. Run development server (with hot-reload)
make dev

# 3. Open http://localhost:8080
```

## Environment Variables

Copy `.env.example` to `.env` and configure:

| Variable | Required | Description |
|----------|----------|-------------|
| `DATABASE_URL` | Yes | PostgreSQL connection string |
| `API_KEY` | Yes | Secret for n8n API endpoints |
| `SESSION_SECRET` | Yes | Secret for session tokens (32+ bytes) |
| `MAILGUN_API_KEY` | No | Mailgun API key for OTP emails |
| `MAILGUN_DOMAIN` | No | Mailgun sending domain |
| `WEBHOOK_DESTINATION_URL` | No | n8n webhook for overload alerts |
| `PORT` | No | HTTP port (default: 8080) |

## Make Commands

| Command | Description |
|---------|-------------|
| `make build` | Compile to `bin/server` |
| `make run` | Build and execute |
| `make dev` | Hot-reload development |
| `make test` | Run test suite |
| `make docker-up` | Start PostgreSQL container |
| `make docker-down` | Stop PostgreSQL container |
| `make init` | Full setup (env + docker) |
| `make fmt` | Format code |
| `make lint` | Run linter |

## Core Concepts

### Entities
- **Person:** Individual with email, title, default capacity
- **Group:** Collection of persons (load = sum of member loads)

### Loads
- Tasks/work items with title, date, source
- Assigned to persons with weights (e.g., 0.5 = half day, 2.0 = two days effort)

### Heatmap Colors
| Load % | Color | Meaning |
|--------|-------|---------|
| < 20% | Green | Light load |
| 20-40% | Lime | Comfortable |
| 40-60% | Amber | Moderate |
| 60-80% | Orange | Heavy |
| 80-100% | Red | Near capacity |
| > 100% | Blood Red | OVERLOAD |

## API Endpoints

### Public
- `GET /` - Heatmap UI
- `GET /login` - Login page
- `POST /auth/request-otp` - Send OTP email
- `POST /auth/verify-otp` - Verify OTP
- `GET /api/entities` - List entities
- `GET /api/heatmap/:entity` - Heatmap data (JSON)

### Protected (Session Required)
- `GET /my-capacity` - Capacity management UI
- `POST /api/my-capacity` - Update own capacity

### Protected (API Key Required)
- `POST /api/loads/upsert` - Create/update load
- `POST /api/entities` - Create entity
- `DELETE /api/entities/:id` - Delete entity
- `POST /api/groups/:id/members` - Add group member

## Sample API Requests

```bash
# Upsert a load
curl -X POST http://localhost:8080/api/loads/upsert \
  -H "x-api-key: YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "external_id": "task-123",
    "title": "Sprint Planning",
    "source": "n8n",
    "date": "2026-01-20",
    "assignees": [
      {"email": "alice@example.com", "weight": 2.0}
    ]
  }'

# List persons
curl http://localhost:8080/api/entities?type=person
```

---

## LLM Verification Checklist

Use this section to verify the project is in a working state.

### 1. File Structure Verification

Required files that MUST exist:

```
cmd/server/main.go
internal/config/config.go
internal/database/postgres.go
internal/database/migrations.go
internal/models/models.go
internal/repository/entity.go
internal/repository/load.go
internal/repository/capacity.go
internal/repository/group.go
internal/service/heatmap.go
internal/service/load.go
internal/service/auth.go
internal/service/capacity.go
internal/service/webhook.go
internal/handler/heatmap.go
internal/handler/api.go
internal/handler/auth.go
internal/handler/capacity.go
internal/middleware/apikey.go
internal/middleware/session.go
templates/base.html
templates/heatmap.html
templates/login.html
templates/capacity_form.html
templates/partials/heatmap_grid.html
templates/partials/day_tasks.html
templates/partials/otp_form.html
go.mod
go.sum
Makefile
.env.example
```

### 2. Dependency Verification

Run: `go mod verify`

Expected: `all modules verified`

Required dependencies in go.mod:
- `github.com/labstack/echo/v4`
- `github.com/jackc/pgx/v5`
- `github.com/go-playground/validator/v10`
- `github.com/google/uuid`
- `github.com/joho/godotenv`
- `github.com/mailgun/mailgun-go/v4`

### 3. Build Verification

Run: `go build -o bin/server ./cmd/server`

Expected: No errors, binary created at `bin/server`

### 4. Database Schema Verification

Required tables (check in migrations.go):
- `entities` (id, title, type, default_capacity, created_at)
- `group_members` (group_id, person_email)
- `loads` (id, external_id, title, source, date, created_at)
- `load_assignments` (id, load_id, person_email, weight)
- `capacity_overrides` (id, entity_id, date, capacity)
- `otp_records` (id, email, otp, expires_at, created_at)
- `sessions` (id, token, email, expires_at, created_at)

Required indexes:
- `idx_loads_date`
- `idx_loads_external_id`
- `idx_load_assignments_person`
- `idx_capacity_overrides_date`
- `idx_sessions_email`

### 5. Configuration Verification

`.env.example` must contain:
```
DATABASE_URL=postgres://user:password@localhost:5432/heatmap?sslmode=disable
API_KEY=dev-api-key-change-in-production
SESSION_SECRET=dev-session-secret-change-in-production-min-32-bytes
MAILGUN_API_KEY=
MAILGUN_DOMAIN=
WEBHOOK_DESTINATION_URL=
PORT=8080
```

### 6. Route Verification

Verify these routes are registered in main.go or handler setup:

| Method | Path | Handler |
|--------|------|---------|
| GET | / | heatmapHandler.Index |
| GET | /login | authHandler.LoginPage |
| POST | /auth/request-otp | authHandler.RequestOTP |
| POST | /auth/verify-otp | authHandler.VerifyOTP |
| POST | /auth/logout | authHandler.Logout |
| GET | /my-capacity | capacityHandler.CapacityPage |
| POST | /api/my-capacity | capacityHandler.UpdateCapacity |
| GET | /api/entities | apiHandler.ListEntities |
| GET | /api/entities/:id | apiHandler.GetEntity |
| POST | /api/entities | apiHandler.CreateEntity |
| DELETE | /api/entities/:id | apiHandler.DeleteEntity |
| GET | /api/heatmap/:entity | heatmapHandler.GetHeatmapData |
| GET | /api/heatmap/:entity/day/:date | heatmapHandler.GetDayTasks |
| POST | /api/loads/upsert | apiHandler.UpsertLoad |
| GET | /api/groups/:id/members | apiHandler.ListGroupMembers |
| POST | /api/groups/:id/members | apiHandler.AddGroupMember |
| DELETE | /api/groups/:id/members/:member | apiHandler.RemoveGroupMember |

### 7. Template Verification

Each template must:
- `base.html`: Define `{{define "base"}}` with `{{template "content" .}}`
- `heatmap.html`: Define `{{define "content"}}` with heatmap grid
- `login.html`: Have form posting to `/auth/request-otp`
- `capacity_form.html`: Have form posting to `/api/my-capacity`
- Partials: Use HTMX attributes (`hx-get`, `hx-post`, `hx-target`, `hx-swap`)

### 8. Service Logic Verification

**Heatmap Service (internal/service/heatmap.go):**
- Must calculate load percentage: `(totalLoad / capacity) * 100`
- Must return color based on percentage thresholds
- Must handle 90-day date range

**Auth Service (internal/service/auth.go):**
- OTP generation: 6 random digits
- OTP expiry: 10 minutes
- Session expiry: 7 days
- Session token: UUID format

**Load Service (internal/service/load.go):**
- Upsert by external_id (update if exists, create if not)
- Trigger webhook on overload (load > capacity for future dates)

### 9. Runtime Verification

Start the server and verify:

```bash
# 1. Server starts without errors
make run

# 2. Health check (if implemented) or index page loads
curl http://localhost:8080/

# 3. API returns entities
curl http://localhost:8080/api/entities

# 4. Protected endpoint rejects without API key
curl -X POST http://localhost:8080/api/loads/upsert
# Expected: 401 Unauthorized

# 5. Protected endpoint accepts with API key
curl -X POST http://localhost:8080/api/loads/upsert \
  -H "x-api-key: dev-api-key-change-in-production" \
  -H "Content-Type: application/json" \
  -d '{"external_id":"test","title":"Test","source":"test","date":"2026-01-20","assignees":[]}'
# Expected: 200 OK
```

---

## Common Issues & Fixes

### Issue: `go build` fails with missing dependencies
**Fix:** Run `go mod tidy` then `go mod download`

### Issue: Database connection fails
**Fix:**
1. Ensure PostgreSQL is running: `make docker-up`
2. Verify DATABASE_URL in `.env`
3. Check connection: `psql $DATABASE_URL -c "SELECT 1"`

### Issue: Templates not found
**Fix:** Ensure templates directory is in working directory when running server. Templates are loaded relative to execution path.

### Issue: API returns 401 Unauthorized
**Fix:**
- For session-protected routes: Login first via `/login`
- For API-key routes: Add header `x-api-key: YOUR_API_KEY`

### Issue: OTP emails not sending
**Fix:** Configure MAILGUN_API_KEY and MAILGUN_DOMAIN in `.env`. In development, OTP is logged to console if Mailgun is not configured.

### Issue: Webhook not triggering on overload
**Fix:**
1. Set WEBHOOK_DESTINATION_URL in `.env`
2. Webhooks only trigger for future dates
3. Load must exceed entity's capacity

### Issue: Heatmap shows no data
**Fix:**
1. Verify entities exist: `curl http://localhost:8080/api/entities`
2. Verify loads exist in database
3. Check date range (heatmap shows 90 days from today)

### Issue: Missing tables after startup
**Fix:** Migrations run automatically. Check server logs for migration errors. Verify DATABASE_URL has correct permissions.

---

## Production Deployment Checklist

- [ ] Set strong `SESSION_SECRET` (32+ random bytes)
- [ ] Set unique `API_KEY` (not the dev default)
- [ ] Use production PostgreSQL with SSL
- [ ] Configure HTTPS and update cookie Secure flag
- [ ] Set up Mailgun for OTP emails
- [ ] Configure webhook destination URL
- [ ] Enable database backups
- [ ] Set up monitoring/alerting
- [ ] Review CORS settings for production domains
