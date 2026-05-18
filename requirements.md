# Product Requirements Document (PRD) — Edge Dispatch Framework

## 1. 产品概述

### 1.1 产品定义

Edge Dispatch Framework 是一个开源边缘分发/加速框架，通过中心调度将用户请求分配到最优边缘节点，实现低延迟、高可用的内容分发。

### 1.2 目标用户

- CDN 运维工程师（部署、监控、管理边缘节点）
- 内容提供商（接入边缘加速服务）
- 开源社区贡献者（扩展调度策略、存储后端、协议支持）

### 1.3 核心价值

| 维度 | 目标 |
|------|------|
| 延迟 | P95 < 50ms（边缘命中），P95 < 200ms（回源） |
| 可用性 | 99.9% 调度可用性，单节点故障 < 30s 摘除 |
| 扩展性 | 单控制面支持 1000+ 边缘节点 |
| 成本 | 利用 NAT/小带宽节点，降低带宽成本 40%+ |

---

## 2. 核心功能需求

### 2.1 控制面 (Control Plane)

#### F-CP-001: 节点注册与鉴权
- [x] v0.1: `POST /v1/nodes/register` 节点注册，返回 UUID + Bearer Token
- [x] v0.1: `DELETE /v1/nodes/{nodeID}` 吊销节点
- [x] v0.1: `GET /v1/nodes/{nodeID}` 查询节点信息
- [x] v0.1: Bearer Token 鉴权中间件

#### F-CP-002: 心跳与状态管理
- [x] v0.1: `POST /v1/nodes/heartbeat` 批量异步处理（50ms 窗口，256 缓冲）
- [x] v0.1: Redis 心跳存储（gzip 压缩 >256B）
- [x] v0.1: PostgreSQL last_seen 更新
- [x] v0.1: 状态机：Registered → Active → Degraded → Quarantined → Offline
- [x] v0.1: 事件回调（node.online / node.degraded / node.offline）
- [x] v0.5: CPU/流量/错误率指标实时计算

#### F-CP-003: 可达性探活
- [x] v0.1: TCP 连接探测 + HTTP GET `/healthz`
- [x] v0.1: 并发探测（信号量控制）
- [x] v0.1: 批量保存探测结果
- [x] v0.1: ReachableScore / HealthScore / RiskScore 计算

#### F-CP-004: Top-K 调度
- [x] v0.1: `POST /v1/dispatch/resolve` 纯 API 调度
- [x] v0.1: `GET /obj/{key}` 302 重定向入口
- [x] v0.1: 多因子评分模型：region(+30)、ISP(+20)、reachability、health、risk
- [x] v0.1: HMAC-SHA256 Token 签名（绑定 resource_key + exp + IP 前缀）
- [x] v0.1: 降级到源站（DegradeToOrigin）
- [x] v0.2: 内容感知调度（hot +25, cached +10）
- [x] v0.3: 隧道亲和度（-15 惩罚分）
- [x] v0.4: 流媒体亲和度（+20 权重）
- [x] v0.6: 带宽感知调度（scoreBandwidth）
- [x] v0.6: 小带宽节点过滤（filterForSmallBandwidth）
- [x] v0.6: Origin Shield 模式（+20 分）

#### F-CP-005: 策略引擎
- [x] v0.1: IP 黑名单（内存）
- [x] v0.1: 节点黑名单（内存）
- [x] v0.1: 全局速率限制（令牌桶，1000 req/s）

#### F-CP-006: 内容索引
- [x] v0.2: Bloom Filter 实现（FNV-128a 双哈希）
- [x] v0.2: 热内容精确表（PostgreSQL 持久化）
- [x] v0.2: 冷内容 Bloom 摘要
- [x] v0.2: 双阶段查找（热内容 → Bloom 过滤）
- [x] v0.2: 内存快速索引（线程安全 map）
- [x] v0.5: Hot Key TTL 过期清理后台协程

#### F-CP-007: 管理 API
- [x] v0.6: `GET/PUT /internal/admin/v1/config` 全局配置管理
- [x] v0.6: `GET /internal/admin/v1/p2p/topology` P2P 拓扑数据
- [x] v0.6: `GET /internal/admin/v1/dashboard` 节点在线/离线统计
- [x] v0.6: `POST /v1/tasks/prewarm` 内容预热下发
- [x] v0.6: 管理面板（Admin UI）：集群概览/节点管理/预热下发
- [ ] v0.7: 节点分组管理
- [ ] v0.7: 操作审计日志持久化

