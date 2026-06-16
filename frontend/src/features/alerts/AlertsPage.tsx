import { useState } from "react";
import { useParams } from "react-router-dom";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Plus, AlertTriangle } from "lucide-react";
import { api } from "@/services/api";
import { useWebSocket } from "@/hooks/useWebSocket";
import {
  Button,
  Card,
  Input,
  Label,
  Select,
} from "@/components/ui/primitives";
import {
  PageHeader,
  Spinner,
  EmptyState,
  SeverityBadge,
  StatusBadge,
  Modal,
} from "@/components/common";
import { relativeTime } from "@/lib/utils";

export function AlertsPage() {
  const { projectId } = useParams();
  const qc = useQueryClient();
  const [filter, setFilter] = useState("");
  const [modal, setModal] = useState(false);

  const alerts = useQuery({
    queryKey: ["alerts", projectId, filter],
    queryFn: () => api.alerts(projectId!, filter || undefined),
  });

  useWebSocket(projectId, (ev) => {
    if (ev.type === "alert") {
      qc.invalidateQueries({ queryKey: ["alerts", projectId] });
    }
  });

  const resolve = useMutation({
    mutationFn: (id: string) => api.resolveAlert(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["alerts", projectId] }),
  });

  return (
    <div>
      <PageHeader
        title="Alerts"
        description="Active and resolved incidents"
        actions={
          <div className="flex items-center gap-2">
            <Select value={filter} onChange={(e) => setFilter(e.target.value)}>
              <option value="">All</option>
              <option value="active">Active</option>
              <option value="resolved">Resolved</option>
            </Select>
            <Button size="sm" onClick={() => setModal(true)}>
              <Plus className="h-4 w-4" /> New alert
            </Button>
          </div>
        }
      />

      {alerts.isLoading ? (
        <Spinner />
      ) : alerts.data && alerts.data.length > 0 ? (
        <div className="space-y-2">
          {alerts.data.map((a) => (
            <Card key={a.id} className="p-4">
              <div className="flex items-start justify-between gap-4">
                <div className="flex items-start gap-3">
                  <AlertTriangle
                    className={`h-4 w-4 mt-0.5 ${
                      a.status === "active" ? "text-danger" : "text-fg-muted"
                    }`}
                  />
                  <div>
                    <div className="flex items-center gap-2 flex-wrap">
                      <span className="text-sm font-medium text-fg">
                        {a.title}
                      </span>
                      <SeverityBadge severity={a.severity} />
                      <StatusBadge status={a.status} />
                    </div>
                    {a.description && (
                      <p className="text-sm text-fg-muted mt-1">
                        {a.description}
                      </p>
                    )}
                    <p className="text-xs text-fg-muted mt-1">
                      {a.service_name && `${a.service_name} · `}
                      {relativeTime(a.created_at)}
                    </p>
                  </div>
                </div>
                {a.status === "active" && (
                  <Button
                    variant="outline"
                    size="sm"
                    disabled={resolve.isPending}
                    onClick={() => resolve.mutate(a.id)}
                  >
                    Resolve
                  </Button>
                )}
              </div>
            </Card>
          ))}
        </div>
      ) : (
        <EmptyState
          icon={<AlertTriangle className="h-8 w-8" />}
          title="No alerts"
          description="Everything looks healthy right now."
        />
      )}

      <CreateAlertModal
        open={modal}
        projectId={projectId!}
        onClose={() => setModal(false)}
        onCreated={() => {
          qc.invalidateQueries({ queryKey: ["alerts", projectId] });
          setModal(false);
        }}
      />
    </div>
  );
}

function CreateAlertModal({
  open,
  projectId,
  onClose,
  onCreated,
}: {
  open: boolean;
  projectId: string;
  onClose: () => void;
  onCreated: () => void;
}) {
  const [title, setTitle] = useState("");
  const [type, setType] = useState("high_latency");
  const [severity, setSeverity] = useState("medium");
  const [serviceId, setServiceId] = useState("");
  const [description, setDescription] = useState("");

  const services = useQuery({
    queryKey: ["services", projectId],
    queryFn: () => api.listServices(projectId),
    enabled: open,
  });

  const mutation = useMutation({
    mutationFn: () =>
      api.createAlert(projectId, {
        title,
        type,
        severity,
        service_id: serviceId || undefined,
        description,
      }),
    onSuccess: onCreated,
  });

  return (
    <Modal open={open} onClose={onClose} title="Create alert">
      <div className="space-y-4">
        <div className="space-y-1.5">
          <Label>Title</Label>
          <Input
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            placeholder="High latency on payment-api"
          />
        </div>
        <div className="grid grid-cols-2 gap-3">
          <div className="space-y-1.5">
            <Label>Type</Label>
            <Select
              className="w-full"
              value={type}
              onChange={(e) => setType(e.target.value)}
            >
              <option value="high_latency">High latency</option>
              <option value="high_error_rate">High error rate</option>
              <option value="service_down">Service down</option>
            </Select>
          </div>
          <div className="space-y-1.5">
            <Label>Severity</Label>
            <Select
              className="w-full"
              value={severity}
              onChange={(e) => setSeverity(e.target.value)}
            >
              <option value="low">Low</option>
              <option value="medium">Medium</option>
              <option value="high">High</option>
              <option value="critical">Critical</option>
            </Select>
          </div>
        </div>
        <div className="space-y-1.5">
          <Label>Service</Label>
          <Select
            className="w-full"
            value={serviceId}
            onChange={(e) => setServiceId(e.target.value)}
          >
            <option value="">None</option>
            {services.data?.map((s) => (
              <option key={s.id} value={s.id}>
                {s.name}
              </option>
            ))}
          </Select>
        </div>
        <div className="space-y-1.5">
          <Label>Description</Label>
          <Input
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            placeholder="Optional"
          />
        </div>
        <div className="flex justify-end gap-2">
          <Button variant="ghost" size="sm" onClick={onClose}>
            Cancel
          </Button>
          <Button
            size="sm"
            disabled={title.length < 1 || mutation.isPending}
            onClick={() => mutation.mutate()}
          >
            Create
          </Button>
        </div>
      </div>
    </Modal>
  );
}
