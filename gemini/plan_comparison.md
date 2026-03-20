# Plan Comparison: Original MVP vs. Gemini Revised Plan

This document compares the two proposed architectures for your high-performance storage system on a resource-constrained Ubuntu server (2 vCPU, 2GB RAM).

## 1. Core Differences

| Feature | **Original MVP Plan** (`plan/mvp-architecture-spec.md`) | **Gemini Revised Plan** (`gemini_private_plan.md`) |
| :--- | :--- | :--- |
| **Frontend Tech** | **Go Templates (Server-Side Rendering)** | **React SPA (Client-Side Rendering)** |
| **Server Load** | **Medium:** Server renders HTML for every page load. | **Low:** Server only sends JSON data. Client renders UI. |
| **Bandwidth** | **Higher:** Sends full HTML on every navigation. | **Lower:** Sends only data (JSON) after initial load. |
| **User Experience** | **Classic Web:** Full page reloads on navigation. | **App-Like:** Instant transitions, no reloads. |
| **Development** | **Simpler:** Single codebase, no separate build step. | **Modern:** Separate frontend build (Vite), richer ecosystem. |
| **API Style** | **Mixed:** HTML + some JSON/API endpoints. | **Pure JSON:** Clean separation of concerns. |
| **Scalability** | **Good:** But UI rendering consumes CPU. | **Better:** UI is static files; API is stateless. |

## 2. Detailed Analysis

### Option A: Original MVP (Go Templates)
**Best for:** Simplicity, small teams, "getting it done" quickly.
*   **Pros:**
    *   **Single Binary:** Everything (UI + API) compiles into one Go executable.
    *   **No Build Step:** No `npm install`, `npm run build`, or managing `node_modules`.
    *   **Less Complexity:** No state management on the client (Redux/Context).
*   **Cons:**
    *   **Server CPU Usage:** Every page view requires the Go server to parse templates and generate HTML strings. On a 2-core machine under load, this eats cycles that could be used for API processing.
    *   **Clunkier UX:** Navigating directories flashes the screen (full reload).

### Option B: Gemini Revised (React SPA)
**Best for:** Performance, user experience, future-proofing.
*   **Pros:**
    *   **Offloaded Rendering:** The user's browser does the heavy lifting of rendering the UI. The server just sends raw data.
    *   **Snappy UX:** Instant directory switching feels like a native file explorer.
    *   **Separation of Concerns:** You can swap the backend (e.g., to Python/Rust) without touching the UI, or swap the UI (e.g., to Vue/Mobile App) without touching the backend.
    *   **Rich Ecosystem:** Access to thousands of React components (e.g., robust file uploaders, image galleries, drag-and-drop libs) that are hard to build with plain templates.
*   **Cons:**
    *   **Setup Complexity:** Requires a Node.js build pipeline (Vite) alongside the Go build.
    *   **Two Projects:** Effectively managing two codebases (frontend/ and backend/).

## 3. Recommendation for "High Performance" on "Bad Hardware"

**Recommendation: Option B (React SPA)**

**Reasoning:**
The primary constraint is the **2 vCPU / 2GB RAM** limit.
*   **CPU:** In Option A, 100 concurrent users browsing folders = 100 concurrent template renders on the server. In Option B, it's 0 renders; Nginx serves static files efficiently, and the Go API just streams JSON from SQLite.
*   **Memory:** Go Templates require loading/parsing templates in memory. A React build is just static files on disk served by Nginx (using OS page cache).

For a storage system where "browsing" is a frequent action, shifting the rendering cost to the *client* is the single most effective performance optimization you can make for the server.

## 4. Next Steps
Based on this comparison, please choose the direction you prefer so we can proceed with implementation planning.
