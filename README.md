# Pulse — Developer-First Observability Platform

[![License: AGPL v3](https://img.shields.io/badge/License-AGPL_v3-blue.svg)](LICENSE)
[![CI](https://github.com/roman0309/pulse/actions/workflows/ci.yml/badge.svg)](https://github.com/roman0309/pulse/actions/workflows/ci.yml)
[![Go 1.24](https://img.shields.io/badge/Go-1.24-00ADD8.svg)](https://go.dev)
[![React 19](https://img.shields.io/badge/React-19-61DAFB.svg)](https://react.dev)

Pulse is a modern observability MVP focused on **incident investigation**, not dashboard building.
Every screen answers one question: **"What changed and why is my system behaving differently?"**

It combines **metrics, logs, alerts, deployments, an incident timeline and root-cause analysis**
into a fast, Linear/Vercel-style UI.

---

## ✨ Features

- **Incident Timeline** — chronological feed that auto-correlates deployments, metric spikes, errors and alerts.
- **Root Cause Analysis** — deterministic `RootCauseAnalyzer` produces a natural-language summary with a confidence score (LLM-ready via the `Analyzer` interface).
- **Metrics** — CPU, memory, request count/rate, error rate, latency P50/P95/P99 stored in ClickHouse with live updates.
- **Logs** — structured logs with full-text search, level/service filters and infinite scrolling.
- **Alerts** — high latency / high error rate / service down, with severity and resolve workflow.
- **Alerting engine** — define rules (e.g. `error_rate > 5% for 5m`); a background evaluator watches live metrics, fires/resolves alerts automatically, adds them to the timeline and notifies Slack/webhook.
- **Deployments** — release history correlated with incidents.
- **Realtime** — WebSocket fan-out for alerts, metrics and timeline events.
- **Auth & RBAC** — JWT access + refresh tokens, password hashing, org roles (owner/admin/member), rate limiting.

---

## 🏗 Architecture

```
┌──────────────┐     REST + WebSocket     ┌──────────────────────┐
│  React 19 SPA│ ───────────────────────▶ │  Go API (Gin)        │
│  Vite + TS   │ ◀─────────────────────── │  Clean Architecture  │
└──────────────┘                          │  ┌────────────────┐  │
                                          │  │ handlers       │  │
   ┌──────────────────────┐               │  │ services (DDD) │  │
   │ OpenTelemetry Collector│ ──OTLP──▶    │  │ repositories   │  │
   └──────────────────────┘               │  │ analyzer       │  │
                                          │  └────────────────┘  │
                                          └───────┬──────┬───────┘
                                                  │      │
                                      ┌───────────▼─┐  ┌─▼──────────────┐
                                      │ PostgreSQL  │  │  ClickHouse     │
                                      │ (entities)  │  │ (metrics, logs) │
                                      └─────────────┘  └─────────────────┘
```

- **PostgreSQL** — users, orgs, team members, projects, services, deployments, alerts, timeline events.
- **ClickHouse** — high-volume time-series metrics and structured logs.
- **Clean Architecture** — `domain/entities`, `domain/repositories` (interfaces), `domain/services` (use cases), `handlers`, with concrete repos in `repositories/postgres` and `repositories/clickhouse`. Dependency injection wired in `cmd/server/main.go`.

### Backend layout

```
backend/
  cmd/server/            # entrypoint + DI
  internal/
    config/              # env config
    domain/
      entities/          # domain models
      repositories/      # repository interfaces
      services/          # auth + core use cases, JWT
    handlers/            # Gin controllers, DTOs, validation, router
    middleware/          # JWT auth, rate limiting
    repositories/
      postgres/          # pgx implementations
      clickhouse/         # clickhouse-go implementations
    analyzer/            # RootCauseAnalyzer (Analyzer interface + deterministic impl)
    ws/                  # WebSocket hub + client
  migrations/
    postgres/            # schema + seed
    clickhouse/          # schema + seed
  api/openapi.yaml       # OpenAPI 3 spec
```

### Frontend layout (feature-based)

```
frontend/src/
  app/ pages/            # (reserved)
  features/              # auth, dashboard, metrics, logs, alerts,
                         # deployments, timeline, services, organizations, projects
  components/ui/         # shadcn-style primitives
  components/common/     # shared widgets (charts, badges, timeline feed)
  layouts/               # AppLayout (sidebar)
  hooks/                 # useWebSocket
  services/              # API client (auto token refresh)
  store/                 # Zustand auth store
  types/ routes/ lib/ styles/
```

---

## 🚀 Install (one command)

Self-host Pulse on any machine with Docker — uses published images, generates
secrets, and starts everything. The backend **migrates the databases itself** on
first boot; no manual steps.

```bash
curl -fsSL https://raw.githubusercontent.com/roman0309/pulse/main/deploy/install.sh | sh
```

Then open **http://localhost** and register the first user (becomes the owner).
To customize, edit the generated `pulse/.env` (see [`deploy/.env.example`](deploy/.env.example))
and `docker compose up -d` again. Production guidance: [DEPLOYMENT.md](DEPLOYMENT.md).

**Connecting servers without a domain?** Pulse ships an optional Tailscale profile —
`docker compose --profile tailscale up -d` exposes it privately at
`https://pulse.<your-tailnet>.ts.net` (auto HTTPS, no public ports). See
[INTEGRATION.md](INTEGRATION.md#private-networking-with-tailscale-no-domain-no-public-ports).

---

## 🛠 Run from source (development)

For hacking on Pulse — builds from source and seeds live demo data:

```bash
docker compose up --build        # SEED_DEMO=true, includes a demo agent + sample app
```

Then open:

| Service          | URL                      |
| ---------------- | ------------------------ |
| Frontend (SPA)   | http://localhost:3000    |
| Backend API      | http://localhost:8080    |
| Health check     | http://localhost:8080/health |
| ClickHouse HTTP  | http://localhost:8123    |
| OTel Collector   | grpc :4317 / http :4318  |

### Demo login

```
email:    demo@metrics.dev
password: demo123456
```

The seed data includes the **payment-api v1.8.2 incident**: a deployment followed by a
latency/error spike, an alert, and matching error logs — perfect for exploring the timeline
and root-cause analysis out of the box.

---

## 🧑‍💻 Local Development (without full Docker)

Run only the databases in Docker, and the app locally for hot reload:

```bash
# 1. Start datastores + collector
docker compose up postgres clickhouse otel-collector

# 2. Backend (Go 1.24+)
cd backend
go run ./cmd/server          # serves on :8080

# 3. Frontend (Node 20+)
cd frontend
npm install
npm run dev                  # serves on :5173, proxies /api -> :8080
```

Environment variables (see `.env.example`):

| Var                  | Default                                  |
| -------------------- | ---------------------------------------- |
| `PORT`               | `8080`                                   |
| `POSTGRES_DSN`       | `postgres://metrics:...@localhost:5432/metrics_db` |
| `CLICKHOUSE_DSN`     | `clickhouse://metrics:...@localhost:9000/metrics_db` |
| `JWT_SECRET`         | dev default (change in prod)             |
| `JWT_REFRESH_SECRET` | dev default (change in prod)             |
| `CORS_ORIGINS`       | `http://localhost:3000,http://localhost:5173` |

---

## 📡 Ingesting your own data

> **Connecting your own server & app?** See the step-by-step
> [INTEGRATION.md](INTEGRATION.md) — host agent, OTel SDK, scraping `/metrics`,
> and Prometheus `remote_write`, with copy-paste configs.

Pulse ingests from the existing ecosystem — point your **OpenTelemetry** instrumentation
or **Prometheus** at it, no re-instrumentation needed. Ingestion is authenticated with a
per-project **ingest key** (`X-Pulse-Key` header). Demo key: `pulse_demo_ingest_key`.

### Native OTLP

The bundled OTel Collector already forwards everything it receives to the backend.
Just send OTLP to the collector (`:4317` gRPC / `:4318` HTTP) — e.g. set your app's
exporter to `http://localhost:4318`. Or POST OTLP/protobuf straight to the backend:

```
POST /otlp/v1/metrics   (X-Pulse-Key: <key>, Content-Type: application/x-protobuf)
POST /otlp/v1/logs
POST /otlp/v1/traces    (accepted; storage lands in Phase 3)
```

Services are **auto-created** on first sight from the `service.name` resource attribute.

### Prometheus remote_write

Drop this into your `prometheus.yml` to dual-write into Pulse — you inherit the entire
exporter ecosystem for free:

```yaml
remote_write:
  - url: http://localhost:8080/api/v1/prom/write
    headers:
      X-Pulse-Key: pulse_demo_ingest_key
```

The `__name__` label becomes the metric name; `service` / `job` / `instance` becomes the
service.

### Live demo: real metrics from a server + its app

`docker compose` runs two real sources that report into the demo project, so the
dashboard shows **live** data (not just the seed):

| Source | Service | Metrics | Layer |
| ------ | ------- | ------- | ----- |
| `sampleapp` | `checkout-api` | request_rate, error_rate, latency p50/p95/p99 | application (RED) |
| `agent` | `demo-host` | cpu_usage, memory_usage, disk_usage, load_1m | infrastructure |

The sample app serves real HTTP traffic, measures its own latency/errors and reports
them — exactly how a service emits application metrics into its project.

### Pulse host agent (real server metrics)

The bundled `agent` service samples **real system metrics** (CPU, memory, disk, load)
and pushes them to its project every 10s — this is the "metrics come from a server,
scoped to a project" model. It runs automatically in `docker compose` against the demo
project and shows up as the `demo-host` service on the dashboard.

To monitor a real server, run the agent there with that project's key:

```bash
PULSE_ENDPOINT=https://pulse.example.com \
PULSE_KEY=<project ingest key> \
PULSE_SERVICE=payment-api \
PULSE_INTERVAL=10s \
./agent          # build: go build ./cmd/agent
```

One agent per host; the ingest key binds it to a project. App-level metrics (latency,
error rate, request rate) come from the app itself via OTLP.

### Quick REST ingestion (JWT auth)

Still available for simple scripted pushes:

```bash
curl -X POST http://localhost:8080/api/v1/projects/$PROJECT/ingest/metrics \
  -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -d '{"points":[{"service_id":"<uuid>","service_name":"payment-api","metric_name":"latency_p95","value":820}]}'
```

### Verify the pipeline

```bash
cd backend
go run ./cmd/ingesttest http://localhost:8080 pulse_demo_ingest_key
# sends a real OTLP gauge + a remote_write sample, prints 202s
```

---

## 🔌 API

Full spec: [`backend/api/openapi.yaml`](backend/api/openapi.yaml).
Import it into Swagger UI / Postman / Insomnia.

Key endpoints (all under `/api/v1`):

- `POST /auth/register | /auth/login | /auth/refresh | /auth/logout`, `GET /auth/me`
- `GET|POST /organizations`, `/organizations/:orgId/projects`
- `GET|PUT|DELETE /projects/:projectId`
- `GET /projects/:projectId/dashboard | /timeline | /analyze | /metrics | /logs`
- `GET|POST /projects/:projectId/services | /deployments | /alerts`
- `POST /alerts/:alertId/resolve`
- `GET /projects/:projectId/ws?token=...` (WebSocket)

---

## 🔒 Security

- Bcrypt password hashing.
- Short-lived JWT access tokens (15 min) + rotating refresh tokens (7 days, hashed at rest, revocable).
- RBAC org membership checks on every project-scoped operation.
- Request validation via struct tags (go-playground/validator).
- Per-IP token-bucket rate limiting.

---

## 🛳 Production Deployment

See [DEPLOYMENT.md](DEPLOYMENT.md) for the full production guide (secrets, TLS, scaling,
managed databases, observability of the platform itself).

---

## 🧪 Build & verify

```bash
cd backend  && go build ./... && go vet ./...
cd frontend && npm run build
```

---

## 🤝 Contributing

Contributions are welcome! See [CONTRIBUTING.md](CONTRIBUTING.md) for the dev setup
and workflow, and our [Code of Conduct](CODE_OF_CONDUCT.md). Found a security issue?
Please follow [SECURITY.md](SECURITY.md) and report it privately.

## 📄 License

Pulse is open source under the **[GNU AGPL-3.0](LICENSE)**. You can self-host, modify,
and run it freely; if you offer it as a network service, your modifications must be
made available under the same license. Copyright © 2026 Pulse contributors.
