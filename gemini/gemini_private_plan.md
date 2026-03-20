# Gemini Private Plan: High-Performance Storage Gateway

## 1. Executive Summary
This plan proposes a lightweight, high-performance storage gateway tailored for a resource-constrained Ubuntu server (2 vCPU, 2GB RAM). It maximizes performance by offloading resource-intensive tasks (file transfers, UI rendering) to the client and external cloud providers.

## 2. Architecture Overview

### Core Principles
*   **Control Plane Only:** The Ubuntu server manages metadata and authorization. It *never* touches file data.
*   **Direct-to-Cloud:** Uploads and downloads occur directly between the User's Browser and Azure Blob Storage via Shared Access Signatures (SAS).
*   **Stateless API:** The Go server is a stateless JSON API.
*   **Client-Side Rendering:** The UI is a static Single Page Application (SPA) served by Nginx, removing rendering load from the server.

### Tech Stack
*   **Backend:** **Go (Golang)**. Compiled, memory-efficient, excellent concurrency.
*   **Frontend:** **React (Vite build)**. Static HTML/JS/CSS files served by Nginx.
*   **Database:** **SQLite** (with `WAL` mode for performance). Stores file metadata, user info, and folder structures.
*   **Storage:** **Azure Blob Storage**.
*   **Reverse Proxy:** **Nginx**. Handles SSL, serves static frontend files, and proxies `/api` requests to Go.

## 3. Detailed Design

### 3.1 Data Flow
1.  **Listing Files:**
    *   User Browser -> `GET /api/files?path=/photos` -> Nginx -> Go Server.
    *   Go Server queries SQLite (indexed by parent_id) -> Returns JSON.
    *   *Performance:* Fast, low memory. 10k files is just a few KB of JSON.

2.  **Viewing/Downloading:**
    *   User Click -> `GET /api/files/:id/download` -> Go Server.
    *   Go Server generates **Azure SAS URL** (valid for 5 mins).
    *   Go Server returns 302 Redirect or JSON URL.
    *   Browser -> Azure Blob Storage (High Bandwidth).
    *   *Server Load:* Negligible.

3.  **Uploading:**
    *   User Drag & Drop -> `POST /api/upload/intent` -> Go Server.
    *   Go Server generates **Azure SAS URL** (Write permission).
    *   Browser -> `PUT <Azure-URL>` (Uploads bytes).
    *   Browser -> `POST /api/upload/confirm` -> Go Server (Updates SQLite).

### 3.2 Database Schema (SQLite)
*   **`users`**: `id`, `username`, `password_hash`
*   **`nodes`**: `id`, `parent_id` (index), `name`, `is_folder`, `size`, `mime_type`, `storage_key`, `created_at`
    *   *Optimization:* `parent_id` index allows O(1) directory listing lookups regardless of total system size.

## 4. Why this is better than Server-Side Templates?
The previous plan suggested Go Templates. We recommend **React SPA** instead:
1.  **Zero CPU Rendering:** Nginx serves static `.js` files. The user's browser does the CPU work to render the HTML.
2.  **Better User Experience:** "App-like" feel, instant directory switching without page reloads.
3.  **Decoupling:** You can update the UI without restarting the Go server.

## 5. Implementation Roadmap

### Phase 1: Foundation (Current Step)
*   [ ] Initialize Go module `github.com/yourname/forge-storage`.
*   [ ] Set up SQLite with `gorm` or raw `sql` migration.
*   [ ] Create basic HTTP server (Gin or Echo framework).

### Phase 2: Core Storage
*   [ ] Implement Azure SDK for Go.
*   [ ] Create `SAS Token` generation service.
*   [ ] Implement `Upload Intent` -> `Direct Upload` -> `Confirm` workflow.

### Phase 3: Frontend
*   [ ] Scaffold React + Vite project.
*   [ ] Build "File Explorer" component (Grid/List view).
*   [ ] Implement file upload queue component.

### Phase 4: Deployment
*   [ ] Configure Nginx (Reverse Proxy).
*   [ ] Create `systemd` service for Go binary.

## 6. Next Step
**Architecture Decision:** Confirm the switch to **React SPA + Go JSON API** (Recommended) vs. Go Templates.

Once confirmed, I will start **Phase 1: Foundation** by initializing the project structure.
