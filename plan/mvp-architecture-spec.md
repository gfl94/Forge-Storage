# MVP Architecture Specification

## 1. Objective

Build a lightweight storage gateway service that runs on a small Ubuntu 20.04 machine, exposes a web UI and simple APIs, stores file data in cloud object storage, and keeps the local host focused on control-plane work rather than bulk data transfer.

This document defines the MVP architecture only. It intentionally stops short of detailed component diagrams, API contracts, and deployment layouts until you review and approve the direction.

## 2. MVP goals

- Browse folders and files from a web browser
- Preview common file types, especially images
- Upload files to cloud storage
- Download files from cloud storage
- Keep CPU and memory usage low on the Ubuntu host
- Reuse the existing Nginx instance
- Keep storage provider choice open for future expansion
- Keep metadata database choice open for future expansion
- Keep the MVP focused on plaintext object workflows only

## 3. Non-goals for MVP

- Native desktop clients
- File sync client
- Full-text indexing/search across file contents
- Multi-region replication orchestration
- Distributed job system
- Rich collaborative editing
- Complex multi-tenant isolation
- Client-side or end-to-end file encryption

## 4. High-level architecture

The system has five logical parts:

1. Browser UI
2. Nginx edge
3. Go application service
4. Metadata database
5. Object storage provider

Nginx fronts the Go service and also serves static frontend assets. The Go service primarily serves JSON APIs. The Go service uses SQLite for auth/session/metadata state in the MVP, with file metadata cache treated as a core feature. File objects live in Azure Blob Storage first. The browser should transfer large file payloads directly to object storage using short-lived signed URLs issued by the Go service.

## 5. Why this shape fits the small Ubuntu host

The Ubuntu machine is resource-constrained, so the design must avoid turning it into a file relay.

- The Go service handles low-bandwidth control operations
- Nginx continues to do the edge work it already does today
- Object storage handles bulk upload/download traffic
- SQLite avoids another always-on service
- The frontend is lightweight and served as static assets by Nginx

This keeps RAM, CPU, and operational complexity low.

## 6. Frontend architecture

### 6.1 Approach

Use a lightweight static frontend served by Nginx, with the browser talking to the Go backend over JSON APIs.

This is preferred over server-rendered templates and over a heavy SPA for MVP because:

- it keeps backend/frontend boundaries clean
- static assets are cheap for Nginx to serve
- the product needs simple workflows first
- the team can delay React-level complexity until real requirements justify it
- the backend can evolve into a stable API platform for a future modern UI

### 6.2 Frontend responsibilities

- Render login page
- Render folder listing page
- Render file preview page
- Submit upload form and show progress
- Trigger downloads via signed URLs
- Show breadcrumbs and metadata
- Call JSON APIs for listing, metadata, and upload/download initiation
- Display errors returned by the backend

### 6.3 Frontend implementation style

- Static HTML/CSS/JS assets, ideally with a very small build output
- Plain TypeScript/JavaScript first, with htmx-style progressive enhancement as an acceptable middle ground if desired
- Keep the UI API-first so React can replace the first frontend later without changing backend contracts
- Avoid a heavy client-side state store for MVP

### 6.4 Frontend UX for MVP

- Login screen
- Main file browser view
- Breadcrumb navigation
- Folder entries and file entries in one list
- Metadata columns such as size and modified time
- Inline image preview
- Browser-native PDF/text preview where practical
- Upload action
- Download action

### 6.5 Future frontend evolution

The initial frontend should not lock the project away from modern UI frameworks.

Recommended direction:

- define stable JSON contracts now
- keep session/auth behavior backend-controlled
- let Nginx serve versioned static assets
- if the UI later grows more interactive, move to a React/Vite frontend without requiring a backend rewrite

This makes the MVP cheap while preserving a clean path to a richer future UI.

## 7. Backend architecture

### 7.1 Core service responsibilities

The Go backend is the control plane for the storage system.

Responsibilities:

