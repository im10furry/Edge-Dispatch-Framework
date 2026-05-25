import { useRef, useCallback } from 'react';

export interface TimePoint {
  time: string;
  value: number;
}

export interface DashboardHistory {
  qps: TimePoint[];
  successRate: TimePoint[];
  hitRate: TimePoint[];
  originRatio: TimePoint[];
  latency: TimePoint[];
  onlineNodes: TimePoint[];
  offlineNodes: TimePoint[];
}

const MAX_POINTS = 60;

export function useDashboardHistory(maxPoints = MAX_POINTS) {
  const historyRef = useRef<DashboardHistory>({
    qps: [],
    successRate: [],
    hitRate: [],
    originRatio: [],
    latency: [],
    onlineNodes: [],
    offlineNodes: [],
  });

  const maxRef = useRef(maxPoints);

  const append = useCallback(
    (data: {
      qps: number;
      success_rate: number;
      hit_rate: number;
      origin_ratio: number;
      p95_latency_ms: number;
      online_nodes: number;
      offline_nodes: number;
    }) => {
      const h = historyRef.current;
      const now = new Date().toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit', second: '2-digit' });

      const push = (arr: TimePoint[], value: number) => {
        arr.push({ time: now, value });
        if (arr.length > maxRef.current) arr.shift();
      };

      push(h.qps, data.qps);
      push(h.successRate, data.success_rate * 100);
      push(h.hitRate, data.hit_rate * 100);
      push(h.originRatio, data.origin_ratio * 100);
      push(h.latency, data.p95_latency_ms);
      push(h.onlineNodes, data.online_nodes);
      push(h.offlineNodes, data.offline_nodes);
    },
    [],
  );

  return { history: historyRef, append };
}
