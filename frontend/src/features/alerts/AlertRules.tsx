import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Plus, Trash2, BellRing, Pencil } from "lucide-react";
import { api } from "@/services/api";
import { toast } from "@/lib/toast";
import {
  Button,
  Card,
  Input,
  Label,
  Select,
  Badge,
} from "@/components/ui/primitives";
import { EmptyState, Spinner, Modal, SeverityBadge } from "@/components/common";
import { METRIC_NAMES, type AlertRule, type Service } from "@/types";

const OP_LABEL: Record<string, string> = { gt: ">", lt: "<", gte: "≥", lte: "≤" };

export function AlertRules({ projectId }: { projectId: string }) {
  const qc = useQueryClient();
  const [modal, setModal] = useState(false);
  const [editing, setEditing] = useState<AlertRule | null>(null);

  const openNew = () => {
    setEditing(null);
    setModal(true);
  };
  const openEdit = (r: AlertRule) => {
    setEditing(r);
    setModal(true);
  };

  const rules = useQuery({
    queryKey: ["alert-rules", projectId],
    queryFn: () => api.listAlertRules(projectId),
  });
  const services = useQuery({
    queryKey: ["services", projectId],
    queryFn: () => api.listServices(projectId),
  });
  const serviceName = (id: string | null) =>
    id ? services.data?.find((s) => s.id === id)?.name ?? "service" : "all services";

  const toggle = useMutation({
    mutationFn: (r: AlertRule) =>
      api.updateAlertRule(projectId, r.id, { ...r, enabled: !r.enabled }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["alert-rules", projectId] }),
  });
  const del = useMutation({
    mutationFn: (id: string) => api.deleteAlertRule(projectId, id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["alert-rules", projectId] });
      toast.success("Rule deleted");
    },
  });

  return (
    <div>
      <div className="flex justify-end mb-3">
        <Button size="sm" onClick={openNew}>
          <Plus className="h-4 w-4" /> New rule
        </Button>
      </div>

      {rules.isLoading ? (
        <Spinner />
      ) : rules.data && rules.data.length > 0 ? (
        <div className="space-y-2">
          {rules.data.map((r) => (
            <Card key={r.id} className="p-4">
              <div className="flex items-start justify-between gap-4">
                <div className="flex items-start gap-3">
                  <BellRing
                    className={`h-4 w-4 mt-0.5 ${r.enabled ? "text-primary" : "text-fg-muted"}`}
                  />
                  <div>
                    <div className="flex items-center gap-2 flex-wrap">
                      <span className="text-sm font-medium text-fg">{r.name}</span>
                      <SeverityBadge severity={r.severity} />
                      {!r.enabled && <Badge tone="muted">disabled</Badge>}
                      {r.notify_type !== "none" && (
                        <Badge tone="info">{r.notify_type}</Badge>
                      )}
                    </div>
                    <p className="text-xs font-mono text-fg-muted mt-1">
                      {r.metric} {OP_LABEL[r.operator]} {r.threshold}
                      {r.for_seconds > 0 && ` for ${r.for_seconds}s`} ·{" "}
                      {serviceName(r.service_id)}
                    </p>
                  </div>
                </div>
                <div className="flex items-center gap-3">
                  <button
                    onClick={() => toggle.mutate(r)}
                    className={`text-xs rounded-md border px-2 h-7 transition ${
                      r.enabled
                        ? "border-success/40 text-success"
                        : "border-border text-fg-muted"
                    }`}
                  >
                    {r.enabled ? "Enabled" : "Disabled"}
                  </button>
                  <button
                    onClick={() => openEdit(r)}
                    className="text-fg-muted hover:text-fg transition"
                    title="Edit rule"
                  >
                    <Pencil className="h-4 w-4" />
                  </button>
                  <button
                    onClick={() => del.mutate(r.id)}
                    className="text-fg-muted hover:text-danger transition"
                    title="Delete rule"
                  >
                    <Trash2 className="h-4 w-4" />
                  </button>
                </div>
              </div>
            </Card>
          ))}
        </div>
      ) : (
        <EmptyState
          icon={<BellRing className="h-8 w-8" />}
          title="No alert rules"
          description="Create a rule to get notified automatically when a metric crosses a threshold."
          action={
            <Button size="sm" onClick={openNew}>
              <Plus className="h-4 w-4" /> Create rule
            </Button>
          }
        />
      )}

      <RuleModal
        key={editing?.id ?? "new"}
        open={modal}
        projectId={projectId}
        rule={editing}
        services={services.data ?? []}
        onClose={() => setModal(false)}
        onSaved={() => {
          qc.invalidateQueries({ queryKey: ["alert-rules", projectId] });
          setModal(false);
        }}
      />
    </div>
  );
}

