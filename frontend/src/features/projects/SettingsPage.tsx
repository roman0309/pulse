import { useEffect, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { RefreshCw, Loader2 } from "lucide-react";
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
} from "@/components/ui/primitives";
import { PageHeader, Spinner } from "@/components/common";
import { NotificationChannels } from "./NotificationChannels";

export function SettingsPage() {
  const { projectId } = useParams();
  const navigate = useNavigate();
  const qc = useQueryClient();
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");

  const project = useQuery({
    queryKey: ["project", projectId],
    queryFn: () => api.getProject(projectId!),
  });

  useEffect(() => {
    if (project.data) {
      setName(project.data.name);
      setDescription(project.data.description);
    }
  }, [project.data]);

  const save = useMutation({
    mutationFn: () => api.updateProject(projectId!, name, description),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["project", projectId] });
      qc.invalidateQueries({ queryKey: ["projects"] });
      toast.success("Project saved");
    },
    onError: () => toast.error("Couldn't save project"),
  });

  const remove = useMutation({
    mutationFn: () => api.deleteProject(projectId!),
    onSuccess: () => navigate("/"),
  });

  if (project.isLoading) return <Spinner />;

  return (
    <div className="max-w-2xl">
      <PageHeader title="Settings" description="Manage your project" />

      <Card className="mb-6">
        <CardHeader>
          <CardTitle>General</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="space-y-1.5">
            <Label>Project name</Label>
            <Input value={name} onChange={(e) => setName(e.target.value)} />
          </div>
          <div className="space-y-1.5">
            <Label>Description</Label>
            <Input
              value={description}
              onChange={(e) => setDescription(e.target.value)}
            />
          </div>
          <div className="flex justify-end">
            <Button
              size="sm"
              disabled={save.isPending}
              onClick={() => save.mutate()}
            >
              {save.isPending ? "Saving…" : "Save changes"}
            </Button>
          </div>
        </CardContent>
      </Card>

      <NotificationChannels projectId={projectId!} />

      <UpdateCard />

      <Card className="border-danger/30">
        <CardHeader>
          <CardTitle className="text-danger">Danger zone</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="flex items-center justify-between">
            <p className="text-sm text-fg-muted">
              Permanently delete this project and all its data.
            </p>
            <Button
              variant="danger"
              size="sm"
              disabled={remove.isPending}
              onClick={() => {
                if (confirm("Delete this project? This cannot be undone.")) {
                  remove.mutate();
                }
              }}
            >
              Delete project
            </Button>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}

// UpdateCard shows an in-app "Update Pulse" button when the backend has the
// Docker socket wired up (meta.self_update).
function UpdateCard() {
  const meta = useQuery({ queryKey: ["meta"], queryFn: api.meta });
  const [updating, setUpdating] = useState(false);

  if (meta.isLoading || !meta.data?.self_update) return null;

  const start = () => {
    if (!confirm("Update Pulse now? The app will pull the latest images and restart (~30–60s).")) return;
    setUpdating(true);
    // Fire-and-forget: the backend is recreated mid-request, then reload.
    api.selfUpdate().catch(() => {});
    setTimeout(() => window.location.reload(), 45000);
  };

  return (
    <Card className="mb-6">
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <RefreshCw className="h-4 w-4" /> Updates
        </CardTitle>
      </CardHeader>
      <CardContent>
        {updating ? (
          <div className="flex items-center gap-2 text-sm text-fg-muted">
            <Loader2 className="h-4 w-4 animate-spin" />
            Pulling the latest images and restarting… this page will reload automatically.
          </div>
        ) : (
          <div className="flex flex-wrap items-center justify-between gap-3">
            <p className="text-sm text-fg-muted">
              Pull the latest Pulse images from the registry and recreate the containers.
            </p>
            <Button size="sm" onClick={start}>
              Update now
            </Button>
          </div>
        )}
      </CardContent>
    </Card>
  );
}
