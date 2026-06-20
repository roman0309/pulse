import { useRouteError, useNavigate } from "react-router-dom";
import { AlertTriangle } from "lucide-react";

// Friendly fallback so a single page's runtime error doesn't blank the whole app.
export function RouteError() {
  const error = useRouteError();
  const navigate = useNavigate();
  const message = error instanceof Error ? error.message : "Unexpected error";

  return (
    <div className="flex min-h-[60vh] flex-col items-center justify-center p-6 text-center">
      <div className="flex h-12 w-12 items-center justify-center rounded-full bg-danger/15 text-danger">
        <AlertTriangle className="h-6 w-6" />
      </div>
      <h2 className="mt-4 text-lg font-semibold text-fg">Something went wrong</h2>
      <p className="mt-1 max-w-md break-words text-sm text-fg-muted">{message}</p>
      <div className="mt-5 flex gap-2">
        <button
          onClick={() => window.location.reload()}
          className="rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-fg transition hover:opacity-90"
        >
          Reload
        </button>
        <button
          onClick={() => navigate(-1)}
          className="rounded-md border border-border px-4 py-2 text-sm text-fg transition hover:bg-surface-2"
        >
          Go back
        </button>
      </div>
    </div>
  );
}
