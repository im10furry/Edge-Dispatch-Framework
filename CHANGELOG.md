# 更新日志（Changelog）

本项目遵循 [Semantic Versioning](https://semver.org/) 和 [Keep a Changelog](https://keepachangelog.com/) 规范。

---

## [v0.5.0] - 2026-04-29

### 新增
- **Prometheus 格式指标端点**
  - 控制面 `/metrics` 端点：dispatch 请求计数、错误计数、心跳/注册计数、节点状态 gauge
  - 边缘节点 `/metrics` 端点（Prometheus 格式，保留 `?format=json` 兼容）
  - 网关 `/metrics` 端点（Prometheus 格式，保留 `?format=json` 兼容）
  - 轻量级 `internal/metrics` 包（零外部依赖，支持 Counter/Gauge，标准 exposition 格式）
- **Hot Key TTL 过期清理**
  - `ContentIndexStore.StartCleanup()` 后台协程定期清理过期热键
  - 自动从 PostgreSQL 删除过期记录并重新加载内存索引
  - 可配置清理间隔（`CP_CI_HOT_KEY_TTL`，默认 5 分钟）

### 修复
- **302 重定向 URL 缺少 `/obj/` 前缀**（Bug v0.1）：`api.go` 中 `redirectURL` 格式修正为 `%s/obj/%s`
- **Gateway Resolver 请求方法与路径不匹配**（Bug v0.3）：
  - 请求方法从 GET 改为 POST
  - URL 路径从 `/api/v1/dispatch/resolve` 修正为 `/v1/dispatch/resolve`
  - URL 路径从 `/api/v1/nodes/{nodeID}` 修正为 `/v1/nodes/{nodeID}`
  - 请求体 JSON 现在正确发送
- **CPU/流量指标硬编码为 0**（Bug v0.1）：
  - CPU：使用 `runtime.NumGoroutine() / runtime.NumCPU()` 作为负载指标
  - Egress：基于 `BytesSent` 增量计算 Mbps（心跳间隔差分）
  - Error Rate：基于错误计数 / 请求计数计算

### 变更
- 边缘节点 Server 新增 `GetMetricsDelta()` 方法，支持指标快照差分
- 控制面 Registry 新增 `CountByStatus()` 方法
- PGStore 新增 `CountByStatus()` 查询方法

---

## [Unreleased]

### Review and hardening
- Hardened Control Plane JSON request parsing with media-type parsing, unknown-field rejection, multi-value rejection, and exact oversized body handling.
- Fixed object key sanitization and redirect path escaping to reject traversal-style keys before dispatch.
- Fixed gateway reverse proxy token forwarding so gateway-selected public and tunnel edge nodes receive dispatch access tokens.
- Removed shared reverse-proxy Director mutation in the gateway request path.
- Fixed DNS adapter Control Plane authentication and split compose defaults for gateway/DNS API tokens.
- Fixed edge cache overwrite accounting, edge metrics double counting, HEAD responses, bounded streaming reads, and origin Range streaming.
- Hardened admin/session auth with random refresh tokens, JWT header validation, bcrypt error handling, and idempotent shutdown helpers.
- Fixed IPv6 endpoint formatting and QUIC test/listener behavior with HTTP/3 ALPN configuration.

### 新增 (v0.6 — 小带宽环境优化)
- **带宽感知调度**：`scoreBandwidth()` 在调度评分中引入节点上行带宽容量、实时流量负载、小带宽节点缓存内容奖励
- **小带宽节点过滤**：`filterForSmallBandwidth()` 排除无缓存内容的小带宽节点，避免无效调度
- **带宽限制器**：`BandwidthLimiter` 基于 token bucket 的流量整形，支持动态调整带宽限制
- **P2P 互助网络**：`P2PFetcher` 节点间内容互助获取，P2P 优先回源策略，`/internal/p2p/obj/{key}` 端点
- **智能预拉取**：`SmartPrefetchManager` 支持夜间/白天双模式、优先级队列、缓存命中检查
- **本地配置 Web UI**：`internal/edgeagent/localconfig` 节点本地可视化配置界面，支持 4 步配置向导、连接测试、磁盘检查、热重载
- **全局配置 API**：`GET/PUT /internal/admin/v1/config` 集中式小带宽/P2P/预拉取配置管理
- **P2P 拓扑 API**：`GET /internal/admin/v1/p2p/topology` P2P 网络拓扑数据
- **Dashboard API**：`GET /internal/admin/v1/dashboard` 节点在线/离线统计
- **配置扩展**：`Capabilities` 新增 `SupportsP2P`/`CurrentEgressMbps`/`CurrentIngressMbps`/`ShieldMode`；`SmallBandwidthConfig` 统一管理小带宽优化配置
- **内容预热下发**：CP `POST /v1/tasks/prewarm` 将内容 Key 推送到所有活跃边缘节点，Edge Agent `/internal/push/prewarm` 接收并回源缓存
- **控制平面管理面板**：`:8082` 端口独立 Admin Dashboard，集群概览/节点管理/预热下发
- **大文件流式缓存**：>10MB 文件使用临时文件替代内存 buffer，WriteTimeout 提升至 300s
- **节点公网 IP 自动检测**：`Reporter.detectPublicIP()` 自动探测公网 IP，支持 `EA_PUBLIC_HOST` 手动指定
- **宽带指标实时上报**：`BandwidthMeter` 统计出入站 Mbps，在 /metrics 和状态页展示
- **一键安装脚本**：`install.sh` 交互式 + 命令行模式，支持 Control Plane / Edge Agent / Origin 三角色
- **UI 全面重设计**：深色主题 + 毛玻璃 + 渐变背景 + 微交互动画，移动端响应式
- **IPv6 双栈支持**：监听失败时自动回退 `[::]:port` 启用 IPv4/v6 双栈
- **Origin Shield 模式**：`EA_SHIELD_MODE=true` 标记为中间缓存层，调度 +20 分，P2P 优先，不限速

### 计划
- HTTP/3 QUIC 支持
- Helm Chart 部署

---

## [v0.4.0] - 2026-04-29

### 新增
- **流媒体支持**：HLS/DASH 分片策略
  - HLS manifest（.m3u8）解析
  - DASH manifest（.mpd）解析
  - 滑动窗口缓存（per-stream 淘汰）
  - 分片请求触发预取（look-ahead prefetch）
  - Manifest 更新触发预取
  - 流感知调度加分（+20 权重）
  - 流媒体指标（chunk 请求/命中/预取/淘汰）
- **边缘节点流媒体集成**：`/obj/` 路径自动检测流媒体请求并路由到 Streaming Handler
- **Streaming 配置**：`StreamingConfig` 支持预取数量、窗口大小、worker 数量等参数

### 变更
- 边缘节点 `/metrics` 端点增加 streaming 指标
- 调度评分模型增加流媒体亲和度因子

---

## [v0.3.0] - 2026-04-29

### 新增
- **网关反向代理**（`cmd/gateway`）
  - HTTP 反向代理（`httputil.ReverseProxy`）
  - 公网节点直连代理
  - NAT 节点隧道代理
  - Hop-by-hop header 剥离
  - X-Forwarded-* 头注入
  - 路径遍历防护
- **反向隧道协议**（`internal/tunnel/`）
  - 二进制隧道协议（10 字节头 + 变长 payload）
  - 8 种消息类型（Register/RegisterOK/Heartbeat/Error/Request/Response/Data/DataEnd）
  - 流多路复用（Stream Multiplexing）
  - 双向 Body 流式传输
  - 隧道客户端（NAT 侧）：连接、心跳、请求转发、自动重连
  - 隧道服务器（网关侧）：监听、Token 认证、最大隧道数限制、请求代理
- **NAT 节点支持**
  - `EA_NAT_MODE` 配置项
  - `EA_TUNNEL_SERVER_ADDR` 配置项
  - `Capabilities.TunnelRequired` 模型字段
  - 调度器隧道亲和度（`-15` 惩罚分）
- **Docker Compose 扩展**
  - `gateway` 服务（HTTP 代理 + 隧道服务器）
  - `edge-agent-nat` 服务（NAT 模式边缘节点）

### 变更
- 调度器增加 `SetTunnelManager()` 接口
- 调度过滤逻辑增加 NAT 节点隧道连通性检查
- 模型增加 `ProxyMode`（direct/tunnel）字段

---

## [v0.2.0] - 2026-04-29

### 新增
- **内容索引**（`internal/contentindex/`）
  - Bloom Filter 实现（FNV-128a 双哈希，可配置 FP 率）
  - 热内容精确表（PostgreSQL 持久化）
  - 冷内容 Bloom 摘要
  - 双阶段查找（热内容 → Bloom 过滤）
  - 内存快速索引（线程安全 map）
  - 启动时全量加载
- **DNS/GSLB 适配器**（`cmd/dns-adapter`、`internal/dns/`）
  - UDP DNS 服务器（完整报文解析）
  - A/AAAA 记录响应（含 name pointer 压缩）
  - NXDOMAIN 响应
  - 域名后缀匹配
  - 缓存刷新循环（定期调用 dispatch API）
  - 控制面集成
- **内容感知调度**
  - `ContentIndexLookup` 接口
  - 调度评分增加内容亲和度（hot +25, cached +10）
  - 心跳处理集成 ContentSummary 上报
- **Docker Compose 扩展**
  - `dns-adapter` 服务

### 变更
- 调度器增加 `SetContentIndex()` 接口
- 心跳处理器增加 `SetContentStore()` 接口
- `HeartbeatRequest` 增加 `ContentSummary` 字段
- 数据模型增加 `ContentSummary`、`ContentIndexEntry`、`DNSQuery/Response/Record`

---

## [v0.1.0] - 2026-04-29

### 新增

#### 控制面（Control Plane）
- **节点注册与鉴权**
  - `POST /v1/nodes/register` — 节点注册，返回 UUID + Bearer Token
  - `DELETE /v1/nodes/{nodeID}` — 吊销节点
  - `GET /v1/nodes/{nodeID}` — 查询节点信息
  - Bearer Token 鉴权中间件
- **心跳与状态上报**
  - `POST /v1/nodes/heartbeat` — 批量异步处理（50ms 窗口，256 缓冲）
  - Redis 心跳存储（gzip 压缩）
  - PostgreSQL last_seen 更新
  - 状态机：Registered → Active → Degraded → Offline
  - 事件回调（node.online / node.degraded / node.offline）
- **可达性探活**
  - TCP 连接探测 + HTTP GET `/healthz`
  - 并发探测（信号量控制，10 goroutines）
  - 批量保存探测结果（32 条一批）
  - 分数计算：ReachableScore、HealthScore、RiskScore
- **Top-K 调度**
  - `POST /v1/dispatch/resolve` — 纯 API 调度
  - `GET /obj/{key}` — 302 重定向入口
  - 多因子评分模型：region(+30)、ISP(+20)、reachability、health、risk
  - HMAC-SHA256 Token 签名（绑定 resource_key + exp + IP 前缀）
  - 降级到源站（`DegradeToOrigin` 配置）
- **策略引擎**
  - IP 黑名单（内存）
  - 节点黑名单（内存）
  - 全局速率限制（令牌桶，1000 req/s，burst 2000）
- **存储层**
  - PostgreSQL：节点表 + 探测结果表 + 7 个索引 + 自动迁移
  - Redis：心跳存储（pipeline 批量 + Lua 原子脚本）+ 事件发布
  - 连接池：PG 50 max/10 min，Redis 50 max
  - 后台维护：VACUUM 6h、探测清理 1h
  - 节点缓存：sync.Map 30s TTL

#### 边缘节点（Edge Agent）
- **HTTP 服务**
  - `GET /obj/{key}` — 对象服务（Token 验证 + 缓存 + 回源）
  - `GET /healthz` — 健康检查
  - `GET /metrics` — JSON 格式指标
  - Range 请求支持（206 Partial Content）
  - Gzip 压缩（按内容类型选择性压缩）
  - TLS 支持
- **磁盘缓存**
  - LRU 淘汰策略（按 lastAccess 排序）
  - MD5 目录结构
  - meta.json 元数据（大小、访问时间）
  - 启动时恢复已有缓存
- **回源**
  - HTTP/2 支持 + 连接池（200 idle，50/host）
  - 请求去重（并发请求合并）
  - 熔断器（closed/open/half-open，阈值 5，恢复 30s）
  - 重试（2 次，指数退避）
  - Range 回源支持
  - Key 路径遍历防护
- **上报**
  - 心跳上报（内存、磁盘、连接数、缓存命中率）
  - 自动注册（无 Token 时自动注册）
  - 内容摘要上报（热内容 + Bloom Filter）

#### 鉴权
- HMAC-SHA256 签名/验证
- IP 前缀绑定（/24 IPv4，/64 IPv6）
- Token 格式：`base64url(payload).base64url(signature)`

#### 源站（Origin）
- `GET /obj/{key}` — 文件服务（Range 支持）
- `GET /healthz` — 健康检查
- 路径遍历防护
- 自动创建示例文件

#### 压力测试工具（Stress）
- 5 种测试模式：direct / dispatch / bench-dispatch / bench-origin / bench-edge
- 并发 worker 模型
- 实时进度报告
- 详细统计：QPS、延迟分布（P50/P95/P99）、吞吐量

#### 基础设施
- Docker Compose 全栈编排（PostgreSQL、Redis、控制面、2 边缘节点、源站）
- 6 个 Dockerfile
- Makefile（构建、测试、Docker、Lint）
- 环境变量配置

#### 数据模型
- Node（5 种状态：Registered/Active/Degraded/Quarantined/Offline）
- Endpoint、Capabilities、NodeScores
- HeartbeatRequest（Runtime/Traffic/Cache）
- DispatchRequest/Response（ClientInfo/ResourceInfo/Candidate）
- ErrorResponse

#### 测试
- Token 签名/验证单元测试 + Benchmark
- Scheduler 单元测试 + Benchmark + 内容感知测试
- Policy 单元测试
- Prober 单元测试
- API 集成测试（需要 PostgreSQL + Redis）
- Cache 单元测试
- Bloom Filter 单元测试

---

## 版本说明

| 版本 | 日期 | 里程碑 | 核心特性 |
|------|------|--------|---------|
| v0.1.0 | 2026-04-29 | M1 | 控制面 + 边缘 + 302 调度 |
| v0.2.0 | 2026-04-29 | M2 | 内容索引 + DNS 适配器 |
| v0.3.0 | 2026-04-29 | M3 | 网关反代 + 隧道 |
| v0.4.0 | 2026-04-29 | — | 流媒体 + 预取 |
| v0.5.0 | 2026-04-29 | — | Prometheus 指标 + Bug 修复 |
