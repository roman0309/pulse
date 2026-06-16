import ReactECharts from "echarts-for-react";
import type { MetricSeries } from "@/types";

interface Props {
  series: MetricSeries[];
  serviceNames?: Record<string, string>;
  unit?: string;
  height?: number;
}

const PALETTE = ["#6366f1", "#22c55e", "#f59e0b", "#ef4444", "#06b6d4", "#a855f7"];

export function MetricChart({ series, serviceNames = {}, unit = "", height = 240 }: Props) {
  const option = {
    grid: { top: 24, right: 16, bottom: 28, left: 44 },
    tooltip: {
      trigger: "axis",
      backgroundColor: "rgba(20,20,24,0.95)",
      borderColor: "#2a2a35",
      textStyle: { color: "#fafafa", fontSize: 12 },
      valueFormatter: (v: number) => `${v.toFixed(1)}${unit}`,
    },
    legend: {
      show: series.length > 1,
      top: 0,
      textStyle: { color: "#9ca3af", fontSize: 11 },
      icon: "roundRect",
    },
    xAxis: {
      type: "time",
      axisLine: { lineStyle: { color: "#2a2a35" } },
      axisLabel: { color: "#6b7280", fontSize: 11 },
      splitLine: { show: false },
    },
    yAxis: {
      type: "value",
      axisLabel: { color: "#6b7280", fontSize: 11, formatter: `{value}${unit}` },
      splitLine: { lineStyle: { color: "#1c1c24" } },
    },
    series: series.map((s, i) => ({
      name: serviceNames[s.service_id] || "series",
      type: "line",
      smooth: true,
      showSymbol: false,
      lineStyle: { width: 2, color: PALETTE[i % PALETTE.length] },
      areaStyle: {
        color: {
          type: "linear",
          x: 0, y: 0, x2: 0, y2: 1,
          colorStops: [
            { offset: 0, color: PALETTE[i % PALETTE.length] + "33" },
            { offset: 1, color: PALETTE[i % PALETTE.length] + "00" },
          ],
        },
      },
      data: s.points.map((p) => [p.timestamp, p.value]),
    })),
  };

  return (
    <ReactECharts
      option={option}
      style={{ height, width: "100%" }}
      notMerge
      lazyUpdate
    />
  );
}
