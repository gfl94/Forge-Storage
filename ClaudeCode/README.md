# Storage Gateway - Cloud Storage Management System

A lightweight, high-performance storage gateway service that provides a web-based interface for managing files in cloud object storage (Azure Blob Storage, with planned support for AWS S3 and Aliyun OSS).

## Overview

This project implements a thin control-plane gateway that runs on resource-constrained hardware while leveraging cloud object storage for bulk data. It follows the detailed system design specifications in the `plan/` directory.

### Key Features

- **Web-based File Browser**: Clean, intuitive interface for browsing and managing files
- **Direct Upload/Download**: Files transfer directly between browser and cloud storage, bypassing the server
- **Smart Caching**: SQLite-based metadata caching for fast browsing
- **Azure Blob Storage**: First-class support with signed URL generation
- **Lightweight**: Designed for 2-core, 2GB RAM Ubuntu machines
- **Secure**: Session-based authentication, secure cookies, path validation
- **Production-Ready**: Systemd integration, structured logging, health checks

## Architecture

```
┌─────────┐       ┌────────┐       ┌──────────┐       ┌───────────┐
│ Browser │ ───── │ Nginx  │ ───── │ Go API   │ ───── │  SQLite   │
└─────────┘       └────────┘       └──────────┘       └───────────┘
     │                                   │
     │                                   │
     └───────── Direct Transfer ─────────┴──────────► Azure Blob
```

### Components

1. **Frontend**: Static HTML/CSS/JavaScript served by Nginx
2. **Backend**: Go service with chi router
3. **Reverse Proxy**: Nginx for TLS termination and static assets
4. **Metadata Store**: SQLite for sessions, users, and cache
5. **Object Storage**: Azure Blob Storage (S3 and OSS planned)

## Technology Stack

### Backend
- **Language**: Go 1.21+
- **HTTP Framework**: chi v5
- **Database**: SQLite 3
- **Cloud SDK**: Azure SDK for Go
- **Auth**: bcrypt password hashing, session cookies

### Frontend
- **HTML/CSS/JavaScript**: Vanilla JS, no framework dependencies
- **API Client**: Fetch API with async/await

### Infrastructure
- **Deployment**: systemd service
- **Reverse Proxy**: Nginx
- **Platform**: Ubuntu 20.04 (or compatible Linux)

## Project Structure

```
ClaudeCode/
├── cmd/
│   └── server/          # Main application entry point
├── internal/
│   ├── auth/            # Authentication and session management
│   ├── config/          # Configuration loading
│   ├── http/            # HTTP handlers and routes
│   │   └── middleware/  # HTTP middleware
│   ├── model/           # Data models
│   ├── observability/   # Logging
│   ├── repository/      # Data persistence interfaces
│   │   └── sqlite/      # SQLite implementation
│   ├── service/         # Business logic layer
│   └── storage/         # Storage provider abstraction
│       └── azure/       # Azure Blob implementation
├── frontend/            # Static web UI
│   ├── index.html       # Main application page
│   ├── login.html       # Login page
│   ├── styles.css       # Styles
│   ├── api.js           # API client
│   ├── app.js           # Main app logic
│   ├── auth.js          # Authentication handler
│   ├── browser.js       # File browser
│   └── upload.js        # Upload handler
├── deployments/         # Deployment configurations
│   ├── deploy.sh        # Deployment script
│   ├── storage-api.service        # Systemd unit
│   ├── storage-api.env.example    # Config template
│   └── nginx.conf.example         # Nginx config
├── go.mod               # Go dependencies
└── README.md            # This file
```

## Prerequisites

- Ubuntu 20.04 or compatible Linux distribution
- Go 1.21 or later
- Nginx (already installed and configured)
- Azure Storage Account with a container
- Root or sudo access for deployment

## Quick Start

### 1. Clone the Repository

```bash
cd /path/to/Forge-Storage
cd ClaudeCode
```

### 2. Set Up Azure Storage

1. Create an Azure Storage Account
2. Create a blob container
3. Note down:
   - Storage account name
   - Container name
   - Storage account key (access key)

### 3. Configure the Application

Copy the example configuration:

```bash
cp deployments/storage-api.env.example deployments/storage-api.env
nano deployments/storage-api.env
```

Update these critical fields:

