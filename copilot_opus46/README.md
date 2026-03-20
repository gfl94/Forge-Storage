# Forge Storage Gateway

A lightweight storage gateway service that runs on a small Ubuntu host, exposes a browser-based file manager, and uses Azure Blob Storage for scalable file storage. The architecture keeps the host out of the data-transfer hot path by using signed URLs for direct browser-to-cloud transfers.

## Architecture Overview

```
┌──────────┐     ┌─────────────────────────────────────────────────┐
│  Browser  │     │                Ubuntu Host                      │
│           │────▶│  Nginx ──▶ Static Frontend (HTML/CSS/JS)       │
│           │     │        ──▶ Go API Service (port 8081)          │
│           │     │              │                                   │
│           │     │              ├── SQLite (auth/sessions/cache)   │
│           │     │              └── StorageProvider interface      │
└─────┬─────┘     └─────────────────────────────────────────────────┘
      │                                    │
      │        Signed upload/download      │
      └──────────────────────────────────▶ Azure Blob Storage
```

**Key design decisions:**
- **Direct transfers**: Large files go directly between the browser and Azure Blob Storage via signed URLs. The Go service only handles control-plane operations.
- **Provider abstraction**: A `StorageProvider` interface isolates Azure-specific code, making it straightforward to add AWS S3 or Aliyun OSS later.
- **Repository abstraction**: All database access goes through repository interfaces, so SQLite can be replaced with PostgreSQL if needed.
- **Metadata caching**: Directory listings and object metadata are cached in SQLite with configurable TTLs to reduce provider round-trips.

## Project Structure

```
copilot_opus46/
├── cmd/
│   └── server/
│       └── main.go                 # Application entry point
├── internal/
│   ├── auth/
│   │   ├── auth.go                 # Password hashing (bcrypt) and ID generation
│   │   └── auth_test.go
│   ├── config/
│   │   ├── config.go               # Environment-based configuration
│   │   └── config_test.go
│   ├── http/
│   │   ├── handler/
│   │   │   ├── auth.go             # POST /api/login, POST /api/logout, GET /api/session
│   │   │   ├── browse.go           # GET /api/list, GET /api/meta
│   │   │   ├── health.go           # GET /health, GET /ready
│   │   │   ├── response.go         # JSON response helpers
│   │   │   └── transfer.go         # POST /api/upload-url, POST /api/download-url, GET /api/preview
│   │   ├── middleware/
│   │   │   ├── auth.go             # Session cookie authentication middleware
│   │   │   └── middleware.go       # Request ID, logging, security headers, CORS
│   │   └── router.go              # chi router with route registration
│   ├── model/
│   │   └── model.go               # Domain types (User, Session, Entry, ObjectInfo, etc.)
│   ├── observability/
│   │   └── logger.go              # Structured JSON logger (slog)
│   ├── repository/
│   │   ├── repository.go          # Repository interfaces
│   │   └── sqlite/
│   │       ├── sqlite.go          # SQLite implementation with auto-migration
│   │       └── sqlite_test.go
│   ├── service/
│   │   ├── errors.go              # Sentinel errors
│   │   ├── service.go             # Business logic (Auth, Browse, File, Upload, Download)
│   │   └── service_test.go
│   └── storage/
│       ├── provider.go            # StorageProvider interface
│       └── azure/
│           └── azure.go           # Azure Blob Storage implementation
├── frontend/
│   ├── index.html                 # Single-page application shell
│   ├── styles.css                 # Responsive CSS
│   ├── api.js                     # Backend API client
│   ├── auth.js                    # Login/logout/session state
│   ├── browser.js                 # File listing, breadcrumbs, preview, download
│   ├── upload.js                  # File upload with progress tracking
│   └── app.js                    # Application initialization
├── deploy/
│   ├── nginx-storage.conf         # Nginx server block configuration
│   ├── storage-api.service        # systemd unit file
│   ├── storage-api.env.example    # Example environment configuration
│   └── install.sh                 # Automated installation script
├── go.mod
├── go.sum
└── README.md
```

## API Endpoints

| Method | Path               | Auth | Description                          |
|--------|--------------------|------|--------------------------------------|
| POST   | `/api/login`       | No   | Authenticate and create session      |
| POST   | `/api/logout`      | Yes  | End session                          |
| GET    | `/api/session`     | Yes  | Check current session status         |
| GET    | `/api/list`        | Yes  | List directory contents              |
| GET    | `/api/meta`        | Yes  | Get file metadata                    |
| POST   | `/api/upload-url`  | Yes  | Generate signed upload URL           |
| POST   | `/api/download-url`| Yes  | Generate signed download URL         |
| GET    | `/api/preview`     | Yes  | Generate signed preview URL          |
| GET    | `/health`          | No   | Liveness check                       |
| GET    | `/ready`           | No   | Readiness check (SQLite + provider)  |

### Error Format

All errors return a consistent JSON envelope:

```json
{
  "error": {
    "code": "invalid_path",
    "message": "Path must start with /"
  }
}
```

Error codes: `unauthorized`, `forbidden`, `invalid_path`, `not_found`, `provider_error`, `database_error`, `rate_limited`, `validation_error`.

## Prerequisites

- **Go 1.21+** (for building the backend)
- **Ubuntu 20.04** (or compatible Linux)
- **Nginx** (already installed on the host)
- **Azure Blob Storage** account with a container created
- **GCC** (required by go-sqlite3 CGO dependency)

## Quick Start (Development)

1. **Clone and build:**

```bash
cd copilot_opus46
go build -o storage-api ./cmd/server
```

2. **Configure environment:**

