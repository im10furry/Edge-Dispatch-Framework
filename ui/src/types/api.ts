export interface PaginatedResponse<T> {
  data: T[];
  total: number;
  limit: number;
  offset: number;
  has_more: boolean;
}

export type NodeStatus = 'REGISTERED' | 'ACTIVE' | 'DEGRADED' | 'QUARANTINED' | 'OFFLINE' | 'DISABLED' | 'MAINTENANCE';

export type UserRole = 'tenant_owner' | 'tenant_admin' | 'project_operator' | 'project_viewer';

export type AdminPolicyType = 'dispatch' | 'block';

export type IngressType = '302' | 'dns' | 'gateway';

export type TaskType = 'prewarm' | 'purge' | 'block';

export type TaskStatus = 'pending' | 'running' | 'completed' | 'failed' | 'cancelled';

export interface Endpoint {
  scheme: string;
  host: string;
  port: number;
}

export interface Capabilities {
  inbound_reachable: boolean;
  cache_disk_gb: number;
  max_uplink_mbps: number;
  supports_https?: boolean;
  nat_type?: string;
  tunnel_required?: boolean;
}

export interface NodeScores {
  reachable_score: number;
  health_score: number;
  risk_score: number;
}

export interface NodeRuntime {
  cpu: number;
  mem_mb: number;
  disk_free_gb: number;
  conn: number;
}

export interface NodeTraffic {
  egress_mbps: number;
  ingress_mbps: number;
  err_5xx_rate: number;
}

export interface NodeCache {
  hit_ratio: number;
}

export interface Node {
  node_id: string;
  tenant_id: string;
  project_id: string;
  name: string;
  region: string;
  isp: string;
  asn: string;
  status: NodeStatus;
  weight: number;
  last_seen_at: string;
  created_at: string;
  updated_at: string;
  endpoints: Endpoint[];
  capabilities: Capabilities;
  scores: NodeScores;
  labels: Record<string, string>;
  disable_reason: string;
  maintain_until: string | null;
  runtime?: NodeRuntime;
  traffic?: NodeTraffic;
  cache?: NodeCache;
}

export interface RegisterRequest {
  node_name: string;
  public_endpoints: Endpoint[];
  region: string;
  isp: string;
  tenant_id?: string;
  project_id?: string;
  capabilities: Capabilities;
}

export interface NodeAuth {
  type: string;
  token: string;
  exp: number;
}

export interface RegisterResponse {
  node_id: string;
  auth: NodeAuth;
}

export interface RoleBinding {
  role: UserRole;
  tenant_id: string;
  project_id: string;
}

export interface User {
  user_id: string;
  tenant_id: string;
  email: string;
  display_name: string;
  created_at: string;
  updated_at: string;
  roles: RoleBinding[];
}

export interface LoginRequest {
  email: string;
  password: string;
}

export interface LoginResponse {
  token: string;
  refresh_token: string;
  user: User;
  expires_at: number;
  roles: RoleBinding[];
}

export interface RefreshRequest {
  refresh_token: string;
}

export interface RefreshResponse {
  token: string;
  expires_at: number;
  refresh_token: string;
}

export interface Tenant {
  tenant_id: string;
  name: string;
  description: string;
  created_at: string;
  updated_at: string;
}

export interface CreateTenantRequest {
  name: string;
  description?: string;
}

export interface UpdateTenantRequest {
  name?: string;
  description?: string;
}

export interface Project {
  project_id: string;
  tenant_id: string;
  name: string;
  description: string;
  created_at: string;
  updated_at: string;
}

export interface CreateProjectRequest {
  name: string;
  description?: string;
}

export interface UpdateProjectRequest {
  name?: string;
  description?: string;
}

export interface CreateUserRequest {
  tenant_id?: string;
  email: string;
  display_name?: string;
  password: string;
  roles?: RoleBinding[];
}

export interface UpdateUserRequest {
  display_name?: string;
  roles?: RoleBinding[];
}

export interface NodeAdminPatch {
  labels?: Record<string, string>;
  weight?: number;
  project_id?: string;
  tenant_id?: string;
  disabled?: boolean;
  disable_reason?: string;
  disable_until?: string;
}

export interface DisableNodeRequest {
  reason: string;
  message?: string;
  until?: string;
}

export interface AdminPolicy {
  policy_id: string;
  name: string;
  type: AdminPolicyType;
  content: unknown;
  description: string;
  tenant_id: string;
  project_id: string;
  version: number;
  is_published: boolean;
  created_at: string;
  updated_at: string;
}

export interface CreatePolicyRequest {
  name: string;
  type?: AdminPolicyType;
  content?: unknown;
  description?: string;
  tenant_id?: string;
  project_id?: string;
}

export interface UpdatePolicyRequest {
  name?: string;
  type?: AdminPolicyType;
  content?: unknown;
  description?: string;
}

export interface AdminPolicyVersion {
  version_id: string;
  policy_id: string;
  version: number;
  content: unknown;
  created_at: string;
}

export interface Ingress {
  ingress_id: string;
  name: string;
  type: IngressType;
  domain: string;
  config: unknown;
  tenant_id: string;
  project_id: string;
  created_at: string;
  updated_at: string;
}

export interface CreateIngressRequest {
  name: string;
  type?: IngressType;
  domain?: string;
  config?: unknown;
  tenant_id?: string;
  project_id?: string;
}

export interface UpdateIngressRequest {
  name?: string;
  type?: IngressType;
  domain?: string;
  config?: unknown;
}

export interface TaskParams {
  target_scope: 'all' | 'region' | 'node_list';
  object_key?: string;
  key_prefix?: string;
  target_nodes?: string[];
  concurrency?: number;
  rate_limit?: number;
}

export interface Task {
  task_id: string;
  tenant_id: string;
  project_id: string;
  creator_id: string;
  type: TaskType;
  status: TaskStatus;
  progress: number;
  total_nodes: number;
  done_nodes: number;
  params: unknown;
  result: unknown;
  created_at: string;
  updated_at: string;
}

export interface DashboardAlert {
  timestamp: string;
  type: 'node_offline' | 'origin_spike' | 'error_spike';
  message: string;
  node_id: string;
}

export interface DashboardMetrics {
  qps: number;
  total_requests: number;
  success_rate: number;
  error_rate: number;
  hit_rate: number;
  origin_ratio: number;
  p95_latency_ms: number;
  online_nodes: number;
  offline_nodes: number;
  total_nodes: number;
  recent_alerts: DashboardAlert[];
}

export interface AdminConfig {
  oidc_enabled: boolean;
  oidc_provider_url: string;
  oidc_client_id: string;
  local_auth_enabled: boolean;
  read_only_mode: boolean;
  ip_allowlist: string[];
  grafana_url: string;
  prometheus_url: string;
  loki_url: string;
}

export interface AuditEvent {
  id: string;
  tenant_id: string;
  project_id: string;
  actor_id: string;
  actor_email: string;
  action: string;
  resource_type: string;
  resource_id: string;
  before: unknown;
  after: unknown;
  request_id: string;
  source_ip: string;
  user_agent: string;
  result: string;
  created_at: string;
}

export interface ErrorDetail {
  code: string;
  message: string;
  request_id: string;
  details?: Record<string, string>;
}

export interface ErrorResponse {
  error: ErrorDetail;
}
