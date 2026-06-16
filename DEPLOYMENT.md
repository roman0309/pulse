# Production Deployment Guide

This guide covers taking Pulse from the local `docker compose up` MVP to a hardened
production deployment.

---

## 1. Secrets & configuration

Never ship the dev defaults. Generate strong secrets:

```bash
openssl rand -hex 32   # JWT_SECRET
openssl rand -hex 32   # JWT_REFRESH_SECRET
```

Required environment variables in production:

| Var                  | Notes                                                |
| -------------------- | ---------------------------------------------------- |
| `JWT_SECRET`         | 32+ random bytes, rotate periodically                |
| `JWT_REFRESH_SECRET` | distinct from `JWT_SECRET`                           |
| `POSTGRES_DSN`       | use `sslmode=require`, dedicated least-privilege user|
| `CLICKHOUSE_DSN`     | TLS-enabled endpoint, scoped user                    |
| `CORS_ORIGINS`       | exact production origin(s), no wildcards             |
| `PUBLIC_INGEST_URL`  | externally reachable Pulse URL remote agents push to (e.g. `https://pulse.example.com`); shown in the Connect page's setup commands |
| `AGENT_IMAGE`        | published agent image shown in the Connect run command (e.g. `ghcr.io/<owner>/pulse-agent:latest`) |
| `PORT`               | behind the reverse proxy                             |

> **Publishing the agent.** Push a `v*` tag to run the `Release agent` workflow
> (`.github/workflows/release-agent.yml`) — it builds a multi-arch image to
> `ghcr.io/<owner>/pulse-agent` and attaches binaries to the GitHub Release. Then set
> `AGENT_IMAGE` so the Connect page hands users a command that pulls your published image.

> **Remote app servers.** Pulse and the monitored apps usually live on different
> hosts. Agents push *outbound* to `PUBLIC_INGEST_URL`, so that URL must be
> reachable from your app servers, and the reverse proxy must route both `/api/`
> and `/otlp/` to the backend (the bundled frontend nginx already does). No inbound
> access into the app servers is needed — that's why the design is push-based.

Store secrets in a manager (AWS Secrets Manager, GCP Secret Manager, Vault, or
Kubernetes Secrets) — not in the image or compose file.

---

## 2. Managed databases (recommended)

Run the datastores as managed services rather than containers:

- **PostgreSQL** → RDS / Cloud SQL / Neon. Enable automated backups + PITR.
- **ClickHouse** → ClickHouse Cloud or a dedicated cluster. Metrics/logs are the
  high-volume path; size disk and set TTLs (the schema already sets a 30-day TTL).

Apply migrations on deploy:

```bash
# Postgres
psql "$POSTGRES_DSN" -f backend/migrations/postgres/001_init.sql
# (run 002_seed.sql only for demo/staging, never production)

# ClickHouse
clickhouse-client --host <host> --queries-file backend/migrations/clickhouse/001_init.sql
```

For repeatable migrations, wrap these in a migration tool (golang-migrate, Atlas, or
dbmate) and run as a pre-deploy job.

---

## 3. Build & ship images

```bash
docker build -t registry.example.com/pulse-backend:$(git rev-parse --short HEAD) ./backend
docker build -t registry.example.com/pulse-frontend:$(git rev-parse --short HEAD) ./frontend
docker push registry.example.com/pulse-backend:...
docker push registry.example.com/pulse-frontend:...
```

Both images are multi-stage and run as non-root. The backend is a static binary on
Alpine; the frontend is static assets served by nginx.

---

## 4. Reverse proxy & TLS

Terminate TLS at a load balancer or nginx/Traefik/Caddy in front of both services:

- Route `/api/*` (including the `/api/v1/projects/:id/ws` WebSocket upgrade) to the backend.
- Serve everything else from the frontend container.
- Enable HTTP/2, HSTS, and gzip/brotli.
- The frontend image already proxies `/api/` to `backend:8080` and forwards
  `Upgrade`/`Connection` headers for WebSockets.

Example: put a single domain `app.example.com` in front; the SPA and API share an origin
so no CORS is needed in production (set `CORS_ORIGINS` only if you split origins).

---

## 5. Kubernetes (optional)

A minimal topology:

- **backend** Deployment (2+ replicas) + Service + HPA on CPU.
  - Liveness/readiness probe: `GET /health`.
  - The WebSocket hub is in-memory per pod — for multi-replica realtime, put a sticky
    session at the LB **or** add a Redis/NATS pub-sub fan-out behind `ws.Hub.Broadcast`
    (the `Hub` is the single integration point to swap).
- **frontend** Deployment + Service (stateless, scale freely).
- **Ingress** with TLS + WebSocket support.
- Secrets via `Secret`, config via `ConfigMap`.

---

## 6. Scaling notes

- **ClickHouse** handles the metric/log write path; batch inserts (the repos already use
  `PrepareBatch`). Increase partitions/shards as volume grows.
- **PostgreSQL** carries low-volume relational data; connection pool is capped at 10 per
  pod (`pgxpool`). Tune `MaxConns` to `pods * conns <= server limit`.
- **Rate limiting** is per-pod in-memory; for global limits move to Redis.
- **Refresh tokens** are stored hashed and revocable; schedule a cleanup job to delete
  expired/revoked rows.

---

## 7. Observability of the platform itself

- The backend emits structured JSON logs (slog) to stdout — ship via your log agent.
- The OpenTelemetry Collector is included; point application/agent OTLP exporters at
  `:4317` (gRPC) or `:4318` (HTTP) and forward to your APM of choice.
- Add `/metrics` Prometheus scraping in front of the collector (`:8888`).

---

## 8. Pre-launch checklist

- [ ] Rotated `JWT_SECRET` / `JWT_REFRESH_SECRET`
- [ ] `CORS_ORIGINS` locked to production origin
- [ ] TLS terminated; WebSocket upgrade verified end-to-end
- [ ] Managed Postgres + ClickHouse with backups
- [ ] Migrations applied (without seed in prod)
- [ ] Health checks wired into the orchestrator
- [ ] Log shipping + alerting on the backend itself
- [ ] Refresh-token cleanup job scheduled
```
