import { useNavigate } from 'react-router-dom';
import {
  Card,
  Descriptions,
  Tag,
  Input,
  Button,
  Space,
  Divider,
  Avatar,
  Modal,
} from 'antd';
import {
  UserOutlined,
  LogoutOutlined,
  MailOutlined,
  LockOutlined,
} from '@ant-design/icons';
import { useAuthStore } from '../store/authStore';
import client from '../api/client';
import type { UserRole } from '../types/api';

const roleLabelMap: Record<UserRole, string> = {
  tenant_owner: '租户所有者',
  tenant_admin: '租户管理员',
  project_operator: '项目操作员',
  project_viewer: '项目查看者',
};

const roleColorMap: Record<UserRole, string> = {
  tenant_owner: 'red',
  tenant_admin: 'orange',
  project_operator: 'blue',
  project_viewer: 'default',
};

function maskPassword(pw: string) {
  if (!pw) return '';
  const len = pw.length;
  if (len <= 2) return pw[0] + '*';
  return pw[0] + '*'.repeat(len - 2) + pw[len - 1];
}

export default function AccountPage() {
  const navigate = useNavigate();
  const { user, roles, logout } = useAuthStore();

  const handleLogout = async () => {
    try {
      await client.post('/logout', undefined, { skipAuthRefresh: true });
    } catch {
      // ignore
    }
    logout();
    navigate('/login', { replace: true });
  };

  if (!user) {
    return (
      <Card>
        <div style={{ textAlign: 'center', padding: 40, color: '#8b949e' }}>
          未获取到用户信息
        </div>
      </Card>
    );
  }

  return (
    <div style={{ maxWidth: 720 }}>
      <Card size="small" title="个人信息" style={{ marginBottom: 24 }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 20, marginBottom: 20 }}>
          <Avatar size={64} icon={<UserOutlined />} style={{ backgroundColor: '#1f6feb' }} />
          <div>
            <div style={{ fontSize: 18, fontWeight: 600 }}>{user.display_name || user.email}</div>
            <div style={{ color: '#8b949e', fontSize: 13 }}>
              <MailOutlined style={{ marginRight: 6 }} />
              {user.email}
            </div>
          </div>
        </div>

        <Descriptions column={1} size="small" colon={false}>
          <Descriptions.Item label="用户 ID">
            <span style={{ fontFamily: 'monospace', fontSize: 13 }}>{user.user_id}</span>
          </Descriptions.Item>
          <Descriptions.Item label="租户 ID">
            <span style={{ fontFamily: 'monospace', fontSize: 13 }}>{user.tenant_id}</span>
          </Descriptions.Item>
          <Descriptions.Item label="注册时间">
            {new Date(user.created_at).toLocaleString('zh-CN')}
          </Descriptions.Item>
        </Descriptions>

        <Divider />

        <div style={{ marginBottom: 8, fontWeight: 500 }}>角色权限</div>
        {roles.length === 0 ? (
          <span style={{ color: '#8b949e' }}>无角色</span>
        ) : (
          <Space wrap size={[6, 6]}>
            {roles.map((r, i) => (
              <Tag key={`${r.role}-${i}`} color={roleColorMap[r.role]}>
                {roleLabelMap[r.role] || r.role}
                <span style={{ marginLeft: 8, opacity: 0.7, fontSize: 12 }}>
                  {r.tenant_id}/{r.project_id || '*'}
                </span>
              </Tag>
            ))}
          </Space>
        )}
      </Card>

      <Card size="small" title="修改密码" style={{ marginBottom: 24 }}>
        <div
          style={{
            padding: '12px 16px',
            background: '#1c2733',
            borderRadius: 6,
            border: '1px solid #30363d',
            marginBottom: 16,
            color: '#d29922',
            fontSize: 13,
          }}
        >
          密码修改功能暂未开放，如需修改密码请联系系统管理员。
        </div>

        <Space direction="vertical" size={16} style={{ width: '100%', maxWidth: 400 }}>
          <Input.Password
            prefix={<LockOutlined />}
            placeholder="当前密码"
            disabled
            value={maskPassword('placeholder')}
          />
          <Input.Password
            prefix={<LockOutlined />}
            placeholder="新密码"
            disabled
          />
          <Input.Password
            prefix={<LockOutlined />}
            placeholder="确认新密码"
            disabled
          />
          <Button type="primary" disabled block>
            修改密码
          </Button>
        </Space>
      </Card>

      <Button
        danger
        icon={<LogoutOutlined />}
        size="large"
        block
        onClick={() => {
          Modal.confirm({
            title: '确认退出',
            content: '确定要退出登录吗？',
            okText: '退出',
            cancelText: '取消',
            okButtonProps: { danger: true },
            onOk: handleLogout,
          });
        }}
      >
        退出登录
      </Button>
    </div>
  );
}