```bash
AZURE_STORAGE_ACCOUNT=your_account_name
AZURE_STORAGE_CONTAINER=your_container_name
AZURE_STORAGE_KEY=your_access_key
```

### 4. Build and Deploy

Run the deployment script:

```bash
sudo ./deployments/deploy.sh
```

This script will:
- Create the `storage-api` user
- Build the Go application
- Install files to `/opt/storage`
- Copy frontend to `/var/www/storage-ui`
- Set up systemd service

### 5. Start the Service

```bash
sudo systemctl start storage-api
sudo systemctl enable storage-api
sudo systemctl status storage-api
```

View logs:

```bash
sudo journalctl -u storage-api -f
```

### 6. Configure Nginx

Copy and edit the Nginx configuration:

```bash
sudo cp deployments/nginx.conf.example /etc/nginx/sites-available/storage-api
sudo nano /etc/nginx/sites-available/storage-api
```

Update:
- `server_name` with your domain
- SSL certificate paths
- Any custom settings

Enable and test:

```bash
sudo ln -s /etc/nginx/sites-available/storage-api /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl reload nginx
```

### 7. Access the Application

Open your browser and navigate to:
- `https://your-domain.com`

Default credentials:
- **Username**: `admin`
- **Password**: `admin123`

**⚠️ IMPORTANT**: Change the default password immediately in production!

## Development

### Local Development

Run the application locally:

```bash
# Set environment variables
export SQLITE_PATH=./storage.db
export AZURE_STORAGE_ACCOUNT=your_account
export AZURE_STORAGE_CONTAINER=your_container
export AZURE_STORAGE_KEY=your_key
export STORAGE_PROVIDER=azure
export APP_ADDR=127.0.0.1:8081

# Build and run
go build -o storage-api ./cmd/server
./storage-api
```

For frontend development, serve the frontend directory with any static file server:

```bash
cd frontend
python3 -m http.server 8080
```

### Running Tests

```bash
go test ./...
```

### Building

```bash
go build -o storage-api ./cmd/server
```

## Configuration Reference

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `APP_ENV` | `production` | Application environment |
| `APP_ADDR` | `127.0.0.1:8081` | Listen address |
| `APP_BASE_URL` | `http://localhost` | Base URL for the app |
| `SESSION_COOKIE_NAME` | `storage_session` | Session cookie name |
| `SESSION_COOKIE_SECURE` | `false` | Use secure cookies (HTTPS only) |
| `SESSION_TTL_HOURS` | `168` | Session lifetime (1 week) |
| `SQLITE_PATH` | `./storage.db` | SQLite database path |
| `METADATA_CACHE_DIR_TTL_SECONDS` | `60` | Directory listing cache TTL |
| `METADATA_CACHE_OBJECT_TTL_SECONDS` | `120` | Object metadata cache TTL |
| `STORAGE_PROVIDER` | `azure` | Storage provider (azure, s3, aliyun) |
| `SIGNED_URL_TTL_SECONDS` | `300` | Signed URL lifetime (5 min) |
| `AZURE_STORAGE_ACCOUNT` | - | Azure storage account name (required) |
| `AZURE_STORAGE_CONTAINER` | - | Azure container name (required) |
| `AZURE_STORAGE_KEY` | - | Azure storage access key (required) |
| `LOG_LEVEL` | `info` | Logging level (debug, info, warn, error) |

## API Documentation

### Authentication Endpoints

#### POST /api/login
Login with username and password.

**Request:**
```json
{
  "username": "admin",
  "password": "admin123"
}
```

**Response:**
```json
{
  "user": {
    "id": "u_1",
    "username": "admin"
  }
}
```

#### POST /api/logout
Logout current session.

**Response:**
```json
{
  "ok": true
}
```

#### GET /api/session
Get current session information.

**Response:**
```json
{
  "authenticated": true,
  "user": {
    "id": "u_1",
    "username": "admin"
  }
}
```

### File Management Endpoints

#### GET /api/list?path=/&limit=100
List files and directories.

**Response:**
```json
{
  "path": "/",
  "entries": [
    {
      "name": "photos",
      "path": "/photos",
      "type": "directory"
    },
    {
      "name": "document.pdf",
      "path": "/document.pdf",
      "type": "file",
      "size": 12345,
      "mimeType": "application/pdf",
      "modifiedAt": "2026-03-20T12:00:00Z",
      "previewable": true
    }
  ],
  "nextCursor": "",
  "source": "cache"
}
```

