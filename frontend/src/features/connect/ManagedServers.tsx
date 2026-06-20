import { useEffect, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  Plus,
  ServerCog,
  Trash2,
  Download,
  StopCircle,
  RefreshCw,
  RotateCw,
  Terminal,
  ChevronRight,
  FileText,
  Boxes,
  HardDrive,
  MemoryStick,
  Cpu,
  Activity,
  Network,
  Gauge,
  Copy,
  Check,
  X,
} from "lucide-react";
import { cn, relativeTime } from "@/lib/utils";
import { api } from "@/services/api";
import { toast } from "@/lib/toast";
import {
  Button,
  Card,
  CardContent,
  CardHeader,
  CardTitle,
  Input,
  Label,
  Select,
  Badge,
} from "@/components/ui/primitives";
import { EmptyState, Spinner, Modal } from "@/components/common";
import type { ManagedServer } from "@/types";

const statusTone: Record<string, "success" | "warning" | "danger" | "muted"> = {
  installed: "success",
  pending: "muted",
  removed: "warning",
  error: "danger",
};

// Diagnostic one-liners run over the same SSH channel. All read-only / safe.
const QUICK: { label: string; icon: React.ReactNode; cmd: string }[] = [
  { label: "Connection", icon: <Network className="h-3.5 w-3.5" />, cmd: `whoami && hostname && echo "CONNECTED"` },
  { label: "Containers", icon: <Boxes className="h-3.5 w-3.5" />, cmd: `docker ps -a --format 'table {{.Names}}\\t{{.Status}}\\t{{.Image}}'` },
  { label: "Disk", icon: <HardDrive className="h-3.5 w-3.5" />, cmd: "df -h" },
  { label: "Memory", icon: <MemoryStick className="h-3.5 w-3.5" />, cmd: "free -h" },
  { label: "CPU / load", icon: <Cpu className="h-3.5 w-3.5" />, cmd: "uptime && echo && top -bn1 | head -n 12" },
  { label: "Docker", icon: <Activity className="h-3.5 w-3.5" />, cmd: `systemctl is-active docker; docker version --format 'server {{.Server.Version}}'` },
];

// Managed servers: Pulse connects over SSH with stored (encrypted) credentials
// and can install the agent or run any command — all from the UI.
export function ManagedServers({ projectId }: { projectId: string }) {
  const qc = useQueryClient();
  const [modal, setModal] = useState(false);

  const servers = useQuery({
    queryKey: ["servers", projectId],
    queryFn: () => api.listServers(projectId),
  });
  const invalidate = () => {
    // Server actions can create/remove services + their metrics, so refresh
    // those views too.
    qc.invalidateQueries({ queryKey: ["servers", projectId] });
    qc.invalidateQueries({ queryKey: ["services", projectId] });
    qc.invalidateQueries({ queryKey: ["dashboard", projectId] });
  };

  const del = useMutation({
    mutationFn: (id: string) => api.deleteServer(projectId, id),
    onSuccess: () => {
      invalidate();
      toast.success("Server removed");
    },
  });

  const count = servers.data?.length ?? 0;

  return (
    <Card className="mb-6">
      <CardHeader className="flex flex-row items-center justify-between">
        <CardTitle className="flex items-center gap-2">
          <ServerCog className="h-4 w-4" /> Servers (SSH)
          {count > 0 && <span className="text-xs font-normal text-fg-muted">· {count}</span>}
        </CardTitle>
        <Button size="sm" onClick={() => setModal(true)}>
          <Plus className="h-4 w-4" /> Add server
        </Button>
      </CardHeader>
      <CardContent>
        {servers.isLoading ? (
          <Spinner />
        ) : servers.data && servers.data.length > 0 ? (
          <div className="space-y-2">
            {servers.data.map((s) => (
              <ServerRow
                key={s.id}
                projectId={projectId}
                server={s}
                onChange={invalidate}
                onDelete={() => del.mutate(s.id)}
              />
            ))}
          </div>
        ) : (
          <EmptyState
            icon={<ServerCog className="h-7 w-7" />}
            title="No servers yet"
            description="Add a server with SSH credentials, then install the agent and manage it from here."
          />
        )}
      </CardContent>

      <AddServerModal
        open={modal}
        projectId={projectId}
        onClose={() => setModal(false)}
        onAdded={() => {
          invalidate();
          setModal(false);
        }}
      />
    </Card>
  );
}

