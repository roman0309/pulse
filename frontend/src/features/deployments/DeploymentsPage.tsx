import { useState } from "react";
import { useParams } from "react-router-dom";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Plus, GitCommitHorizontal } from "lucide-react";
import { api } from "@/services/api";
import {
  Badge,
  Button,
  Card,
  Input,
  Label,
  Select,
} from "@/components/ui/primitives";
import { PageHeader, Spinner, EmptyState, Modal } from "@/components/common";
import { formatDateTime } from "@/lib/utils";

export function DeploymentsPage() {
  const { projectId } = useParams();
  const qc = useQueryClient();
  const [serviceId, setServiceId] = useState("");
  const [modal, setModal] = useState(false);

  const services = useQuery({
    queryKey: ["services", projectId],
    queryFn: () => api.listServices(projectId!),
  });
  const deployments = useQuery({
    queryKey: ["deployments", projectId, serviceId],
    queryFn: () => api.deployments(projectId!, serviceId || undefined),
  });

  const statusTone = (s: string) =>
    s === "success" ? "success" : s === "rolled_back" ? "warning" : "danger";

  return (
    <div>
      <PageHeader
        title="Deployments"
        description="Release history correlated with incidents"
        actions={
          <div className="flex items-center gap-2">
            <Select
              value={serviceId}
              onChange={(e) => setServiceId(e.target.value)}
            >
              <option value="">All services</option>
              {services.data?.map((s) => (
                <option key={s.id} value={s.id}>
                  {s.name}
                </option>
              ))}
            </Select>
            <Button size="sm" onClick={() => setModal(true)}>
              <Plus className="h-4 w-4" /> New deployment
            </Button>
          </div>
        }
      />

      {deployments.isLoading ? (
        <Spinner />
      ) : deployments.data && deployments.data.length > 0 ? (
        <Card className="overflow-hidden">
          <div className="divide-y divide-border">
            {deployments.data.map((d) => (
              <div
                key={d.id}
                className="flex items-center justify-between gap-4 px-4 py-3 hover:bg-surface-2 transition"
              >
                <div className="flex items-center gap-3 min-w-0">
                  <GitCommitHorizontal className="h-4 w-4 text-primary shrink-0" />
                  <div className="min-w-0">
                    <div className="flex items-center gap-2">
                      <span className="font-mono text-sm font-medium text-fg">
                        {d.version}
                      </span>
                      <Badge tone={statusTone(d.status)}>{d.status}</Badge>
                    </div>
                    <p className="text-xs text-fg-muted truncate">
                      {d.service_name} · {d.environment} ·{" "}
                      {d.deployed_by || "unknown"}
                      {d.commit_sha && ` · ${d.commit_sha.slice(0, 7)}`}
                    </p>
                  </div>
                </div>
                <span className="text-xs text-fg-muted whitespace-nowrap">
                  {formatDateTime(d.created_at)}
                </span>
              </div>
            ))}
          </div>
        </Card>
      ) : (
        <EmptyState
          icon={<GitCommitHorizontal className="h-8 w-8" />}
          title="No deployments yet"
          description="Record a deployment to correlate it with metric changes."
        />
      )}

      <CreateDeploymentModal
        open={modal}
        projectId={projectId!}
        services={services.data ?? []}
        onClose={() => setModal(false)}
        onCreated={() => {
          qc.invalidateQueries({ queryKey: ["deployments", projectId] });
          qc.invalidateQueries({ queryKey: ["timeline", projectId] });
          setModal(false);
        }}
      />
    </div>
  );
}

function CreateDeploymentModal({
  open,
  projectId,
  services,
  onClose,
  onCreated,
}: {
  open: boolean;
  projectId: string;
  services: { id: string; name: string }[];
  onClose: () => void;
  onCreated: () => void;
}) {
  const [serviceId, setServiceId] = useState("");
  const [version, setVersion] = useState("");
  const [commitSha, setCommitSha] = useState("");
  const [deployedBy, setDeployedBy] = useState("");

  const mutation = useMutation({
    mutationFn: () =>
      api.createDeployment(projectId, {
        service_id: serviceId,
        version,
        commit_sha: commitSha,
        deployed_by: deployedBy,
        environment: "production",
      }),
    onSuccess: onCreated,
  });

  return (
    <Modal open={open} onClose={onClose} title="Record deployment">
      <div className="space-y-4">
        <div className="space-y-1.5">
          <Label>Service</Label>
          <Select
            className="w-full"
            value={serviceId}
            onChange={(e) => setServiceId(e.target.value)}
          >
            <option value="">Select…</option>
            {services.map((s) => (
              <option key={s.id} value={s.id}>
                {s.name}
              </option>
            ))}
          </Select>
        </div>
        <div className="grid grid-cols-2 gap-3">
          <div className="space-y-1.5">
            <Label>Version</Label>
            <Input
              value={version}
              onChange={(e) => setVersion(e.target.value)}
              placeholder="v1.8.3"
            />
          </div>
          <div className="space-y-1.5">
            <Label>Commit SHA</Label>
            <Input
              value={commitSha}
              onChange={(e) => setCommitSha(e.target.value)}
              placeholder="a1b2c3d"
            />
          </div>
        </div>
        <div className="space-y-1.5">
          <Label>Deployed by</Label>
          <Input
            value={deployedBy}
            onChange={(e) => setDeployedBy(e.target.value)}
            placeholder="your name"
          />
        </div>
        <div className="flex justify-end gap-2">
          <Button variant="ghost" size="sm" onClick={onClose}>
            Cancel
          </Button>
          <Button
            size="sm"
            disabled={!serviceId || version.length < 1 || mutation.isPending}
            onClick={() => mutation.mutate()}
          >
            Record
          </Button>
        </div>
      </div>
    </Modal>
  );
}
