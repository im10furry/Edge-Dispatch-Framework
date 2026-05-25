import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  Button, Table, Tag, Space, Modal, Form, Input, Select, App, Spin, Empty,
} from 'antd';
import { PlusOutlined, EditOutlined, DeleteOutlined } from '@ant-design/icons';
import client from '../../api/client';
import { getApiErrorMessage } from '../../api/errors';
import type { User, Tenant, UserRole, RoleBinding, CreateUserRequest, UpdateUserRequest } from '../../types/api';
import dayjs from 'dayjs';

const roleLabels: Record<UserRole, string> = {
  tenant_owner: '租户所有者',
  tenant_admin: '租户管理员',
  project_operator: '项目运维',
  project_viewer: '项目查看',
};

const roleColorMap: Record<UserRole, string> = {
  tenant_owner: '#f85149',
  tenant_admin: '#d29922',
  project_operator: '#1f6feb',
  project_viewer: '#3fb950',
};

const allRoles: UserRole[] = ['tenant_owner', 'tenant_admin', 'project_operator', 'project_viewer'];

export default function UserListPage() {
  const queryClient = useQueryClient();
  const { message } = App.useApp();
  const [createOpen, setCreateOpen] = useState(false);
  const [editUser, setEditUser] = useState<User | null>(null);
  const [deleteUser, setDeleteUser] = useState<User | null>(null);
  const [createForm] = Form.useForm();
  const [editForm] = Form.useForm();

  const { data: users, isLoading } = useQuery({
    queryKey: ['users'],
    queryFn: async () => {
      const res = await client.get<User[]>('/users');
      return res.data;
    },
  });

  const { data: tenants } = useQuery({
    queryKey: ['tenants'],
    queryFn: async () => {
      const res = await client.get<Tenant[]>('/tenants');
      return res.data;
    },
  });

  const createMutation = useMutation({
    mutationFn: (body: CreateUserRequest) => client.post('/users', body),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['users'] });
      message.success('用户创建成功');
      setCreateOpen(false);
      createForm.resetFields();
    },
    onError: (err: unknown) => {
      message.error(getApiErrorMessage(err, '创建失败'));
    },
  });

  const updateMutation = useMutation({
    mutationFn: ({ id, body }: { id: string; body: UpdateUserRequest }) =>
      client.put(`/users/${id}`, body),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['users'] });
      message.success('用户更新成功');
      setEditUser(null);
      editForm.resetFields();
    },
    onError: (err: unknown) => {
      message.error(getApiErrorMessage(err, '更新失败'));
    },
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => client.delete(`/users/${id}`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['users'] });
      message.success('用户已删除');
      setDeleteUser(null);
    },
    onError: (err: unknown) => {
      message.error(getApiErrorMessage(err, '删除失败'));
    },
  });

  const handleCreate = () => {
    createForm.validateFields().then((values) => {
      const roles: RoleBinding[] = (values.roles || []).map((role: UserRole) => ({
        role,
        tenant_id: values.tenant_id || '',
        project_id: '',
      }));
      createMutation.mutate({
        email: values.email,
        password: values.password,
        display_name: values.display_name,
        tenant_id: values.tenant_id || undefined,
        roles,
      });
    });
  };

  const handleEdit = () => {
    editForm.validateFields().then((values) => {
      if (!editUser) return;
      const roles: RoleBinding[] = (values.roles || []).map((role: UserRole) => ({
        role,
        tenant_id: values.tenant_id || editUser.tenant_id || '',
        project_id: '',
      }));
      const body: UpdateUserRequest = { display_name: values.display_name, roles };
      updateMutation.mutate({ id: editUser.user_id, body });
    });
  };

  const handleDelete = () => {
    if (!deleteUser) return;
    deleteMutation.mutate(deleteUser.user_id);
  };

  const openEditModal = async (user: User) => {
    try {
      const res = await client.get<User>(`/users/${user.user_id}`);
      const detail = res.data;
      setEditUser(detail);
      editForm.setFieldsValue({
        display_name: detail.display_name,
        roles: detail.roles?.map((r) => r.role) || [],
        tenant_id: detail.tenant_id,
      });
    } catch {
      setEditUser(user);
      editForm.setFieldsValue({
        display_name: user.display_name,
        roles: user.roles?.map((r) => r.role) || [],
        tenant_id: user.tenant_id,
      });
    }
  };

  const getTenantName = (tenantId: string) => {
    const t = tenants?.find((t) => t.tenant_id === tenantId);
    return t?.name || tenantId;
  };

  const columns = [
    { title: '邮箱', dataIndex: 'email', key: 'email', width: 220 },
    {
      title: '显示名',
      dataIndex: 'display_name',
      key: 'display_name',
      width: 140,
      render: (v: string) => v || '-',
    },
    {
      title: '角色',
      dataIndex: 'roles',
      key: 'roles',
      width: 280,
      render: (roles: RoleBinding[]) => (
        <Space size={[0, 4]} wrap>
          {roles?.map((r, i) => (
            <Tag key={i} color={roleColorMap[r.role]}>{roleLabels[r.role]}</Tag>
          )) || '-'}
        </Space>
      ),
    },
    {
      title: '租户',
      dataIndex: 'tenant_id',
      key: 'tenant_id',
      width: 160,
      render: (v: string) => (v ? getTenantName(v) : '-'),
    },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      key: 'created_at',
      width: 180,
      render: (v: string) => dayjs(v).format('YYYY-MM-DD HH:mm'),
    },
    {
      title: '操作',
      key: 'actions',
      width: 160,
      render: (_: unknown, record: User) => (
        <Space>
          <Button type="link" size="small" icon={<EditOutlined />} onClick={() => openEditModal(record)}>
            编辑
          </Button>
          <Button type="link" danger size="small" icon={<DeleteOutlined />} onClick={() => setDeleteUser(record)}>
            删除
          </Button>
        </Space>
      ),
    },
  ];

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
        <h2 style={{ color: '#e6edf3', margin: 0, fontSize: 20, fontWeight: 600 }}>用户管理</h2>
        <Button type="primary" icon={<PlusOutlined />} onClick={() => setCreateOpen(true)}>
          新建用户
        </Button>
      </div>

      {!users || users.length === 0 ? (
        <Empty description="暂无用户" />
      ) : (
        <Table
          rowKey="user_id"
          dataSource={users}
          columns={columns}
          pagination={false}
          size="middle"
        />
      )}

      <Modal
        title="新建用户"
        open={createOpen}
        onCancel={() => { setCreateOpen(false); createForm.resetFields(); }}
        onOk={handleCreate}
        confirmLoading={createMutation.isPending}
        okText="创建"
        cancelText="取消"
        width={520}
      >
        <Form form={createForm} layout="vertical" style={{ marginTop: 16 }}>
          <Form.Item name="email" label="邮箱" rules={[{ required: true, message: '请输入邮箱' }, { type: 'email', message: '请输入有效的邮箱' }]}>
            <Input placeholder="请输入邮箱" />
          </Form.Item>
          <Form.Item name="password" label="密码" rules={[{ required: true, message: '请输入密码' }]}>
            <Input.Password placeholder="请输入密码" />
          </Form.Item>
          <Form.Item name="display_name" label="显示名">
            <Input placeholder="请输入显示名（选填）" />
          </Form.Item>
          <Form.Item name="tenant_id" label="所属租户">
            <Select placeholder="请选择租户" allowClear showSearch optionFilterProp="label">
              {tenants?.map((t) => (
                <Select.Option key={t.tenant_id} value={t.tenant_id} label={t.name}>
                  {t.name}
                </Select.Option>
              ))}
            </Select>
          </Form.Item>
          <Form.Item name="roles" label="角色">
            <Form.Item noStyle name="roles">
              <RoleSelect />
            </Form.Item>
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title="编辑用户"
        open={!!editUser}
        onCancel={() => { setEditUser(null); editForm.resetFields(); }}
        onOk={handleEdit}
        confirmLoading={updateMutation.isPending}
        okText="保存"
        cancelText="取消"
        width={520}
      >
        <Form form={editForm} layout="vertical" style={{ marginTop: 16 }}>
          <Form.Item name="display_name" label="显示名">
            <Input placeholder="请输入显示名" />
          </Form.Item>
          <Form.Item name="roles" label="角色">
            <Form.Item noStyle name="roles">
              <RoleSelect />
            </Form.Item>
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title="删除用户"
        open={!!deleteUser}
        onCancel={() => setDeleteUser(null)}
        onOk={handleDelete}
        confirmLoading={deleteMutation.isPending}
        okText="确认删除"
        cancelText="取消"
        okButtonProps={{ danger: true }}
      >
        <p>确定要删除用户 <strong>{deleteUser?.email}</strong> 吗？此操作不可撤销。</p>
      </Modal>
    </div>
  );
}

function RoleSelect(props: { value?: UserRole[]; onChange?: (v: UserRole[]) => void }) {
  const current = props.value || [];

  const toggle = (role: UserRole) => {
    if (current.includes(role)) {
      props.onChange?.(current.filter((r) => r !== role));
    } else {
      props.onChange?.([...current, role]);
    }
  };

  return (
    <Space wrap>
      {allRoles.map((role) => (
        <Tag.CheckableTag
          key={role}
          checked={current.includes(role)}
          onChange={() => toggle(role)}
          style={current.includes(role) ? { borderColor: roleColorMap[role], color: roleColorMap[role] } : undefined}
        >
          {roleLabels[role]}
        </Tag.CheckableTag>
      ))}
    </Space>
  );
}
