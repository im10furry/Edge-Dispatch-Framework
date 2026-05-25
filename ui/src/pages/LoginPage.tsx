import { useState } from 'react';
import { useLocation, useNavigate } from 'react-router-dom';
import { Card, Input, Button, Checkbox, Typography, Alert, Space } from 'antd';
import { MailOutlined, LockOutlined } from '@ant-design/icons';
import { AxiosError } from 'axios';
import client from '../api/client';
import { useAuthStore } from '../store/authStore';
import type { LoginRequest, LoginResponse, ErrorResponse } from '../types/api';

const { Title, Text } = Typography;

export default function LoginPage() {
  const navigate = useNavigate();
  const location = useLocation();
  const setAuth = useAuthStore((s) => s.setAuth);

  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [remember, setRemember] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleSubmit = async () => {
    if (!email || !password) {
      setError('请输入邮箱和密码');
      return;
    }
    setError(null);
    setLoading(true);
    try {
      const res = await client.post<LoginResponse>('/login', {
        email,
        password,
      } as LoginRequest, { skipAuthRefresh: true });
      const { token, refresh_token, user, roles, must_change_password } = res.data;
      setAuth(token, refresh_token, user, roles, remember);
      if (must_change_password) {
        navigate('/account?setup=1', { replace: true });
        return;
      }
      const from = (location.state as { from?: string } | null)?.from;
      navigate(from && from !== '/login' ? from : '/', { replace: true });
    } catch (err) {
      const axiosError = err as AxiosError<ErrorResponse>;
      setError(axiosError.response?.data?.error?.message || '登录失败，请重试');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div
      style={{
        minHeight: '100vh',
        display: 'flex',
        justifyContent: 'center',
        alignItems: 'center',
        padding: 24,
      }}
    >
      <Card
        style={{ width: '100%', maxWidth: 420 }}
        styles={{ body: { padding: 40 } }}
      >
        <Space direction="vertical" size="large" style={{ width: '100%' }}>
          <div style={{ textAlign: 'center' }}>
            <Title level={2} style={{ color: '#58a6ff', marginBottom: 4, letterSpacing: 4 }}>
              EDF
            </Title>
            <Text type="secondary">Edge Dispatch Framework 管理中心</Text>
          </div>

          {error && (
            <Alert
              message={error}
              type="error"
              showIcon
              closable
              onClose={() => setError(null)}
            />
          )}

          <Input
            size="large"
            prefix={<MailOutlined />}
            placeholder="邮箱"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            onPressEnter={handleSubmit}
          />

          <Input.Password
            size="large"
            prefix={<LockOutlined />}
            placeholder="密码"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            onPressEnter={handleSubmit}
          />

          <Checkbox checked={remember} onChange={(e) => setRemember(e.target.checked)}>
            记住我
          </Checkbox>

          <Button
            type="primary"
            size="large"
            block
            loading={loading}
            onClick={handleSubmit}
          >
            登录
          </Button>
        </Space>
      </Card>
    </div>
  );
}