- authenticate the user
- normalize and validate paths
- list folders/files
- retrieve and compose metadata
- maintain and serve file metadata cache entries
- generate upload/download/view signed URLs
- apply authorization and policy rules
- persist lightweight metadata and session state
- translate internal operations to provider-specific APIs
- expose health/readiness endpoints and structured operational logs

### 7.2 Backend design principles

- stateless service layer where practical
- explicit interfaces between app logic and infrastructure
- no direct provider SDK usage from handlers
- no direct SQL from handlers
- no large file streaming through the app unless absolutely necessary
- API-first contracts so frontend technology can change independently

### 7.3 Suggested internal layers

- HTTP layer: `chi` router, handlers, request parsing, auth/session extraction, health endpoints
- Service layer: application logic for browse, preview, upload, download
- Storage provider layer: object storage abstraction and provider implementations
- Metadata repository layer: database abstraction and concrete persistence implementation
- Static frontend served outside the app by Nginx

## 8. Storage provider abstraction

### 8.1 Why it is needed

You want Azure first, but AWS S3 and Aliyun OSS may be added later. If the code directly depends on Azure concepts everywhere, migration and extension will be costly. The MVP should therefore define a provider abstraction immediately.

### 8.2 Design goal

The app should operate in terms of normalized virtual paths and object metadata, not raw provider SDK types.

### 8.3 Proposed abstraction boundary

Define an internal `StorageProvider` interface that covers only what the MVP needs:

- list objects by normalized path/prefix
- get object metadata
- create signed upload request
- create signed download/view request
- delete object if needed later
- store/retrieve provider-specific object identifiers only inside the provider implementation

### 8.4 Provider implementations roadmap

- MVP: `AzureBlobProvider`
- Future: `S3Provider`
- Future: `AliyunOSSProvider`

### 8.5 Important provider design rules

- Keep path normalization outside provider-specific code
- Keep signed URL policy generation inside each provider implementation
- Keep provider config isolated
- Avoid leaking Azure-specific naming or SDK types above the provider layer

## 9. Metadata database abstraction

### 9.1 Why it is needed

SQLite is good for MVP, but future growth may require a stronger metadata database. The service should therefore hide persistence details behind a repository boundary from the start.

### 9.2 What metadata belongs in the database

MVP database contents should stay small and purposeful, but file metadata cache is a core feature rather than an afterthought:

- users
- password hashes
- sessions or auth tokens
- saved application settings
- cached file and directory metadata
- upload records or audit records if needed

The database should not be treated as the source of truth for file bytes. Object storage remains the source of truth for stored files.

### 9.3 Proposed abstraction boundary

Define repositories such as:

- `UserRepository`
- `SessionRepository`
- `MetadataRepository`
- optional `AuditRepository`

Handlers and services call repository interfaces, not SQLite directly.

### 9.4 Migration path

If SQLite later becomes limiting, keep the service layer unchanged and introduce a new repository implementation backed by PostgreSQL or another database.

### 9.5 Metadata cache role

The metadata cache should be treated as a first-class optimization because storage listing and metadata lookups are likely to dominate perceived latency more than HTML rendering would.

Recommended uses:

- cache normalized directory listings
- cache object metadata needed for the browser view
- support pagination and lightweight sorting
- reduce repeated provider round-trips for common browsing flows

Object storage remains the source of truth. Cache invalidation rules should stay simple for MVP.

## 10. Path and namespace model

### 10.1 Internal path model

Use normalized virtual paths such as `/photos/2026/cat.jpg`.

The app should treat these as canonical regardless of provider.

### 10.2 Folder semantics

Folders are logical, not physical. The system infers them from object prefixes.

This is important because:

- Azure Blob commonly uses prefix-based directory emulation
- S3 also commonly uses prefix-based directory emulation
- OSS works well with the same general model

Using a virtual path model gives the cleanest cross-provider behavior.

## 11. Upload and download strategy

### 11.1 Uploads

The browser asks the backend for an upload authorization. The backend validates the request and returns a provider-specific signed upload target. The browser uploads directly to the object storage provider.

Benefits:

- the Ubuntu host does not relay the file body
- upload throughput scales with the provider
- server CPU and memory stay low

### 11.2 Downloads and previews

