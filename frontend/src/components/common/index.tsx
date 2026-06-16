import * as React from "react";
import { Badge } from "@/components/ui/primitives";
import type { AlertSeverity, AlertStatus, ServiceStatus } from "@/types";
import { cn } from "@/lib/utils";

export function PageHeader({
  title,
  description,
  actions,
}: {
  title: string;
  description?: string;
  actions?: React.ReactNode;
}) {
  return (
    <div className="flex items-start justify-between gap-4 mb-6">
      <div>
        <h1 className="text-xl font-semibold text-fg">{title}</h1>
        {description && (
          <p className="text-sm text-fg-muted mt-1">{description}</p>
        )}
      </div>
      {actions && <div className="flex items-center gap-2">{actions}</div>}
    </div>
  );
}

export function EmptyState({
  icon,
  title,
  description,
  action,
}: {
  icon?: React.ReactNode;
  title: string;
  description?: string;
  action?: React.ReactNode;
}) {
  return (
    <div className="flex flex-col items-center justify-center py-16 text-center">
      {icon && <div className="text-fg-muted mb-3">{icon}</div>}
      <p className="text-sm font-medium text-fg">{title}</p>
      {description && (
        <p className="text-sm text-fg-muted mt-1 max-w-sm">{description}</p>
      )}
      {action && <div className="mt-4">{action}</div>}
    </div>
  );
}

export function Spinner({ className }: { className?: string }) {
  return (
    <div
      className={cn(
        "h-4 w-4 animate-spin rounded-full border-2 border-fg-muted/30 border-t-fg",
        className
      )}
    />
  );
}

export function SeverityBadge({ severity }: { severity: AlertSeverity }) {
  const tone =
    severity === "critical" || severity === "high"
      ? "danger"
      : severity === "medium"
        ? "warning"
        : "muted";
  return <Badge tone={tone}>{severity}</Badge>;
}

export function StatusBadge({ status }: { status: AlertStatus }) {
  return (
    <Badge tone={status === "active" ? "danger" : "success"}>{status}</Badge>
  );
}

export function ServiceStatusDot({ status }: { status: ServiceStatus }) {
  const color =
    status === "healthy"
      ? "bg-success"
      : status === "degraded"
        ? "bg-warning"
        : "bg-danger";
  return (
    <span className="inline-flex items-center gap-1.5">
      <span className={cn("h-2 w-2 rounded-full", color)} />
      <span className="text-xs capitalize text-fg-muted">{status}</span>
    </span>
  );
}

export function Modal({
  open,
  onClose,
  title,
  children,
}: {
  open: boolean;
  onClose: () => void;
  title: string;
  children: React.ReactNode;
}) {
  if (!open) return null;
  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 p-4 animate-in"
      onClick={onClose}
    >
      <div
        className="w-full max-w-md rounded-lg border border-border bg-surface shadow-xl"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="border-b border-border px-5 py-4">
          <h2 className="text-sm font-semibold text-fg">{title}</h2>
        </div>
        <div className="p-5">{children}</div>
      </div>
    </div>
  );
}
