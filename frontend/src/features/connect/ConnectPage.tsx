import { useMemo, useState } from "react";
import { useParams } from "react-router-dom";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  Plus,
  KeyRound,
  Trash2,
  Copy,
  Check,
  ServerCog,
  Boxes,
  Activity,
  CheckCircle2,
  AlertTriangle,
  Loader2,
} from "lucide-react";
import { api } from "@/services/api";
import {
  Button,
  Card,
  CardContent,
  CardHeader,
  CardTitle,
  Input,
  Label,
  Badge,
} from "@/components/ui/primitives";
import { PageHeader, Spinner, EmptyState, Modal } from "@/components/common";
import { relativeTime } from "@/lib/utils";
import { ManagedServers } from "./ManagedServers";
import type { IngestKey } from "@/types";

// Fallback ingest base for local dev: the backend's OTLP/ingest port on the
// same host. In production the operator sets PUBLIC_INGEST_URL (served via
// /api/v1/meta) so remote agents get the externally reachable address.
const fallbackBase = `${location.protocol}//${location.hostname}:8080`;

export function ConnectPage() {
  const { projectId } = useParams();
  const qc = useQueryClient();
  const [createOpen, setCreateOpen] = useState(false);
  // `created` keeps the just-created key (with its one-time token) for the rest
  // of the session so the setup command stays filled in. `showToken` only
  // controls the show-once modal — closing it does NOT clear the filled command.
  const [created, setCreated] = useState<IngestKey | null>(null);
  const [showToken, setShowToken] = useState(false);

  const keys = useQuery({
    queryKey: ["ingest-keys", projectId],
    queryFn: () => api.listIngestKeys(projectId!),
    refetchInterval: 5000, // live: reflect a server connecting without a manual refresh
  });

  const meta = useQuery({ queryKey: ["meta"], queryFn: api.meta });
  const endpoint = meta.data?.ingest_url || fallbackBase;
  const agentImage = meta.data?.agent_image || "ghcr.io/acme/pulse-agent:latest";
  const isLocalEndpoint = /localhost|127\.0\.0\.1|backend:/.test(endpoint);

  // Connection state derived from when keys were last used.
  const lastUsedMs = (keys.data ?? [])
    .map((k) => (k.last_used_at ? new Date(k.last_used_at).getTime() : 0))
    .reduce((a, b) => Math.max(a, b), 0);
  const hasKeys = (keys.data?.length ?? 0) > 0;
  const everUsed = lastUsedMs > 0;
  const liveNow = everUsed && Date.now() - lastUsedMs < 90_000;

  const del = useMutation({
    mutationFn: (id: string) => api.deleteIngestKey(projectId!, id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["ingest-keys", projectId] }),
  });

  // The token to show in setup snippets: the freshly created one, else a placeholder.
  const token = created?.token ?? "YOUR_INGEST_KEY";

  return (
    <div>
      <PageHeader
        title="Connect a server"
        description="Install an agent on your servers — metrics and services appear here automatically"
        actions={
          <Button size="sm" onClick={() => setCreateOpen(true)}>
            <Plus className="h-4 w-4" /> New ingest key
          </Button>
        }
      />

      {/* Live connection status */}
      {hasKeys && (
        <div
          className={`flex items-center gap-2 rounded-lg border px-4 py-3 mb-4 text-sm ${
            liveNow
              ? "border-success/30 bg-success/10 text-success"
              : everUsed
                ? "border-border bg-surface text-fg-muted"
                : "border-warning/30 bg-warning/10 text-warning"
          }`}
        >
          {liveNow ? (
            <>
              <CheckCircle2 className="h-4 w-4" />
              <span className="text-fg">
                Connected — receiving data from your server
              </span>
            </>
          ) : everUsed ? (
            <>
              <Activity className="h-4 w-4" />
              <span>Last data received {relativeTime(new Date(lastUsedMs))}</span>
            </>
          ) : (
            <>
              <Loader2 className="h-4 w-4 animate-spin" />
              <span>
                Waiting for first data… run the command below on your server.
              </span>
            </>
          )}
        </div>
      )}

      {/* Local-address warning */}
      {isLocalEndpoint && (
        <div className="flex items-start gap-2 rounded-lg border border-warning/30 bg-warning/10 px-4 py-3 mb-6 text-sm text-warning">
          <AlertTriangle className="h-4 w-4 mt-0.5 shrink-0" />
          <span>
            The ingest address is <code className="font-mono">{endpoint}</code> — a
            local address that <strong>remote servers can't reach</strong>. Set{" "}
            <code className="font-mono">PUBLIC_INGEST_URL</code> to your public Pulse
            domain so the commands below work from anywhere.
          </span>
        </div>
      )}

      {/* How it works */}
      <div className="grid sm:grid-cols-3 gap-3 mb-6">
        <Step n={1} icon={<KeyRound className="h-4 w-4" />} title="Create a key">
          One key per project. It binds a server to this project.
        </Step>
        <Step n={2} icon={<ServerCog className="h-4 w-4" />} title="Run the agent">
          Paste the command on each server. No code changes needed.
        </Step>
        <Step n={3} icon={<Boxes className="h-4 w-4" />} title="See it live">
          Services auto-appear; metrics stream into the dashboard.
        </Step>
      </div>

      {/* Keys */}
      <Card className="mb-6">
        <CardHeader>
          <CardTitle>Ingest keys</CardTitle>
        </CardHeader>
        <CardContent>
          {keys.isLoading ? (
            <Spinner />
          ) : keys.data && keys.data.length > 0 ? (
            <div className="divide-y divide-border">
              {keys.data.map((k) => (
                <div key={k.id} className="flex items-center justify-between py-2.5">
                  <div className="flex items-center gap-3">
                    <KeyRound className="h-4 w-4 text-fg-muted" />
                    <div>
                      <p className="text-sm font-medium text-fg">{k.name}</p>
                      <p className="text-xs font-mono text-fg-muted">
                        {k.prefix}••••••••
                      </p>
                    </div>
                  </div>
                  <div className="flex items-center gap-3">
                    <Badge tone={k.last_used_at ? "success" : "muted"}>
                      {k.last_used_at ? `seen ${relativeTime(k.last_used_at)}` : "never used"}
                    </Badge>
                    <button
                      onClick={() => del.mutate(k.id)}
                      className="text-fg-muted hover:text-danger transition"
                      title="Revoke"
                    >
                      <Trash2 className="h-4 w-4" />
                    </button>
                  </div>
                </div>
              ))}
            </div>
          ) : (
            <EmptyState
              icon={<KeyRound className="h-7 w-7" />}
              title="No ingest keys yet"
              description="Create one to connect your first server."
            />
          )}
        </CardContent>
      </Card>

      {/* Auto-managed servers (Tailscale SSH) */}
      <ManagedServers projectId={projectId!} />

      {/* Setup instructions */}
      <SetupInstructions
        token={token}
        endpoint={endpoint}
        agentImage={agentImage}
        highlight={!!created}
      />

      {/* Create key modal */}
      <CreateKeyModal
        open={createOpen}
        projectId={projectId!}
        onClose={() => setCreateOpen(false)}
        onCreated={(key) => {
          setCreated(key);
          setShowToken(true);
          setCreateOpen(false);
          qc.invalidateQueries({ queryKey: ["ingest-keys", projectId] });
        }}
      />

      {/* Show-once token modal — closing it leaves the command below filled in */}
      <Modal open={showToken} onClose={() => setShowToken(false)} title="Copy your ingest key">
        <p className="text-sm text-fg-muted mb-3">
          This is the only time the full key is shown. The setup command below stays
          filled in with it until you leave this page.
        </p>
        {created?.token && <CopyBox value={created.token} />}
        <div className="flex justify-end mt-4">
          <Button size="sm" onClick={() => setShowToken(false)}>
            Done
          </Button>
        </div>
      </Modal>
    </div>
  );
}

