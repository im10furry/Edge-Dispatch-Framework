import type { NodeStatus } from '../../types/api';

export const NODE_STATUS_MAP: Record<NodeStatus, string> = {
  REGISTERED: '已注册',
  ACTIVE: '在线',
  DEGRADED: '降级',
  QUARANTINED: '隔离',
  OFFLINE: '离线',
  DISABLED: '已禁用',
  MAINTENANCE: '维护中',
};
