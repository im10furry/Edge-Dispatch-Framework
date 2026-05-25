import { useState, useEffect } from 'react';
import {
  Card, Tag, Button, Space, Typography, Descriptions, Spin, App, Modal, Input, Timeline,
} from 'antd';
import {
  EditOutlined, SendOutlined, RollbackOutlined, SaveOutlined, ArrowLeftOutlined,
} from '@ant-design/icons';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useParams, useNavigate } from 'react-router-dom';
import client from '../../api/client';
import type { AdminPolicy, AdminPolicyType, AdminPolicyVersion } from '../../types/api';

const { Text, Title } = Typography;
const { TextArea } = Input;

const TYPE_LABELS: Record<AdminPolicyType, string> = { dispatch: '调度', block: '封禁' };
const TYPE_COLORS: Record<AdminPolicyType, string> = { dispatch: 'blue', block: 'red' };

export default function PolicyDetailPage() {
  const { policyID } = useParams<{ policyID: string }>();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const { message } = App.useApp();
  const [editing, setEditing] = useState(false);
  const [editContent, setEditContent] = useState('');
  const [publishOpen, setPublishOpen] = useState(false);
  const [rollbackOpen, setRollbackOpen] = useState(false);
  const [selectedVersion, setSelectedVersion] = useState<AdminPolicyVersion | null>(null);

  const { data: policy, isLoading } = useQuery({
    queryKey: ['policy', policyID],
    queryFn: async () => {
      const res = await client.get<AdminPolicy>(`/policies/${policyID}`);
      return res.data;
    },
    enabled: !!policyID,
  });

  const { data: versions } = useQuery({
    queryKey: ['policyVersions', policyID],
    queryFn: async () => {
      const res = await client.get<AdminPolicyVersion[]>(`/policies/${policyID}/versions`);
      return res.data || [];
    },
    enabled: !!policyID,
  });

  useEffect(() => {
    if (policy) {
      setEditContent(
        typeof policy.content === 'string'
          ? policy.content
          : JSON.stringify(policy.content, null, 2),
      );
    }
  }, [policy]);

  const updateMutation = useMutation({
    mutationFn: (content: unknown) => client.put(`/policies/${policyID}`, { content }),
    onSuccess: () => {
      message.success('草稿已保存');
      setEditing(false);
      queryClient.invalidateQueries({ queryKey: ['policy', policyID] });
      queryClient.invalidateQueries({ queryKey: ['policies'] });
    },
    onError: () => message.error('保存失败'),
  });

  const publishMutation = useMutation({
    mutationFn: () => client.post(`/policies/${policyID}:publish`),
    onSuccess: () => {
      message.success('发布成功');
      setPublishOpen(false);
      queryClient.invalidateQueries({ queryKey: ['policy', policyID] });
      queryClient.invalidateQueries({ queryKey: ['policies'] });
    },
    onError: () => message.error('发布失败'),
  });

  const rollbackMutation = useMutation({
    mutationFn: (version: number) =>
      client.post(`/policies/${policyID}:rollback`, null, { params: { to: version } }),
    onSuccess: () => {
      message.success('回滚成功');
      setRollbackOpen(false);
      setSelectedVersion(null);
      queryClient.invalidateQueries({ queryKey: ['policy', policyID] });
      queryClient.invalidateQueries({ queryKey: ['policyVersions', policyID] });
      queryClient.invalidateQueries({ queryKey: ['policies'] });
    },
    onError: () => message.error('回滚失败'),
  });

  if (isLoading) {
    return <div style={{ display: 'flex', justifyContent: 'center', paddingTop: 120 }}><Spin size="large" /></div>;
  }

  if (!policy) {
    return <Text type="secondary">策略不存在</Text>;
  }

  const formatContent = (c: unknown) =>
    typeof c === 'string' ? c : JSON.stringify(c, null, 2);

  return (
    <>
      <div style={{ marginBottom: 16 }}>
        <Button type="text" icon={<ArrowLeftOutlined />} onClick={() => navigate('/policies')}>
          返回列表
        </Button>
      </div>

      <div style={{ display: 'flex', alignItems: 'center', gap: 12, marginBottom: 20 }}>
        <Title level={4} style={{ margin: 0 }}>{policy.name}</Title>
        <Tag color={TYPE_COLORS[policy.type]}>{TYPE_LABELS[policy.type]}</Tag>
        <Tag>v{policy.version}</Tag>
        {policy.is_published ? (
          <Tag color="green">已发布</Tag>
        ) : (
          <Tag color="default">草稿</Tag>
        )}
      </div>

      <Space style={{ marginBottom: 20 }}>
        {!editing ? (
          <Button icon={<EditOutlined />} onClick={() => setEditing(true)}>编辑内容</Button>
        ) : (
          <Button
            type="primary"
            icon={<SaveOutlined />}
            loading={updateMutation.isPending}
            onClick={() => {
              let parsed: unknown = editContent;
              try { parsed = JSON.parse(editContent); } catch { /* send as string */ }
              updateMutation.mutate(parsed);
            }}
          >
            保存草稿
          </Button>
        )}
        <Button
          icon={<SendOutlined />}
          disabled={policy.is_published}
          onClick={() => setPublishOpen(true)}
        >
          发布
        </Button>
        <Button icon={<RollbackOutlined />} onClick={() => setRollbackOpen(true)}>
          回滚
        </Button>
      </Space>

      <Card title="策略信息" style={{ marginBottom: 20 }}>
        <Descriptions column={2} size="small">
          <Descriptions.Item label="策略 ID">{policy.policy_id}</Descriptions.Item>
          <Descriptions.Item label="版本">v{policy.version}</Descriptions.Item>
          <Descriptions.Item label="租户 ID">{policy.tenant_id}</Descriptions.Item>
          <Descriptions.Item label="项目 ID">{policy.project_id}</Descriptions.Item>
          <Descriptions.Item label="描述">{policy.description || '-'}</Descriptions.Item>
          <Descriptions.Item label="状态">
            {policy.is_published ? <Tag color="green">已发布</Tag> : <Tag color="default">草稿</Tag>}
          </Descriptions.Item>
          <Descriptions.Item label="创建时间">
            {new Date(policy.created_at).toLocaleString('zh-CN')}
          </Descriptions.Item>
          <Descriptions.Item label="更新时间">
            {new Date(policy.updated_at).toLocaleString('zh-CN')}
          </Descriptions.Item>
        </Descriptions>
      </Card>

      <Card
        title="策略内容"
        extra={editing ? (
          <Button
            size="small"
            onClick={() => {
              if (policy) {
                setEditContent(formatContent(policy.content));
              }
              setEditing(false);
            }}
          >
            取消
          </Button>
        ) : null}
        style={{ marginBottom: 20 }}
      >
        {editing ? (
          <TextArea
            value={editContent}
            onChange={(e) => setEditContent(e.target.value)}
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
            <code>{formatContent(policy.content)}</code>
          </pre>
        )}
      </Card>

      <Card title={`版本历史 (${(versions || []).length})`} style={{ marginBottom: 20 }}>
        {(versions || []).length === 0 ? (
          <Text type="secondary">暂无版本记录</Text>
        ) : (
          <Timeline
            items={(versions || []).map((v) => ({
              color: v.version === policy.version ? 'blue' : 'gray',
              children: (
                <Card size="small" style={{ marginBottom: 8 }}>
                  <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                    <Space>
                      <Tag color={v.version === policy.version ? 'blue' : 'default'}>v{v.version}</Tag>
                      <Text type="secondary" style={{ fontSize: 12 }}>
                        {new Date(v.created_at).toLocaleString('zh-CN')}
                      </Text>
                    </Space>
                    {v.version !== policy.version && (
                      <Button
                        size="small"
                        type="link"
                        icon={<RollbackOutlined />}
                        onClick={() => {
                          setSelectedVersion(v);
                          setRollbackOpen(true);
                        }}
                      >
                        回滚到此版本
                      </Button>
                    )}
                  </div>
                  <pre style={{
                    background: '#0a0e14',
                    color: '#e6edf3',
                    padding: 8,
                    borderRadius: 4,
                    fontFamily: 'monospace',
                    fontSize: 12,
                    margin: '8px 0 0',
                    maxHeight: 120,
                    overflow: 'auto',
                  }}>
                    {typeof v.content === 'string' ? v.content.slice(0, 800) : JSON.stringify(v.content, null, 2).slice(0, 800)}
                  </pre>
                </Card>
              ),
            }))}
          />
        )}
      </Card>

      {/* Publish Confirmation */}
      <Modal
        title="确认发布"
        open={publishOpen}
        onCancel={() => setPublishOpen(false)}
        onOk={() => publishMutation.mutate()}
        confirmLoading={publishMutation.isPending}
      >
        <p>确认发布策略 <Text strong>{policy.name}</Text>？发布后将立即生效。</p>
      </Modal>

      {/* Rollback Modal */}
      <Modal
        title="确认回滚"
        open={rollbackOpen}
        onCancel={() => { setRollbackOpen(false); setSelectedVersion(null); }}
        onOk={() => {
          if (selectedVersion) {
            rollbackMutation.mutate(selectedVersion.version);
          }
        }}
        confirmLoading={rollbackMutation.isPending}
        okText="确认回滚"
        width={700}
      >
        {selectedVersion ? (
          <>
            <p>
              当前版本: <Tag>v{policy.version}</Tag>
              {' '}→{' '}
              回滚至: <Tag color="orange">v{selectedVersion.version}</Tag>
            </p>
            <Text type="secondary" style={{ display: 'block', marginBottom: 8 }}>
              {new Date(selectedVersion.created_at).toLocaleString('zh-CN')}
            </Text>
            <pre style={{
              background: '#0a0e14',
              color: '#e6edf3',
              padding: 12,
              borderRadius: 4,
              fontFamily: 'monospace',
              fontSize: 12,
              maxHeight: 300,
              overflow: 'auto',
              border: '1px solid #30363d',
            }}>
              {typeof selectedVersion.content === 'string'
                ? selectedVersion.content
                : JSON.stringify(selectedVersion.content, null, 2)}
            </pre>
          </>
        ) : (
          <>
            <p>将回滚策略 <Text strong>{policy.name}</Text> 到上一个版本。</p>
            {(versions || []).length > 1 && (
              <p>
                将回滚至 <Tag>v{(versions || []).filter((v) => v.version !== policy.version).sort((a, b) => b.version - a.version)[0]?.version}</Tag>
              </p>
            )}
          </>
        )}
      </Modal>
    </>
  );
}
