# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

A multiplayer party game platform (Kahoot-style) built with a Go WebSocket backend and Vue 3 frontend. The first game mode "2種人" (Two Types) is implemented — players are classified into types based on their answers. The project is in Traditional Chinese (zh-TW).

## Development Commands

### Backend (Go)
```bash
cd kahoot-game/backend
go mod download          # Install dependencies
go run cmd/main.go       # Start server on :8080
```
The backend runs without Redis/PostgreSQL by falling back to in-memory mode (set `DB_HOST=skip` in `.env`).

### Frontend (Vue 3)
```bash
cd kahoot-game/frontend
npm install              # Install dependencies
npm run dev              # Start dev server on :3333 (proxies /api and /ws to backend)
npm run build            # Type-check + production build
npm run type-check       # vue-tsc --noEmit
```

### Both at once
```bash
./start.sh               # Kills old processes, starts backend + frontend
```

### No automated test suite
There is no test framework configured. The `test-*.js` files in the root are manual/ad-hoc WebSocket test scripts.

## Architecture

### Backend (`kahoot-game/backend/`)

**Hub-and-Spoke WebSocket pattern:**
- `internal/websocket/hub.go` — Central Hub manages all connections, rooms, and message routing via Go channels
- `internal/websocket/client.go` — Each WebSocket connection gets a Client with `readPump()`/`writePump()` goroutines and message handlers (`handleCreateRoom`, `handleJoinRoom`, `handleStartGame`, `handleSubmitAnswer`, etc.)

**Service layer:**
- `internal/services/game_service.go` — Game lifecycle, scoring logic, host rotation, question progression
- `internal/services/room_service.go` — Room CRUD, player join/leave
- `internal/services/question_service.go` — Question queries; built-in dataset in `two_types_questions.go`
- `internal/handlers/` — REST endpoints (`/api/rooms`, `/api/games`, `/api/questions`) and WebSocket upgrade (`/ws`, `/ws/:roomId`)
- `internal/config/config.go` — All configuration via environment variables with defaults
- `internal/models/models.go` — Shared data structures (Player, Room, Question, Answer, etc.)

**Entry point:** `cmd/main.go` — initializes config → database connections (with fallback) → services → Hub → Gin routes → graceful shutdown.

### Frontend (`kahoot-game/frontend/`)

**State management (Pinia stores):**
- `src/stores/socket.ts` — WebSocket connection, auto-reconnect with exponential backoff, session recovery from localStorage (`ricky_game_session`), message routing
- `src/stores/game.ts` — Room state, player info, questions, scores, game lifecycle (`waiting`|`playing`|`show_result`|`finished`)
- `src/stores/ui.ts` — Global loading/error/success toast state

**Views (9 routes):**
- `/` HomeView, `/create` CreateRoomView, `/join` JoinRoomView, `/join/:roomId` direct join
- `/lobby/:roomId` LobbyView, `/game/host/:roomId` GameHostView, `/game/player/:roomId` GamePlayerView
- `/results/:roomId` ResultsView, `/about` AboutView

**Key config:**
- `vite.config.ts` — Dev proxy to backend, manual chunk splitting (vendor, socket)
- Path alias: `@/*` → `src/*`
- Environment: `VITE_API_URL`, `VITE_WS_URL`, `VITE_WS_PORT`, `VITE_WS_PATH`

## WebSocket Protocol

All messages are JSON with `{ type, payload }` structure.

**Client → Server:** `CREATE_ROOM`, `JOIN_ROOM`, `START_GAME`, `SUBMIT_ANSWER`, `LEAVE_ROOM`, `PING`
**Server → Client:** `ROOM_CREATED`, `PLAYER_JOINED`, `GAME_STARTED`, `NEW_QUESTION`, `TIMER_UPDATE`, `SCORES_UPDATE`, `GAME_FINISHED`, `ERROR`, `PONG`

See `SOCKET_FLOW_DOCUMENTATION.md` for detailed message flows.

## Coding Conventions

- **Vue:** `<script setup lang="ts">` with Composition API
- **Go:** Standard gofmt, always handle errors
- **CSS:** TailwindCSS utilities only (no custom CSS files)
- **File naming:** PascalCase for Vue components, camelCase for TS files, snake_case for Go files
- **Web framework:** Gin (Go), Axios for HTTP client (frontend)

## Deployment

- **Primary:** Render.com via `render.yaml` (backend Go service + frontend static site + Redis)
- **Alternative:** Vercel for frontend (`vercel.json`), Google Cloud Run for backend
- **Database:** Supabase PostgreSQL (optional — works in memory mode without it)
