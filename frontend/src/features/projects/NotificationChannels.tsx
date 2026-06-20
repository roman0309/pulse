import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Plus, Trash2, Send, Check, AlertTriangle, Bell } from "lucide-react";
import { api } from "@/services/api";
import {
  Button,
  Card,
  CardContent,
  CardHeader,
  CardTitle,
  Input,
  Label,
  Select,
  Badge,
} from "@/components/ui/primitives";
import { EmptyState, Spinner, Modal } from "@/components/common";
import { toast } from "@/lib/toast";
import type { NotificationChannel } from "@/types";

// Reusable notification channels (Telegram / Slack / webhook). Configure a bot
// once here, then pick it in any alert rule.
export function NotificationChannels({ projectId }: { projectId: string }) {
  const qc = useQueryClient();
  const [modal, setModal] = useState(false);

  const channels = useQuery({
    queryKey: ["channels", projectId],
    queryFn: () => api.listChannels(projectId),
  });
  const invalidate = () => qc.invalidateQueries({ queryKey: ["channels", projectId] });

  const del = useMutation({
    mutationFn: (id: string) => api.deleteChannel(projectId, id),
    onSuccess: () => {
      invalidate();
      toast.success("Channel deleted");
    },
  });

  return (
    <Card className="mb-6">
      <CardHeader className="flex flex-row items-center justify-between">
        <CardTitle className="flex items-center gap-2">
          <Bell className="h-4 w-4" /> Notification channels
        </CardTitle>
        <Button size="sm" onClick={() => setModal(true)}>
          <Plus className="h-4 w-4" /> Add channel
        </Button>
      </CardHeader>
      <CardContent>
        {channels.isLoading ? (
          <Spinner />
        ) : channels.data && channels.data.length > 0 ? (
          <div className="divide-y divide-border">
            {channels.data.map((ch) => (
              <ChannelRow key={ch.id} projectId={projectId} channel={ch} onDelete={() => del.mutate(ch.id)} />
            ))}
          </div>
        ) : (
          <EmptyState
            icon={<Bell className="h-7 w-7" />}
            title="No channels yet"
            description="Add a Telegram bot, Slack webhook or generic webhook, then select it in alert rules."
          />
        )}
      </CardContent>

      <AddChannelModal
        open={modal}
        projectId={projectId}
        onClose={() => setModal(false)}
        onAdded={() => {
          invalidate();
          setModal(false);
        }}
      />
    </Card>
  );
}

function ChannelRow({
  projectId,
  channel,
  onDelete,
}: {
  projectId: string;
  channel: NotificationChannel;
  onDelete: () => void;
}) {
  const [result, setResult] = useState<{ ok: boolean; msg: string } | null>(null);
  const test = useMutation({
    mutationFn: () => api.testChannel(projectId, channel.id),
    onSuccess: () => {
      setResult({ ok: true, msg: "Sent — check the channel" });
      toast.success(`Test sent to ${channel.name}`);
    },
    onError: (e) => {
      const msg = e instanceof Error ? e.message : "failed";
      setResult({ ok: false, msg });
      toast.error(`Test failed: ${msg}`);
    },
  });

  return (
    <div className="py-2.5">
      <div className="flex items-center justify-between gap-3">
        <div className="min-w-0">
          <div className="flex items-center gap-2">
            <span className="text-sm font-medium text-fg">{channel.name}</span>
            <Badge tone="info">{channel.type}</Badge>
          </div>
          <p className="truncate text-xs text-fg-muted">{channel.hint}</p>
        </div>
        <div className="flex shrink-0 items-center gap-1.5">
          <Button variant="outline" size="sm" disabled={test.isPending} onClick={() => test.mutate()}>
            <Send className="h-3.5 w-3.5" /> {test.isPending ? "Sending…" : "Test"}
          </Button>
          <button
            onClick={() => {
              if (confirm(`Delete channel "${channel.name}"?`)) onDelete();
            }}
            className="rounded p-1.5 text-fg-muted transition hover:bg-surface-2 hover:text-danger"
            title="Delete channel"
          >
            <Trash2 className="h-4 w-4" />
          </button>
        </div>
      </div>
      {result && (
        <p className={`mt-1.5 flex items-center gap-1 text-xs ${result.ok ? "text-success" : "text-danger"}`}>
          {result.ok ? <Check className="h-3.5 w-3.5" /> : <AlertTriangle className="h-3.5 w-3.5" />}
          {result.msg}
        </p>
      )}
    </div>
  );
}

