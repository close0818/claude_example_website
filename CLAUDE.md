# CLAUDE.md - LibraryMS Project

## Project Overview
A library management system with a Golang backend (Gin) and a Vue.js/Vite frontend.

## Backend (Golang)
Location: `libraryms/backend`

### Commands
- **Run Server**: `go run cmd/server/main.go`
- **Run Tests**: `go test ./...`
- **Format Code**: `go fmt ./...`
- **Install Dependencies**: `go mod tidy`

### Coding Standards
- Follow standard Go idioms and `effective go`.
- Use Gin for routing and middleware.
- Organize code with `internal/` for private logic.
- Use `models` for database structures.

## Frontend (Vue.js)
Location: `libraryms/frontend`

### Commands
- **Dev Server**: `npm run dev`
- **Build Project**: `npm run build`
- **Preview Build**: `npm run preview`

### Coding Standards
- Use Vue 3 Composition API.
- Use Vite for bundling and development.
- Use Lucide icons where appropriate.
- Maintain component-based architecture.