function Step({
  n,
  icon,
  title,
  children,
}: {
  n: number;
  icon: React.ReactNode;
  title: string;
  children: React.ReactNode;
}) {
  return (
    <Card className="p-4">
      <div className="flex items-center gap-2 text-fg">
        <span className="flex h-5 w-5 items-center justify-center rounded-full bg-primary/15 text-[11px] font-semibold text-primary">
          {n}
        </span>
        {icon}
        <span className="text-sm font-medium">{title}</span>
      </div>
      <p className="text-xs text-fg-muted mt-2">{children}</p>
    </Card>
  );
}

const TABS = ["Host metrics", "App metrics", "App (zero-code)", "Prometheus"] as const;
type Tab = (typeof TABS)[number];

function SetupInstructions({
  token,
  endpoint,
  agentImage,
  highlight,
}: {
  token: string;
  endpoint: string;
  agentImage: string;
  highlight: boolean;
}) {
  const [tab, setTab] = useState<Tab>("Host metrics");

  const snippet = useMemo(() => {
    switch (tab) {
      case "Host metrics":
        return `docker run -d --name pulse-agent \\
  -e PULSE_ENDPOINT=${endpoint} \\
  -e PULSE_KEY=${token} \\
  -e PULSE_SERVICE=my-server \\
  -e HOST_PROC=/host/proc -e HOST_SYS=/host/sys \\
  -v /proc:/host/proc:ro -v /sys:/host/sys:ro \\
  ${agentImage}`;
      case "App metrics":
        return `# Push your app's RED metrics (request rate, errors, latency).
# Works from any language — POST JSON with your ingest key:
curl -X POST ${endpoint}/api/v1/ingest/metrics \\
  -H "X-Pulse-Key: ${token}" \\
  -H "Content-Type: application/json" \\
  -d '{"points":[
    {"service":"my-app","metric":"request_rate","value":58},
    {"service":"my-app","metric":"error_rate","value":1.2},
    {"service":"my-app","metric":"latency_p95","value":142}
  ]}'

# Go service? Drop in a ready reporter (no dependencies):
#   curl -fsSL https://raw.githubusercontent.com/roman0309/pulse/main/examples/go-instrumentation/pulse.go -o pulse/pulse.go
#   pulse.Start("${endpoint}", "${token}", "my-app")
#   http.ListenAndServe(addr, pulse.Middleware(mux))`;
      case "App (zero-code)":
        return `# Zero-code app metrics via eBPF (Grafana Beyla) — no code changes.
# Run on the same host as your app. Set BEYLA_OPEN_PORT to the app's listen port.
docker run -d --name pulse-beyla --restart unless-stopped \\
  --privileged --pid=host \\
  -e BEYLA_OPEN_PORT=8080 \\
  -e OTEL_SERVICE_NAME=my-app \\
  -e OTEL_EXPORTER_OTLP_ENDPOINT=${endpoint}/otlp \\
  -e OTEL_EXPORTER_OTLP_HEADERS=X-Pulse-Key=${token} \\
  -e OTEL_EXPORTER_OTLP_METRICS_TEMPORALITY_PREFERENCE=delta \\
  grafana/beyla:latest`;
      case "Prometheus":
        return `# prometheus.yml
remote_write:
  - url: ${endpoint}/api/v1/prom/write
    headers:
      X-Pulse-Key: ${token}`;
    }
  }, [tab, token, endpoint, agentImage]);

  return (
    <Card className={highlight ? "border-primary/40" : ""}>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <Activity className="h-4 w-4 text-primary" />
          Setup instructions
        </CardTitle>
      </CardHeader>
      <CardContent>
        <div className="flex gap-1 mb-3">
          {TABS.map((t) => (
            <button
              key={t}
              onClick={() => setTab(t)}
              className={`rounded-md px-3 h-8 text-xs font-medium transition ${
                tab === t
                  ? "bg-surface-2 text-fg"
                  : "text-fg-muted hover:text-fg"
              }`}
            >
              {t}
            </button>
          ))}
        </div>
        <CopyBox value={snippet} />
        {token === "YOUR_INGEST_KEY" && (
          <p className="text-xs text-fg-muted mt-2">
            Create an ingest key above to fill in the command automatically.
          </p>
        )}
      </CardContent>
    </Card>
  );
}