function AddChannelModal({
  open,
  projectId,
  onClose,
  onAdded,
}: {
  open: boolean;
  projectId: string;
  onClose: () => void;
  onAdded: () => void;
}) {
  const [type, setType] = useState("telegram");
  const [name, setName] = useState("");
  const [token, setToken] = useState("");
  const [chatId, setChatId] = useState("");
  const [url, setUrl] = useState("");
  const [err, setErr] = useState("");

  const reset = () => {
    setName("");
    setToken("");
    setChatId("");
    setUrl("");
    setErr("");
  };

  const create = useMutation({
    mutationFn: () => {
      const config: Record<string, string> =
        type === "telegram"
          ? { token: token.trim(), chat_id: chatId.trim() }
          : { url: url.trim() };
      return api.createChannel(projectId, { name: name.trim(), type, config });
    },
    onSuccess: () => {
      reset();
      onAdded();
      toast.success("Channel added");
    },
    onError: (e) => setErr(e instanceof Error ? e.message : "failed"),
  });

  const valid =
    type === "telegram" ? token.trim() && chatId.trim() : url.trim();

  return (
    <Modal open={open} onClose={onClose} title="Add notification channel">
      <div className="space-y-3">
        <div className="grid grid-cols-2 gap-2">
          <div className="space-y-1.5">
            <Label>Type</Label>
            <Select className="w-full" value={type} onChange={(e) => setType(e.target.value)}>
              <option value="telegram">Telegram</option>
              <option value="slack">Slack</option>
              <option value="webhook">Webhook</option>
            </Select>
          </div>
          <div className="space-y-1.5">
            <Label>Name (optional)</Label>
            <Input value={name} onChange={(e) => setName(e.target.value)} placeholder="team-alerts" />
          </div>
        </div>

        {type === "telegram" ? (
          <>
            <div className="space-y-1.5">
              <Label>Bot token</Label>
              <Input type="password" value={token} onChange={(e) => setToken(e.target.value)} placeholder="123456:ABC-DEF…" />
            </div>
            <div className="space-y-1.5">
              <Label>Chat ID</Label>
              <Input value={chatId} onChange={(e) => setChatId(e.target.value)} placeholder="-1001234567890" />
            </div>
            <div className="rounded-md border border-border bg-surface-2 p-2.5 text-xs text-fg-muted">
              <p className="font-medium text-fg">How to get these</p>
              <ol className="ml-4 mt-1 list-decimal space-y-0.5">
                <li>Create a bot via <code className="font-mono">@BotFather</code> → copy the token.</li>
                <li>Send your bot a message (or add it to a group).</li>
                <li>
                  Open <code className="font-mono">https://api.telegram.org/bot&lt;TOKEN&gt;/getUpdates</code> →
                  copy <code className="font-mono">chat.id</code>.
                </li>
              </ol>
            </div>
          </>
        ) : (
          <div className="space-y-1.5">
            <Label>{type === "slack" ? "Slack webhook URL" : "Webhook URL"}</Label>
            <Input
              value={url}
              onChange={(e) => setUrl(e.target.value)}
              placeholder={type === "slack" ? "https://hooks.slack.com/services/…" : "https://example.com/webhook"}
            />
          </div>
        )}

        <p className="text-xs text-fg-muted">Secrets are stored encrypted (AES-256-GCM). Use “Test” after saving.</p>
        {err && <p className="text-xs text-danger">{err}</p>}
        <div className="flex justify-end gap-2">
          <Button variant="ghost" size="sm" onClick={onClose}>
            Cancel
          </Button>
          <Button
            size="sm"
            disabled={!valid || create.isPending}
            onClick={() => {
              setErr("");
              create.mutate();
            }}
          >
            {create.isPending ? "Saving…" : "Add channel"}
          </Button>
        </div>
      </div>
    </Modal>
  );
}