function ServerRow({
  projectId,
  server,
  onChange,
  onDelete,
}: {
  projectId: string;
  server: ManagedServer;
  onChange: () => void;
  onDelete: () => void;
}) {
  const [open, setOpen] = useState(false);
  const [cmd, setCmd] = useState("");
  const [ports, setPorts] = useState("8080");
  const [out, setOut] = useState(server.last_result || "");
  const [label, setLabel] = useState(server.last_result ? "Last result" : "");
  const [at, setAt] = useState<Date | null>(null);
  const [copied, setCopied] = useState(false);

  const onResult = (text: string, refetch = false) => {
    setOut(text);
    setAt(new Date());
    if (refetch) onChange();
  };
  const errText = (e: unknown) => (e instanceof Error ? e.message : "error");

  // statusOut holds the latest `docker ps` for pulse-agent/pulse-beyla;
  // null = not checked yet (pills show "checking…").
  const [statusOut, setStatusOut] = useState<string | null>(null);

  // Refresh the live container status. Silent by default (updates the pills
  // only); showOutput renders it in the Output panel too.
  const statusCheck = useMutation({
    mutationFn: (_v?: { showOutput?: boolean }) => api.serverAction(projectId, server.id, "status"),
    onSuccess: (s, v) => {
      setStatusOut(s.last_result ?? "");
      onChange();
      if (v?.showOutput) {
        setLabel("Status");
        setOut(s.last_result || "(no Pulse containers running)");
        setAt(new Date());
      }
    },
    onError: (e, v) => {
      if (v?.showOutput) onResult(errText(e));
    },
  });
  const refreshStatus = () => statusCheck.mutate({ showOutput: false });

  const action = useMutation({
    mutationFn: (act: "install" | "remove") => api.serverAction(projectId, server.id, act),
    onSuccess: (s) => {
      onResult(s.last_result || s.status, true);
      refreshStatus();
    },
    onError: (e) => onResult(errText(e)),
  });
  const run = useMutation({
    mutationFn: (command: string) => api.runServerCommand(projectId, server.id, command),
    onSuccess: (s) => onResult(s.last_result || "(no output)"),
    onError: (e) => onResult(errText(e)),
  });
  const beyla = useMutation({
    mutationFn: (p: string) => api.installBeyla(projectId, server.id, p),
    onSuccess: (s) => {
      onResult(s.last_result || "(no output)", true);
      refreshStatus();
    },
    onError: (e) => onResult(errText(e)),
  });
  const beylaRemove = useMutation({
    mutationFn: () => api.removeBeyla(projectId, server.id),
    onSuccess: (s) => {
      onResult(s.last_result || "(no output)", true);
      refreshStatus();
    },
    onError: (e) => onResult(errText(e)),
  });
  const busy = action.isPending || run.isPending || beyla.isPending || beylaRemove.isPending || statusCheck.isPending;

  // Auto-check status the first time the console is opened.
  useEffect(() => {
    if (open && statusOut === null && !statusCheck.isPending) refreshStatus();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open]);

  const agentState = statusOut === null ? "unknown" : statusOut.includes("pulse-agent") ? "running" : "stopped";
  const beylaState = statusOut === null ? "unknown" : statusOut.includes("pulse-beyla") ? "running" : "stopped";

  function act(lbl: string, a: "install" | "remove") {
    setLabel(lbl);
    setOpen(true);
    action.mutate(a);
  }
  function exec(lbl: string, command: string) {
    if (!command.trim()) return;
    setLabel(lbl);
    setOpen(true);
    run.mutate(command);
  }
  function installBeyla() {
    if (!ports.trim()) return;
    setLabel("Install app metrics (Beyla)");
    setOpen(true);
    beyla.mutate(ports);
  }
  function removeBeyla() {
    if (!confirm("Remove app metrics (Beyla)?\n\nThis stops pulse-beyla and deletes the services, charts and metrics it created.")) return;
    setLabel("Remove app metrics (Beyla)");
    setOpen(true);
    beylaRemove.mutate();
  }
  function copyOut() {
    navigator.clipboard.writeText(out).then(() => {
      setCopied(true);
      setTimeout(() => setCopied(false), 1200);
    });
  }

  return (
    <div className="rounded-lg border border-border">
      {/* Header — always visible */}
      <div className="flex items-center justify-between gap-3 p-3 flex-wrap">
        <button onClick={() => setOpen((o) => !o)} className="flex min-w-0 items-center gap-2 text-left">
          <ChevronRight className={cn("h-4 w-4 shrink-0 text-fg-muted transition-transform", open && "rotate-90")} />
          <div className="min-w-0">
            <div className="flex items-center gap-2">
              <span className="text-sm font-medium text-fg">{server.name}</span>
              <Badge tone={statusTone[server.status] ?? "muted"}>{server.status}</Badge>
            </div>
            <p className="truncate font-mono text-xs text-fg-muted">{server.ssh_target}</p>
          </div>
        </button>
        <div className="flex shrink-0 items-center gap-1.5">
          <Button
            variant="outline"
            size="sm"
            disabled={busy}
            onClick={() => {
              setOpen(true);
              statusCheck.mutate({ showOutput: true });
            }}
            title="Check containers (pulse-agent, pulse-beyla)"
          >
            <RefreshCw className={cn("h-3.5 w-3.5", busy && "animate-spin")} /> Status
          </Button>
          <button
            onClick={() => {
              if (
                confirm(
                  `Delete server "${server.name}"?\n\nThis removes it from Pulse, revokes its ingest keys, stops its agents, and deletes the services & metrics they created.`
                )
              )
                onDelete();
            }}
            className="rounded p-1.5 text-fg-muted transition hover:bg-surface-2 hover:text-danger"
            title="Delete server"
          >
            <Trash2 className="h-4 w-4" />
          </button>
        </div>
      </div>

      {/* Management console */}
      {open && (
        <div className="space-y-4 border-t border-border p-3">
          {/* Two independent components, each with its own lifecycle */}
          <div className="grid gap-3 md:grid-cols-2">
            {/* Host agent */}
            <CompCard
              icon={<ServerCog className="h-4 w-4" />}
              title="Host agent"
              desc="CPU · memory · disk · network"
              status={<StatusPill state={agentState} />}
            >
              <Button variant="primary" size="sm" disabled={busy} onClick={() => act("Install / update host agent", "install")}>
                <Download className="h-3.5 w-3.5" /> Install / update
              </Button>
              <Button variant="ghost" size="sm" disabled={busy} onClick={() => exec("Restart host agent", "docker restart pulse-agent 2>&1 || echo 'pulse-agent not found'")}>
                <RotateCw className="h-3.5 w-3.5" /> Restart
              </Button>
              <Button variant="ghost" size="sm" disabled={busy} onClick={() => exec("Host agent logs", "docker logs --tail 200 pulse-agent 2>&1 || echo 'pulse-agent not found'")}>
                <FileText className="h-3.5 w-3.5" /> Logs
              </Button>
              <Button
                variant="ghost"
                size="sm"
                disabled={busy}
                className="text-danger hover:text-danger"
                onClick={() => {
                  if (
                    confirm(
                      "Remove the host agent?\n\nThis stops pulse-agent and deletes the host/container services and metrics it created."
                    )
                  )
                    act("Remove host agent", "remove");
                }}
              >
                <StopCircle className="h-3.5 w-3.5" /> Remove
              </Button>
            </CompCard>

            {/* App metrics (Beyla, zero-code) */}
            <CompCard
              icon={<Gauge className="h-4 w-4" />}
              title="App metrics — Beyla"
              desc="zero-code RED metrics, no app changes"
              status={<StatusPill state={beylaState} />}
            >
              <div className="flex w-full items-center gap-1.5">
                <Input
                  value={ports}
                  onChange={(e) => setPorts(e.target.value)}
                  placeholder="8080,8083,8084,8085"
                  title="App ports to instrument (comma-separated or a range like 8080-8090)"
                  className="h-8 flex-1 font-mono text-xs"
                />
                <span className="shrink-0 text-[11px] text-fg-muted">ports</span>
              </div>
              <Button variant="primary" size="sm" disabled={busy || !ports.trim()} onClick={installBeyla}>
                <Download className="h-3.5 w-3.5" /> Install / update
              </Button>
              <Button variant="ghost" size="sm" disabled={busy} onClick={() => exec("Restart Beyla", "docker restart pulse-beyla 2>&1 || echo 'pulse-beyla not found'")}>
                <RotateCw className="h-3.5 w-3.5" /> Restart
              </Button>
              <Button variant="ghost" size="sm" disabled={busy} onClick={() => exec("Beyla logs", "docker logs --tail 200 pulse-beyla 2>&1 || echo 'pulse-beyla not found'")}>
                <FileText className="h-3.5 w-3.5" /> Logs
              </Button>
              <Button variant="ghost" size="sm" disabled={busy} className="text-danger hover:text-danger" onClick={removeBeyla}>
                <StopCircle className="h-3.5 w-3.5" /> Remove
              </Button>
            </CompCard>
          </div>

          {/* Diagnostics */}
          <Section title="Diagnostics">
            {QUICK.map((q) => (
              <Button key={q.label} variant="ghost" size="sm" disabled={busy} onClick={() => exec(q.label, q.cmd)} title={q.cmd}>
                {q.icon} {q.label}
              </Button>
            ))}
          </Section>

          {/* Free-form command */}
          <Section title="Run command">
            <div className="flex w-full items-center gap-1.5">
              <Terminal className="h-3.5 w-3.5 shrink-0 text-fg-muted" />
              <Input
                value={cmd}
                onChange={(e) => setCmd(e.target.value)}
                placeholder="run any command on this server…"
                className="h-8 font-mono text-xs"
                onKeyDown={(e) => {
                  if (e.key === "Enter" && cmd && !busy) exec(cmd, cmd);
                }}
              />
              <Button variant="outline" size="sm" disabled={busy || !cmd} onClick={() => exec(cmd, cmd)}>
                Run
              </Button>
            </div>
          </Section>

          {/* Output */}
          {(busy || out) && (
            <div>
              <div className="mb-1.5 flex items-center justify-between">
                <p className="text-[11px] font-semibold uppercase tracking-wide text-fg-muted">
                  Output
                  {label && <span className="font-normal normal-case text-fg-muted"> · {label}</span>}
                  {at && !busy && <span className="font-normal normal-case text-fg-muted"> · {relativeTime(at)}</span>}
                </p>
                {!busy && out && (
                  <div className="flex items-center gap-1">
                    <button onClick={copyOut} className="rounded p-1 text-fg-muted transition hover:text-fg" title="Copy">
                      {copied ? <Check className="h-3.5 w-3.5 text-success" /> : <Copy className="h-3.5 w-3.5" />}
                    </button>
                    <button onClick={() => { setOut(""); setLabel(""); }} className="rounded p-1 text-fg-muted transition hover:text-fg" title="Clear">
                      <X className="h-3.5 w-3.5" />
                    </button>
                  </div>
                )}
              </div>
              {busy ? (
                <p className="text-xs text-fg-muted">running…</p>
              ) : (
                <pre className="max-h-72 overflow-auto whitespace-pre-wrap break-all rounded bg-surface-2 p-2 text-[11px] font-mono text-fg-muted">
                  {out}
                </pre>
              )}
            </div>
          )}
        </div>
      )}
    </div>
  );
}

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div>
      <p className="mb-1.5 text-[11px] font-semibold uppercase tracking-wide text-fg-muted">{title}</p>
      <div className="flex flex-wrap items-center gap-1.5">{children}</div>
    </div>
  );
}