#### F-CP-008: 可观测性
- [x] v0.5: Prometheus `/metrics` 端点
- [x] v0.5: dispatch 请求计数、错误计数、心跳/注册计数、节点状态 gauge

---

### 2.2 边缘节点 (Edge Agent)

#### F-EA-001: HTTP 对象服务
- [x] v0.1: `GET /obj/{key}` 对象服务（Token 验证 + 缓存 + 回源）
- [x] v0.1: `GET /healthz` 健康检查
- [x] v0.1: Range 请求（206 Partial Content）
- [x] v0.1: Gzip 压缩
- [x] v0.1: TLS 支持
- [x] v0.6: 大文件流式缓存（>10MB）
- [x] v0.6: HEAD 请求响应修正
- [x] v0.6: `/internal/p2p/obj/{key}` P2P 端点
- [x] v0.6: `/internal/push/prewarm` 预热端点

#### F-EA-002: 磁盘缓存
- [x] v0.1: LRU 淘汰策略
- [x] v0.1: MD5 目录结构
- [x] v0.1: meta.json 元数据
- [x] v0.1: 启动恢复
- [x] v0.6: 缓存覆写计量修正

#### F-EA-003: 回源获取
- [x] v0.1: HTTP/2 支持 + 连接池
- [x] v0.1: 请求去重（并发合并）
- [x] v0.1: 熔断器（closed/open/half-open）
- [x] v0.1: 重试（2 次，指数退避）
- [x] v0.1: Range 回源支持
- [x] v0.1: Key 路径遍历防护

#### F-EA-004: 心跳上报
- [x] v0.1: 心跳上报（内存、磁盘、连接数、缓存命中率）
- [x] v0.1: 自动注册
- [x] v0.1: 内容摘要上报（热内容 + Bloom Filter）
- [x] v0.6: 公网 IP 自动检测
- [x] v0.6: 宽带指标上报（BandwidthMeter）

#### F-EA-005: 带宽管理
- [x] v0.6: BandwidthLimiter token bucket 限速
- [x] v0.6: 动态带宽限制调整
- [x] v0.6: 小带宽适配（SmallBandwidthConfig）

#### F-EA-006: P2P 互助
- [x] v0.6: P2PFetcher 节点间内容获取
- [x] v0.6: P2P 优先回源策略
- [x] v0.6: P2P 拓扑发现

#### F-EA-007: 智能预取
- [x] v0.6: SmartPrefetchManager 优先级队列
- [x] v0.6: 日间/夜间双模式
- [x] v0.6: 缓存命中检查

#### F-EA-008: 本地配置
- [x] v0.6: Web UI 本地配置界面
- [x] v0.6: 4 步配置向导
- [x] v0.6: 连接测试、磁盘检查
- [x] v0.6: 热重载

#### F-EA-009: 可观测性
- [x] v0.5: Prometheus `/metrics` 端点
- [x] v0.5: 流媒体指标
- [x] v0.6: 宽带指标实时展示

---

### 2.3 网关 (Gateway)

#### F-GW-001: HTTP 反向代理
- [x] v0.3: `httputil.ReverseProxy` 实现
- [x] v0.3: 公网节点直连代理
- [x] v0.3: Hop-by-hop header 剥离
- [x] v0.3: X-Forwarded-* 头注入
- [x] v0.3: 路径遍历防护
- [x] v0.6: Token 转发修正
- [x] v0.6: Director mutation 修复

#### F-GW-002: 隧道代理
- [x] v0.3: NAT 节点隧道代理
- [x] v0.3: 隧道服务器 Token 认证
- [x] v0.3: 最大隧道数限制

#### F-GW-003: QUIC/HTTP3
- [x] v0.6: HTTP/3 服务器（build tag `quic`）
- [x] v0.6: Alt-Svc 中间件
- [x] v0.6: TLS + ALPN 配置
- [ ] planned: QUIC 客户端（网关→边缘节点 QUIC 加速）
- [ ] planned: 0-RTT 重连

#### F-GW-004: 调度集成
- [x] v0.3: 控制面 dispatch resolve 客户端
- [x] v0.3: 节点查找 + 缓存
- [x] v0.5: 请求方法/路径修正

