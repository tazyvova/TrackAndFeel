# Training Insights — Monorepo Skeleton

This is a starter skeleton for a learning project: Go backend + Vue 3 frontend,
containerized and set up for local dev with Vite proxying `/api` to the Go server.

## Quick start (local)

Requirements:
- Go 1.22+
- Node.js 20+ (npm 10+)
- VS Code (recommended) with Go, Vue, ESLint, Prettier extensions

```bash
# 1) Backend — run the API on :8080
cd backend
go mod tidy
go run ./cmd/server

# 2) Frontend — dev server on :5173
cd ../frontend
npm ci
npm run dev
```

Open http://localhost:5173 and check `/healthz` works via proxy.
