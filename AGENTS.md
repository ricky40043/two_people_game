# Repository Guidelines

This document provides guidelines for AI agents working in this repository.

## Project Structure

- **Root directory**: Static mini-games collection (index.html entry point, games in subfolders like `bomb-topic/`, `guess-number/`, `two-player-quiz/`, `who-is-spy/`)
- **kahoot-game/**: Kahoot-style multiplayer quiz game
  - `backend/`: Go 1.21+ backend with Gin + Gorilla WebSocket
  - `frontend/`: Vue 3 + TypeScript + Vite frontend with Pinia, Vue Router, TailwindCSS

## Build, Test, and Development Commands

### Root Static Games
```bash
npm run dev      # Start Python HTTP server at http://localhost:3000
npm run build    # No build needed for static site
npm run start    # Alias for npm run dev
```

### Kahoot Frontend (kahoot-game/frontend)
```bash
cd kahoot-game/frontend
npm install              # Install dependencies
npm run dev              # Start Vite dev server
npm run build            # Type-check (vue-tsc) + Vite build
npm run type-check       # Run vue-tsc --noEmit for type checking only
npm run preview          # Preview production build
```

### Kahoot Backend (kahoot-game/backend)
```bash
cd kahoot-game/backend
go mod download          # Download Go dependencies
go run cmd/main.go       # Run backend server
go build -o server cmd/main.go  # Build binary
```

## Coding Style

### General
- 4-space indentation (no tabs)
- Use semicolons and trailing commas
- Avoid unnecessary whitespace in template literals

### JavaScript/TypeScript
- Use ES modules (`import`/`export`)
- Prefer `const` over `let`, avoid `var`
- Use TypeScript strict mode; define interfaces for all data structures
- Vue components use `<script setup lang="ts">` with Composition API
- Path aliases: `@/*` maps to `src/*` (configured in tsconfig.json)
- File naming: PascalCase for components (`.vue`), camelCase for utilities (`.ts`)
- Import ordering: Vue/Router/Pinia imports first, then third-party, then local imports

### Go
- Standard Go formatting (`gofmt`)
- Error handling: check errors immediately, return or log with context
- Use context.Context for cancellation and timeouts
- Package structure: `internal/` for core logic, `cmd/` for entry points
- Group imports by type: standard library, third-party, internal packages

### CSS/Tailwind
- TailwindCSS utility classes for styling
- Semantic HTML elements with appropriate classes
- Mobile-first responsive design
- Avoid custom CSS when Tailwind utility exists

### File Naming
- **Vue components**: `PascalCase.vue` (e.g., `GamePlayerView.vue`)
- **TypeScript files**: `camelCase.ts` (e.g., `gameService.ts`)
- **Go files**: `snake_case.go` (e.g., `room_handler.go`)
- **Large datasets**: `*_complete.js` pattern for automatic loading

### Vue Component Structure
```vue
<template>...</template>
<script setup lang="ts">
  // Imports (Vue → Router/Pinia → Third-party → Local)
  // Types/Interfaces
  // Props/Emits
  // Store/composable initialization
  // Computed properties
  // Watchers
  // Lifecycle hooks
  // Methods
</script>
<style scoped>...</style>
```

### DOM Elements
- Use semantic IDs/classes (e.g., `#game-page`, `.option-btn`)
- Add comments for complex logic flows

## Error Handling

- **Frontend**: Use Pinia stores (`useUIStore`) for global error/success messages
- **Backend**: Return proper HTTP status codes; log errors with context
- **WebSocket**: Handle connection failures gracefully; implement reconnection logic
- Go: Always handle errors inline; use `if err != nil { return err }` pattern

## WebSocket Conventions

- Message format: JSON with `type` and `payload` fields
- Client: Native WebSocket API with Pinia store for state management
- Server: Gorilla WebSocket with hub pattern for broadcasting
- Reconnection: Auto-reconnect with exponential backoff on client

## Testing

- No automated tests currently configured
- Manual testing: Run `npm run dev` and verify in multiple browsers
- For question banks: Call exposed functions like `getQuestionsByFilter('medium','all',20)` in browser console to verify diversity
- Test responsive layouts on mobile viewport sizes

## Commit & Pull Request Guidelines

- Commit messages: "verb + purpose" in Chinese or English (e.g., `feat: expand global quiz dataset`)
- PR description: Include summary, testing instructions, related links, screenshots/videos for UI changes
- Before PR: Check code style, asset paths, deployment scripts; remove debug logs

## Security

- Never commit API keys or user data
- Use `.env.local` for local secrets; deploy secrets via platform environment variables
- Reference secrets via `process.env` in frontend, `os.Getenv` in Go
- Validate all user inputs on backend

## Environment Configuration

- Frontend: Vite with `import.meta.env` for env vars
- Backend: `godotenv` for local dev, platform env vars for production
- Required backend env vars: `DATABASE_URL`, `REDIS_URL`, `PORT`, `CORS_ORIGINS`

## Additional Documentation

- `README.md`: Project overview and deployment guide
- `TODO_DEVELOPMENT_PLAN.md`: Development roadmap
- `FULL_SYSTEM_TEST.md`: Testing procedures
- `SOCKET_FLOW_DOCUMENTATION.md`: WebSocket message flows
- `kahoot-game/docs/`: Architecture, tech stack, message specs

## Development Workflow

1. Create feature branch from main
2. Make changes following coding style guidelines
3. Run `npm run type-check` (frontend) before committing
4. Test locally with both backend and frontend running
5. Commit with descriptive message
6. Push and create PR with description