#### F-GW-005: 可观测性
- [x] v0.5: Prometheus `/metrics` 端点
- [ ] planned: 请求级别分布式追踪

---

### 2.4 DNS 适配器 (DNS Adapter)

#### F-DNS-001: UDP DNS 服务器
- [x] v0.2: 完整报文解析
- [x] v0.2: A/AAAA 记录响应（name pointer 压缩）
- [x] v0.2: NXDOMAIN 响应
- [x] v0.2: 域名后缀匹配
- [x] v0.2: 缓存刷新循环
- [x] v0.6: 控制面认证和 Token 分离

#### F-DNS-002: GSLB 集成
- [x] v0.2: 定期调用 dispatch API
- [ ] planned: 加权轮询 DNS
- [ ] planned: GeoDNS 地理位置解析

---

### 2.5 流媒体 (Streaming)

#### F-STR-001: 协议支持
- [x] v0.4: HLS（.m3u8）manifest 解析
- [x] v0.4: DASH（.mpd）manifest 解析
- [ ] planned: DASH 分段数动态化（当前硬编码 50）
- [ ] planned: 预取 URL 动态化（当前硬编码）
- [ ] planned: LL-HLS（低延迟 HLS）

#### F-STR-002: 缓存策略
- [x] v0.4: 滑动窗口缓存（per-stream 淘汰）
- [x] v0.4: 分片请求触发预取（look-ahead）
- [x] v0.4: Manifest 更新触发预取
- [x] v0.4: 流媒体指标

#### F-STR-003: 调度优化
- [x] v0.4: 流感知调度加分（+20 权重）

---

### 2.6 隧道 (Tunnel)

#### F-TUN-001: 协议
- [x] v0.3: 二进制协议（10 字节头 + 变长 payload）
- [x] v0.3: 8 种消息类型
- [x] v0.3: 流多路复用
- [x] v0.3: 双向 Body 流式传输

#### F-TUN-002: 客户端
- [x] v0.3: 连接、心跳、请求转发
- [x] v0.3: TLS 支持
- [x] v0.3: 自动重连

#### F-TUN-003: 服务器
- [x] v0.3: 监听、Token 认证
- [x] v0.3: 最大隧道数限制
- [x] v0.3: 请求代理

---

### 2.7 安全

#### F-SEC-001: Token 鉴权
- [x] v0.1: HMAC-SHA256 签名/验证
- [x] v0.1: IP 前缀绑定（/24 IPv4, /64 IPv6）
- [x] v0.1: Token 格式：`base64url(payload).base64url(signature)`
- [x] v0.6: JWT header 验证加固

#### F-SEC-002: Admin 安全
- [x] v0.6: HMAC nonce 重放保护
- [x] v0.6: JWT session（HS256）+ 随机 refresh token
- [x] v0.6: bcrypt 错误处理加固
- [x] v0.6: 登录速率限制
- [x] v0.6: 幂等 shutdown helpers

#### F-SEC-003: API 安全
- [x] v0.1: 请求体大小限制（32KB）
- [x] v0.5: JSON 解析加固（media-type / unknown-field / multi-value）
- [x] v0.6: 对象 key 清理和路径转义
- [ ] planned: Rate limit per tenant
- [ ] planned: DDoS 防护

---

### 2.8 部署

#### F-DEP-001: Docker
- [x] v0.1: 6 个 Dockerfile（多阶段构建）
- [x] v0.1: docker-compose.yml（全栈本地演示）
- [x] v0.3: docker-compose.cp.yml（生产控制面）
- [x] v0.3: docker-compose.edge.yml（生产边缘节点）

#### F-DEP-002: 一键安装
- [x] v0.6: `install.sh` 交互式 + 命令行模式
- [x] v0.6: 三角色支持（CP / EA / Origin）

#### F-DEP-003: Kubernetes
- [ ] planned: Helm Chart（charts/edge-dispatch/）

#### F-DEP-004: IPv6
- [x] v0.6: IPv6 双栈支持（自动回退 `[::]:port`）
- [x] v0.6: IPv6 endpoint 格式化修正

---

## 3. 非功能需求

### 3.1 性能

