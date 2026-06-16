import { Activity } from "lucide-react";

export function AuthShell({
  title,
  subtitle,
  children,
  footer,
}: {
  title: string;
  subtitle: string;
  children: React.ReactNode;
  footer: React.ReactNode;
}) {
  return (
    <div className="min-h-screen flex items-center justify-center bg-bg px-4">
      <div className="w-full max-w-sm animate-in">
        <div className="flex items-center gap-2 mb-8 justify-center">
          <Activity className="h-6 w-6 text-primary" />
          <span className="text-lg font-semibold text-fg">Pulse</span>
        </div>
        <div className="rounded-lg border border-border bg-surface p-6">
          <h1 className="text-lg font-semibold text-fg">{title}</h1>
          <p className="text-sm text-fg-muted mt-1 mb-6">{subtitle}</p>
          {children}
        </div>
        <p className="text-center text-sm text-fg-muted mt-6">{footer}</p>
      </div>
    </div>
  );
}
