# Connecting a Server & Application to Pulse

How to get metrics from a real server (and the app running on it) into a Pulse project.

> **Push vs pull.** Pulse is **push-based**: an agent/collector on your server sends
> metrics in (authenticated by a project ingest key). To get a classic Prometheus
> *pull* feel, run a collector **on the server** that scrapes your app's `/metrics`
> locally and forwards to Pulse — covered in Recipe C below. (Pulse scraping your
> targets directly is on the roadmap, Phase 2.)

---

## Step 1 — Get your project's ingest key

Every server/agent authenticates with a per-project key sent in the `X-Pulse-Key`
header. In the app, open your project → **Connect** → **New ingest key**. The full key
is shown once (copy it then) and the page renders a ready-to-paste command with the key
and endpoint already filled in.

The seeded demo project key (for trying things out) is:

```
pulse_demo_ingest_key
```

The key binds everything a server sends to that one project. `service.name` (OTLP) or
the scrape `job` / `service` label (Prometheus) splits incoming data into **services**
inside the project — created automatically on first sight.

---

## Step 2 — Pick how the server reports

| You have… | Use | Gets you |
| --------- | --- | -------- |
| Just a server, want infra metrics | **Recipe A** — Pulse host agent | CPU, memory, disk, load |
| Source code you can instrument | **Recipe B** — OpenTelemetry SDK | app metrics (counters/gauges) |
| An app exposing `/metrics` (Prometheus) | **Recipe C** — collector scrape | whatever the app exports |
| An existing Prometheus server | **Recipe D** — `remote_write` | everything Prometheus already has |

---

## Recipe A — Pulse host agent (CPU / RAM / disk / load)

The simplest path for server infrastructure metrics. The exact command (with your key
and endpoint pre-filled) is shown on the project's **Connect** page — copy it from there.

**Docker** (reads the host through `/proc`, `/sys`):

```bash
docker run -d --name pulse-agent \
  -e PULSE_ENDPOINT=http://YOUR_PULSE_HOST:8080 \
  -e PULSE_KEY=YOUR_PROJECT_KEY \
  -e PULSE_SERVICE=prod-server-1 \
  -e HOST_PROC=/host/proc -e HOST_SYS=/host/sys \
  -v /proc:/host/proc:ro -v /sys:/host/sys:ro \
  ghcr.io/<owner>/pulse-agent:latest
```

