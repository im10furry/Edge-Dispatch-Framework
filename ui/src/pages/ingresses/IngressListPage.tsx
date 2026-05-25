import { useState } from 'react';
import {
  Table, Button, Space, Tag, Modal, Form, Input, Select, App, Typography, Tooltip,
} from 'antd';
import { PlusOutlined, EditOutlined, DeleteOutlined } from '@ant-design/icons';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useNavigate } from 'react-router-dom';
import type { ColumnsType } from 'antd/es/table';
import client from '../../api/client';
import type { Ingress, IngressType, PaginatedResponse } from '../../types/api';

const { Text } = Typography;
const { TextArea } = Input;

const TYPE_LABELS: Record<IngressType, string> = { '302': '302 重定向', dns: 'DNS', gateway: 'Gateway' };
const TYPE_COLORS: Record<IngressType, string> = { '302': 'cyan', dns: 'purple', gateway: 'green' };

export default function IngressListPage() {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const { message } = App.useApp();
  const [createOpen, setCreateOpen] = useState(false);
  const [editOpen, setEditOpen] = useState(false);
  const [deleteTarget, setDeleteTarget] = useState<Ingress | null>(null);
  const [editingIngress, setEditingIngress] = useState<Ingress | null>(null);
  const [form] = Form.useForm();

  const { data, isLoading } = useQuery({
    queryKey: ['ingresses'],
    queryFn: async () => {
      const res = await client.get<PaginatedResponse<Ingress>>('/ingresses', {
        params: { limit: 100, offset: 0 },
      });
      return res.data;
    },
  });

  const createMutation = useMutation({
    mutationFn: (vals: Record<string, unknown>) => client.post('/ingresses', vals),
    onSuccess: () => {
      message.success('入口创建成功');
      setCreateOpen(false);
      form.resetFields();
      queryClient.invalidateQueries({ queryKey: ['ingresses'] });
    },
    onError: () => message.error('创建失败'),
  });

  const updateMutation = useMutation({
    mutationFn: ({ id, vals }: { id: string; vals: Record<string, unknown> }) =>
      client.put(`/ingresses/${id}`, vals),
    onSuccess: () => {
      message.success('入口更新成功');
      setEditOpen(false);
      setEditingIngress(null);
      form.resetFields();
      queryClient.invalidateQueries({ queryKey: ['ingresses'] });
    },
    onError: () => message.error('更新失败'),
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => client.delete(`/ingresses/${id}`),
    onSuccess: () => {
      message.success('删除成功');
      setDeleteTarget(null);
      queryClient.invalidateQueries({ queryKey: ['ingresses'] });
    },
    onError: () => message.error('删除失败'),
  });

  const handleEdit = (record: Ingress) => {
    setEditingIngress(record);
    form.setFieldsValue({
      name: record.name,
      type: record.type,
      domain: record.domain,
      config: typeof record.config === 'string' ? record.config : JSON.stringify(record.config, null, 2),
      tenant_id: record.tenant_id,
      project_id: record.project_id,
    });
    setEditOpen(true);
  };

  const columns: ColumnsType<Ingress> = [
    {
      title: '名称',
      dataIndex: 'name',
      width: 200,
      render: (v: string, r) => (
        <Space direction="vertical" size={0}>
          <Text strong>{v}</Text>
          <Text type="secondary" style={{ fontSize: 12 }}>{r.ingress_id}</Text>
        </Space>
      ),
    },
    {
      title: '类型',
      dataIndex: 'type',
      width: 120,
      render: (t: IngressType) => (
        <Tag color={TYPE_COLORS[t]}>{TYPE_LABELS[t]}</Tag>
      ),
    },
    {
      title: '域名',
      dataIndex: 'domain',
      width: 200,
      ellipsis: true,
      render: (v: string) => v || '-',
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
      title: '创建时间',
      dataIndex: 'created_at',
      width: 180,
      render: (v: string) => new Date(v).toLocaleString('zh-CN'),
    },
    {
      title: '操作',
      key: 'actions',
      width: 140,
      render: (_, r) => (
        <Space size={4}>
          <Tooltip title="编辑">
            <Button type="link" size="small" icon={<EditOutlined />} onClick={(e) => { e.stopPropagation(); handleEdit(r); }}>
              编辑
            </Button>
          </Tooltip>
          <Tooltip title="删除">
            <Button
              type="link"
              size="small"
              danger
              icon={<DeleteOutlined />}
              onClick={(e) => { e.stopPropagation(); setDeleteTarget(r); }}
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
        <Typography.Title level={4} style={{ margin: 0 }}>入口管理</Typography.Title>
        <Button type="primary" icon={<PlusOutlined />} onClick={() => { form.resetFields(); setCreateOpen(true); }}>
          新建入口
        </Button>
      </div>

      <Table
        rowKey="ingress_id"
        columns={columns}
        dataSource={data?.data}
        loading={isLoading}
        pagination={{ pageSize: 20, showSizeChanger: true }}
        onRow={(r) => ({ onClick: () => navigate(`/ingresses/${r.ingress_id}`), style: { cursor: 'pointer' } })}
      />

      {/* Create Modal */}
      <Modal
        title="新建入口"
        open={createOpen}
        onCancel={() => setCreateOpen(false)}
        onOk={() => form.submit()}
        confirmLoading={createMutation.isPending}
        width={600}
        destroyOnClose
      >
        <Form form={form} layout="vertical" onFinish={(vals) => {
          const payload = { ...vals };
          if (payload.config && typeof payload.config === 'string') {
            try { payload.config = JSON.parse(payload.config); } catch { /* send as string */ }
          }
          createMutation.mutate(payload);
        }}>
          <Form.Item name="name" label="入口名称" rules={[{ required: true, message: '请输入入口名称' }]}>
            <Input placeholder="输入入口名称" />
          </Form.Item>
          <Form.Item name="type" label="类型" initialValue="gateway">
            <Select
              options={[
                { label: '302 重定向', value: '302' },
                { label: 'DNS', value: 'dns' },
                { label: 'Gateway', value: 'gateway' },
              ]}
            />
          </Form.Item>
          <Form.Item name="domain" label="域名">
            <Input placeholder="example.com" />
          </Form.Item>
          <Form.Item name="config" label="配置 (JSON)">
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
        title="编辑入口"
        open={editOpen}
        onCancel={() => { setEditOpen(false); setEditingIngress(null); }}
        onOk={() => form.submit()}
        confirmLoading={updateMutation.isPending}
        width={600}
        destroyOnClose
      >
        <Form form={form} layout="vertical" onFinish={(vals) => {
          if (!editingIngress) return;
          const payload = { ...vals };
          if (payload.config && typeof payload.config === 'string') {
            try { payload.config = JSON.parse(payload.config); } catch { /* send as string */ }
          }
          updateMutation.mutate({ id: editingIngress.ingress_id, vals: payload });
        }}>
          <Form.Item name="name" label="入口名称" rules={[{ required: true, message: '请输入入口名称' }]}>
            <Input />
          </Form.Item>
          <Form.Item name="type" label="类型">
            <Select
              options={[
                { label: '302 重定向', value: '302' },
                { label: 'DNS', value: 'dns' },
                { label: 'Gateway', value: 'gateway' },
              ]}
            />
          </Form.Item>
          <Form.Item name="domain" label="域名">
            <Input />
          </Form.Item>
          <Form.Item name="config" label="配置 (JSON)">
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

      {/* Delete Modal */}
      <Modal
        title="确认删除"
        open={!!deleteTarget}
        onCancel={() => setDeleteTarget(null)}
        onOk={() => deleteTarget && deleteMutation.mutate(deleteTarget.ingress_id)}
        confirmLoading={deleteMutation.isPending}
        okButtonProps={{ danger: true }}
        okText="删除"
      >
        <p>确认删除入口 <Text strong>{deleteTarget?.name}</Text>？此操作不可撤销。</p>
      </Modal>
    </>
  );
}
