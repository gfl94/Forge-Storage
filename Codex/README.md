# Forge Storage (Codex)

Lightweight storage gateway built from the detailed system design: Go/chi backend with Azure Blob provider, SQLite-backed auth/session/metadata cache, and a static API-first frontend.

## Project layout

```
Codex/
  backend/   Go API service (chi, SQLite, Azure provider)
  frontend/  Vite + TypeScript static UI
```

## Backend

### Configuration

Environment variables (defaults in parentheses):

- `APP_ADDR` (`127.0.0.1:8081`) – listen address
- `APP_BASE_URL` (`http://127.0.0.1:8081`) – external base URL
- `SESSION_COOKIE_NAME` (`storage_session`)
- `SESSION_COOKIE_SECURE` (`false`)
- `SESSION_TTL_HOURS` (`24h`)
- `SQLITE_PATH` (`./storage.db`)
- `METADATA_CACHE_DIR_TTL_SECONDS` (`60s`)
- `METADATA_CACHE_OBJECT_TTL_SECONDS` (`120s`)
- `STORAGE_PROVIDER` (`azure`)
- `AZURE_STORAGE_ACCOUNT` (required)
- `AZURE_STORAGE_KEY` (required)
- `AZURE_STORAGE_CONTAINER` (required)
- `SIGNED_URL_TTL_SECONDS` (`5m`)
- `LOG_LEVEL` (`info`)
- `DEFAULT_ADMIN_USERNAME` (`admin`)
- `DEFAULT_ADMIN_PASSWORD` (required to seed admin on first boot)

### Running locally

```bash
cd Codex/backend
export AZURE_STORAGE_ACCOUNT=...
export AZURE_STORAGE_KEY=...
export AZURE_STORAGE_CONTAINER=...
export DEFAULT_ADMIN_PASSWORD=choose-a-strong-secret
go run ./cmd/server
```

The service automatically creates `storage.db` (SQLite), applies migrations, and seeds the admin user if `DEFAULT_ADMIN_PASSWORD` is set.

### API surface

- `POST /api/login` – issue session cookie
- `POST /api/logout` – drop session
- `GET /api/session` – session status
- `GET /api/list?path=/docs&limit=100&cursor=` – directory listing with cache
- `GET /api/meta?path=/docs/file.jpg` – metadata lookup with cache
- `POST /api/upload-url` – signed upload target (direct to Azure)
- `POST /api/download-url` – signed download target
- `GET /api/preview?path=/docs/file.jpg` – preview redirect target
- `GET /health` – liveness
- `GET /ready` – readiness (SQLite + provider probe)

### Internals

- `chi` router with request ID, logging, and auth middleware
- SQLite repositories for users, sessions, directory cache, object cache
- Cache TTL defaults: 60s (directories), 120s (objects)
- Azure provider uses signed URLs for uploads/downloads and prefix-based listings
- JSON error model: `{ "error": { "code": "...", "message": "..." } }`

## Frontend

Vite + TypeScript static UI; consumes the JSON APIs and assumes same-origin requests via Nginx.

### Run/build

```bash
cd Codex/frontend
npm install
npm run dev     # local dev server
npm run build   # production build to dist/
```

### Features

- Login/logout against backend
- Path-aware folder browser with size/modified columns
- Metadata panel with preview/download actions
- Direct upload flow using backend-issued signed URLs
- Status bar for errors/progress

## Deployment notes

- Serve `Codex/frontend/dist` via Nginx; proxy `/api/` to the Go service (`APP_ADDR`).
- Run the API as a `systemd` service; keep secrets in an env file (`/etc/storage-api/storage-api.env`).
- Ensure the SQLite path is writable by the service user.
- Use TLS termination at Nginx; set `SESSION_COOKIE_SECURE=true` in production.

## Health and testing

```bash
# Backend
cd Codex/backend && go test ./...

# Frontend
cd Codex/frontend && npm run build
```

Health checks: `/health` (process) and `/ready` (SQLite + provider).
