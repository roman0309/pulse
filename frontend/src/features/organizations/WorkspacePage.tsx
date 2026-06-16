import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Activity, Building2, Plus, FolderGit2, ArrowRight } from "lucide-react";
import { api } from "@/services/api";
import { useAuthStore } from "@/store/auth";
import {
  Button,
  Badge,
  Input,
  Label,
} from "@/components/ui/primitives";
import { EmptyState, Modal, Spinner } from "@/components/common";
import { relativeTime } from "@/lib/utils";

export function WorkspacePage() {
  const navigate = useNavigate();
  const qc = useQueryClient();
  const { activeOrgId, setActiveOrg, setActiveProject, user, logout } =
    useAuthStore();
  const [orgModal, setOrgModal] = useState(false);
  const [projModal, setProjModal] = useState(false);

  const orgsQuery = useQuery({ queryKey: ["orgs"], queryFn: api.listOrgs });

  // default the active org to the first available
  useEffect(() => {
    if (!activeOrgId && orgsQuery.data && orgsQuery.data.length > 0) {
      setActiveOrg(orgsQuery.data[0].id);
    }
  }, [activeOrgId, orgsQuery.data, setActiveOrg]);

  const projectsQuery = useQuery({
    queryKey: ["projects", activeOrgId],
    queryFn: () => api.listProjects(activeOrgId!),
    enabled: !!activeOrgId,
  });

  const enterProject = (projectId: string) => {
    setActiveProject(projectId);
    navigate(`/projects/${projectId}/overview`);
  };

  return (
    <div className="min-h-screen bg-bg">
      <header className="border-b border-border bg-surface">
        <div className="mx-auto max-w-5xl px-6 h-14 flex items-center justify-between">
          <div className="flex items-center gap-2">
            <Activity className="h-5 w-5 text-primary" />
            <span className="font-semibold text-fg">Pulse</span>
          </div>
          <div className="flex items-center gap-3">
            <span className="text-sm text-fg-muted">{user?.email}</span>
            <Button
              variant="ghost"
              size="sm"
              onClick={() => {
                logout();
                navigate("/login");
              }}
            >
              Sign out
            </Button>
          </div>
        </div>
      </header>

      <main className="mx-auto max-w-5xl px-6 py-10">
        {/* Organizations */}
        <section className="mb-10">
          <div className="flex items-center justify-between mb-4">
            <div>
              <h2 className="text-base font-semibold text-fg flex items-center gap-2">
                <Building2 className="h-4 w-4" /> Organizations
              </h2>
              <p className="text-sm text-fg-muted">
                Select an organization to view its projects
              </p>
            </div>
            <Button size="sm" onClick={() => setOrgModal(true)}>
              <Plus className="h-4 w-4" /> New
            </Button>
          </div>

          {orgsQuery.isLoading ? (
            <Spinner />
          ) : orgsQuery.data && orgsQuery.data.length > 0 ? (
            <div className="flex flex-wrap gap-2">
              {orgsQuery.data.map((org) => (
                <button
                  key={org.id}
                  onClick={() => setActiveOrg(org.id)}
                  className={`rounded-md border px-4 py-2 text-sm transition ${
                    activeOrgId === org.id
                      ? "border-primary bg-primary/10 text-fg"
                      : "border-border bg-surface text-fg-muted hover:text-fg"
                  }`}
                >
                  <span className="font-medium">{org.name}</span>
                  {org.role && (
                    <Badge tone="muted" className="ml-2">
                      {org.role}
                    </Badge>
                  )}
                </button>
              ))}
            </div>
          ) : (
            <EmptyState
              icon={<Building2 className="h-8 w-8" />}
              title="No organizations yet"
              description="Create your first organization to get started."
              action={
                <Button size="sm" onClick={() => setOrgModal(true)}>
                  <Plus className="h-4 w-4" /> Create organization
                </Button>
              }
            />
          )}
        </section>

        {/* Projects */}
        {activeOrgId && (
          <section>
            <div className="flex items-center justify-between mb-4">
              <h2 className="text-base font-semibold text-fg flex items-center gap-2">
                <FolderGit2 className="h-4 w-4" /> Projects
              </h2>
              <Button size="sm" onClick={() => setProjModal(true)}>
                <Plus className="h-4 w-4" /> New project
              </Button>
            </div>

            {projectsQuery.isLoading ? (
              <Spinner />
            ) : projectsQuery.data && projectsQuery.data.length > 0 ? (
              <div className="grid gap-3 sm:grid-cols-2">
                {projectsQuery.data.map((p) => (
                  <button
                    key={p.id}
                    onClick={() => enterProject(p.id)}
                    className="group text-left rounded-lg border border-border bg-surface p-5 hover:border-primary/50 transition"
                  >
                    <div className="flex items-center justify-between">
                      <span className="font-medium text-fg">{p.name}</span>
                      <ArrowRight className="h-4 w-4 text-fg-muted group-hover:text-primary transition" />
                    </div>
                    <p className="text-sm text-fg-muted mt-1 line-clamp-2">
                      {p.description || "No description"}
                    </p>
                    <p className="text-xs text-fg-muted mt-3">
                      Created {relativeTime(p.created_at)}
                    </p>
                  </button>
                ))}
              </div>
            ) : (
              <EmptyState
                icon={<FolderGit2 className="h-8 w-8" />}
                title="No projects yet"
                description="Create a project to start tracking services, metrics and incidents."
                action={
                  <Button size="sm" onClick={() => setProjModal(true)}>
                    <Plus className="h-4 w-4" /> Create project
                  </Button>
                }
              />
            )}
          </section>
        )}
      </main>

      <CreateOrgModal
        open={orgModal}
        onClose={() => setOrgModal(false)}
        onCreated={(org) => {
          qc.invalidateQueries({ queryKey: ["orgs"] });
          setActiveOrg(org.id);
          setOrgModal(false);
        }}
      />
      <CreateProjectModal
        open={projModal}
        orgId={activeOrgId}
        onClose={() => setProjModal(false)}
        onCreated={() => {
          qc.invalidateQueries({ queryKey: ["projects", activeOrgId] });
          setProjModal(false);
        }}
      />
    </div>
  );
}

