import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Plus, ServerCog, Trash2, Download, StopCircle, RefreshCw } from "lucide-react";
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
import { EmptyState, Spinner, Modal } from "@/components/common";
import type { ManagedServer } from "@/types";

const statusTone: Record<string, "success" | "warning" | "danger" | "muted"> = {
  installed: "success",
  pending: "muted",
  removed: "warning",
  error: "danger",
};

export function ManagedServers({ projectId }: { projectId: string }) {
  const qc = useQueryClient();
  const [modal, setModal] = useState(false);

  const servers = useQuery({
    queryKey: ["servers", projectId],
    queryFn: () => api.listServers(projectId),
  });
  const invalidate = () => qc.invalidateQueries({ queryKey: ["servers", projectId] });

  const action = useMutation({
    mutationFn: ({ id, act }: { id: string; act: "install" | "remove" | "status" }) =>
      api.serverAction(projectId, id, act),
    onSuccess: invalidate,
  });
  const del = useMutation({
    mutationFn: (id: string) => api.deleteServer(projectId, id),
    onSuccess: invalidate,
  });

  const busyId = action.isPending ? action.variables?.id : undefined;

  return (
    <Card className="mb-6">
      <CardHeader className="flex flex-row items-center justify-between">
        <CardTitle className="flex items-center gap-2">
          <ServerCog className="h-4 w-4" /> Servers (auto-manage via Tailscale SSH)
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
              <div key={s.id} className="rounded-md border border-border p-3">
                <div className="flex items-center justify-between gap-3">
                  <div className="min-w-0">
                    <div className="flex items-center gap-2">
                      <span className="text-sm font-medium text-fg">{s.name}</span>
                      <Badge tone={statusTone[s.status] ?? "muted"}>{s.status}</Badge>
                    </div>
                    <p className="text-xs font-mono text-fg-muted truncate">{s.ssh_target}</p>
                  </div>
                  <div className="flex items-center gap-1.5 shrink-0">
                    <Button variant="outline" size="sm" disabled={busyId === s.id}
                      onClick={() => action.mutate({ id: s.id, act: "install" })}>
                      <Download className="h-3.5 w-3.5" /> Install
                    </Button>
                    <Button variant="ghost" size="sm" disabled={busyId === s.id}
                      onClick={() => action.mutate({ id: s.id, act: "status" })} title="Check status">
                      <RefreshCw className="h-3.5 w-3.5" />
                    </Button>
                    <Button variant="ghost" size="sm" disabled={busyId === s.id}
                      onClick={() => action.mutate({ id: s.id, act: "remove" })} title="Remove agent">
                      <StopCircle className="h-3.5 w-3.5" />
                    </Button>
                    <button onClick={() => del.mutate(s.id)} className="text-fg-muted hover:text-danger transition" title="Delete">
                      <Trash2 className="h-4 w-4" />
                    </button>
                  </div>
                </div>
                {s.last_result && (
                  <pre className="mt-2 max-h-32 overflow-auto rounded bg-surface-2 p-2 text-[11px] font-mono text-fg-muted whitespace-pre-wrap break-all">
                    {s.last_result}
                  </pre>
                )}
              </div>
            ))}
          </div>
        ) : (
          <EmptyState
            icon={<ServerCog className="h-7 w-7" />}
            title="No managed servers"
            description="Add a server (Tailscale name) and install the agent with one click — no SSH keys stored."
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
  const [name, setName] = useState("");
  const [target, setTarget] = useState("");
  const [err, setErr] = useState("");
  const mutation = useMutation({
    mutationFn: () => api.addServer(projectId, name, target),
    onSuccess: onAdded,
    onError: (e) => setErr(e instanceof Error ? e.message : "failed"),
  });
  return (
    <Modal open={open} onClose={onClose} title="Add server">
      <div className="space-y-4">
        <div className="space-y-1.5">
          <Label>Name (service name in Pulse)</Label>
          <Input value={name} onChange={(e) => setName(e.target.value)} placeholder="messenger" autoFocus />
        </div>
        <div className="space-y-1.5">
          <Label>SSH target (Tailscale)</Label>
          <Input value={target} onChange={(e) => setTarget(e.target.value)} placeholder="root@messenger-host" />
          <p className="text-xs text-fg-muted">
            user@host on your tailnet. No password/keys — access is by Tailscale identity
            (enable <code className="font-mono">tailscale up --ssh</code> on the target + allow it in ACLs).
          </p>
        </div>
        {err && <p className="text-xs text-danger">{err}</p>}
        <div className="flex justify-end gap-2">
          <Button variant="ghost" size="sm" onClick={onClose}>Cancel</Button>
          <Button size="sm" disabled={target.length < 3 || mutation.isPending} onClick={() => { setErr(""); mutation.mutate(); }}>
            Add
          </Button>
        </div>
      </div>
    </Modal>
  );
}
