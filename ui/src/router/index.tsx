import { createBrowserRouter } from 'react-router-dom';
import { lazy } from 'react';
import { Navigate } from 'react-router-dom';
import { LazyPage, ProtectedRoute, PublicRoute } from './routeGuards';

const LoginPage = lazy(() => import('../pages/LoginPage'));
const DashboardPage = lazy(() => import('../pages/DashboardPage'));
const NodeListPage = lazy(() => import('../pages/nodes/NodeListPage'));
const NodeDetailPage = lazy(() => import('../pages/nodes/NodeDetailPage'));
const TenantListPage = lazy(() => import('../pages/tenants/TenantListPage'));
const TenantDetailPage = lazy(() => import('../pages/tenants/TenantDetailPage'));
const UserListPage = lazy(() => import('../pages/users/UserListPage'));
const UserDetailPage = lazy(() => import('../pages/users/UserDetailPage'));
const PolicyListPage = lazy(() => import('../pages/policies/PolicyListPage'));
const PolicyDetailPage = lazy(() => import('../pages/policies/PolicyDetailPage'));
const IngressListPage = lazy(() => import('../pages/ingresses/IngressListPage'));
const IngressDetailPage = lazy(() => import('../pages/ingresses/IngressDetailPage'));
const TaskListPage = lazy(() => import('../pages/tasks/TaskListPage'));
const TaskDetailPage = lazy(() => import('../pages/tasks/TaskDetailPage'));
const AuditPage = lazy(() => import('../pages/AuditPage'));
const P2PTopologyPage = lazy(() => import('../pages/P2PTopologyPage'));
const GlobalConfigPage = lazy(() => import('../pages/GlobalConfigPage'));
const SettingsPage = lazy(() => import('../pages/SettingsPage'));
const AccountPage = lazy(() => import('../pages/AccountPage'));

export const router = createBrowserRouter(
  [

  {
    path: '/login',
    element: (
      <PublicRoute>
        <LazyPage>
          <LoginPage />
        </LazyPage>
      </PublicRoute>
    ),
  },
  {
    element: <ProtectedRoute />,
    children: [
      {
        index: true,
        element: (
          <LazyPage>
            <DashboardPage />
          </LazyPage>
        ),
      },
      {
        path: 'nodes',
        element: (
          <LazyPage>
            <NodeListPage />
          </LazyPage>
        ),
      },
      {
        path: 'nodes/:nodeID',
        element: (
          <LazyPage>
            <NodeDetailPage />
          </LazyPage>
        ),
      },
      {
        path: 'tenants',
        element: (
          <LazyPage>
            <TenantListPage />
          </LazyPage>
        ),
      },
      {
        path: 'tenants/:tenantID',
        element: (
          <LazyPage>
            <TenantDetailPage />
          </LazyPage>
        ),
      },
      {
        path: 'users',
        element: (
          <LazyPage>
            <UserListPage />
          </LazyPage>
        ),
      },
      {
        path: 'users/:userID',
        element: (
          <LazyPage>
            <UserDetailPage />
          </LazyPage>
        ),
      },
      {
        path: 'policies',
        element: (
          <LazyPage>
            <PolicyListPage />
          </LazyPage>
        ),
      },
      {
        path: 'policies/:policyID',
        element: (
          <LazyPage>
            <PolicyDetailPage />
          </LazyPage>
        ),
      },
      {
        path: 'ingresses',
        element: (
          <LazyPage>
            <IngressListPage />
          </LazyPage>
        ),
      },
      {
        path: 'ingresses/:ingressID',
        element: (
          <LazyPage>
            <IngressDetailPage />
          </LazyPage>
        ),
      },
      {
        path: 'tasks',
        element: (
          <LazyPage>
            <TaskListPage />
          </LazyPage>
        ),
      },
      {
        path: 'tasks/:taskID',
        element: (
          <LazyPage>
            <TaskDetailPage />
          </LazyPage>
        ),
      },
      {
        path: 'audit',
        element: (
          <LazyPage>
            <AuditPage />
          </LazyPage>
        ),
      },
      {
        path: 'p2p',
        element: (
          <LazyPage>
            <P2PTopologyPage />
          </LazyPage>
        ),
      },
      {
        path: 'config',
        element: (
          <LazyPage>
            <GlobalConfigPage />
          </LazyPage>
        ),
      },
      {
        path: 'settings',
        element: (
          <LazyPage>
            <SettingsPage />
          </LazyPage>
        ),
      },
      {
        path: 'account',
        element: (
          <LazyPage>
            <AccountPage />
          </LazyPage>
        ),
      },
      {
        path: '*',
        element: <Navigate to="/" replace />,
      },
      ],
    },
  ],
  { basename: '/admin' }
);

export default router;
