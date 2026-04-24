# FlowDesk — Flexible Workplace Booking Platform

FlowDesk is a web application for booking office desks and workspaces. Employees can browse floor plans, reserve desks

---

## Environments

| Environment | URL | Purpose |
|-------------|-----|---------|
| **Production** | [http://oasis-space.duckdns.org](http://oasis-space.duckdns.org) | Stable release for end users |
| **Testing** | [http://test-oasis-space.duckdns.org](http://test-oasis-space.duckdns.org) | Pre-release testing |
| **Grafana (Test)** | [http://test-oasis-space.duckdns.org/grafana](http://test-oasis-space.duckdns.org/grafana) | Monitoring dashboard (test) |
| **Grafana (Prod)** | [http://oasis-space.duckdns.org/grafana](http://oasis-space.duckdns.org/grafana) | Monitoring dashboard (prod) |

**FOR BOTH ENVS GRAFANA CREDENTIALS ARE:**
```
login: admin
password: tovarish_dru
```

---

### Tech Stack

| Layer | Technology |
|-------|-----------|
| **Frontend** | React 18, Vite, TypeScript, Tailwind CSS, shadcn/ui, Framer Motion |
| **Backend** | Go 1.24, Chi router, JWT (access + refresh tokens), golang-migrate |
| **Database** | PostgreSQL 16 |
| **Infrastructure** | Kubernetes (k3s), Kustomize, NGINX Ingress Controller |
| **Monitoring** | Prometheus, Grafana, Node Exporter |
| **CI/CD** | GitHub Actions |

## Monitoring

The platform includes a full observability stack:

- **Prometheus** — scrapes backend `/metrics` and node-exporter
- **Grafana** — dashboard with monitoring panels
- **Node Exporter** — VM hardware metrics

## CI/CD

Automated via GitHub Actions:

| Workflow | Trigger | Action |
|----------|---------|--------|
| **CI** | Push to any branch | Lint, test, build |
| **Deploy Test** | Push to `main` | Build images → deploy to test VM |
| **Deploy Prod** | Manual dispatch | Build images → deploy to prod VM |

## Kubernetes

Both environments run on **k3s** with:

- **Kustomize** base + overlay pattern for environment separation
- **NGINX Ingress Controller** with direct port binding
- **golang-migrate** initContainer for automatic database migrations

---

## Documentation

Detailed project documentation is available in the [`docs/`](docs/) directory:

- [Lean Canvas](docs/lean-canvas.md) — Business model overview
- [Technical Requirements](docs/technical-requirements.md) — System requirements and constraints
- [Use Cases](docs/use-cases.md) — Detailed use case descriptions
- [User Stories](docs/user-stories.md) — User stories and acceptance criteria
- [Test Plan](docs/test-plan.md) — Testing strategy and test cases