```bash
export APP_ENV=development
export APP_ADDR=127.0.0.1:8081
export SQLITE_PATH=./storage.db
export AZURE_STORAGE_ACCOUNT=your_account
export AZURE_STORAGE_CONTAINER=files
export AZURE_STORAGE_KEY=your_key
export ADMIN_PASSWORD=your_secure_password
```

3. **Run:**

```bash
./storage-api
```

4. **Open the frontend:**

Serve the `frontend/` directory with any static file server, or configure Nginx to point to it. For quick development:

```bash
# In another terminal
cd frontend
python3 -m http.server 8080
```

Then visit `http://localhost:8080` and login with `admin` / `your_secure_password`.

## Production Deployment

### Automated Install

```bash
cd copilot_opus46
go build -o storage-api ./cmd/server
sudo bash deploy/install.sh
```

### Manual Steps

1. **Build the binary:**

```bash
cd copilot_opus46
CGO_ENABLED=1 go build -o storage-api ./cmd/server
```

2. **Install files:**

```bash
sudo cp storage-api /opt/storage/bin/
sudo cp frontend/* /var/www/storage-ui/
sudo cp deploy/storage-api.service /etc/systemd/system/
sudo cp deploy/storage-api.env.example /etc/storage-api/storage-api.env
sudo chmod 600 /etc/storage-api/storage-api.env
```

3. **Configure:**

Edit `/etc/storage-api/storage-api.env` with your Azure credentials:

```
AZURE_STORAGE_ACCOUNT=your_account_name
AZURE_STORAGE_CONTAINER=your_container
AZURE_STORAGE_KEY=your_account_key
ADMIN_PASSWORD=a_strong_password
SESSION_COOKIE_SECURE=true
```

4. **Set up Nginx:**

```bash
sudo cp deploy/nginx-storage.conf /etc/nginx/sites-available/storage
sudo ln -sf /etc/nginx/sites-available/storage /etc/nginx/sites-enabled/
# Edit server_name and TLS certificate paths
sudo nginx -t && sudo systemctl reload nginx
```

5. **Start the service:**

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now storage-api
sudo journalctl -u storage-api -f
```

## Configuration Reference

| Variable                          | Default              | Description                                     |
|-----------------------------------|----------------------|-------------------------------------------------|
| `APP_ENV`                         | `development`        | Environment (`production` or `development`)     |
| `APP_ADDR`                        | `127.0.0.1:8081`     | Listen address                                  |
| `APP_BASE_URL`                    | `http://localhost:8081` | External base URL                            |
| `SESSION_COOKIE_NAME`             | `storage_session`    | Session cookie name                             |
| `SESSION_COOKIE_SECURE`           | `false`              | Secure cookie flag (set `true` in production)   |
| `SESSION_TTL_HOURS`               | `168` (7 days)       | Session lifetime in hours                       |
| `SQLITE_PATH`                     | `storage.db`         | SQLite database file path                       |
| `METADATA_CACHE_DIR_TTL_SECONDS`  | `60`                 | Directory listing cache TTL                     |
| `METADATA_CACHE_OBJECT_TTL_SECONDS`| `120`               | Object metadata cache TTL                       |
| `STORAGE_PROVIDER`                | `azure`              | Storage backend (`azure`)                       |
| `AZURE_STORAGE_ACCOUNT`           | (required)           | Azure Storage account name                      |
| `AZURE_STORAGE_CONTAINER`         | `files`              | Azure Blob container name                       |
| `AZURE_STORAGE_KEY`               | (required)           | Azure Storage account key                       |
| `SIGNED_URL_TTL_SECONDS`          | `300` (5 min)        | Signed URL expiry                               |
| `LOG_LEVEL`                       | `info`               | Log level (`debug`, `info`, `warn`, `error`)    |
| `ADMIN_PASSWORD`                  | `changeme`           | Default admin password (first run only)         |

## Database Schema

SQLite is used as the metadata store with automatic schema migration on startup. Tables:

- **`users`** – User accounts with bcrypt password hashes
- **`sessions`** – Active sessions with expiry tracking
- **`directory_cache`** – Cached directory listings (JSON payloads with TTL)
- **`object_cache`** – Cached object metadata (with TTL)
- **`audit_events`** – Optional audit trail (table created, ready for future use)

## Frontend Features

The lightweight static frontend provides:

- **Login screen** with form validation
- **File browser** with sortable table view
- **Breadcrumb navigation** for folder hierarchy
- **File metadata** display (size, type, modified date)
- **Image preview** in a modal overlay
- **PDF/text preview** via iframe
- **File upload** with progress bar and multi-file support
- **File download** via signed URLs
- **Responsive design** for mobile and desktop

## Security

- Session cookies are `HttpOnly` with `SameSite=Lax`
- Passwords hashed with bcrypt (cost 12)
- Signed URLs are short-lived (5 minutes default)
- Security headers: `X-Content-Type-Options`, `X-Frame-Options`, `Referrer-Policy`
- Path traversal protection (`..\` and `..` rejected)
- Provider credentials never exposed to the frontend
- systemd service runs as non-root with `NoNewPrivileges`, `PrivateTmp`, `ProtectSystem`

## Running Tests

```bash
cd copilot_opus46
go test ./...
```

## Future Extensions

Per the design documents, the following are planned for future phases:

- **AWS S3 provider** – Implement `storage.Provider` for S3
- **Aliyun OSS provider** – Implement `storage.Provider` for OSS
- **PostgreSQL repository** – Replace SQLite for higher concurrency
- **React frontend** – Replace the static frontend with a modern SPA
- **Resumable uploads** – Multipart upload support for large files
- **Audit logging** – Record all user actions (schema already in place)
- **Thumbnail generation** – On-demand thumbnails with LRU cache
- **Rate limiting** – Login endpoint rate limiting (Nginx config included)
- **Client-side encryption** – Optional browser-side encryption mode

## License

This project is part of the Forge-Storage system.
