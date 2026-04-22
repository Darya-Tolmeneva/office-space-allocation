# FlowDesk — Frontend

Liquid-glass UI for FlowDesk hot-desking system. Built with React 18 + Vite + Tailwind.
The app is a **pure SPA** that talks to an external FlowDesk backend over HTTP/JSON
(see [`docs/flowdesk-openapi.yaml`](docs/flowdesk-openapi.yaml), spec v1.2.0).

## Configuration

The frontend reads a single environment variable at **build time**:

| Variable | Default | Description |
|---|---|---|
| `VITE_API_BASE_URL` | `http://localhost:8080/v1` | Base URL of the FlowDesk API (must include the `/v1` prefix). |

Copy `.env.example` to `.env` and adjust before building.

## Local development

```bash
npm install
npm run dev   # Vite dev server on http://localhost:8080
```

## Docker

Multi-stage `Dockerfile`: builds the SPA with Node, then serves it via nginx with
SPA fallback (deep links like `/login` work on refresh).

```bash
# Build & run with docker compose (reads .env / .env.example defaults)
docker compose up --build

# Frontend will be available on http://localhost:3000
```

Or build directly:

```bash
docker build \
  --build-arg VITE_API_BASE_URL=https://api.flowdesk.example.com/v1 \
  -t flowdesk-frontend .

docker run --rm -p 3000:80 flowdesk-frontend
```

> ⚠️ `VITE_API_BASE_URL` is **baked into the build**. To point the same image at a
> different backend, rebuild with a new `--build-arg`.

## Auth

- `/login` and `/register` are public.
- All other routes are gated behind `ProtectedRoute` and require a valid JWT.
- Tokens (access + refresh) are stored in `localStorage` and the API client
  auto-refreshes via `POST /auth/refresh` on 401.