The image is published by the `Release agent` GitHub Action on each version tag (see
[Publishing the agent](#publishing-the-agent)). To use it before publishing, build it
locally with `make agent-image` (tags `pulse-agent:local`).

**Binary** (no Docker — pre-built binaries are attached to each GitHub Release, or build
your own with `make agent-binary`):

```bash
PULSE_ENDPOINT=http://YOUR_PULSE_HOST:8080 \
PULSE_KEY=YOUR_PROJECT_KEY \
PULSE_SERVICE=prod-server-1 \
./pulse-agent
```

It emits the canonical metric names `cpu_usage`, `memory_usage`, `disk_usage`,
`load_1m` — so they show up directly on the dashboard.

### Publishing the agent

`make agent-publish AGENT_IMAGE=ghcr.io/<owner>/pulse-agent:v0.1.0`, or push a `v*` tag
to run the `Release agent` workflow (multi-arch image → GHCR + binaries → Release).
Point Pulse at your published image by setting `AGENT_IMAGE` on the backend — the Connect
page then shows that image in the run command automatically.

---

## Recipe B — Instrument your app with OpenTelemetry

Point your app's OTLP exporter at Pulse (or at a local collector that forwards to
Pulse). Pulse exposes a standard OTLP/HTTP endpoint at `/otlp`.

**Environment (works for any OTel SDK):**

```bash
export OTEL_EXPORTER_OTLP_ENDPOINT=http://YOUR_PULSE_HOST:8080/otlp
export OTEL_EXPORTER_OTLP_HEADERS=X-Pulse-Key=YOUR_PROJECT_KEY
export OTEL_SERVICE_NAME=payment-api
export OTEL_EXPORTER_OTLP_PROTOCOL=http/protobuf
```

**Go (excerpt):**

```go
exp, _ := otlpmetrichttp.New(ctx,
    otlpmetrichttp.WithEndpointURL("http://YOUR_PULSE_HOST:8080/otlp/v1/metrics"),
    otlpmetrichttp.WithHeaders(map[string]string{"X-Pulse-Key": "YOUR_PROJECT_KEY"}),
)
provider := metric.NewMeterProvider(metric.WithReader(metric.NewPeriodicReader(exp)))
```

**Node.js (excerpt):**

```js
import { OTLPMetricExporter } from "@opentelemetry/exporter-metrics-otlp-http";
const exporter = new OTLPMetricExporter({
  url: "http://YOUR_PULSE_HOST:8080/otlp/v1/metrics",
  headers: { "X-Pulse-Key": "YOUR_PROJECT_KEY" },
});
```

> **Note on latency percentiles:** the MVP backend ingests gauge/sum data points.
> SDK *histograms* (the usual source of p95/p99) are not yet decoded into percentiles —
> either export pre-computed gauges (as the bundled `sampleapp` does) or wait for
> histogram support (roadmap). Counters and gauges work today.

---

## Recipe C — Scrape an existing `/metrics` endpoint (the "pull" path)

If your app already exposes Prometheus metrics, run a collector **on the server** that
scrapes it locally and pushes to Pulse. A ready, validated config is provided:
[`examples/otel-agent.yaml`](examples/otel-agent.yaml).

Edit three things in it — `YOUR_PULSE_HOST`, `YOUR_PROJECT_KEY`, and the scrape
`targets` (your app's `host:port`) — then run:

```bash
docker run -d --name pulse-agent --network host \
  -v $PWD/examples/otel-agent.yaml:/etc/otelcol-contrib/config.yaml \
  otel/opentelemetry-collector-contrib:0.103.0
```

It scrapes your app **and** collects host metrics, forwarding both to Pulse. The scrape
`job_name` becomes the service name. (Scraped metric names are kept as-is; rename them to
Pulse's canonical names with a `metricstransform` processor if you want them on the
fixed dashboard cards.)

---

## Recipe D — Existing Prometheus → `remote_write`

Already running Prometheus? Dual-write into Pulse — no agent needed, you inherit your
whole exporter setup:

```yaml
# prometheus.yml
remote_write:
  - url: http://YOUR_PULSE_HOST:8080/api/v1/prom/write
    headers:
      X-Pulse-Key: YOUR_PROJECT_KEY
```

The `__name__` label becomes the metric name; `service` / `job` / `instance` becomes the
service.

---

## Private networking with Tailscale (no domain, no public ports)

The cleanest way to connect agents on other servers to Pulse without a domain or
exposing anything to the internet. Pulse joins your private Tailscale network and is
reachable at `https://<host>.<your-tailnet>.ts.net` with **automatic HTTPS**.

**One-time tailnet setup** (in the [Tailscale admin](https://login.tailscale.com/admin)):
enable **MagicDNS** and **HTTPS certificates**, then create a reusable
[auth key](https://login.tailscale.com/admin/settings/keys).

**On the Pulse server** — set these in `deploy/.env`:

```ini
TS_AUTHKEY=tskey-auth-xxxxxxxx
TS_HOSTNAME=pulse
PUBLIC_INGEST_URL=https://pulse.your-tailnet.ts.net
CORS_ORIGINS=https://pulse.your-tailnet.ts.net
```

### One command

The installer does it all — pass your auth key and it brings Pulse up on the tailnet,
auto-detects the `*.ts.net` address, and wires `PUBLIC_INGEST_URL` for you:

```bash
curl -fsSL https://raw.githubusercontent.com/roman0309/pulse/main/deploy/install.sh \
  | TS_AUTHKEY=tskey-auth-xxxxxxxx sh
```

It prints the private URL (e.g. `https://pulse.your-tailnet.ts.net`) when ready.

### Or manually

```bash
# in deploy/.env: TS_AUTHKEY, TS_HOSTNAME, PUBLIC_INGEST_URL, CORS_ORIGINS
docker compose --profile tailscale up -d
```

Either way, no public ports are needed — you can firewall off 80/8080 entirely. Pulse is
reachable at **`https://<TS_HOSTNAME>.your-tailnet.ts.net`** for anyone on your tailnet.

**On each agent / app server** — join the tailnet, then point the agent at the private URL:

```bash
curl -fsSL https://tailscale.com/install.sh | sh && sudo tailscale up

docker run -d --name pulse-agent --restart unless-stopped \
  -e PULSE_ENDPOINT=https://pulse.your-tailnet.ts.net \
  -e PULSE_KEY=YOUR_PROJECT_KEY \
  -e PULSE_SERVICE=payment-api \
  -e HOST_PROC=/host/proc -e HOST_SYS=/host/sys \
  -v /proc:/host/proc:ro -v /sys:/host/sys:ro \
  ghcr.io/roman0309/pulse-agent:latest
```

No domains, no open ports, traffic encrypted by WireGuard. View the UI from any device
on your tailnet at the same URL.

## Automated agent rollout

### One host — one line

Instead of the long `docker run`, use the agent installer:

```bash
curl -fsSL https://raw.githubusercontent.com/roman0309/pulse/main/deploy/install-agent.sh \
  | PULSE_ENDPOINT=https://pulse.your-tailnet.ts.net PULSE_KEY=YOUR_PROJECT_KEY sh
```

`PULSE_SERVICE` defaults to the host's name. Re-running it upgrades the agent.

### Many hosts — over SSH from one place

Roll it out to a fleet from the Pulse host (or any box with SSH access):

```bash
PULSE_ENDPOINT=https://pulse.your-tailnet.ts.net PULSE_KEY=YOUR_PROJECT_KEY \
  ./deploy/provision-agents.sh root@web1 root@web2 root@db1
# or:  ... ./deploy/provision-agents.sh -f hosts.txt
```

It SSHes to each host and runs the installer there. Each agent reports under that
host's own name.

**Keyless with Tailscale SSH** (recommended — no SSH keys to manage, pairs with the
[tailscale profile](#private-networking-with-tailscale-no-domain-no-public-ports)):
enable `tailscale up --ssh` on the targets and allow it in your tailnet ACLs, then:

```bash
SSH="tailscale ssh" PULSE_ENDPOINT=... PULSE_KEY=... \
  ./deploy/provision-agents.sh root@web1 root@web2
```

### Large fleets

For serious fleets, call `install-agent.sh` from your existing tooling — an Ansible
task, a cloud-init `runcmd`, or a Kubernetes DaemonSet — rather than the SSH loop.
The installer is the stable primitive; the orchestration is yours to choose.

## Step 3 — Verify it's flowing

```bash
# Get an access token
TOKEN=$(curl -s http://YOUR_PULSE_HOST:8080/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"demo@metrics.dev","password":"demo123456"}' | jq -r .access_token)

# List services in the project — your new server/app should appear
curl -s http://YOUR_PULSE_HOST:8080/api/v1/projects/<project-uuid>/services \
  -H "Authorization: Bearer $TOKEN" | jq '.services[].name'

# Query a metric
curl -s "http://YOUR_PULSE_HOST:8080/api/v1/projects/<project-uuid>/metrics?metric=cpu_usage" \
  -H "Authorization: Bearer $TOKEN" | jq
```

Then open the dashboard → **Metrics**, filter by your service, and watch it live.
