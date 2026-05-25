import { useState } from 'react';
import {
  Table, Button, Space, Tag, Modal, Form, Input, Select, App, Typography, Tooltip,
} from 'antd';
import { PlusOutlined, EditOutlined, SendOutlined, RollbackOutlined, DeleteOutlined } from '@ant-design/icons';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useNavigate } from 'react-router-dom';
import type { ColumnsType } from 'antd/es/table';
import client from '../../api/client';
import type {
  AdminPolicy, AdminPolicyType, AdminPolicyVersion, PaginatedResponse,
} from '../../types/api';

const { Text } = Typography;
const { TextArea } = Input;

const TYPE_LABELS: Record<AdminPolicyType, string> = { dispatch: '调度', block: '封禁' };
const TYPE_COLORS: Record<AdminPolicyType, string> = { dispatch: 'blue', block: 'red' };

export default function PolicyListPage() {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const { message } = App.useApp();
  const [createOpen, setCreateOpen] = useState(false);
  const [editOpen, setEditOpen] = useState(false);
  const [publishTarget, setPublishTarget] = useState<AdminPolicy | null>(null);
  const [rollbackTarget, setRollbackTarget] = useState<AdminPolicy | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<AdminPolicy | null>(null);
  const [editingPolicy, setEditingPolicy] = useState<AdminPolicy | null>(null);
  const [versions, setVersions] = useState<AdminPolicyVersion[]>([]);
  const [selectedVersionId, setSelectedVersionId] = useState<string | null>(null);
  const [form] = Form.useForm();

  const { data, isLoading } = useQuery({
    queryKey: ['policies'],
    queryFn: async () => {
      const res = await client.get<PaginatedResponse<AdminPolicy>>('/policies', {
        params: { limit: 100, offset: 0 },
      });
      return res.data;
    },
  });

  const createMutation = useMutation({
    mutationFn: (vals: Record<string, unknown>) => client.post('/policies', vals),
    onSuccess: () => {
      message.success('策略创建成功');
      setCreateOpen(false);
      form.resetFields();
      queryClient.invalidateQueries({ queryKey: ['policies'] });
    },
    onError: () => message.error('创建失败'),
  });

  const updateMutation = useMutation({
    mutationFn: ({ id, vals }: { id: string; vals: Record<string, unknown> }) =>
      client.put(`/policies/${id}`, vals),
    onSuccess: () => {
      message.success('策略更新成功');
      setEditOpen(false);
      setEditingPolicy(null);
      form.resetFields();
      queryClient.invalidateQueries({ queryKey: ['policies'] });
    },
    onError: () => message.error('更新失败'),
  });

  const publishMutation = useMutation({
    mutationFn: (id: string) => client.post(`/policies/${id}:publish`),
    onSuccess: () => {
      message.success('发布成功');
      setPublishTarget(null);
      queryClient.invalidateQueries({ queryKey: ['policies'] });
    },
    onError: () => message.error('发布失败'),
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => client.delete(`/policies/${id}`),
    onSuccess: () => {
      message.success('删除成功');
      setDeleteTarget(null);
      queryClient.invalidateQueries({ queryKey: ['policies'] });
    },
    onError: () => message.error('删除失败'),
  });

  const rollbackMutation = useMutation({
    mutationFn: ({ pid, version }: { pid: string; version: number }) =>
      client.post(`/policies/${pid}:rollback`, null, { params: { to: version } }),
    onSuccess: () => {
      message.success('回滚成功');
      setRollbackTarget(null);
      setVersions([]);
      setSelectedVersionId(null);
      queryClient.invalidateQueries({ queryKey: ['policies'] });
    },
    onError: () => message.error('回滚失败'),
  });

  const fetchVersions = async (policyId: string) => {
    try {
      const res = await client.get<AdminPolicyVersion[]>(`/policies/${policyId}/versions`);
      setVersions(res.data || []);
    } catch {
      message.error('获取版本列表失败');
    }
  };

  const handleRollback = (record: AdminPolicy) => {
    setRollbackTarget(record);
    setSelectedVersionId(null);
    fetchVersions(record.policy_id);
  };

  const handleEdit = (record: AdminPolicy) => {
    setEditingPolicy(record);
    form.setFieldsValue({
      name: record.name,
      type: record.type,
      description: record.description,
      content: typeof record.content === 'string' ? record.content : JSON.stringify(record.content, null, 2),
      tenant_id: record.tenant_id,
      project_id: record.project_id,
    });
    setEditOpen(true);
  };

  const columns: ColumnsType<AdminPolicy> = [
    {
      title: '策略名称 / ID',
      key: 'name',
      width: 200,
      render: (_, r) => (
        <Space direction="vertical" size={0}>
          <Text strong>{r.name}</Text>
          <Text type="secondary" style={{ fontSize: 12 }}>{r.policy_id}</Text>
        </Space>
      ),
    },
    {
      title: '类型',
      dataIndex: 'type',
      width: 90,
      render: (t: AdminPolicyType) => (
        <Tag color={TYPE_COLORS[t]}>{TYPE_LABELS[t]}</Tag>
      ),
    },
    {
      title: '租户/项目',
      key: 'scope',
      width: 180,
      render: (_, r) => (
        <Space direction="vertical" size={0}>
          <Text style={{ fontSize: 12 }}>租户: {r.tenant_id}</Text>
          <Text type="secondary" style={{ fontSize: 12 }}>项目: {r.project_id}</Text>
        </Space>
      ),
    },
    {
      title: '版本',
      dataIndex: 'version',
      width: 70,
      align: 'center',
      render: (v: number) => <Tag>v{v}</Tag>,
    },
    {
      title: '状态',
      dataIndex: 'is_published',
      width: 90,
      align: 'center',
      render: (v: boolean) => (
        v ? <Tag color="green">已发布</Tag> : <Tag color="default">草稿</Tag>
      ),
    },
    {
      title: '更新时间',
      dataIndex: 'updated_at',
      width: 180,
      render: (v: string) => new Date(v).toLocaleString('zh-CN'),
    },
    {
      title: '操作',
      key: 'actions',
      width: 220,
      render: (_, r) => (
        <Space size={4}>
          <Tooltip title="编辑">
            <Button type="link" size="small" icon={<EditOutlined />} onClick={() => handleEdit(r)}>
              编辑
            </Button>
          </Tooltip>
          <Tooltip title="发布">
            <Button
              type="link"
              size="small"
              icon={<SendOutlined />}
              disabled={r.is_published}
              onClick={() => setPublishTarget(r)}
            >
              发布
            </Button>
          </Tooltip>
          <Tooltip title="回滚">
            <Button type="link" size="small" icon={<RollbackOutlined />} onClick={() => handleRollback(r)}>
              回滚
            </Button>
          </Tooltip>
          <Tooltip title="删除">
            <Button
              type="link"
              size="small"
              danger
              icon={<DeleteOutlined />}
              onClick={() => setDeleteTarget(r)}
            >
              删除
            </Button>
          </Tooltip>
        </Space>
      ),
    },
  ];

  return (
    <>
      <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 16 }}>
        <Typography.Title level={4} style={{ margin: 0 }}>策略管理</Typography.Title>
        <Button type="primary" icon={<PlusOutlined />} onClick={() => { form.resetFields(); setCreateOpen(true); }}>
          新建策略
        </Button>
      </div>

      <Table
        rowKey="policy_id"
        columns={columns}
        dataSource={data?.data}
        loading={isLoading}
        pagination={{ pageSize: 20, showSizeChanger: true }}
        onRow={(r) => ({ onClick: () => navigate(`/policies/${r.policy_id}`), style: { cursor: 'pointer' } })}
      />

      {/* Create Modal */}
      <Modal
        title="新建策略"
        open={createOpen}
        onCancel={() => setCreateOpen(false)}
        onOk={() => form.submit()}
        confirmLoading={createMutation.isPending}
        width={600}
        destroyOnClose
      >
        <Form form={form} layout="vertical" onFinish={(vals) => {
          const payload = { ...vals };
          if (payload.content && typeof payload.content === 'string') {
            try { payload.content = JSON.parse(payload.content); } catch { /* send as string */ }
          }
          createMutation.mutate(payload);
        }}>
          <Form.Item name="name" label="策略名称" rules={[{ required: true, message: '请输入策略名称' }]}>
            <Input placeholder="输入策略名称" />
          </Form.Item>
          <Form.Item name="type" label="类型" initialValue="dispatch">
            <Select
              options={[
                { label: '调度 (dispatch)', value: 'dispatch' },
                { label: '封禁 (block)', value: 'block' },
              ]}
            />
          </Form.Item>
          <Form.Item name="description" label="描述">
            <TextArea rows={2} placeholder="输入策略描述" />
          </Form.Item>
          <Form.Item name="content" label="内容 (JSON)">
            <TextArea rows={8} placeholder='输入 JSON 配置内容' style={{ fontFamily: 'monospace' }} />
          </Form.Item>
          <Space size="middle">
            <Form.Item name="tenant_id" label="租户 ID">
              <Input placeholder="租户 ID" style={{ width: 240 }} />
            </Form.Item>
            <Form.Item name="project_id" label="项目 ID">
              <Input placeholder="项目 ID" style={{ width: 240 }} />
            </Form.Item>
          </Space>
        </Form>
      </Modal>

      {/* Edit Modal */}
      <Modal
        title="编辑策略"
        open={editOpen}
        onCancel={() => { setEditOpen(false); setEditingPolicy(null); }}
        onOk={() => form.submit()}
        confirmLoading={updateMutation.isPending}
        width={600}
        destroyOnClose
      >
        <Form form={form} layout="vertical" onFinish={(vals) => {
          if (!editingPolicy) return;
          const payload = { ...vals };
          if (payload.content && typeof payload.content === 'string') {
            try { payload.content = JSON.parse(payload.content); } catch { /* send as string */ }
          }
          updateMutation.mutate({ id: editingPolicy.policy_id, vals: payload });
        }}>
          <Form.Item name="name" label="策略名称" rules={[{ required: true, message: '请输入策略名称' }]}>
            <Input />
          </Form.Item>
          <Form.Item name="type" label="类型">
            <Select
              options={[
                { label: '调度 (dispatch)', value: 'dispatch' },
                { label: '封禁 (block)', value: 'block' },
              ]}
            />
          </Form.Item>
          <Form.Item name="description" label="描述">
            <TextArea rows={2} />
          </Form.Item>
          <Form.Item name="content" label="内容 (JSON)">
            <TextArea rows={8} style={{ fontFamily: 'monospace' }} />
          </Form.Item>
          <Space size="middle">
            <Form.Item name="tenant_id" label="租户 ID">
              <Input style={{ width: 240 }} />
            </Form.Item>
            <Form.Item name="project_id" label="项目 ID">
              <Input style={{ width: 240 }} />
            </Form.Item>
          </Space>
        </Form>
      </Modal>

      {/* Publish Confirm Modal */}
      <Modal
        title="确认发布"
        open={!!publishTarget}
        onCancel={() => setPublishTarget(null)}
        onOk={() => publishTarget && publishMutation.mutate(publishTarget.policy_id)}
        confirmLoading={publishMutation.isPending}
      >
        <p>确认发布策略 <Text strong>{publishTarget?.name}</Text>？发布后将立即生效。</p>
      </Modal>

      {/* Rollback Modal */}
      <Modal
        title={`回滚策略 — ${rollbackTarget?.name || ''}`}
        open={!!rollbackTarget}
        onCancel={() => { setRollbackTarget(null); setVersions([]); setSelectedVersionId(null); }}
        onOk={() => {
          if (rollbackTarget && selectedVersionId) {
            const version = versions.find((item) => item.version_id === selectedVersionId)?.version;
            if (version !== undefined) {
              rollbackMutation.mutate({ pid: rollbackTarget.policy_id, version });
            }
          }
        }}
        confirmLoading={rollbackMutation.isPending}
        okText="确认回滚"
        width={700}
      >
        <div style={{ marginBottom: 12 }}>
          <Text type="secondary">当前版本: v{rollbackTarget?.version}</Text>
        </div>
        <Table
          rowKey="version_id"
          size="small"
          dataSource={versions}
          rowSelection={{
            type: 'radio',
            selectedRowKeys: selectedVersionId ? [selectedVersionId] : [],
            onChange: (keys) => setSelectedVersionId(keys[0] as string),
          }}
          columns={[
            { title: '版本', dataIndex: 'version', width: 80, render: (v: number) => <Tag>v{v}</Tag> },
            {
              title: '内容预览',
              dataIndex: 'content',
              ellipsis: true,
              render: (c: unknown) => (
                <Text style={{ fontFamily: 'monospace', fontSize: 12 }}>
                  {typeof c === 'string' ? c.slice(0, 120) : JSON.stringify(c).slice(0, 120)}
                </Text>
              ),
            },
            {
              title: '创建时间',
              dataIndex: 'created_at',
              width: 180,
              render: (v: string) => new Date(v).toLocaleString('zh-CN'),
            },
          ]}
          pagination={false}
        />
      </Modal>

      {/* Delete Confirm Modal */}
      <Modal
        title="确认删除"
        open={!!deleteTarget}
        onCancel={() => setDeleteTarget(null)}
        onOk={() => deleteTarget && deleteMutation.mutate(deleteTarget.policy_id)}
        confirmLoading={deleteMutation.isPending}
        okButtonProps={{ danger: true }}
        okText="删除"
      >
        <p>确认删除策略 <Text strong>{deleteTarget?.name}</Text>？此操作不可撤销。</p>
      </Modal>
    </>
  );
}
