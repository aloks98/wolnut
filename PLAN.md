# WoL-NUT Frontend Migration Plan

## Overview
Migrate from server-side Go templates (HTMX + Alpine.js) to a modern Svelte frontend with shadcn-svelte components.

---

## Phase 1: Project Setup & Backend API ✅
**Goal:** Set up Svelte project and convert Go backend to JSON API

### Task 1.1: Initialize Svelte Project ✅
- [x] Create SvelteKit project in `web/` directory
- [x] Configure Vite for static build
- [x] Add static adapter for embedding in Go
- [x] Configure path aliases

### Task 1.2: Add shadcn-svelte & Tailwind ✅
- [x] Install and configure Tailwind CSS v4
- [x] Initialize shadcn-svelte
- [x] Configure dark theme as default
- [x] Add base components: Button, Card, Input, Dialog, Badge, Sonner (Toast)

### Task 1.3: Convert Go Backend to JSON API ✅
- [x] Create `api.go` with JSON-only handlers
- [x] Update routes to `/api/` prefix
- [x] Add CORS middleware for development
- [x] Keep health endpoint
- [x] Updated main.go to serve frontend

### Task 1.4: Embed Frontend in Go Binary ✅
- [x] Configure Go embed for `web/build/`
- [x] Serve static files from embedded FS
- [x] Fallback to index.html for SPA routing

---

## Phase 2: Core UI Components ✅
**Goal:** Build reusable components and layout

### Task 2.1: Layout & Navigation ✅
- [x] Create app shell with navbar (`Nav.svelte`)
- [x] Add navigation links (Dashboard, Devices, UPS, Settings)
- [x] Implement mobile-responsive hamburger menu
- [x] Add dark mode (default) with light toggle via mode-watcher

### Task 2.2: Toast/Notification System ✅
- [x] Using svelte-sonner for toasts
- [x] Integrated with shadcn Sonner component
- [x] Success/error variants working
- [x] Auto-dismiss enabled

### Task 2.3: API Client ✅
- [x] Create typed API client (`lib/api/client.ts`)
- [x] Add fetch wrapper with error handling
- [x] Define TypeScript interfaces for all models (`lib/api/types.ts`)
- [x] Loading states in page components

---

## Phase 3: Dashboard Page ✅
**Goal:** Build the main dashboard with UPS monitoring and device controls

### Task 3.1: UPS Status Cards ✅
- [x] Create UPS card component
- [x] Display status badge (Online/On Battery/Offline)
- [x] Show battery %, runtime, load, input voltage
- [x] Add error state display

### Task 3.2: Battery Chart ✅
- [x] Add charting library (Layerchart)
- [x] Create donut chart for battery level
- [x] Create donut chart for load percentage
- [x] Color coding: green >50%, yellow 20-50%, red <20%

### Task 3.3: Auto-Refresh ✅
- [x] Implement polling with 30s interval
- [x] Add manual refresh button
- [x] Show "last updated" timestamp
- [x] Pause polling when tab is not visible

### Task 3.4: Device Wake Cards ✅
- [x] Create device card component
- [x] Wake button with loading state
- [x] Show success/error toast on wake
- [x] Link to device management

---

## Phase 4: Device Management Page ✅
**Goal:** CRUD interface for WoL devices

### Task 4.1: Device List ✅
- [x] Fetch and display all devices
- [x] Show name, MAC address
- [x] Empty state with call-to-action

### Task 4.2: Add Device Dialog ✅
- [x] Create dialog with form (name, MAC)
- [x] MAC address validation (backend)
- [x] Submit and refresh list
- [x] Form validation feedback

### Task 4.3: Delete Device ✅
- [x] Add delete button to each device
- [x] Delete with toast confirmation
- [x] Immediate UI update

### Task 4.4: Wake from Device Page ✅
- [x] Wake button on each device card
- [x] Same functionality as dashboard

---

## Phase 5: UPS Management Page ✅
**Goal:** CRUD interface for NUT UPS connections

### Task 5.1: UPS List ✅
- [x] Fetch and display all UPS entries
- [x] Show name, host, UPS name
- [x] Empty state with call-to-action

### Task 5.2: Add UPS Dialog ✅
- [x] Create dialog with form (name, host, ups_name)
- [x] Default port hint (3493)
- [x] Submit and refresh list

### Task 5.3: Delete UPS ✅
- [x] Add delete button to each UPS
- [x] Delete with toast confirmation
- [x] Immediate UI update

### Task 5.4: Test Connection (Optional)
- [ ] Add "Test" button before saving
- [ ] Show connection result inline
*Deferred for future enhancement*

---

## Phase 6: Settings Page ✅
**Goal:** Import/export configuration

### Task 6.1: Export Configuration ✅
- [x] Download button for data.json backup
- [x] Trigger file download

### Task 6.2: Import Configuration ✅
- [x] File upload input
- [x] Backend validates JSON structure
- [x] Reload app after import

---

## Phase 7: Polish & Production
**Goal:** Final touches and production readiness

### Task 7.1: Loading States
- [ ] Add skeleton loaders for cards
- [ ] Button loading spinners
- [ ] Page transition animations

### Task 7.2: Error Handling
- [ ] Global error boundary
- [ ] API error display
- [ ] Offline detection

### Task 7.3: Responsive Design
- [ ] Test on mobile viewports
- [ ] Adjust grid layouts
- [ ] Touch-friendly buttons

### Task 7.4: Build & Deployment ✅
- [x] Update Dockerfile for frontend build
- [x] Update GitHub Actions workflow
- [x] Test embedded binary
- [x] Install scripts already correct

### Task 7.5: Documentation ✅
- [x] Update README with new architecture
- [x] Add development instructions
- [x] Document API endpoints

---

## API Endpoints (Final)

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/devices` | List all devices |
| POST | `/api/devices` | Add device |
| DELETE | `/api/devices/{id}` | Delete device |
| POST | `/api/devices/{id}/wake` | Wake device |
| GET | `/api/ups` | List all UPS entries |
| POST | `/api/ups` | Add UPS |
| DELETE | `/api/ups/{id}` | Delete UPS |
| GET | `/api/ups/status` | Get all UPS statuses |
| GET | `/api/config/export` | Export data.json |
| POST | `/api/config/import` | Import data.json |
| GET | `/health` | Health check |

---

## Tech Stack Summary

| Layer | Technology |
|-------|------------|
| Frontend Framework | SvelteKit (static) |
| UI Components | shadcn-svelte |
| Styling | Tailwind CSS |
| Charts | Chart.js or Layerchart |
| State Management | Svelte stores |
| Backend | Go (net/http) |
| Data Storage | JSON file |
| Embedding | Go embed |

---

## Timeline Estimate

| Phase | Tasks | Complexity |
|-------|-------|------------|
| Phase 1 | 4 | Medium |
| Phase 2 | 3 | Low |
| Phase 3 | 4 | Medium |
| Phase 4 | 4 | Low |
| Phase 5 | 4 | Low |
| Phase 6 | 2 | Low |
| Phase 7 | 5 | Medium |

---

## Notes

- Keep the existing Go backend structure, just remove template code
- Use TypeScript for type safety in frontend
- All API responses should be JSON
- Frontend build output goes to `web/build/`
- Go embeds `web/build/` directory
- Development: run Vite dev server + Go backend separately
- Production: single binary with embedded frontend
