// Tiny dependency-free toast store. Call `toast.success("…")` from anywhere
// (including mutation callbacks); <Toaster/> renders the queue.
export type ToastType = "success" | "error" | "info";
export interface Toast {
  id: number;
  type: ToastType;
  message: string;
}

let items: Toast[] = [];
let nextId = 1;
const listeners = new Set<(t: Toast[]) => void>();

function emit() {
  for (const l of listeners) l(items);
}

export function subscribeToasts(l: (t: Toast[]) => void) {
  listeners.add(l);
  l(items);
  return () => {
    listeners.delete(l);
  };
}

export function dismissToast(id: number) {
  items = items.filter((t) => t.id !== id);
  emit();
}

function push(type: ToastType, message: string, ms = 4000) {
  const id = nextId++;
  items = [...items, { id, type, message }];
  emit();
  setTimeout(() => dismissToast(id), ms);
}

export const toast = {
  success: (m: string) => push("success", m),
  error: (m: string) => push("error", m, 6000),
  info: (m: string) => push("info", m),
};
