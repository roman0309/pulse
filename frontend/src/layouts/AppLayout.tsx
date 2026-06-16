import { NavLink, Outlet, useNavigate, useParams } from "react-router-dom";
import {
  Activity,
  AlertTriangle,
  GitCommitHorizontal,
  LayoutDashboard,
  LineChart,
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

const nav = [
  { to: "overview", label: "Overview", icon: LayoutDashboard },
  { to: "timeline", label: "Timeline", icon: Waypoints },
  { to: "metrics", label: "Metrics", icon: LineChart },
  { to: "logs", label: "Logs", icon: ScrollText },
  { to: "alerts", label: "Alerts", icon: AlertTriangle },
  { to: "deployments", label: "Deployments", icon: GitCommitHorizontal },
  { to: "services", label: "Services", icon: Boxes },
  { to: "connect", label: "Connect", icon: ServerCog },
  { to: "settings", label: "Settings", icon: Settings },
];

export function AppLayout() {
  const { projectId } = useParams();
  const { user, refreshToken, logout } = useAuthStore();
  const navigate = useNavigate();

  const handleLogout = async () => {
    if (refreshToken) await api.logout(refreshToken).catch(() => {});
    logout();
    navigate("/login");
  };

  return (
    <div className="flex h-screen bg-bg">
      {/* Sidebar */}
      <aside className="hidden md:flex w-60 flex-col border-r border-border bg-surface">
        <div className="flex items-center gap-2 px-5 h-14 border-b border-border">
          <Activity className="h-5 w-5 text-primary" />
          <span className="font-semibold text-fg">Pulse</span>
        </div>
        <nav className="flex-1 p-3 space-y-1">
          {nav.map(({ to, label, icon: Icon }) => (
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
            </NavLink>
          ))}
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
          <div className="mx-auto max-w-7xl px-4 sm:px-6 py-6">
            <Outlet />
          </div>
        </main>
        {/* Mobile bottom nav */}
        <nav className="md:hidden flex items-center justify-around border-t border-border bg-surface py-2">
          {nav.slice(0, 5).map(({ to, icon: Icon }) => (
            <NavLink
              key={to}
              to={`/projects/${projectId}/${to}`}
              className={({ isActive }) =>
                cn(
                  "p-2 rounded-md",
                  isActive ? "text-primary" : "text-fg-muted"
                )
              }
            >
              <Icon className="h-5 w-5" />
            </NavLink>
          ))}
        </nav>
      </div>
    </div>
  );
}
