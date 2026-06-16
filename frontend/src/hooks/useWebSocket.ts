import { useEffect, useRef } from "react";
import { useAuthStore } from "@/store/auth";

export interface WSEvent {
  type: "alert" | "metric" | "timeline";
  project_id: string;
  payload: unknown;
}

/**
 * Subscribes to the project's realtime channel. Calls `onEvent` for each
 * message. Reconnects automatically with a short backoff.
 */
export function useWebSocket(
  projectId: string | null | undefined,
  onEvent: (ev: WSEvent) => void
) {
  const handlerRef = useRef(onEvent);
  handlerRef.current = onEvent;

  useEffect(() => {
    if (!projectId) return;
    const token = useAuthStore.getState().accessToken;
    if (!token) return;

    let ws: WebSocket | null = null;
    let closed = false;
    let retry: ReturnType<typeof setTimeout>;

    const connect = () => {
      const proto = location.protocol === "https:" ? "wss" : "ws";
      ws = new WebSocket(
        `${proto}://${location.host}/api/v1/projects/${projectId}/ws?token=${token}`
      );
      ws.onmessage = (e) => {
        try {
          handlerRef.current(JSON.parse(e.data) as WSEvent);
        } catch {
          /* ignore malformed */
        }
      };
      ws.onclose = () => {
        if (!closed) retry = setTimeout(connect, 3000);
      };
    };
    connect();

    return () => {
      closed = true;
      clearTimeout(retry);
      ws?.close();
    };
  }, [projectId]);
}
