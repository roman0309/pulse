# Contributing to Pulse

Thanks for your interest in improving Pulse! This guide gets you from zero to a
running dev environment and a merged pull request.

## Code of Conduct

This project follows the [Contributor Covenant](CODE_OF_CONDUCT.md). By
participating you agree to uphold it.

## Ways to contribute

- 🐛 **Report bugs** — open an issue with steps to reproduce.
- 💡 **Suggest features** — open a feature-request issue describing the problem first.
- 📖 **Improve docs** — README, INTEGRATION, DEPLOYMENT, code comments.
- 🔧 **Send code** — fix a bug or implement an accepted feature.

For anything large, **open an issue to discuss before writing code** so we agree on
the approach.

## Development setup

Prerequisites: **Docker**, **Go 1.24+**, **Node 20+**.

```bash
git clone https://github.com/roman0309/pulse.git
cd pulse

# Option 1 — full stack in Docker
docker compose up -d
# UI http://localhost:3000  ·  demo@metrics.dev / demo123456

# Option 2 — datastores in Docker, app locally for hot reload
docker compose up -d postgres clickhouse otel-collector
cd backend && go run ./cmd/server      # :8080
cd frontend && npm install && npm run dev   # :5173
```

## Project layout

- `backend/` — Go API (clean architecture), migrations, agent, sample app.
- `frontend/` — React 19 SPA (feature-based).
- `examples/` — ready-to-use agent/collector configs.

See the [README](README.md#-architecture) for the architecture overview.

## Before you open a PR

Run the same checks CI runs:

```bash
cd backend  && go build ./... && go vet ./... && go test ./...
cd frontend && npm run build
```

- Keep changes focused; one logical change per PR.
- Match the surrounding code style (the repo has no heavy linters — read nearby code).
- Update docs when you change behaviour.
- Add a clear PR description: what changed and why.

## Commit & PR conventions

- Write descriptive commit messages (imperative mood: "Add X", "Fix Y").
- Reference issues with `Fixes #123` where relevant.
- PRs are squash-merged; keep the title meaningful.

## License of contributions

Pulse is licensed under **AGPL-3.0**. By submitting a contribution you agree that it
will be licensed under the same terms.

Happy hacking! 🟢
