# EDF Webmanager

> **[Edge Dispatch Framework](https://github.com/DarkInno/Edge-Dispatch-Framework)** 的管理控制台 —— 一个开源边缘内容分发与加速框架，通过中心调度将用户请求分配到最优边缘节点，支持 302 重定向、DNS/GSLB、网关反代等多种接入方式。

[![React](https://img.shields.io/badge/React-18-61DAFB?logo=react)](https://react.dev/)
[![TypeScript](https://img.shields.io/badge/TypeScript-5.6-3178C6?logo=typescript)](https://www.typescriptlang.org/)
[![Vite](https://img.shields.io/badge/Vite-6-646CFF?logo=vite)](https://vitejs.dev/)
[![Ant Design](https://img.shields.io/badge/Ant%20Design-5-0170FE)](https://ant.design/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

## 功能特性

- **实时仪表盘** — QPS、延迟、命中率、错误率、节点状态，每 30 秒轮询刷新
- **边缘节点管理** — 节点列表、详情、启停、标签编辑
- **多租户管理** — 租户、项目、用户的 CRUD，基于角色的访问控制（RBAC）
- **策略管理** — 分发策略与拦截策略，支持版本历史
- **入口管理** — DNS、302 重定向、网关反向代理等入口配置
- **缓存任务** — 预热、清除、封禁操作，支持进度跟踪
- **审计日志** — 变更前后对比，支持 CSV/JSON 导出
- **系统设置** — OIDC 配置、可观测性 URL、IP 白名单

## 技术栈

| 层级 | 技术 |
|------|------|
| 框架 | React 18 + TypeScript 5.6 |
| 构建工具 | Vite 6 |
| UI 组件库 | Ant Design 5 |
| 数据请求 | @tanstack/react-query 5 |
| 状态管理 | Zustand 5 |
| 路由 | React Router 6 |
| 图表 | @ant-design/charts |
| 测试 | Vitest + Playwright + MSW |

## 环境要求

- Node.js >= 18
- [EDF 后端](https://github.com/DarkInno/Edge-Dispatch-Framework)（Go）运行在 `localhost:8080`

## 快速开始

```bash
# 克隆仓库
git clone https://github.com/im10furry/Edge-Dispatch-Framework-Webmanager.git
cd Edge-Dispatch-Framework-Webmanager

# 安装依赖
npm install

# 启动开发服务器（监听 0.0.0.0:8443，API 代理到 localhost:8080）
npm run dev
```

打开浏览器访问 [http://localhost:8443](http://localhost:8443)。

## 环境变量

开发服务器会将 API 请求代理到 EDF 控制面。代理配置在 `vite.config.ts` 中：

| 代理路径 | 目标 | 说明 |
|---------|------|------|
| `/v1/*` | `http://localhost:8080` | 控制面 API |
| `/internal/*` | `http://localhost:8080` | 内部管理 API |
| `/healthz` | `http://localhost:8080` | 健康检查 |

生产环境需设置以下环境变量指向后端 API：

```bash
# .env
VITE_API_BASE_URL=http://<cp-host>:8080
```

## 脚本说明

| 命令 | 描述 |
|------|------|
| `npm run dev` | 启动 Vite 开发服务器 |
| `npm run build` | TypeScript 检查 + 生产构建 |
| `npm run preview` | 预览生产构建 |
| `npm run lint` | ESLint 代码检查 |
| `npm run test` | Vitest 单元测试 |
| `npm run test:ui` | Vitest 可视化界面 |
| `npm run test:e2e` | Playwright 端到端测试 |

## 项目结构

```
src/
├── api/client.ts              # Axios 实例（含 JWT 自动刷新）
├── components/layout/         # 应用布局（侧边栏 + 顶栏）
├── pages/                     # 页面组件
│   ├── DashboardPage.tsx      # 仪表盘
│   ├── LoginPage.tsx          # 登录
│   ├── nodes/                 # 节点列表与详情
│   ├── tenants/               # 租户列表与详情
│   ├── users/                 # 用户列表与详情
│   ├── policies/              # 策略列表与详情
│   ├── ingresses/             # 入口列表与详情
│   ├── tasks/                 # 缓存任务列表与详情
│   ├── AuditPage.tsx          # 审计日志
│   ├── SettingsPage.tsx       # 系统设置
│   └── AccountPage.tsx        # 个人中心
├── router/index.tsx           # 路由定义与鉴权守卫
├── store/authStore.ts         # Zustand 认证状态（JWT + RBAC）
├── types/api.ts               # TypeScript API 类型定义
└── theme.ts                   # Ant Design 暗色主题
```

## 部署

### 构建

```bash
npm run build
```

输出目录为 `dist/`，可使用任意静态文件服务器（nginx、Caddy 等）部署。

### Docker

```bash
# 构建镜像
docker build -t edf-webmanager .

# 运行
docker run -d \
  -p 8443:80 \
  -e VITE_API_BASE_URL=http://<cp-host>:8080 \
  edf-webmanager
```

### Nginx 配置示例

```nginx
server {
    listen 80;
    server_name manager.example.com;
    root /var/www/edf-webmanager;
    index index.html;

    location / {
        try_files $uri $uri/ /index.html;
    }

    location /v1/ {
        proxy_pass http://<cp-host>:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }

    location /internal/ {
        proxy_pass http://<cp-host>:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

## API 接口

Webmanager 通过 EDF 控制面 API 进行交互。完整 API 规范见 [openapi.json](openapi.json)。

| 接口 | 说明 |
|------|------|
| `POST /v1/login` | 管理员登录（邮箱 + 密码） |
| `POST /v1/refresh` | 刷新 JWT Token |
| `GET /v1/admin/dashboard` | 仪表盘指标 |
| `GET /v1/admin/nodes` | 节点列表 |
| `GET /v1/admin/nodes/:id` | 节点详情 |
| `PATCH /v1/admin/nodes/:id` | 更新节点（标签、权重、启停） |
| `GET /v1/admin/tenants` | 租户列表 |
| `POST /v1/admin/tenants` | 创建租户 |
| `GET /v1/admin/users` | 用户列表 |
| `POST /v1/admin/users` | 创建用户 |
| `GET /v1/admin/policies` | 策略列表 |
| `POST /v1/admin/policies` | 创建策略 |
| `GET /v1/admin/ingresses` | 入口列表 |
| `POST /v1/admin/ingresses` | 创建入口配置 |
| `GET /v1/admin/tasks` | 缓存任务列表 |
| `POST /v1/admin/tasks` | 创建缓存任务 |
| `GET /v1/admin/audit` | 审计日志 |
| `GET /v1/admin/settings` | 系统设置 |
| `PUT /v1/admin/settings` | 更新设置 |

## 贡献

1. Fork 本仓库
2. 创建功能分支 (`git checkout -b feature/amazing-feature`)
3. 提交更改 (`git commit -m 'Add amazing feature'`)
4. 推送分支 (`git push origin feature/amazing-feature`)
5. 发起 Pull Request

## 相关项目

- [Edge Dispatch Framework](https://github.com/DarkInno/Edge-Dispatch-Framework) — 核心后端（Go）

## 许可证

[MIT](LICENSE) © 2026 im10furry