| 指标 | 目标 | 状态 |
|------|------|------|
| Dispatch API QPS | > 10,000（单实例） | [ ] 待验证 |
| 心跳处理延迟 | < 50ms（批量窗口内） | [x] v0.1 |
| 调度延迟 | < 5ms（p99） | [ ] 待验证 |
| 边缘节点吞吐 | > 1 Gbps（公网节点） | [ ] 待验证 |
| 磁盘缓存命中率 | > 80%（典型工作负载） | [ ] 待验证 |

### 3.2 可靠性

| 指标 | 目标 | 状态 |
|------|------|------|
| 控制面 HA | 支持多实例部署 | [ ] planned |
| 故障检测 | < 30s 摘除离线节点 | [x] v0.1 |
| 熔断恢复 | 30s 恢复半开探测 | [x] v0.1 |
| 数据持久化 | PostgreSQL + Redis 双存储 | [x] v0.1 |

### 3.3 可维护性

| 指标 | 目标 | 状态 |
|------|------|------|
| 测试覆盖率 | > 70% 核心包 | [ ] 当前 ~40% |
| 结构化日志 | `log/slog` | [x] v0.1 |
| 配置管理 | 环境变量 | [x] v0.1 |
| OpenAPI 规范 | 3.0 完整文档 | [x] v0.5 |

### 3.4 安全性

| 指标 | 目标 | 状态 |
|------|------|------|
| 传输加密 | TLS 1.3 | [x] v0.1 |
| API 鉴权 | HMAC + JWT + Bearer Token | [x] v0.1 |
| 密钥管理 | 环境变量注入（非硬编码） | [x] v0.1 |
| 输入校验 | JSON 加固 + 路径清理 | [x] v0.6 |

---

## 4. 版本路线图

| 版本 | 状态 | 里程碑 |
|------|------|--------|
| v0.1 | ✅ 完成 | 控制面 + 边缘节点 + 302 调度 + Token 鉴权 |
| v0.2 | ✅ 完成 | 内容索引 (Bloom Filter) + DNS/GSLB 适配器 + 内容感知调度 |
| v0.3 | ✅ 完成 | 网关反代 + 反向隧道协议 + NAT 节点支持 |
| v0.4 | ⚠️ 部分 | 流媒体 HLS/DASH + 滑动窗口缓存 + 预取（DASH 50段硬编码待修复） |
| v0.5 | ✅ 完成 | Prometheus 指标 + 多个 Bug 修复 |
| v0.6 | ⚠️ 开发中 | 小带宽优化 + P2P + 智能预取 + Admin UI + 预热下发 + IPv6 |
| v0.7 | 📋 计划 | Helm Chart + QUIC 客户端 + 分布式追踪 + GeoDNS + 多租户限流 + 控制面 HA |

---

## 5. 已知限制与遗留问题

### 5.1 功能限制

| 问题 | 影响 | 优先级 |
|------|------|--------|
| DASH manifest 分段数硬编码为 50 | 长视频超出 50 段时无法正确预取 | 中 |
| 预取 URL 硬编码 | 不支持自定义预取路径 | 中 |
| QUIC 仅服务端实现，缺客户端 | 网关→边缘节点无法使用 QUIC 加速 | 低 |
| 无分布式追踪 | 跨服务请求难以排查延迟瓶颈 | 低 |
| 控制面单实例 | 无 HA 能力，存在单点故障 | 中 |
| 无多租户隔离 | Rate limit 仅全局，不支持按租户限流 | 低 |

### 5.2 测试覆盖缺口

| 包 | 当前覆盖 | 目标 |
|------|------|------|
| `internal/store/` | 0% | > 60% |
| `internal/edgeagent/` | ~10%（仅 cache.go） | > 50% |
| `internal/gateway/` | ~33%（仅 resolver.go） | > 60% |
| `internal/config/` | 0% | > 50% |
| `internal/autotls/` | 0% | > 50% |
| `internal/controlplane/config_manager` | 0% | > 40% |
| `internal/controlplane/adminui` | 0% | > 40% |

---

## 6. 接口规范

### 6.1 控制面 API

详见 `openapi.json`（OpenAPI 3.0 规范）。

### 6.2 数据模型

详见 `internal/models/models.go`。

### 6.3 隧道协议

详见 `internal/tunnel/message.go`。

---

## 7. 参考

- [Semantic Versioning](https://semver.org/)
- [Keep a Changelog](https://keepachangelog.com/)
- 项目 Wiki: https://github.com/darkinno/edge-dispatch-framework/wiki
