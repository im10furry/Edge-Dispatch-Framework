import { useState, useEffect } from 'react';
import {
  Card, Tag, Button, Space, Typography, Descriptions, Spin, App, Input,
} from 'antd';
import { EditOutlined, ArrowLeftOutlined, SaveOutlined } from '@ant-design/icons';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useParams, useNavigate } from 'react-router-dom';
import client from '../../api/client';
import type { Ingress, IngressType } from '../../types/api';

const { Text, Title } = Typography;
const { TextArea } = Input;

const TYPE_LABELS: Record<IngressType, string> = { '302': '302 重定向', dns: 'DNS', gateway: 'Gateway' };
const TYPE_COLORS: Record<IngressType, string> = { '302': 'cyan', dns: 'purple', gateway: 'green' };

export default function IngressDetailPage() {
  const { ingressID } = useParams<{ ingressID: string }>();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const { message } = App.useApp();
  const [editing, setEditing] = useState(false);
  const [editConfig, setEditConfig] = useState('');

  const { data: ingress, isLoading } = useQuery({
    queryKey: ['ingress', ingressID],
    queryFn: async () => {
      const res = await client.get<Ingress>(`/ingresses/${ingressID}`);
      return res.data;
    },
    enabled: !!ingressID,
  });

  useEffect(() => {
    if (ingress) {
      setEditConfig(
        typeof ingress.config === 'string'
          ? ingress.config
          : JSON.stringify(ingress.config, null, 2),
      );
    }
  }, [ingress]);

  const updateMutation = useMutation({
    mutationFn: (vals: Record<string, unknown>) => client.put(`/ingresses/${ingressID}`, vals),
    onSuccess: () => {
      message.success('配置已保存');
      setEditing(false);
      queryClient.invalidateQueries({ queryKey: ['ingress', ingressID] });
      queryClient.invalidateQueries({ queryKey: ['ingresses'] });
    },
    onError: () => message.error('保存失败'),
  });

  if (isLoading) {
    return <div style={{ display: 'flex', justifyContent: 'center', paddingTop: 120 }}><Spin size="large" /></div>;
  }

  if (!ingress) {
    return <Text type="secondary">入口不存在</Text>;
  }

  const formatConfig = (c: unknown) =>
    typeof c === 'string' ? c : JSON.stringify(c, null, 2);

  return (
    <>
      <div style={{ marginBottom: 16 }}>
        <Button type="text" icon={<ArrowLeftOutlined />} onClick={() => navigate('/ingresses')}>
          返回列表
        </Button>
      </div>

      <div style={{ display: 'flex', alignItems: 'center', gap: 12, marginBottom: 20 }}>
        <Title level={4} style={{ margin: 0 }}>{ingress.name}</Title>
        <Tag color={TYPE_COLORS[ingress.type]}>{TYPE_LABELS[ingress.type]}</Tag>
      </div>

      <Space style={{ marginBottom: 20 }}>
        {!editing ? (
          <Button icon={<EditOutlined />} onClick={() => setEditing(true)}>编辑配置</Button>
        ) : (
          <>
            <Button
              type="primary"
              icon={<SaveOutlined />}
              loading={updateMutation.isPending}
              onClick={() => {
                let parsed: unknown = editConfig;
                try { parsed = JSON.parse(editConfig); } catch { /* send as string */ }
                updateMutation.mutate({ config: parsed });
              }}
            >
              保存
            </Button>
            <Button onClick={() => { setEditing(false); setEditConfig(formatConfig(ingress.config)); }}>
              取消
            </Button>
          </>
        )}
      </Space>

      <Card title="入口信息" style={{ marginBottom: 20 }}>
        <Descriptions column={2} size="small">
          <Descriptions.Item label="入口 ID">{ingress.ingress_id}</Descriptions.Item>
          <Descriptions.Item label="类型">
            <Tag color={TYPE_COLORS[ingress.type]}>{TYPE_LABELS[ingress.type]}</Tag>
          </Descriptions.Item>
          <Descriptions.Item label="域名">{ingress.domain || '-'}</Descriptions.Item>
          <Descriptions.Item label="租户 ID">{ingress.tenant_id}</Descriptions.Item>
          <Descriptions.Item label="项目 ID">{ingress.project_id}</Descriptions.Item>
          <Descriptions.Item label="创建时间">
            {new Date(ingress.created_at).toLocaleString('zh-CN')}
          </Descriptions.Item>
          <Descriptions.Item label="更新时间" span={2}>
            {new Date(ingress.updated_at).toLocaleString('zh-CN')}
          </Descriptions.Item>
        </Descriptions>
      </Card>

      <Card title="入口配置">
        {editing ? (
          <TextArea
            value={editConfig}
            onChange={(e) => setEditConfig(e.target.value)}
            rows={16}
            style={{ fontFamily: 'monospace', background: '#0a0e14', color: '#e6edf3' }}
          />
        ) : (
          <pre
            style={{
              background: '#0a0e14',
              color: '#e6edf3',
              padding: 16,
              borderRadius: 6,
              fontFamily: '"Fira Code", "Cascadia Code", Consolas, monospace',
              fontSize: 13,
              lineHeight: 1.6,
              overflow: 'auto',
              maxHeight: 500,
              margin: 0,
              border: '1px solid #30363d',
            }}
          >
            <code>{formatConfig(ingress.config)}</code>
          </pre>
        )}
      </Card>
    </>
  );
}
