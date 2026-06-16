import { createBrowserRouter, Navigate, Outlet } from "react-router-dom";
import { useAuthStore } from "@/store/auth";
import { AppLayout } from "@/layouts/AppLayout";
import { LoginPage } from "@/features/auth/LoginPage";
import { RegisterPage } from "@/features/auth/RegisterPage";
import { WorkspacePage } from "@/features/organizations/WorkspacePage";
import { OverviewPage } from "@/features/dashboard/OverviewPage";
import { TimelinePage } from "@/features/timeline/TimelinePage";
import { MetricsPage } from "@/features/metrics/MetricsPage";
import { LogsPage } from "@/features/logs/LogsPage";
import { AlertsPage } from "@/features/alerts/AlertsPage";
import { DeploymentsPage } from "@/features/deployments/DeploymentsPage";
import { ServicesPage } from "@/features/services/ServicesPage";
import { ConnectPage } from "@/features/connect/ConnectPage";
import { SettingsPage } from "@/features/projects/SettingsPage";

function RequireAuth() {
  const token = useAuthStore((s) => s.accessToken);
  if (!token) return <Navigate to="/login" replace />;
  return <Outlet />;
}

export const router = createBrowserRouter([
  { path: "/login", element: <LoginPage /> },
  { path: "/register", element: <RegisterPage /> },
  {
    element: <RequireAuth />,
    children: [
      { path: "/", element: <WorkspacePage /> },
      {
        path: "/projects/:projectId",
        element: <AppLayout />,
        children: [
          { index: true, element: <Navigate to="overview" replace /> },
          { path: "overview", element: <OverviewPage /> },
          { path: "timeline", element: <TimelinePage /> },
          { path: "metrics", element: <MetricsPage /> },
          { path: "logs", element: <LogsPage /> },
          { path: "alerts", element: <AlertsPage /> },
          { path: "deployments", element: <DeploymentsPage /> },
          { path: "services", element: <ServicesPage /> },
          { path: "connect", element: <ConnectPage /> },
          { path: "settings", element: <SettingsPage /> },
        ],
      },
    ],
  },
  { path: "*", element: <Navigate to="/" replace /> },
]);
