import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  Button, Card, Row, Col, Typography, Modal, Form, Input, Dropdown, Space, App, Empty, Spin,
} from 'antd';
import {
  PlusOutlined, MoreOutlined, EditOutlined, DeleteOutlined, TeamOutlined,
} from '@ant-design/icons';
import client from '../../api/client';
import { getApiErrorMessage } from '../../api/errors';
import type { Tenant, CreateTenantRequest, UpdateTenantRequest } from '../../types/api';
import dayjs from 'dayjs';

const { Text, Paragraph } = Typography;

export default function TenantListPage() {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const { message } = App.useApp();
  const [createOpen, setCreateOpen] = useState(false);
  const [editTenant, setEditTenant] = useState<Tenant | null>(null);
  const [deleteTenant, setDeleteTenant] = useState<Tenant | null>(null);
  const [createForm] = Form.useForm();
  const [editForm] = Form.useForm();

  const { data: tenants, isLoading } = useQuery({
    queryKey: ['tenants'],
    queryFn: async () => {
      const res = await client.get<Tenant[]>('/tenants');
      return res.data;
    },
  });

  const createMutation = useMutation({
    mutationFn: (body: CreateTenantRequest) => client.post('/tenants', body),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['tenants'] });
      message.success('租户创建成功');
      setCreateOpen(false);
      createForm.resetFields();
    },
    onError: (err: unknown) => {
      message.error(getApiErrorMessage(err, '创建失败'));
    },
  });

  const updateMutation = useMutation({
    mutationFn: ({ id, body }: { id: string; body: UpdateTenantRequest }) =>
      client.put(`/tenants/${id}`, body),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['tenants'] });
      message.success('租户更新成功');
      setEditTenant(null);
      editForm.resetFields();
    },
    onError: (err: unknown) => {
      message.error(getApiErrorMessage(err, '更新失败'));
    },
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => client.delete(`/tenants/${id}`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['tenants'] });
      message.success('租户已删除');
      setDeleteTenant(null);
    },
    onError: (err: unknown) => {
      message.error(getApiErrorMessage(err, '删除失败'));
    },
  });

  const handleCreate = () => {
    createForm.validateFields().then((values) => {
      createMutation.mutate({ name: values.name, description: values.description });
    });
  };

  const handleEdit = () => {
    editForm.validateFields().then((values) => {
      if (!editTenant) return;
      updateMutation.mutate({ id: editTenant.tenant_id, body: { name: values.name, description: values.description } });
    });
  };

  const handleDelete = () => {
    if (!deleteTenant) return;
    deleteMutation.mutate(deleteTenant.tenant_id);
  };

  const openEditModal = (tenant: Tenant) => {
    setEditTenant(tenant);
    editForm.setFieldsValue({ name: tenant.name, description: tenant.description });
  };

  if (isLoading) {
    return (
      <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: 300 }}>
        <Spin size="large" />
      </div>
    );
  }

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 24 }}>
        <h2 style={{ color: '#e6edf3', margin: 0, fontSize: 20, fontWeight: 600 }}>租户管理</h2>
        <Button type="primary" icon={<PlusOutlined />} onClick={() => setCreateOpen(true)}>
          新建租户
        </Button>
      </div>

      {!tenants || tenants.length === 0 ? (
        <Empty description="暂无租户" />
      ) : (
        <Row gutter={[16, 16]}>
          {tenants.map((tenant) => (
            <Col key={tenant.tenant_id} xs={24} sm={12} lg={8} xl={6}>
              <Card
                hoverable
                style={{ cursor: 'pointer' }}
                bodyStyle={{ padding: 20 }}
                onClick={() => navigate(`/tenants/${tenant.tenant_id}`)}
                title={
                  <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                    <Space>
                      <TeamOutlined style={{ color: '#58a6ff' }} />
                      <Text strong style={{ color: '#e6edf3', fontSize: 15 }}>{tenant.name}</Text>
                    </Space>
                    <Dropdown
                      menu={{
                        items: [
                          {
                            key: 'edit',
                            icon: <EditOutlined />,
                            label: '编辑',
                            onClick: (e) => { e.domEvent.stopPropagation(); openEditModal(tenant); },
                          },
                          {
                            key: 'delete',
                            icon: <DeleteOutlined />,
                            label: '删除',
                            danger: true,
                            onClick: (e) => { e.domEvent.stopPropagation(); setDeleteTenant(tenant); },
                          },
                        ],
                      }}
                      trigger={['click']}
                    >
                      <Button
                        type="text"
                        size="small"
                        icon={<MoreOutlined />}
                        onClick={(e) => e.stopPropagation()}
                      />
                    </Dropdown>
                  </div>
                }
              >
                <Paragraph
                  ellipsis={{ rows: 2 }}
                  style={{ color: '#8b949e', marginBottom: 12, minHeight: 40 }}
                >
                  {tenant.description || '暂无描述'}
                </Paragraph>
                <Text type="secondary" style={{ fontSize: 12 }}>
                  创建于 {dayjs(tenant.created_at).format('YYYY-MM-DD HH:mm')}
                </Text>
              </Card>
            </Col>
          ))}
        </Row>
      )}

      <Modal
        title="新建租户"
        open={createOpen}
        onCancel={() => { setCreateOpen(false); createForm.resetFields(); }}
        onOk={handleCreate}
        confirmLoading={createMutation.isPending}
        okText="创建"
        cancelText="取消"
      >
        <Form form={createForm} layout="vertical" style={{ marginTop: 16 }}>
          <Form.Item name="name" label="名称" rules={[{ required: true, message: '请输入租户名称' }]}>
            <Input placeholder="请输入租户名称" />
          </Form.Item>
          <Form.Item name="description" label="描述">
            <Input.TextArea rows={3} placeholder="请输入租户描述（选填）" />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title="编辑租户"
        open={!!editTenant}
        onCancel={() => { setEditTenant(null); editForm.resetFields(); }}
        onOk={handleEdit}
        confirmLoading={updateMutation.isPending}
        okText="保存"
        cancelText="取消"
      >
        <Form form={editForm} layout="vertical" style={{ marginTop: 16 }}>
          <Form.Item name="name" label="名称" rules={[{ required: true, message: '请输入租户名称' }]}>
            <Input placeholder="请输入租户名称" />
          </Form.Item>
          <Form.Item name="description" label="描述">
            <Input.TextArea rows={3} placeholder="请输入租户描述（选填）" />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title="删除租户"
        open={!!deleteTenant}
        onCancel={() => setDeleteTenant(null)}
        onOk={handleDelete}
        confirmLoading={deleteMutation.isPending}
        okText="确认删除"
        cancelText="取消"
        okButtonProps={{ danger: true }}
      >
        <div style={{ padding: '12px 0', color: '#f85149' }}>
          删除租户将同时移除其下所有项目、用户关联
        </div>
        <Text>确定要删除租户 <Text strong>{deleteTenant?.name}</Text> 吗？此操作不可撤销。</Text>
      </Modal>
    </div>
  );
}