// CompCard groups one server component (host agent / Beyla) with its own
// description, a live status pill and a row of lifecycle actions.
function CompCard({
  icon,
  title,
  desc,
  status,
  children,
}: {
  icon: React.ReactNode;
  title: string;
  desc: string;
  status?: React.ReactNode;
  children: React.ReactNode;
}) {
  return (
    <div className="rounded-md border border-border bg-surface-2/40 p-3">
      <div className="flex items-start justify-between gap-2">
        <div className="flex items-start gap-2">
          <span className="mt-0.5 text-fg-muted">{icon}</span>
          <div className="min-w-0">
            <p className="text-sm font-medium text-fg">{title}</p>
            <p className="text-xs text-fg-muted">{desc}</p>
          </div>
        </div>
        {status && <div className="shrink-0">{status}</div>}
      </div>
      <div className="mt-2.5 flex flex-wrap items-center gap-1.5">{children}</div>
    </div>
  );
}

function StatusPill({ state }: { state: "running" | "stopped" | "unknown" }) {
  if (state === "unknown") {
    return <span className="text-[11px] text-fg-muted">checking…</span>;
  }
  return <Badge tone={state === "running" ? "success" : "muted"}>{state}</Badge>;
}

function AddServerModal({
  open,
  projectId,
  onClose,
  onAdded,
}: {
  open: boolean;
  projectId: string;
  onClose: () => void;
  onAdded: () => void;
}) {
  const [f, setF] = useState({
    name: "",
    host: "",
    port: "22",
    user: "pulse",
    auth_method: "key",
    secret: "",
  });
  const set = (k: string, v: string) => setF((p) => ({ ...p, [k]: v }));
  const [err, setErr] = useState("");

  const mutation = useMutation({
    mutationFn: () =>
      api.addServer(projectId, {
        name: f.name.trim(),
        host: f.host.trim(),
        port: Number(f.port) || 22,
        user: f.user.trim(),
        auth_method: f.auth_method,
        secret: f.secret.trim(),
      }),
    onSuccess: () => {
      toast.success("Server added");
      onAdded();
    },
    onError: (e) => setErr(e instanceof Error ? e.message : "failed"),
  });

  return (
    <Modal open={open} onClose={onClose} title="Add server (SSH)">
      <div className="space-y-3">
        <div className="grid grid-cols-3 gap-2">
          <div className="col-span-2 space-y-1.5">
            <Label>Host / IP</Label>
            <Input value={f.host} onChange={(e) => set("host", e.target.value)} placeholder="203.0.113.5 or host.tailnet.ts.net" autoFocus />
          </div>
          <div className="space-y-1.5">
            <Label>Port</Label>
            <Input value={f.port} onChange={(e) => set("port", e.target.value)} />
          </div>
        </div>
        <div className="grid grid-cols-2 gap-2">
          <div className="space-y-1.5">
            <Label>User</Label>
            <Input value={f.user} onChange={(e) => set("user", e.target.value)} placeholder="pulse" />
          </div>
          <div className="space-y-1.5">
            <Label>Name (optional)</Label>
            <Input value={f.name} onChange={(e) => set("name", e.target.value)} placeholder="messenger" />
          </div>
        </div>
        <div className="space-y-1.5">
          <Label>Auth method</Label>
          <Select className="w-full" value={f.auth_method} onChange={(e) => set("auth_method", e.target.value)}>
            <option value="key">Private key</option>
            <option value="password">Password</option>
          </Select>
        </div>
        <div className="space-y-1.5">
          <Label>{f.auth_method === "key" ? "Private key (PEM)" : "Password"}</Label>
          {f.auth_method === "key" ? (
            <textarea
              value={f.secret}
              onChange={(e) => set("secret", e.target.value)}
              rows={4}
              className="w-full rounded-md border border-border bg-surface-2 px-3 py-2 text-xs font-mono text-fg focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/50"
              placeholder={"-----BEGIN OPENSSH PRIVATE KEY-----\n…paste the whole key, including BEGIN/END lines…\n-----END OPENSSH PRIVATE KEY-----"}
            />
          ) : (
            <Input type="password" value={f.secret} onChange={(e) => set("secret", e.target.value)} />
          )}
          <p className="text-xs text-fg-muted">
            Stored encrypted (AES-256-GCM). Use the dedicated <code className="font-mono">pulse</code> user from Step 1 — never root.
          </p>
        </div>
        {err && <p className="text-xs text-danger">{err}</p>}
        <div className="flex justify-end gap-2">
          <Button variant="ghost" size="sm" onClick={onClose}>
            Cancel
          </Button>
          <Button
            size="sm"
            disabled={!f.host.trim() || !f.user.trim() || !f.secret.trim() || mutation.isPending}
            onClick={() => {
              setErr("");
              mutation.mutate();
            }}
          >
            Add server
          </Button>
        </div>
      </div>
    </Modal>
  );
}
