import { useParams } from "react-router-dom";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Sparkles, RefreshCw } from "lucide-react";
import { api } from "@/services/api";
import { useWebSocket } from "@/hooks/useWebSocket";
import {
  Button,
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/primitives";
import { PageHeader, Spinner } from "@/components/common";
import { TimelineFeed } from "./TimelineFeed";

export function TimelinePage() {
  const { projectId } = useParams();
  const qc = useQueryClient();

  const timeline = useQuery({
    queryKey: ["timeline", projectId],
    queryFn: () => api.timeline(projectId!),
  });
  const rca = useQuery({
    queryKey: ["rca", projectId],
    queryFn: () => api.analyze(projectId!, 120),
  });

  useWebSocket(projectId, () => {
    qc.invalidateQueries({ queryKey: ["timeline", projectId] });
  });

  return (
    <div>
      <PageHeader
        title="Incident Timeline"
        description="Chronological view of deployments, spikes, errors and alerts"
        actions={
          <Button
            variant="outline"
            size="sm"
            onClick={() => {
              timeline.refetch();
              rca.refetch();
            }}
          >
            <RefreshCw className="h-4 w-4" /> Refresh
          </Button>
        }
      />

      {/* Root Cause Analysis */}
      <Card className="mb-6 border-primary/30 bg-gradient-to-br from-primary/5 to-transparent">
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Sparkles className="h-4 w-4 text-primary" />
            Root Cause Analysis
          </CardTitle>
        </CardHeader>
        <CardContent>
          {rca.isLoading ? (
            <Spinner />
          ) : rca.data ? (
            <div>
              <div className="flex items-start justify-between gap-4">
                <p className="text-sm text-fg leading-relaxed flex-1">
                  {rca.data.summary}
                </p>
                <div className="text-right shrink-0">
                  <div className="text-2xl font-semibold text-primary">
                    {Math.round(rca.data.confidence * 100)}%
                  </div>
                  <div className="text-xs text-fg-muted">confidence</div>
                </div>
              </div>
              {rca.data.evidence.length > 0 && (
                <div className="mt-4 border-t border-border pt-3">
                  <p className="text-xs font-medium text-fg-muted mb-2">
                    Evidence
                  </p>
                  <ul className="space-y-1">
                    {rca.data.evidence.map((e, i) => (
                      <li
                        key={i}
                        className="text-xs text-fg-muted flex gap-2"
                      >
                        <span className="text-primary">→</span>
                        {e}
                      </li>
                    ))}
                  </ul>
                </div>
              )}
            </div>
          ) : (
            <p className="text-sm text-fg-muted">Analysis unavailable.</p>
          )}
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Event Sequence</CardTitle>
        </CardHeader>
        <CardContent>
          {timeline.isLoading ? (
            <Spinner />
          ) : (
            <TimelineFeed events={timeline.data ?? []} />
          )}
        </CardContent>
      </Card>
    </div>
  );
}
