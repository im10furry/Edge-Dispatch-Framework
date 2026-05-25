# EDF Webmanager

> Admin web console for **[Edge Dispatch Framework](https://github.com/DarkInno/Edge-Dispatch-Framework)** — an open-source edge content delivery and acceleration framework that routes user requests to the optimal edge node via centralized dispatch.

[![React](https://img.shields.io/badge/React-18-61DAFB?logo=react)](https://react.dev/)
[![TypeScript](https://img.shields.io/badge/TypeScript-5.6-3178C6?logo=typescript)](https://www.typescriptlang.org/)
[![Vite](https://img.shields.io/badge/Vite-6-646CFF?logo=vite)](https://vitejs.dev/)
[![Ant Design](https://img.shields.io/badge/Ant%20Design-5-0170FE)](https://ant.design/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

## Features

- **Real-time Dashboard** — QPS, latency, hit rate, error rate, and node status with 30s polling
- **Edge Node Management** — List, detail, enable/disable, and label editing per node
- **Multi-tenant Management** — CRUD for tenants, projects, and users with role-based access control (RBAC)
- **Policy Management** — Dispatch and block policies with versioning
- **Ingress Management** — DNS, 302 redirect, and gateway reverse proxy configurations
- **Cache Tasks** — Prewarm, purge, and block operations with progress tracking
- **Audit Logging** — Before/after change diffing with CSV/JSON export
- **System Settings** — OIDC, observability URLs, and IP allowlists

## Tech Stack

| Layer | Technology |
|-------|------------|
| Framework | React 18 + TypeScript 5.6 |
| Build Tool | Vite 6 |
| UI Library | Ant Design 5 |
| Data Fetching | @tanstack/react-query 5 |
| State Management | Zustand 5 |
| Routing | React Router 6 |
| Charts | @ant-design/charts |
| Testing | Vitest + Playwright + MSW |

## Prerequisites

- Node.js >= 18
- The [EDF backend](https://github.com/DarkInno/Edge-Dispatch-Framework) (Go) running on `localhost:8080`

## Getting Started

```bash
# Clone
git clone https://github.com/im10furry/Edge-Dispatch-Framework-Webmanager.git
cd Edge-Dispatch-Framework-Webmanager

# Install dependencies
npm install

# Start dev server (binds to 0.0.0.0:8443, proxies API to localhost:8080)
npm run dev
```

Open [http://localhost:8443](http://localhost:8443) in your browser.

## Environment Variables

The dev server proxies API requests to the EDF control plane. The proxy is configured in `vite.config.ts`:

| Proxy Path | Target | Description |
|------------|--------|-------------|
| `/v1/*` | `http://localhost:8080` | Control Plane API |
| `/internal/*` | `http://localhost:8080` | Internal Admin API |
| `/healthz` | `http://localhost:8080` | Health check |

For production, set the following environment variable to point to the backend API:

```bash
# .env
VITE_API_BASE_URL=http://<your-cp-host>:8080
```

## Scripts

| Command | Description |
|---------|-------------|
| `npm run dev` | Start Vite dev server |
| `npm run build` | TypeScript check + production build |
| `npm run preview` | Preview production build |
| `npm run lint` | ESLint |
| `npm run test` | Vitest unit tests |
| `npm run test:ui` | Vitest UI |
| `npm run test:e2e` | Playwright E2E tests |

## Project Structure

```
src/
├── api/client.ts              # Axios instance with JWT refresh
├── components/layout/         # App layout (sidebar + header)
├── pages/                     # Page components
│   ├── DashboardPage.tsx
│   ├── LoginPage.tsx
│   ├── nodes/                 # Node list & detail
│   ├── tenants/               # Tenant list & detail
│   ├── users/                 # User list & detail
│   ├── policies/              # Policy list & detail
│   ├── ingresses/             # Ingress list & detail
│   ├── tasks/                 # Cache task list & detail
│   ├── AuditPage.tsx
│   ├── SettingsPage.tsx
│   └── AccountPage.tsx
├── router/index.tsx           # Route definitions & auth guard
├── store/authStore.ts         # Zustand auth state (JWT + RBAC)
├── types/api.ts               # TypeScript API type definitions
└── theme.ts                   # Ant Design dark theme tokens
```

## Deployment

### Build

```bash
npm run build
```

Output is in `dist/`. Serve it with any static file server (nginx, Caddy, etc.).

### Docker

```bash
# Build image
docker build -t edf-webmanager .

# Run
docker run -d \
  -p 8443:80 \
  -e VITE_API_BASE_URL=http://<cp-host>:8080 \
  edf-webmanager
```

### Nginx Example

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

## API Reference

The Webmanager interacts with the EDF Control Plane API. See [openapi.json](openapi.json) for the full API specification.

| Endpoint | Description |
|----------|-------------|
| `POST /v1/login` | Admin login (email + password) |
| `POST /v1/refresh` | Refresh JWT token |
| `GET /v1/admin/dashboard` | Dashboard metrics |
| `GET /v1/admin/nodes` | List edge nodes |
| `GET /v1/admin/nodes/:id` | Node detail |
| `PATCH /v1/admin/nodes/:id` | Update node (labels, weight, disable) |
| `GET /v1/admin/tenants` | List tenants |
| `POST /v1/admin/tenants` | Create tenant |
| `GET /v1/admin/users` | List users |
| `POST /v1/admin/users` | Create user |
| `GET /v1/admin/policies` | List policies |
| `POST /v1/admin/policies` | Create policy |
| `GET /v1/admin/ingresses` | List ingress configs |
| `POST /v1/admin/ingresses` | Create ingress config |
| `GET /v1/admin/tasks` | List cache tasks |
| `POST /v1/admin/tasks` | Create cache task |
| `GET /v1/admin/audit` | Audit log |
| `GET /v1/admin/settings` | System settings |
| `PUT /v1/admin/settings` | Update settings |

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## Related Projects

- [Edge Dispatch Framework](https://github.com/DarkInno/Edge-Dispatch-Framework) — The core backend (Go)

## License

[MIT](LICENSE) © 2026 im10furry
