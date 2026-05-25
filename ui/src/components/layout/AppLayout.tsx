import { Layout, Menu, Button, theme as antTheme } from 'antd';
import {
  DashboardOutlined,
  CloudServerOutlined,
  TeamOutlined,
  UserOutlined,
  SafetyCertificateOutlined,
  GlobalOutlined,
  SyncOutlined,
  AuditOutlined,
  SettingOutlined,
  MenuFoldOutlined,
  MenuUnfoldOutlined,
} from '@ant-design/icons';
import { useState, useEffect } from 'react';
import { useNavigate, useLocation, Outlet } from 'react-router-dom';
import { useAuthStore } from '../../store/authStore';
import HeaderBar from './HeaderBar';

const { Sider, Content } = Layout;

const menuItems = [
  { key: '/', icon: <DashboardOutlined />, label: '仪表盘' },
  { key: '/nodes', icon: <CloudServerOutlined />, label: '节点管理' },
  { key: '/tenants', icon: <TeamOutlined />, label: '租户管理' },
  { key: '/users', icon: <UserOutlined />, label: '用户管理' },
  { key: '/policies', icon: <SafetyCertificateOutlined />, label: '策略管理' },
  { key: '/ingresses', icon: <GlobalOutlined />, label: '入口管理' },
  { key: '/tasks', icon: <SyncOutlined />, label: '缓存任务' },
  { key: '/audit', icon: <AuditOutlined />, label: '审计日志' },
  { key: '/settings', icon: <SettingOutlined />, label: '系统设置' },
];

export default function AppLayout() {
  const [collapsed, setCollapsed] = useState(false);
  const navigate = useNavigate();
  const location = useLocation();
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated);
  const { token: themeToken } = antTheme.useToken();

  useEffect(() => {
    if (!isAuthenticated) {
      navigate('/login', { replace: true });
    }
  }, [isAuthenticated, navigate]);

  if (!isAuthenticated) return null;

  const selectedKey = '/' + (location.pathname.split('/')[1] || '');

  const logoSrc = collapsed ? '' : undefined;

  return (
    <Layout style={{ height: '100vh' }}>
      <Sider
        trigger={null}
        collapsible
        collapsed={collapsed}
        width={240}
        style={{
          borderRight: `1px solid ${themeToken.colorBorderSecondary}`,
          height: '100vh',
          position: 'fixed',
          left: 0,
          top: 0,
          bottom: 0,
          zIndex: 100,
        }}
        theme="dark"
      >
        <div
          style={{
            height: 56,
            display: 'flex',
            alignItems: 'center',
            justifyContent: collapsed ? 'center' : 'flex-start',
            padding: collapsed ? 0 : '0 20px',
            borderBottom: `1px solid ${themeToken.colorBorderSecondary}`,
          }}
        >
          {!collapsed && (
            <span style={{ color: '#58a6ff', fontWeight: 700, fontSize: 18, letterSpacing: 1 }}>
              {logoSrc ? <img alt="EDF" height={28} src={logoSrc} /> : 'EDF'}
            </span>
          )}
          {collapsed && (
            <span style={{ color: '#58a6ff', fontWeight: 700, fontSize: 18 }}>E</span>
          )}
        </div>
        <Menu
          theme="dark"
          mode="inline"
          selectedKeys={[selectedKey]}
          items={menuItems}
          onClick={({ key }) => navigate(key)}
          style={{ borderInlineEnd: 'none', marginTop: 8 }}
        />
        <div style={{ position: 'absolute', bottom: 0, width: '100%', padding: 12 }}>
          <Button
            type="text"
            icon={collapsed ? <MenuUnfoldOutlined /> : <MenuFoldOutlined />}
            onClick={() => setCollapsed(!collapsed)}
            style={{ color: '#8b949e', width: '100%' }}
          />
        </div>
      </Sider>
      <Layout style={{ marginLeft: collapsed ? 80 : 240, transition: 'margin-left 0.2s' }}>
        <HeaderBar collapsed={collapsed} onToggle={() => setCollapsed(!collapsed)} />
        <Content
          style={{
            marginTop: 56,
            padding: 24,
            minHeight: 'calc(100vh - 56px)',
            overflow: 'auto',
            background: themeToken.colorBgLayout,
          }}
        >
          <div style={{ maxWidth: 1400, margin: '0 auto' }}>
            <Outlet />
          </div>
        </Content>
      </Layout>
    </Layout>
  );
}
