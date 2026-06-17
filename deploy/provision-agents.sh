#!/usr/bin/env sh
# ============================================================
# Pulse — roll the agent out to many hosts over SSH.
#
# From the machine running Pulse (or any box with SSH access), install the
# agent on each target host by running install-agent.sh remotely.
#
# Usage:
#   PULSE_ENDPOINT=https://pulse.your-tailnet.ts.net PULSE_KEY=pk_xxx \
#     ./provision-agents.sh root@web1 root@web2 root@db1
#
#   # or read targets (one per line) from a file:
#   PULSE_ENDPOINT=... PULSE_KEY=... ./provision-agents.sh -f hosts.txt
#
# Keyless via Tailscale SSH (recommended — pairs with the tailscale profile):
#   SSH="tailscale ssh" PULSE_ENDPOINT=... PULSE_KEY=... ./provision-agents.sh root@web1 ...
#
# Each agent reports under the remote host's own hostname (PULSE_SERVICE).
# ============================================================
set -eu

err() { printf '\033[31m✗\033[0m %s\n' "$1" >&2; exit 1; }

: "${PULSE_ENDPOINT:?set PULSE_ENDPOINT}"
: "${PULSE_KEY:?set PULSE_KEY}"
SSH="${SSH:-ssh}"
INSTALLER="${INSTALLER_URL:-https://raw.githubusercontent.com/roman0309/pulse/main/deploy/install-agent.sh}"

# Collect target hosts from args or a -f file.
if [ "${1:-}" = "-f" ]; then
  [ -n "${2:-}" ] || err "usage: -f <hostsfile>"
  HOSTS=$(grep -v '^[[:space:]]*#' "$2" | sed '/^[[:space:]]*$/d')
else
  [ "$#" -gt 0 ] || err "no hosts given (pass user@host ... or -f hosts.txt)"
  HOSTS="$*"
fi

ok=0; failed=0
for host in $HOSTS; do
  printf '\033[36m›\033[0m %s … ' "$host"
  # Pipe the installer into the remote shell, passing config via `sudo env`.
  if echo "curl -fsSL '$INSTALLER' | sudo env PULSE_ENDPOINT='$PULSE_ENDPOINT' PULSE_KEY='$PULSE_KEY' sh" \
       | $SSH "$host" sh >/dev/null 2>&1; then
    printf '\033[32mok\033[0m\n'; ok=$((ok + 1))
  else
    printf '\033[31mfailed\033[0m\n'; failed=$((failed + 1))
  fi
done

printf '\nDone: %d ok, %d failed.\n' "$ok" "$failed"
[ "$failed" -eq 0 ]
