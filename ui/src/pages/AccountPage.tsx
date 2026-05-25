import { useState } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { Card, Descriptions, Tag, Button, Input, Alert, Typography, Divider, Modal, message } from 'antd';
import { UserOutlined, LockOutlined, LogoutOutlined, KeyOutlined } from '@ant-design/icons';
import { useQuery, useMutation } from '@tanstack/react-query';
import client from '../api/client';
import { useAuthStore } from '../store/authStore';
import type { User, ChangePasswordRequest, ErrorResponse } from '../types/api';
import { AxiosError } from 'axios';

const { Title, Text } = Typography;

const ROLE_LABELS: Record<string, { color: string; label: string }> = {
  tenant_owner: { color: 'gold', label: '超级管理员' },
  tenant_admin: { color: 'blue', label: '管理员' },
  project_operator: { color: 'green', label: '操作员' },
  project_viewer: { color: 'default', label: '观察者' },
};

export default function AccountPage() {
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const isSetup = searchParams.get('setup') === '1';
  const { user, logout, setUser } = useAuthStore();
  const [logoutModal, setLogoutModal] = useState(false);

  const [currentPwd, setCurrentPwd] = useState('');
  const [newPwd, setNewPwd] = useState('');
  const [confirmPwd, setConfirmPwd] = useState('');
  const [pwdError, setPwdError] = useState<string | null>(null);
  const [pwdSuccess, setPwdSuccess] = useState(false);

  const { data: freshUser } = useQuery({
    queryKey: ['me'],
    queryFn: () => client.get<User>('/me').then((r) => r.data),
  });

  const displayUser = freshUser || user;

  const changeMutation = useMutation({
    mutationFn: (body: ChangePasswordRequest) => client.put('/me/password', body),
    onSuccess: () => {
      setPwdSuccess(true);
      setPwdError(null);
      setCurrentPwd('');
      setNewPwd('');
      setConfirmPwd('');
      if (freshUser) {
        setUser({ ...freshUser, must_change_password: false });
      }
      message.success('密码已修改，请重新登录');
      setTimeout(() => {
        logout();
        navigate('/login', { replace: true });
      }, 2000);
    },
    onError: (err: AxiosError<ErrorResponse>) => {
      setPwdError(err.response?.data?.error?.message || '密码修改失败');
    },
  });

  const handleChangePassword = () => {
    setPwdError(null);
    if (!currentPwd) {
      setPwdError('请输入当前密码');
      return;
    }
    if (!newPwd || newPwd.length < 8) {
      setPwdError('新密码至少 8 个字符');
      return;
    }
    if (newPwd !== confirmPwd) {
      setPwdError('两次输入的新密码不一致');
      return;
    }
    changeMutation.mutate({ current_password: currentPwd, new_password: newPwd });
  };

  const handleLogout = async () => {
    try {
      await client.post('/logout');
    } catch { /* ignore */ }
    logout();
    navigate('/login', { replace: true });
  };

  if (!displayUser) return null;

  return (
    <div>
      <Title level={4} style={{ marginBottom: 24 }}>
        <UserOutlined style={{ marginRight: 8 }} />
        个人信息
      </Title>

      {isSetup && !pwdSuccess && (
        <Alert
          type="warning"
          message="首次登录 — 请修改默认密码"
          description="检测到您正在使用默认密码。出于安全考虑，请立即修改密码。"
          showIcon
          style={{ marginBottom: 24 }}
        />
      )}

      {pwdSuccess && (
        <Alert
          type="success"
          message="密码修改成功，即将跳转到登录页..."
          showIcon
          style={{ marginBottom: 24 }}
        />
      )}

      <Card style={{ marginBottom: 24 }}>
        <Descriptions column={1} size="small" labelStyle={{ width: 100 }}>
          <Descriptions.Item label="用户名">{displayUser.display_name}</Descriptions.Item>
          <Descriptions.Item label="邮箱">{displayUser.email}</Descriptions.Item>
          <Descriptions.Item label="用户 ID">
            <Text code style={{ fontSize: 11 }}>{displayUser.user_id}</Text>
          </Descriptions.Item>
          <Descriptions.Item label="租户 ID">
            <Text code style={{ fontSize: 11 }}>{displayUser.tenant_id}</Text>
          </Descriptions.Item>
          <Descriptions.Item label="创建时间">
            {new Date(displayUser.created_at).toLocaleString('zh-CN')}
          </Descriptions.Item>
          <Descriptions.Item label="角色">
            {displayUser.roles.map((r, i) => {
              const cfg = ROLE_LABELS[r.role] || { color: 'default', label: r.role };
              return <Tag key={i} color={cfg.color}>{cfg.label}</Tag>;
            })}
          </Descriptions.Item>
        </Descriptions>
      </Card>

      <Card
        title={<span><KeyOutlined style={{ marginRight: 6 }} />{isSetup ? '设置新密码' : '修改密码'}</span>}
        style={{ marginBottom: 24 }}
      >
        {isSetup && (
          <Alert
            type="info"
            message="密码要求：至少 8 个字符，最多 72 个字符"
            showIcon={false}
            style={{ marginBottom: 16 }}
          />
        )}
        {pwdError && (
          <Alert type="error" message={pwdError} showIcon closable onClose={() => setPwdError(null)} style={{ marginBottom: 16 }} />
        )}

        <div style={{ maxWidth: 400 }}>
          <div style={{ marginBottom: 16 }}>
            <Text type="secondary" style={{ display: 'block', marginBottom: 4 }}>当前密码</Text>
            <Input.Password
              prefix={<LockOutlined />}
              placeholder="输入当前密码"
              value={currentPwd}
              onChange={(e) => setCurrentPwd(e.target.value)}
              onPressEnter={handleChangePassword}
            />
          </div>

          <div style={{ marginBottom: 16 }}>
            <Text type="secondary" style={{ display: 'block', marginBottom: 4 }}>新密码</Text>
            <Input.Password
              prefix={<KeyOutlined />}
              placeholder="输入新密码（至少 8 位）"
              value={newPwd}
              onChange={(e) => setNewPwd(e.target.value)}
              onPressEnter={handleChangePassword}
            />
          </div>

          <div style={{ marginBottom: 16 }}>
            <Text type="secondary" style={{ display: 'block', marginBottom: 4 }}>确认新密码</Text>
            <Input.Password
              prefix={<KeyOutlined />}
              placeholder="再次输入新密码"
              value={confirmPwd}
              onChange={(e) => setConfirmPwd(e.target.value)}
              onPressEnter={handleChangePassword}
            />
          </div>

          <Button
            type="primary"
            block
            icon={<KeyOutlined />}
            loading={changeMutation.isPending}
            onClick={handleChangePassword}
          >
            {isSetup ? '设置密码并登录' : '修改密码'}
          </Button>
        </div>
      </Card>

      <Divider />

      <Button
        danger
        icon={<LogoutOutlined />}
        onClick={() => setLogoutModal(true)}
      >
        退出登录
      </Button>

      <Modal
        title="确认退出"
        open={logoutModal}
        onOk={handleLogout}
        onCancel={() => setLogoutModal(false)}
        okText="退出"
        cancelText="取消"
      >
        确定要退出登录吗？
      </Modal>
    </div>
  );
}
