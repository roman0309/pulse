#!/usr/bin/env sh
# ============================================================
# Pulse agent — one-line installer for a single host.
#
#   curl -fsSL https://raw.githubusercontent.com/roman0309/pulse/main/deploy/install-agent.sh \
#     | PULSE_ENDPOINT=https://pulse.your-tailnet.ts.net PULSE_KEY=pk_xxx sh
#
# Runs the host agent as a restart-on-boot container. PULSE_SERVICE defaults to
# this machine's hostname. Requires Docker.
# ============================================================
set -eu

err() { printf '\033[31m✗\033[0m %s\n' "$1" >&2; exit 1; }

: "${PULSE_ENDPOINT:?set PULSE_ENDPOINT (e.g. https://pulse.your-tailnet.ts.net)}"
: "${PULSE_KEY:?set PULSE_KEY (project ingest key from the Connect page)}"
PULSE_SERVICE="${PULSE_SERVICE:-$(hostname)}"
PULSE_INTERVAL="${PULSE_INTERVAL:-10s}"
AGENT_IMAGE="${AGENT_IMAGE:-ghcr.io/roman0309/pulse-agent:latest}"

command -v docker >/dev/null 2>&1 || err "Docker is required on this host."

# Replace any existing agent (idempotent re-runs).
docker rm -f pulse-agent >/dev/null 2>&1 || true

docker run -d --name pulse-agent --restart unless-stopped \
  -e PULSE_ENDPOINT="$PULSE_ENDPOINT" \
  -e PULSE_KEY="$PULSE_KEY" \
  -e PULSE_SERVICE="$PULSE_SERVICE" \
  -e PULSE_INTERVAL="$PULSE_INTERVAL" \
  -e HOST_PROC=/host/proc -e HOST_SYS=/host/sys \
  -v /proc:/host/proc:ro -v /sys:/host/sys:ro \
  -v /var/run/docker.sock:/var/run/docker.sock \
  "$AGENT_IMAGE" >/dev/null

printf '\033[32m✓\033[0m pulse-agent running  (service=%s → %s)\n' "$PULSE_SERVICE" "$PULSE_ENDPOINT"
