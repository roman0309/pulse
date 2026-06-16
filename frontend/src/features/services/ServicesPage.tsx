import { useState } from "react";
import { useParams } from "react-router-dom";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Plus, Boxes, Trash2 } from "lucide-react";
import { api } from "@/services/api";
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
  Modal,
  ServiceStatusDot,
} from "@/components/common";
import type { Service } from "@/types";

export function ServicesPage() {
  const { projectId } = useParams();
  const qc = useQueryClient();
  const [modal, setModal] = useState(false);

  const services = useQuery({
    queryKey: ["services", projectId],
    queryFn: () => api.listServices(projectId!),
  });

  const del = useMutation({
    mutationFn: (id: string) => api.deleteService(id),
    onSuccess: () =>
      qc.invalidateQueries({ queryKey: ["services", projectId] }),
  });

  const update = useMutation({
    mutationFn: (s: Service) =>
      api.updateService(s.id, {
        name: s.name,
        environment: s.environment,
        status: s.status,
      }),
    onSuccess: () =>
      qc.invalidateQueries({ queryKey: ["services", projectId] }),
  });

  return (
    <div>
      <PageHeader
        title="Services"
        description="Register and manage the services you monitor"
        actions={
          <Button size="sm" onClick={() => setModal(true)}>
            <Plus className="h-4 w-4" /> New service
          </Button>
        }
      />

      {services.isLoading ? (
        <Spinner />
      ) : services.data && services.data.length > 0 ? (
        <div className="grid sm:grid-cols-2 lg:grid-cols-3 gap-4">
          {services.data.map((s) => (
            <Card key={s.id} className="p-4">
              <div className="flex items-start justify-between">
                <div>
                  <p className="font-medium text-fg">{s.name}</p>
                  <p className="text-xs text-fg-muted">{s.environment}</p>
                </div>
                <button
                  onClick={() => del.mutate(s.id)}
                  className="text-fg-muted hover:text-danger transition"
                >
                  <Trash2 className="h-4 w-4" />
                </button>
              </div>
              <div className="mt-3 flex items-center justify-between">
                <ServiceStatusDot status={s.status} />
                <Select
                  className="h-7 text-xs"
                  value={s.status}
                  onChange={(e) =>
                    update.mutate({
                      ...s,
                      status: e.target.value as Service["status"],
                    })
                  }
                >
                  <option value="healthy">healthy</option>
                  <option value="degraded">degraded</option>
                  <option value="down">down</option>
                </Select>
              </div>
            </Card>
          ))}
        </div>
      ) : (
        <EmptyState
          icon={<Boxes className="h-8 w-8" />}
          title="No services yet"
          description="Register a service like payment-api or auth-api."
          action={
            <Button size="sm" onClick={() => setModal(true)}>
              <Plus className="h-4 w-4" /> Register service
            </Button>
          }
        />
      )}

      <CreateServiceModal
        open={modal}
        projectId={projectId!}
        onClose={() => setModal(false)}
        onCreated={() => {
          qc.invalidateQueries({ queryKey: ["services", projectId] });
          setModal(false);
        }}
      />
    </div>
  );
}

function CreateServiceModal({
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
  const [name, setName] = useState("");
  const [environment, setEnvironment] = useState("production");
  const [status, setStatus] = useState("healthy");

  const mutation = useMutation({
    mutationFn: () =>
      api.createService(projectId, { name, environment, status }),
    onSuccess: onCreated,
  });

  return (
    <Modal open={open} onClose={onClose} title="Register service">
      <div className="space-y-4">
        <div className="space-y-1.5">
          <Label>Name</Label>
          <Input
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="payment-api"
            autoFocus
          />
        </div>
        <div className="grid grid-cols-2 gap-3">
          <div className="space-y-1.5">
            <Label>Environment</Label>
            <Input
              value={environment}
              onChange={(e) => setEnvironment(e.target.value)}
            />
          </div>
          <div className="space-y-1.5">
            <Label>Status</Label>
            <Select
              className="w-full"
              value={status}
              onChange={(e) => setStatus(e.target.value)}
            >
              <option value="healthy">healthy</option>
              <option value="degraded">degraded</option>
              <option value="down">down</option>
            </Select>
          </div>
        </div>
        <div className="flex justify-end gap-2">
          <Button variant="ghost" size="sm" onClick={onClose}>
            Cancel
          </Button>
          <Button
            size="sm"
            disabled={name.length < 1 || mutation.isPending}
            onClick={() => mutation.mutate()}
          >
            Register
          </Button>
        </div>
      </div>
    </Modal>
  );
}
