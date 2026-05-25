import { Layout, Breadcrumb, Dropdown, Avatar, Space } from 'antd';
import { UserOutlined, LogoutOutlined, SettingOutlined } from '@ant-design/icons';
import { useNavigate, useLocation } from 'react-router-dom';
import { useAuthStore } from '../../store/authStore';
import client from '../../api/client';

const { Header } = Layout;

const pathMap: Record<string, string> = {
  '/': '仪表盘',
  '/nodes': '节点管理',
  '/tenants': '租户管理',
  '/users': '用户管理',
  '/policies': '策略管理',
  '/ingresses': '入口管理',
  '/tasks': '缓存任务',
  '/audit': '审计日志',
  '/settings': '系统设置',
  '/account': '个人中心',
};

interface HeaderBarProps {
  collapsed: boolean;
  onToggle: () => void;
}

export default function HeaderBar({ collapsed, onToggle }: HeaderBarProps) {
  void onToggle;
  const navigate = useNavigate();
  const location = useLocation();
  const { user, logout } = useAuthStore();

  const segments = location.pathname.split('/').filter(Boolean);
  const breadcrumbItems = [
    { title: '首页', path: '/' },
    ...segments.map((seg, i) => {
      const path = '/' + segments.slice(0, i + 1).join('/');
      return { title: pathMap[path] || seg, path };
    }),
  ];

  const handleLogout = async () => {
    try {
      await client.post('/logout', undefined, { skipAuthRefresh: true });
    } catch {
      // ignore
    }
    logout();
    navigate('/login', { replace: true });
  };

  const userMenuItems = {
    items: [
      {
        key: 'account',
        icon: <UserOutlined />,
        label: '个人中心',
        onClick: () => navigate('/account'),
      },
      {
        key: 'settings',
        icon: <SettingOutlined />,
        label: '系统设置',
        onClick: () => navigate('/settings'),
      },
      { type: 'divider' as const },
      {
        key: 'logout',
        icon: <LogoutOutlined />,
        label: '退出登录',
        danger: true,
        onClick: handleLogout,
      },
    ],
  };

  return (
    <Header
      style={{
        height: 56,
        lineHeight: '56px',
        padding: '0 24px',
        background: '#0d1117',
        borderBottom: '1px solid #21262d',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'space-between',
        position: 'fixed',
        top: 0,
        right: 0,
        left: collapsed ? 80 : 240,
        zIndex: 99,
        transition: 'left 0.2s',
      }}
    >
      <Breadcrumb
        items={breadcrumbItems.map((item, index) => ({
          title: index === breadcrumbItems.length - 1 ? (
            <span style={{ color: '#e6edf3' }}>{item.title}</span>
          ) : (
            <a style={{ color: '#58a6ff' }} onClick={() => navigate(item.path)}>
              {item.title}
            </a>
          ),
        }))}
      />
      <Dropdown menu={userMenuItems} trigger={['click']}>
        <Space style={{ cursor: 'pointer' }}>
          <Avatar size={28} icon={<UserOutlined />} style={{ backgroundColor: '#1f6feb' }} />
          <span style={{ color: '#e6edf3', fontSize: 14 }}>{user?.display_name || user?.email}</span>
        </Space>
      </Dropdown>
    </Header>
  );
}
