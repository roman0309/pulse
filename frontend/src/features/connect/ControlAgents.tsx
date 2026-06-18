import { useState } from "react";
import { useMutation, useQuery } from "@tanstack/react-query";
import { Radio, Activity, Download, StopCircle } from "lucide-react";
import { api } from "@/services/api";
import {
  Button,
  Card,
  CardContent,
  CardHeader,
  CardTitle,
  Input,
  Badge,
} from "@/components/ui/primitives";
import { EmptyState, Spinner } from "@/components/common";

// Live agents connected over the control channel — manage them with buttons,
// no SSH or VPN: the agent dials out and Pulse sends commands down the channel.
export function ControlAgents({ projectId }: { projectId: string }) {
  const agents = useQuery({
    queryKey: ["control-agents", projectId],
    queryFn: () => api.listAgents(projectId),
    refetchInterval: 5000,
  });

  return (
    <Card className="mb-6">
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <Radio className="h-4 w-4 text-success" /> Connected agents
          <span className="text-xs font-normal text-fg-muted">
            · live control channel (no SSH/VPN)
          </span>
        </CardTitle>
      </CardHeader>
      <CardContent>
        {agents.isLoading ? (
          <Spinner />
        ) : agents.data && agents.data.length > 0 ? (
          <div className="space-y-2">
            {agents.data.map((id) => (
              <AgentRow key={id} projectId={projectId} agentId={id} />
            ))}
          </div>
        ) : (
          <EmptyState
            icon={<Radio className="h-7 w-7" />}
            title="No agents connected"
            description="Install the host agent on a server — it dials back here automatically, then you manage it from these buttons."
          />
        )}
      </CardContent>
    </Card>
  );
}

function AgentRow({ projectId, agentId }: { projectId: string; agentId: string }) {
  const [port, setPort] = useState("8080");
  const [out, setOut] = useState("");
  const cmd = useMutation({
    mutationFn: (c: "status" | "install_beyla" | "remove") =>
      api.sendAgentCommand(projectId, agentId, c, c === "install_beyla" ? { port } : undefined),
    onSuccess: (r) => setOut(r.output || (r.ok ? "ok" : "failed")),
    onError: (e) => setOut(e instanceof Error ? e.message : "error"),
  });

  return (
    <div className="rounded-md border border-border p-3">
      <div className="flex items-center justify-between gap-3 flex-wrap">
        <div className="flex items-center gap-2">
          <Activity className="h-4 w-4 text-success" />
          <span className="text-sm font-medium text-fg">{agentId}</span>
          <Badge tone="success">online</Badge>
        </div>
        <div className="flex items-center gap-1.5">
          <Input
            value={port}
            onChange={(e) => setPort(e.target.value)}
            className="h-8 w-20 text-xs"
            title="App port to instrument"
          />
          <Button variant="outline" size="sm" disabled={cmd.isPending}
            onClick={() => cmd.mutate("install_beyla")}>
            <Download className="h-3.5 w-3.5" /> App metrics
          </Button>
          <Button variant="ghost" size="sm" disabled={cmd.isPending}
            onClick={() => cmd.mutate("status")}>
            Status
          </Button>
          <Button variant="ghost" size="sm" disabled={cmd.isPending}
            onClick={() => cmd.mutate("remove")} title="Remove app-metrics agent">
            <StopCircle className="h-3.5 w-3.5" />
          </Button>
        </div>
      </div>
      {cmd.isPending && <p className="text-xs text-fg-muted mt-2">running…</p>}
      {out && !cmd.isPending && (
        <pre className="mt-2 max-h-28 overflow-auto rounded bg-surface-2 p-2 text-[11px] font-mono text-fg-muted whitespace-pre-wrap break-all">
          {out}
        </pre>
      )}
    </div>
  );
}