#### GET /api/meta?path=/document.pdf
Get metadata for a specific file.

**Response:**
```json
{
  "path": "/document.pdf",
  "type": "file",
  "size": 12345,
  "mimeType": "application/pdf",
  "modifiedAt": "2026-03-20T12:00:00Z",
  "etag": "abc123",
  "previewable": true
}
```

#### POST /api/upload-url
Generate signed URL for file upload.

**Request:**
```json
{
  "path": "/document.pdf",
  "size": 12345,
  "mimeType": "application/pdf"
}
```

**Response:**
```json
{
  "provider": "azure",
  "method": "PUT",
  "url": "https://account.blob.core.windows.net/...",
  "headers": {
    "x-ms-blob-type": "BlockBlob",
    "Content-Type": "application/pdf"
  },
  "expiresAt": "2026-03-20T12:10:00Z"
}
```

#### POST /api/download-url
Generate signed URL for file download.

**Request:**
```json
{
  "path": "/document.pdf"
}
```

**Response:**
```json
{
  "url": "https://account.blob.core.windows.net/...",
  "expiresAt": "2026-03-20T12:10:00Z"
}
```

#### GET /api/preview?path=/document.pdf
Get preview information for a file.

**Response:**
```json
{
  "path": "/document.pdf",
  "mode": "redirect",
  "url": "https://account.blob.core.windows.net/...",
  "expiresAt": "2026-03-20T12:10:00Z"
}
```

### Health Endpoints

#### GET /health
Basic health check.

**Response:**
```json
{
  "ok": true
}
```

#### GET /ready
Readiness check with dependency status.

**Response:**
```json
{
  "ok": true,
  "checks": {
    "sqlite": "ok",
    "storageProvider": "ok"
  }
}
```

## Security Considerations

1. **Change Default Credentials**: The default admin password must be changed
2. **Use HTTPS**: Always use TLS in production
3. **Secure Cookies**: Enable `SESSION_COOKIE_SECURE=true` for HTTPS
4. **Access Keys**: Keep Azure storage keys secure and rotate regularly
5. **Rate Limiting**: Consider adding rate limiting for auth endpoints
6. **Path Validation**: Path traversal attempts are blocked
7. **Short-lived URLs**: Signed URLs expire in 5 minutes by default

## Operational Guidance

### Monitoring

Check service status:
```bash
sudo systemctl status storage-api
```

View recent logs:
```bash
sudo journalctl -u storage-api -n 100
```

Follow logs in real-time:
```bash
sudo journalctl -u storage-api -f
```

### Backup

The SQLite database contains sessions and cache. To backup:

```bash
sudo sqlite3 /var/lib/storage-api/storage.db ".backup '/path/to/backup.db'"
```

### Performance Tuning

1. **Cache TTL**: Adjust `METADATA_CACHE_*_TTL_SECONDS` based on your usage
2. **Signed URL TTL**: Increase for larger file transfers
3. **Session TTL**: Adjust based on security requirements

### Troubleshooting

**Service won't start:**
```bash
sudo journalctl -u storage-api -n 50
```

**Connection refused:**
- Check if service is running
- Verify `APP_ADDR` in config
- Check firewall rules

**Upload/download fails:**
- Verify Azure credentials
- Check signed URL expiration
- Verify network connectivity to Azure

**Cache issues:**
- Clear cache by deleting database: `sudo rm /var/lib/storage-api/storage.db`
- Service will recreate on next start

## Future Enhancements

- [ ] AWS S3 provider implementation
- [ ] Aliyun OSS provider implementation
- [ ] React-based frontend
- [ ] Thumbnail generation for images
- [ ] Full-text search
- [ ] Resumable uploads for large files
- [ ] Optional client-side encryption
- [ ] Multi-user support with permissions
- [ ] Audit logging

## Design Documentation

For detailed system design information, see the `plan/` directory:
- `detailed-system-design.md`: Comprehensive implementation guide
- `mvp-architecture-spec.md`: Architecture decisions and rationale
- `overall-plan.md`: High-level project overview

## License

This project is part of the Forge-Storage repository.

## Support

For issues, questions, or contributions, please refer to the main Forge-Storage repository.

## Acknowledgments

Built following industry best practices for:
- Cloud-native architecture
- Separation of concerns
- Provider abstraction
- Operational excellence
