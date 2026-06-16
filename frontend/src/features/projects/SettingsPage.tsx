import { useEffect, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { api } from "@/services/api";
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
    },
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
