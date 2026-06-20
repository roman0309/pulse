import { useEffect, useState } from "react";
import { createPortal } from "react-dom";
import { CheckCircle2, AlertTriangle, Info, X } from "lucide-react";
import { cn } from "@/lib/utils";
import { subscribeToasts, dismissToast, type Toast } from "@/lib/toast";

const TONE: Record<Toast["type"], { icon: typeof Info; cls: string }> = {
  success: { icon: CheckCircle2, cls: "border-success/30 text-success" },
  error: { icon: AlertTriangle, cls: "border-danger/30 text-danger" },
  info: { icon: Info, cls: "border-border text-fg" },
};

export function Toaster() {
  const [items, setItems] = useState<Toast[]>([]);
  useEffect(() => subscribeToasts(setItems), []);

  return createPortal(
    <div className="pointer-events-none fixed bottom-4 right-4 z-[60] flex w-80 max-w-[calc(100vw-2rem)] flex-col gap-2">
      {items.map((t) => {
        const { icon: Icon, cls } = TONE[t.type];
        return (
          <div
            key={t.id}
            className={cn(
              "pointer-events-auto flex items-start gap-2 rounded-lg border bg-surface px-3 py-2.5 text-sm shadow-lg",
              cls
            )}
          >
            <Icon className="mt-0.5 h-4 w-4 shrink-0" />
            <span className="flex-1 break-words text-fg">{t.message}</span>
            <button onClick={() => dismissToast(t.id)} className="text-fg-muted transition hover:text-fg">
              <X className="h-3.5 w-3.5" />
            </button>
          </div>
        );
      })}
    </div>,
    document.body
  );
}