The browser asks the backend for a download or preview authorization. The backend validates the request and returns a short-lived signed URL. The browser then fetches the file directly from object storage.

### 11.3 When proxying is allowed

Proxying content through the Go service should be the exception, not the default. It may be acceptable later for narrow cases like:

- inline content filtering
- audit enforcement
- special transformations

But that should not be part of the MVP hot path.

## 12. Security model

### 12.1 Authentication

For MVP, use a small local auth model:

- single-user or very small user list
- password hashes using Argon2id or bcrypt
- session cookie or equivalent lightweight session handling

### 12.2 Authorization

Keep authorization simple in MVP, but centralize it in the service layer so it is not scattered across handlers.

### 12.3 Signed URL policy

- short expiration
- least privilege
- scope to one object or one upload target
- no long-lived storage secrets in the browser

### 12.4 Operational security

- reuse existing Nginx TLS setup
- add security headers
- rate-limit auth-sensitive endpoints
- log access and failures

## 13. Operational model

### 13.1 Runtime model

- Nginx receives public traffic
- Nginx serves static frontend assets and routes API requests to the Go app
- Go app runs as a systemd service
- SQLite lives on local disk
- object data lives in cloud object storage

### 13.2 Resource expectations

This design is intentionally conservative for a 2-core, 2-GB host.

- Go app should remain lightweight
- SQLite load should be modest
- Nginx already exists, so no additional edge service is needed
- direct-to-object-store transfers keep memory usage predictable

## 14. Operations and observability

The MVP should include a minimal but real operational baseline.

Recommended from day one:

- `GET /health` for liveness
- `GET /ready` for readiness if startup dependencies matter
- structured logs
- request IDs propagated from Nginx if available
- clear error logging for provider and database failures

These are low-cost additions that significantly improve maintainability.

## 15. Thumbnails and previews

Thumbnail generation does not need to be a mandatory MVP feature, but it should be acknowledged because image browsing is a likely use case.

Recommended position:

- do not make server-side thumbnail generation a hard MVP dependency
- if image browsing quality becomes important, add lazy thumbnail generation later
- store generated thumbnails in object storage or a small local cache
- use an LRU cache if local thumbnail reuse becomes valuable

This keeps the first release simpler while leaving a practical path for better gallery UX.

## 16. Suggested MVP module breakdown

Suggested internal modules:

- `cmd/server`
- `internal/http`
- `internal/service`
- `internal/storage`
- `internal/storage/azure`
- `internal/storage/s3` later
- `internal/storage/aliyun` later
- `internal/repository`
- `internal/repository/sqlite`
- `internal/auth`
- `internal/observability`

The exact package layout can change, but the boundary idea should remain.

## 17. MVP risks and mitigations

### Risk: provider abstraction becomes too generic

Mitigation: keep the interface small and driven by real MVP use cases only.

### Risk: SQLite becomes a bottleneck or operational concern

Mitigation: confine DB access to repositories and keep schema narrow.

### Risk: frontend becomes too dynamic for templates

Mitigation: keep the backend API-first, start with a static lightweight frontend, and move to React later if the product demands it.

### Risk: signed URL handling differs across providers

Mitigation: keep signed URL creation fully inside provider implementations and expose normalized request/response models to the service layer.

### Risk: metadata cache becomes stale or overcomplicated

Mitigation: keep object storage as source of truth, scope cache behavior to browsing needs, and avoid complex invalidation rules in MVP.

## 18. Optional client-side encryption mode

This section is intentionally out of MVP scope. It remains here only as future-direction guidance so later design work does not conflict with current decisions.

### 16.1 Goal

If you want the storage provider to receive only encrypted file content, encryption must happen on the client before upload. In practice, this means the browser encrypts file bytes before sending them to Azure Blob, S3, or OSS.

This should be treated as an optional mode layered onto the architecture, not as an afterthought.

### 16.2 Recommended practice

Use browser-side envelope encryption:

- generate a random per-file data key in the browser
- encrypt the file with an authenticated symmetric algorithm such as AES-GCM
- wrap the per-file key with a user-level master key
- upload ciphertext to the storage provider
- store only ciphertext, wrapped key, and encryption metadata outside the plaintext file body

