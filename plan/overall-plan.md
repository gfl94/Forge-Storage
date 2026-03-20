Problem

Build a storage service that runs on a low-spec Ubuntu 20.04 machine with about 2 CPU cores and 2 GB RAM, exposes a browser-based file view and simple storage APIs, supports upload/download and file/photo viewing, and scales storage beyond the local machine by using external object storage.

Constraints and assumptions

- The Ubuntu host already runs Nginx and should continue to use it as the reverse proxy
- The Ubuntu host should not be the primary file store
- Azure Blob Storage is the first backend provider
- AWS S3 and Aliyun OSS are planned as future backup/alternate providers
- SQLite is acceptable for MVP metadata and auth state, but the design should not lock the service to SQLite forever
- Client-side encryption is explicitly out of MVP scope and should be treated as a later enhancement

Proposed approach

Use the Ubuntu machine as a thin control plane and web gateway.

- Frontend: lightweight static web frontend served by Nginx and talking to the backend over JSON APIs; keep it simple now and compatible with a future React UI
- Backend: Go service that handles auth, listing, metadata lookup, file metadata caching, signed URL issuance, health endpoints, and policy checks
- Reverse proxy: existing Nginx instance
- External storage: Azure Blob Storage first, with an internal provider interface for later S3 and OSS implementations
- Metadata database: SQLite first, with a repository abstraction for future migration and file metadata cache as a core feature

Architectural direction

1. The browser talks to the Go backend for login, folder listing, metadata reads, and obtaining upload/download/view URLs.
2. The Go backend talks to the storage provider and the metadata repository.
3. Large file transfers go directly between the browser and the object storage provider whenever possible.
4. The backend remains responsible for permissions, path normalization, metadata composition, and provider-specific translation.

Suggested MVP stack

- Backend language: Go
- HTTP stack: `chi`
- Reverse proxy / TLS: existing Nginx instance
- Frontend: static assets served by Nginx; start with plain TypeScript/JavaScript or htmx-style enhancement, keep React as a later UI evolution
- Storage provider implementation: Azure Blob SDK for Go
- Metadata database: SQLite with a core file metadata cache
- Deployment: systemd service on Ubuntu

Important design decisions

- Define a `StorageProvider` interface from day one so the API/service layer is not tied to Azure-specific code
- Define a `MetadataRepository` interface from day one so SQLite can later be replaced by PostgreSQL or another database
- Represent folders as normalized virtual paths rather than relying on a provider-native directory model
- Keep the Ubuntu host out of the hot path for large uploads/downloads
- Do not let future encryption requirements complicate the first MVP; treat encryption as a later extension
- Keep the frontend API-first so a future React application can replace the initial UI without changing backend contracts
- Add health endpoints, request IDs, and structured logging from the beginning

MVP capabilities

- Login/logout
- Folder browsing
- File metadata display
- File metadata cache backed by SQLite
- Image preview
- Browser-native preview for supported text/PDF files
- Upload
- Download
- Signed URL generation
- Health endpoint and structured operational logs

Phases

Phase 1: confirm scope and finalize interfaces
- single-user vs multi-user
- storage path model
- provider abstraction
- metadata repository abstraction

Phase 2: build MVP
- backend APIs
- lightweight frontend
- Azure provider
- SQLite repository
- file metadata cache
- Nginx + systemd deployment

Phase 3: hardening
- pagination
- rate limiting
- audit logging
- caching
- thumbnail generation and LRU thumbnail cache if image browsing proves important
- resumable upload if needed
- optional client-side encryption mode with encrypted object metadata handling and adjusted preview rules

Phase 4: extension
- AWS S3 provider
- Aliyun OSS provider
- stronger metadata database
- richer frontend only if needed
- optional client-side encryption mode later

Initial recommendation

Start with Go/chi + existing Nginx + Azure Blob + SQLite metadata cache + static API-driven frontend, but make both the storage backend and metadata backend replaceable behind interfaces.