function CreateOrgModal({
  open,
  onClose,
  onCreated,
}: {
  open: boolean;
  onClose: () => void;
  onCreated: (org: { id: string }) => void;
}) {
  const [name, setName] = useState("");
  const mutation = useMutation({
    mutationFn: () => api.createOrg(name),
    onSuccess: onCreated,
  });
  return (
    <Modal open={open} onClose={onClose} title="Create organization">
      <div className="space-y-4">
        <div className="space-y-1.5">
          <Label>Name</Label>
          <Input
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="Acme Inc"
            autoFocus
          />
        </div>
        <div className="flex justify-end gap-2">
          <Button variant="ghost" size="sm" onClick={onClose}>
            Cancel
          </Button>
          <Button
            size="sm"
            disabled={name.length < 2 || mutation.isPending}
            onClick={() => mutation.mutate()}
          >
            Create
          </Button>
        </div>
      </div>
    </Modal>
  );
}

function CreateProjectModal({
  open,
  orgId,
  onClose,
  onCreated,
}: {
  open: boolean;
  orgId: string | null;
  onClose: () => void;
  onCreated: () => void;
}) {
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const mutation = useMutation({
    mutationFn: () => api.createProject(orgId!, name, description),
    onSuccess: onCreated,
  });
  return (
    <Modal open={open} onClose={onClose} title="Create project">
      <div className="space-y-4">
        <div className="space-y-1.5">
          <Label>Name</Label>
          <Input
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="Production Platform"
            autoFocus
          />
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
            disabled={name.length < 2 || !orgId || mutation.isPending}
            onClick={() => mutation.mutate()}
          >
            Create
          </Button>
        </div>
      </div>
    </Modal>
  );
}