This keeps plaintext away from:

- the storage provider
- the Ubuntu host
- the reverse proxy

### 16.3 Where encryption should happen

The browser should do the encryption if the goal is true end-to-end or provider-blind encryption.

Reasons:

- server-side encryption still exposes plaintext to the Go service
- client-side encryption preserves the current thin-server architecture
- it avoids turning the small Ubuntu host into a CPU-heavy encryption gateway

### 16.4 Performance impact on clients

Yes, client-side encryption adds overhead, but it is usually reasonable on modern desktop and laptop hardware if implemented correctly.

Recommended implementation patterns:

- use the Web Crypto API
- encrypt in chunks rather than loading the whole file at once
- use Web Workers for large files to avoid blocking the UI
- upload encrypted chunks or encrypted file output directly to object storage

Expected tradeoff:

- modest client CPU cost
- low constant memory usage if chunked properly
- more implementation complexity than plain upload

For typical personal or small-team usage, the user experience is usually acceptable. Older devices and very large files will feel the impact more.

### 16.5 Product tradeoffs introduced by client-side encryption

Client-side encryption affects several user-facing features:

- server cannot inspect plaintext files
- server cannot generate plaintext thumbnails by itself
- provider-side preview logic is no longer useful
- content search becomes much harder
- sharing becomes a key-management problem, not just an access-control problem
- encrypted filenames and metadata may be needed if metadata privacy matters

### 16.6 Preview strategy with encrypted files

There are three practical options:

1. decrypt locally in the browser and render previews there
2. upload client-generated encrypted thumbnails or preview artifacts separately
3. disable some preview features in encrypted mode

For MVP, the cleanest rule is:

- plaintext mode supports the normal preview experience
- encrypted mode uses browser-side decryption for supported files and may offer a reduced preview experience

### 16.7 Metadata considerations

Even with encrypted file bodies, metadata can still leak information.

Potentially exposed items include:

- filenames
- folder structure
- file sizes
- timestamps
- MIME types

If metadata privacy matters, the design should allow:

- encrypted filenames or opaque object names
- encrypted per-object metadata blobs
- logical folder structure stored in encrypted metadata rather than directly in object names

This is a deeper privacy design choice and may be phased after the initial encrypted mode.

### 16.8 Architectural impact

If optional encrypted mode is supported, the architecture should add:

- a client crypto module in the frontend
- key-derivation and key-wrapping logic in the browser
- encrypted object metadata handling in the backend and repository layer
- clear file-mode metadata so the system knows whether an object is plaintext or client-encrypted

The storage provider abstraction should remain unchanged at a high level, because it still stores objects and returns signed URLs. The main differences are in what the client uploads and what metadata the application tracks.

### 16.9 Recommended scope decision

Recommended approach:

- keep client-side encryption out of the MVP implementation
- do not let it drive first-release API or UX complexity
- revisit it only after the plain MVP path is stable

This keeps the first release simpler while preserving a clean path to stronger privacy.

## 19. Decisions proposed for approval

Please review these MVP decisions:

1. Use Go for the backend service.
2. Reuse the existing Nginx instance as the reverse proxy.
3. Use a lightweight static frontend for MVP, served by Nginx and backed by JSON APIs.
4. Use Azure Blob as the first storage provider.
5. Add a storage-provider abstraction now so AWS S3 and Aliyun OSS can be implemented later.
6. Use SQLite for MVP metadata/auth state.
7. Treat file metadata cache as a core feature and keep a repository abstraction so the metadata database can change later.
8. Use `chi` as the HTTP router/framework.
9. Add health endpoints, structured logging, and request IDs from the beginning.
10. Keep large uploads/downloads direct between browser and object storage.
11. Keep client-side encryption out of MVP scope and revisit it later as an optional design extension.

## 20. Next document after approval

After you review and approve this MVP architecture spec, the next step should be the more detailed system design package:

- component diagram
- API contracts
- deployment layout
- configuration model
- schema outline
