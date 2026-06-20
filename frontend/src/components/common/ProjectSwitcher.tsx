import { useState } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import { Activity, Check, ChevronsUpDown, LayoutGrid } from "lucide-react";
import { api } from "@/services/api";
import { cn } from "@/lib/utils";

// Sidebar header: shows the current project and a dropdown to switch between
// sibling projects (or jump back to the workspace).
export function ProjectSwitcher() {
  const { projectId } = useParams();
  const navigate = useNavigate();
  const [open, setOpen] = useState(false);

  const project = useQuery({
    queryKey: ["project", projectId],
    queryFn: () => api.getProject(projectId!),
    enabled: !!projectId,
  });
  const orgId = project.data?.organization_id;
  const projects = useQuery({
    queryKey: ["projects", orgId],
    queryFn: () => api.listProjects(orgId!),
    enabled: !!orgId,
  });

  const go = (id: string) => {
    setOpen(false);
    navigate(`/projects/${id}/overview`);
  };

  return (
    <div className="relative">
      <button
        onClick={() => setOpen((o) => !o)}
        className="flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-left transition hover:bg-surface-2"
      >
        <Activity className="h-5 w-5 shrink-0 text-primary" />
        <div className="min-w-0 flex-1">
          <p className="truncate text-sm font-semibold text-fg">{project.data?.name ?? "Pulse"}</p>
          <p className="truncate text-[11px] text-fg-muted">Pulse</p>
        </div>
        <ChevronsUpDown className="h-4 w-4 shrink-0 text-fg-muted" />
      </button>

      {open && (
        <>
          <div className="fixed inset-0 z-10" onClick={() => setOpen(false)} />
          <div className="absolute inset-x-0 top-full z-20 mt-1 rounded-md border border-border bg-surface p-1 shadow-xl">
            <p className="px-2 py-1 text-[10px] font-semibold uppercase tracking-wide text-fg-muted">Projects</p>
            <div className="max-h-64 overflow-auto">
              {(projects.data ?? []).map((p) => (
                <button
                  key={p.id}
                  onClick={() => go(p.id)}
                  className={cn(
                    "flex w-full items-center gap-2 rounded px-2 py-1.5 text-sm transition",
                    p.id === projectId ? "bg-surface-2 text-fg" : "text-fg-muted hover:bg-surface-2 hover:text-fg"
                  )}
                >
                  <span className="truncate">{p.name}</span>
                  {p.id === projectId && <Check className="ml-auto h-3.5 w-3.5 text-primary" />}
                </button>
              ))}
            </div>
            <div className="my-1 border-t border-border" />
            <button
              onClick={() => {
                setOpen(false);
                navigate("/");
              }}
              className="flex w-full items-center gap-2 rounded px-2 py-1.5 text-sm text-fg-muted transition hover:bg-surface-2 hover:text-fg"
            >
              <LayoutGrid className="h-3.5 w-3.5" /> All projects
            </button>
          </div>
        </>
      )}
    </div>
  );
}