function RuleModal({
  open,
  projectId,
  rule,
  services,
  onClose,
  onSaved,
}: {
  open: boolean;
  projectId: string;
  rule?: AlertRule | null;
  services: Service[];
  onClose: () => void;
  onSaved: () => void;
}) {
  const editing = !!rule;
  const [form, setForm] = useState(() => ({
    name: rule?.name ?? "",
    metric: rule?.metric ?? "error_rate",
    operator: rule?.operator ?? "gt",
    threshold: rule?.threshold ?? 5,
    for_seconds: rule?.for_seconds ?? 0,
    severity: rule?.severity ?? "high",
    type: rule?.type ?? "high_error_rate",
    service_id: rule?.service_id ?? "",
    // "" = off | channel id | "__inline__"
    notify: rule?.notify_channel_id
      ? rule.notify_channel_id
      : rule && rule.notify_type !== "none"
        ? "__inline__"
        : "",
    notify_type: rule && rule.notify_type !== "none" ? rule.notify_type : "none",
    notify_url: rule?.notify_url ?? "",
  }));
  const set = (k: string, v: string | number) => setForm((f) => ({ ...f, [k]: v }));

  const channels = useQuery({
    queryKey: ["channels", projectId],
    queryFn: () => api.listChannels(projectId),
  });

  const mutation = useMutation({
    mutationFn: () => {
      const inline = form.notify === "__inline__";
      const channelId = inline || form.notify === "" ? null : form.notify;
      const body = {
        name: form.name,
        metric: form.metric,
        operator: form.operator,
        threshold: Number(form.threshold),
        for_seconds: Number(form.for_seconds),
        severity: form.severity,
        type: form.type,
        service_id: form.service_id || null,
        notify_channel_id: channelId,
        notify_type: inline ? form.notify_type : "none",
        notify_url: inline ? form.notify_url : "",
      };
      return editing
        ? api.updateAlertRule(projectId, rule!.id, body as never)
        : api.createAlertRule(projectId, body as never);
    },
    onSuccess: () => {
      toast.success(editing ? "Alert rule saved" : "Alert rule created");
      onSaved();
    },
    onError: () => toast.error(editing ? "Couldn't save rule" : "Couldn't create rule"),
  });

  return (
    <Modal open={open} onClose={onClose} title={editing ? "Edit alert rule" : "New alert rule"}>
      <div className="space-y-3">
        <div className="space-y-1.5">
          <Label>Name</Label>
          <Input
            value={form.name}
            onChange={(e) => set("name", e.target.value)}
            placeholder="High error rate on checkout"
            autoFocus
          />
        </div>
        <div className="grid grid-cols-3 gap-2">
          <div className="space-y-1.5 col-span-1">
            <Label>Metric</Label>
            <Select className="w-full" value={form.metric} onChange={(e) => set("metric", e.target.value)}>
              {METRIC_NAMES.map((m) => (
                <option key={m} value={m}>
                  {m}
                </option>
              ))}
            </Select>
          </div>
          <div className="space-y-1.5">
            <Label>Op</Label>
            <Select className="w-full" value={form.operator} onChange={(e) => set("operator", e.target.value)}>
              <option value="gt">&gt;</option>
              <option value="gte">≥</option>
              <option value="lt">&lt;</option>
              <option value="lte">≤</option>
            </Select>
          </div>
          <div className="space-y-1.5">
            <Label>Threshold</Label>
            <Input
              type="number"
              value={form.threshold}
              onChange={(e) => set("threshold", e.target.value)}
            />
          </div>
        </div>
        <div className="grid grid-cols-3 gap-2">
          <div className="space-y-1.5">
            <Label>For (s)</Label>
            <Input
              type="number"
              value={form.for_seconds}
              onChange={(e) => set("for_seconds", e.target.value)}
            />
          </div>
          <div className="space-y-1.5">
            <Label>Severity</Label>
            <Select className="w-full" value={form.severity} onChange={(e) => set("severity", e.target.value)}>
              <option value="low">low</option>
              <option value="medium">medium</option>
              <option value="high">high</option>
              <option value="critical">critical</option>
            </Select>
          </div>
          <div className="space-y-1.5">
            <Label>Type</Label>
            <Select className="w-full" value={form.type} onChange={(e) => set("type", e.target.value)}>
              <option value="high_latency">high_latency</option>
              <option value="high_error_rate">high_error_rate</option>
              <option value="service_down">service_down</option>
            </Select>
          </div>
        </div>
        <div className="space-y-1.5">
          <Label>Service</Label>
          <Select className="w-full" value={form.service_id} onChange={(e) => set("service_id", e.target.value)}>
            <option value="">All services</option>
            {services.map((s) => (
              <option key={s.id} value={s.id}>
                {s.name}
              </option>
            ))}
          </Select>
        </div>
        <div className="space-y-1.5">
          <Label>Notify</Label>
          <Select className="w-full" value={form.notify} onChange={(e) => set("notify", e.target.value)}>
            <option value="">Off</option>
            {(channels.data ?? []).map((ch) => (
              <option key={ch.id} value={ch.id}>
                {ch.name} · {ch.type}
              </option>
            ))}
            <option value="__inline__">Custom inline URL…</option>
          </Select>
          {(channels.data ?? []).length === 0 && (
            <p className="text-xs text-fg-muted">
              Tip: save a Telegram bot / Slack channel in Settings → Notification channels, then reuse it here.
            </p>
          )}
        </div>
        {form.notify === "__inline__" && (
          <div className="grid grid-cols-2 gap-2">
            <div className="space-y-1.5">
              <Label>Type</Label>
              <Select className="w-full" value={form.notify_type} onChange={(e) => set("notify_type", e.target.value)}>
                <option value="slack">Slack</option>
                <option value="telegram">Telegram</option>
                <option value="webhook">Webhook</option>
              </Select>
            </div>
            <div className="space-y-1.5">
              <Label>URL</Label>
              <Input
                value={form.notify_url}
                onChange={(e) => set("notify_url", e.target.value)}
                placeholder={
                  form.notify_type === "telegram"
                    ? "https://api.telegram.org/bot<TOKEN>/sendMessage?chat_id=<ID>"
                    : form.notify_type === "slack"
                      ? "https://hooks.slack.com/…"
                      : "https://example.com/webhook"
                }
              />
            </div>
          </div>
        )}
        <div className="flex justify-end gap-2 pt-1">
          <Button variant="ghost" size="sm" onClick={onClose}>
            Cancel
          </Button>
          <Button
            size="sm"
            disabled={form.name.length < 1 || mutation.isPending}
            onClick={() => mutation.mutate()}
          >
            {editing ? "Save changes" : "Create rule"}
          </Button>
        </div>
      </div>
    </Modal>
  );
}