function CopyBox({ value }: { value: string }) {
  const [copied, setCopied] = useState(false);
  const copy = async () => {
    await navigator.clipboard.writeText(value);
    setCopied(true);
    setTimeout(() => setCopied(false), 1500);
  };
  return (
    <div className="relative">
      <pre className="overflow-x-auto rounded-md border border-border bg-surface-2 p-3 pr-12 text-xs font-mono text-fg whitespace-pre-wrap break-all">
        {value}
      </pre>
      <button
        onClick={copy}
        className="absolute right-2 top-2 rounded-md border border-border bg-surface p-1.5 text-fg-muted hover:text-fg transition"
        title="Copy"
      >
        {copied ? (
          <Check className="h-3.5 w-3.5 text-success" />
        ) : (
          <Copy className="h-3.5 w-3.5" />
        )}
      </button>
    </div>
  );
}

function CreateKeyModal({
  open,
  projectId,
  onClose,
  onCreated,
}: {
  open: boolean;
  projectId: string;
  onClose: () => void;
  onCreated: (key: IngestKey) => void;
}) {
  const [name, setName] = useState("");
  const mutation = useMutation({
    mutationFn: () => api.createIngestKey(projectId, name || "server"),
    onSuccess: onCreated,
  });
  return (
    <Modal open={open} onClose={onClose} title="New ingest key">
      <div className="space-y-4">
        <div className="space-y-1.5">
          <Label>Name</Label>
          <Input
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="prod-server-1"
            autoFocus
          />
          <p className="text-xs text-fg-muted">
            A label to recognise this key (e.g. the server or environment).
          </p>
        </div>
        <div className="flex justify-end gap-2">
          <Button variant="ghost" size="sm" onClick={onClose}>
            Cancel
          </Button>
          <Button size="sm" disabled={mutation.isPending} onClick={() => mutation.mutate()}>
            Create key
          </Button>
        </div>
      </div>
    </Modal>
  );
}
