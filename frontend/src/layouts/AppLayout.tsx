import { NavLink, Outlet, useLocation, useNavigate, useParams } from "react-router-dom";
import {
  Activity,
  AlertTriangle,
  GitCommitHorizontal,
  LayoutDashboard,
  LineChart,
  Lock,
  LogOut,
  ScrollText,
  Settings,
  Waypoints,
  Boxes,
  ServerCog,
} from "lucide-react";
import { useAuthStore } from "@/store/auth";
import { api } from "@/services/api";
import { cn } from "@/lib/utils";
import { useProjectReady } from "@/hooks/useProjectReady";
import { Spinner } from "@/components/common";

type NavItem = {
  to: string;
  label: string;
  icon: typeof LayoutDashboard;
  gate?: boolean; // locked until the project is receiving data
};

// Data views are gated until the project is connected. Connect is always
// available (it's where you get connected); Settings stays reachable too.
const nav: NavItem[] = [
  { to: "overview", label: "Overview", icon: LayoutDashboard, gate: true },
  { to: "timeline", label: "Timeline", icon: Waypoints, gate: true },
  { to: "metrics", label: "Metrics", icon: LineChart, gate: true },
  { to: "logs", label: "Logs", icon: ScrollText, gate: true },
  { to: "alerts", label: "Alerts", icon: AlertTriangle, gate: true },
  { to: "deployments", label: "Deployments", icon: GitCommitHorizontal, gate: true },
  { to: "services", label: "Services", icon: Boxes, gate: true },
  { to: "connect", label: "Connect", icon: ServerCog },
  { to: "settings", label: "Settings", icon: Settings },
];

export function AppLayout() {
  const { projectId } = useParams();
  const location = useLocation();
  const { user, refreshToken, logout } = useAuthStore();
  const navigate = useNavigate();
  const { ready, loading } = useProjectReady(projectId);

  const handleLogout = async () => {
    if (refreshToken) await api.logout(refreshToken).catch(() => {});
    logout();
    navigate("/login");
  };

  const connectPath = `/projects/${projectId}/connect`;
  const currentTab = location.pathname.split("/")[3] ?? "";
  const gatedTabs = new Set(nav.filter((n) => n.gate).map((n) => n.to));
  const onGatedTab = gatedTabs.has(currentTab);
  const locked = (item: NavItem) => !!item.gate && !ready;

  // Gate: a data view requested before the project is ready shows a friendly
  // prompt to connect (covers direct URLs and the index → overview redirect)
  // instead of rendering an empty dashboard.
  let content: React.ReactNode;
  if (onGatedTab && loading) {
    content = (
      <div className="flex justify-center py-20">
        <Spinner />
      </div>
    );
  } else if (onGatedTab && !ready) {
    content = (
      <div className="flex min-h-[60vh] flex-col items-center justify-center text-center">
        <div className="flex h-12 w-12 items-center justify-center rounded-full bg-surface-2 text-fg-muted">
          <ServerCog className="h-6 w-6" />
        </div>
        <h2 className="mt-4 text-lg font-semibold text-fg">No data yet</h2>
        <p className="mt-1 max-w-sm text-sm text-fg-muted">
          Connect a server and install the agent — metrics, logs and alerts unlock
          automatically once data starts flowing.
        </p>
        <button
          onClick={() => navigate(connectPath)}
          className="mt-5 inline-flex items-center gap-2 rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-fg transition hover:opacity-90"
        >
          <ServerCog className="h-4 w-4" /> Connect a server
        </button>
      </div>
    );
  } else {
    content = <Outlet />;
  }

  return (
    <div className="flex h-screen bg-bg">
      {/* Sidebar */}
      <aside className="hidden md:flex w-60 flex-col border-r border-border bg-surface">
        <div className="flex items-center gap-2 px-5 h-14 border-b border-border">
          <Activity className="h-5 w-5 text-primary" />
          <span className="font-semibold text-fg">Pulse</span>
        </div>
        <nav className="flex-1 p-3 space-y-1">
          {nav.map((item) => {
            const { to, label, icon: Icon } = item;
            if (locked(item)) {
              return (
                <div
                  key={to}
                  title="Connect a server first — no data yet"
                  className="flex cursor-not-allowed items-center gap-3 rounded-md px-3 py-2 text-sm text-fg-muted/40 select-none"
                >
                  <Icon className="h-4 w-4" />
                  {label}
                  <Lock className="ml-auto h-3.5 w-3.5" />
                </div>
              );
            }
            return (
              <NavLink
                key={to}
                to={`/projects/${projectId}/${to}`}
                className={({ isActive }) =>
                  cn(
                    "flex items-center gap-3 rounded-md px-3 py-2 text-sm transition-colors",
                    isActive
                      ? "bg-surface-2 text-fg"
                      : "text-fg-muted hover:text-fg hover:bg-surface-2"
                  )
                }
              >
                <Icon className="h-4 w-4" />
                {label}
                {to === "connect" && !ready && (
                  <span className="ml-auto rounded-full bg-primary/15 px-1.5 py-0.5 text-[10px] font-semibold text-primary">
                    start
                  </span>
                )}
              </NavLink>
            );
          })}
        </nav>
        <div className="border-t border-border p-3">
          <div className="flex items-center justify-between gap-2 px-2">
            <div className="min-w-0">
              <p className="truncate text-sm font-medium text-fg">
                {user?.name}
              </p>
              <p className="truncate text-xs text-fg-muted">{user?.email}</p>
            </div>
            <button
              onClick={handleLogout}
              className="text-fg-muted hover:text-danger transition-colors"
              title="Logout"
            >
              <LogOut className="h-4 w-4" />
            </button>
          </div>
        </div>
      </aside>

      {/* Mobile top bar */}
      <div className="flex flex-1 flex-col overflow-hidden">
        <header className="md:hidden flex items-center gap-2 px-4 h-14 border-b border-border bg-surface">
          <Activity className="h-5 w-5 text-primary" />
          <span className="font-semibold text-fg">Pulse</span>
        </header>
        <main className="flex-1 overflow-y-auto">
          <div className="mx-auto max-w-7xl px-4 sm:px-6 py-6">{content}</div>
        </main>
        {/* Mobile bottom nav */}
        <nav className="md:hidden flex items-center justify-around border-t border-border bg-surface py-2">
          {!ready ? (
            <NavLink
              to={connectPath}
              className={({ isActive }) =>
                cn(
                  "flex items-center gap-2 rounded-md px-3 py-1.5 text-sm",
                  isActive ? "text-primary" : "text-fg-muted"
                )
              }
            >
              <ServerCog className="h-5 w-5" /> Connect a server
            </NavLink>
          ) : (
            nav
              .filter((n) => !n.gate || ready)
              .slice(0, 5)
              .map(({ to, icon: Icon }) => (
                <NavLink
                  key={to}
                  to={`/projects/${projectId}/${to}`}
                  className={({ isActive }) =>
                    cn("p-2 rounded-md", isActive ? "text-primary" : "text-fg-muted")
                  }
                >
                  <Icon className="h-5 w-5" />
                </NavLink>
              ))
          )}
        </nav>
      </div>
    </div>
  );
}
