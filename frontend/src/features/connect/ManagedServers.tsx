import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  Plus,
  ServerCog,
  Trash2,
  Download,
  StopCircle,
  RefreshCw,
  Terminal,
  FileText,
  RotateCw,
  Boxes,
  HardDrive,
  MemoryStick,
  Clock,
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

// Managed servers: Pulse connects over SSH with stored (encrypted) credentials
// and can install the agent or run any command — all from the UI.
export function ManagedServers({ projectId }: { projectId: string }) {
  const qc = useQueryClient();
  const [modal, setModal] = useState(false);

  const servers = useQuery({
    queryKey: ["servers", projectId],
    queryFn: () => api.listServers(projectId),
  });
  const invalidate = () => qc.invalidateQueries({ queryKey: ["servers", projectId] });

  const del = useMutation({
    mutationFn: (id: string) => api.deleteServer(projectId, id),
    onSuccess: invalidate,
  });

  return (
    <Card className="mb-6">
      <CardHeader className="flex flex-row items-center justify-between">
        <CardTitle className="flex items-center gap-2">
          <ServerCog className="h-4 w-4" /> Servers (SSH)
          <span className="text-xs font-normal text-fg-muted">· install &amp; run commands</span>
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
              <ServerRow key={s.id} projectId={projectId} server={s} onChange={invalidate} onDelete={() => del.mutate(s.id)} />
            ))}
          </div>
        ) : (
          <EmptyState
            icon={<ServerCog className="h-7 w-7" />}
            title="No servers"
            description="Add a server with SSH credentials, then install the agent or run commands from here."
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
  const [cmd, setCmd] = useState("");
  const [out, setOut] = useState(server.last_result || "");

  const action = useMutation({
    mutationFn: (act: "install" | "remove" | "status") => api.serverAction(projectId, server.id, act),
    onSuccess: (s) => {
      setOut(s.last_result || s.status);
      onChange();
    },
    onError: (e) => setOut(e instanceof Error ? e.message : "error"),
  });
  const run = useMutation({
    mutationFn: (command: string) => api.runServerCommand(projectId, server.id, command),
    onSuccess: (s) => setOut(s.last_result || "(no output)"),
    onError: (e) => setOut(e instanceof Error ? e.message : "error"),
  });
  const busy = action.isPending || run.isPending;

  // Predefined one-click operations, executed via the same SSH run channel.
  const quick: { label: string; icon: React.ReactNode; cmd: string }[] = [
    { label: "Logs", icon: <FileText className="h-3.5 w-3.5" />, cmd: "docker logs --tail 120 pulse-agent 2>&1" },
    { label: "Restart", icon: <RotateCw className="h-3.5 w-3.5" />, cmd: "docker restart pulse-agent && echo restarted" },
    { label: "Containers", icon: <Boxes className="h-3.5 w-3.5" />, cmd: "docker ps --format 'table {{.Names}}\\t{{.Status}}\\t{{.Image}}'" },
    { label: "Disk", icon: <HardDrive className="h-3.5 w-3.5" />, cmd: "df -h /" },
    { label: "Memory", icon: <MemoryStick className="h-3.5 w-3.5" />, cmd: "free -h" },
    { label: "Uptime", icon: <Clock className="h-3.5 w-3.5" />, cmd: "uptime && echo && uname -a" },
  ];

  return (
    <div className="rounded-md border border-border p-3">
      <div className="flex items-center justify-between gap-3 flex-wrap">
        <div className="min-w-0">
          <div className="flex items-center gap-2">
            <span className="text-sm font-medium text-fg">{server.name}</span>
            <Badge tone={statusTone[server.status] ?? "muted"}>{server.status}</Badge>
          </div>
          <p className="text-xs font-mono text-fg-muted truncate">{server.ssh_target}</p>
        </div>
        <div className="flex items-center gap-1.5 shrink-0">
          <Button variant="outline" size="sm" disabled={busy} onClick={() => action.mutate("install")}>
            <Download className="h-3.5 w-3.5" /> Install
          </Button>
          <Button variant="ghost" size="sm" disabled={busy} onClick={() => action.mutate("status")} title="Status">
            <RefreshCw className="h-3.5 w-3.5" />
          </Button>
          <Button variant="ghost" size="sm" disabled={busy} onClick={() => action.mutate("remove")} title="Remove agent">
            <StopCircle className="h-3.5 w-3.5" />
          </Button>
          <button onClick={onDelete} className="text-fg-muted hover:text-danger transition" title="Delete server">
            <Trash2 className="h-4 w-4" />
          </button>
        </div>
      </div>

      {/* Quick actions — one-click ops over SSH */}
      <div className="mt-2 flex flex-wrap items-center gap-1.5">
        {quick.map((q) => (
          <Button
            key={q.label}
            variant="ghost"
            size="sm"
            disabled={busy}
            onClick={() => run.mutate(q.cmd)}
            title={q.cmd}
          >
            {q.icon} {q.label}
          </Button>
        ))}
      </div>

      {/* Run arbitrary command */}
      <div className="mt-2 flex items-center gap-1.5">
        <Terminal className="h-3.5 w-3.5 text-fg-muted shrink-0" />
        <Input
          value={cmd}
          onChange={(e) => setCmd(e.target.value)}
          placeholder="run a command on this server…"
          className="h-8 text-xs font-mono"
          onKeyDown={(e) => {
            if (e.key === "Enter" && cmd && !busy) run.mutate(cmd);
          }}
        />
        <Button variant="outline" size="sm" disabled={busy || !cmd} onClick={() => run.mutate(cmd)}>
          Run
        </Button>
      </div>

      {busy && <p className="text-xs text-fg-muted mt-2">running…</p>}
      {out && !busy && (
        <pre className="mt-2 max-h-40 overflow-auto rounded bg-surface-2 p-2 text-[11px] font-mono text-fg-muted whitespace-pre-wrap break-all">
          {out}
        </pre>
      )}
    </div>
  );
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
        name: f.name,
        host: f.host,
        port: Number(f.port) || 22,
        user: f.user,
        auth_method: f.auth_method,
        secret: f.secret,
      }),
    onSuccess: onAdded,
    onError: (e) => setErr(e instanceof Error ? e.message : "failed"),
  });

  return (
    <Modal open={open} onClose={onClose} title="Add server (SSH)">
      <div className="space-y-3">
        <div className="grid grid-cols-3 gap-2">
          <div className="space-y-1.5 col-span-2">
            <Label>Host / IP</Label>
            <Input value={f.host} onChange={(e) => set("host", e.target.value)} placeholder="10.0.0.5 or host.tailnet.ts.net" autoFocus />
          </div>
          <div className="space-y-1.5">
            <Label>Port</Label>
            <Input value={f.port} onChange={(e) => set("port", e.target.value)} />
          </div>
        </div>
        <div className="grid grid-cols-2 gap-2">
          <div className="space-y-1.5">
            <Label>User</Label>
            <Input value={f.user} onChange={(e) => set("user", e.target.value)} />
          </div>
          <div className="space-y-1.5">
            <Label>Name (optional)</Label>
            <Input value={f.name} onChange={(e) => set("name", e.target.value)} placeholder="messenger" />
          </div>
        </div>
        <div className="space-y-1.5">
          <Label>Auth method</Label>
          <Select className="w-full" value={f.auth_method} onChange={(e) => set("auth_method", e.target.value)}>
            <option value="password">Password</option>
            <option value="key">Private key</option>
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
              placeholder="-----BEGIN OPENSSH PRIVATE KEY-----"
            />
          ) : (
            <Input type="password" value={f.secret} onChange={(e) => set("secret", e.target.value)} />
          )}
          <p className="text-xs text-fg-muted">Stored encrypted (AES-GCM). Pulse will SSH in to run commands.</p>
        </div>
        {err && <p className="text-xs text-danger">{err}</p>}
        <div className="flex justify-end gap-2">
          <Button variant="ghost" size="sm" onClick={onClose}>Cancel</Button>
          <Button size="sm" disabled={!f.host || !f.user || !f.secret || mutation.isPending} onClick={() => { setErr(""); mutation.mutate(); }}>
            Add
          </Button>
        </div>
      </div>
    </Modal>
  );
}
