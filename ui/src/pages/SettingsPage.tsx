import { useState, useEffect } from 'react';
import {
  Card,
  Collapse,
  Descriptions,
  Switch,
  Tag,
  Input,
  Button,
  Space,
  App,
  Spin,
} from 'antd';
import {
  LinkOutlined,
  SaveOutlined,
  SafetyCertificateOutlined,
  GlobalOutlined,
  EyeOutlined,
} from '@ant-design/icons';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import client from '../api/client';
import type { AdminConfig } from '../types/api';

function fetchSettings() {
  return client.get<AdminConfig>('/settings').then((r) => r.data);
}

function updateSettings(data: Partial<AdminConfig>) {
  return client.put('/settings', data).then((r) => r.data);
}

export default function SettingsPage() {
  const { message } = App.useApp();
  const queryClient = useQueryClient();

  const { data: settings, isLoading } = useQuery({
    queryKey: ['settings'],
    queryFn: fetchSettings,
  });

  const [grafanaUrl, setGrafanaUrl] = useState('');
  const [prometheusUrl, setPrometheusUrl] = useState('');
  const [lokiUrl, setLokiUrl] = useState('');

  useEffect(() => {
    if (settings) {
      setGrafanaUrl(settings.grafana_url || '');
      setPrometheusUrl(settings.prometheus_url || '');
      setLokiUrl(settings.loki_url || '');
    }
  }, [settings]);

  const saveMutation = useMutation({
    mutationFn: updateSettings,
    onSuccess: () => {
      message.success('设置更新已提交');
      queryClient.invalidateQueries({ queryKey: ['settings'] });
    },
    onError: () => {
      message.error('设置更新失败');
    },
  });

  const handleSave = () => {
    saveMutation.mutate({
      grafana_url: grafanaUrl,
      prometheus_url: prometheusUrl,
      loki_url: lokiUrl,
    });
  };

  if (isLoading) {
    return (
      <div style={{ display: 'flex', justifyContent: 'center', padding: 80 }}>
        <Spin size="large" />
      </div>
    );
  }

  const collapseItems = [
    {
      key: 'auth',
      label: (
        <Space>
          <SafetyCertificateOutlined />
          <span>认证设置</span>
        </Space>
      ),
      children: (
        <Descriptions column={1} size="small" colon={false} style={{ marginTop: 8 }}>
          <Descriptions.Item label="OIDC 已启用">
            <Tag color={settings?.oidc_enabled ? 'green' : 'default'}>
              {settings?.oidc_enabled ? '是' : '否'}
            </Tag>
          </Descriptions.Item>
          <Descriptions.Item label="OIDC 提供者 URL">
            <span style={{ fontFamily: 'monospace', fontSize: 13 }}>
              {settings?.oidc_provider_url || '-'}
            </span>
          </Descriptions.Item>
          <Descriptions.Item label="OIDC 客户端 ID">
            <span style={{ fontFamily: 'monospace', fontSize: 13 }}>
              {settings?.oidc_client_id || '-'}
            </span>
          </Descriptions.Item>
          <Descriptions.Item label="本地认证">
            <Switch
              checked={settings?.local_auth_enabled ?? true}
              disabled
              size="small"
            />
          </Descriptions.Item>
          <Descriptions.Item label="只读模式">
            <Switch
              checked={settings?.read_only_mode ?? false}
              disabled
              size="small"
            />
          </Descriptions.Item>
        </Descriptions>
      ),
    },
    {
      key: 'network',
      label: (
        <Space>
          <GlobalOutlined />
          <span>网络设置</span>
        </Space>
      ),
      children: (
        <Descriptions column={1} size="small" colon={false} style={{ marginTop: 8 }}>
          <Descriptions.Item label="IP 白名单">
            <Space size={[4, 4]} wrap>
              {settings?.ip_allowlist?.length ? (
                settings.ip_allowlist.map((ip) => (
                  <Tag key={ip} color="blue">
                    {ip}
                  </Tag>
                ))
              ) : (
                <span style={{ color: '#8b949e' }}>未配置</span>
              )}
            </Space>
          </Descriptions.Item>
        </Descriptions>
      ),
    },
    {
      key: 'observability',
      label: (
        <Space>
          <EyeOutlined />
          <span>可观测性</span>
        </Space>
      ),
      children: (
        <div style={{ marginTop: 8, display: 'flex', flexDirection: 'column', gap: 16 }}>
          <div>
            <div style={{ marginBottom: 6, color: '#8b949e', fontSize: 13 }}>Grafana URL</div>
            <Input.Search
              value={grafanaUrl}
              onChange={(e) => setGrafanaUrl(e.target.value)}
              placeholder="https://grafana.example.com"
              enterButton={
                <Button
                  type="link"
                  icon={<LinkOutlined />}
                  disabled={!grafanaUrl}
                  onClick={() => window.open(grafanaUrl, '_blank')}
                >
                  打开
                </Button>
              }
              style={{ maxWidth: 600 }}
            />
          </div>
          <div>
            <div style={{ marginBottom: 6, color: '#8b949e', fontSize: 13 }}>Prometheus URL</div>
            <Input.Search
              value={prometheusUrl}
              onChange={(e) => setPrometheusUrl(e.target.value)}
              placeholder="https://prometheus.example.com"
              enterButton={
                <Button
                  type="link"
                  icon={<LinkOutlined />}
                  disabled={!prometheusUrl}
                  onClick={() => window.open(prometheusUrl, '_blank')}
                >
                  打开
                </Button>
              }
              style={{ maxWidth: 600 }}
            />
          </div>
          <div>
            <div style={{ marginBottom: 6, color: '#8b949e', fontSize: 13 }}>Loki URL</div>
            <Input.Search
              value={lokiUrl}
              onChange={(e) => setLokiUrl(e.target.value)}
              placeholder="https://loki.example.com"
              enterButton={
                <Button
                  type="link"
                  icon={<LinkOutlined />}
                  disabled={!lokiUrl}
                  onClick={() => window.open(lokiUrl, '_blank')}
                >
                  打开
                </Button>
              }
              style={{ maxWidth: 600 }}
            />
          </div>
        </div>
      ),
    },
  ];

  return (
    <div>
      <Card size="small" styles={{ body: { padding: '8px 16px' } }}>
        <Collapse
          defaultActiveKey={['auth', 'network', 'observability']}
          ghost
          items={collapseItems}
        />
      </Card>

      <div style={{ marginTop: 24, display: 'flex', justifyContent: 'flex-end' }}>
        <Button
          type="primary"
          icon={<SaveOutlined />}
          loading={saveMutation.isPending}
          onClick={handleSave}
          size="large"
        >
          保存设置
        </Button>
      </div>
    </div>
  );
}
