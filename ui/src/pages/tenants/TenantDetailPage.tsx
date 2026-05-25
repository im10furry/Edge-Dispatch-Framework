import { useState } from 'react';
import { useParams } from 'react-router-dom';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  Tabs, Card, Descriptions, Button, Table, Modal, Form, Input, Tag, Space, Spin, App, Empty, Statistic, Row, Col,
} from 'antd';
import { PlusOutlined, EditOutlined, DeleteOutlined, TeamOutlined, ProjectOutlined, UserOutlined } from '@ant-design/icons';
import client from '../../api/client';
import { getApiErrorMessage } from '../../api/errors';
import type {
  Tenant, Project, User, UpdateTenantRequest, CreateProjectRequest, UpdateProjectRequest, UserRole, RoleBinding,
} from '../../types/api';
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

export default function TenantDetailPage() {
  const { tenantID } = useParams<{ tenantID: string }>();
  const queryClient = useQueryClient();
  const { message } = App.useApp();
  const [editOpen, setEditOpen] = useState(false);
  const [editForm] = Form.useForm();
  const [projectModalOpen, setProjectModalOpen] = useState(false);
  const [editingProject, setEditingProject] = useState<Project | null>(null);
  const [projectForm] = Form.useForm();
  const [editingUser, setEditingUser] = useState<User | null>(null);
  const [userForm] = Form.useForm();

  const { data: tenant, isLoading } = useQuery({
    queryKey: ['tenant', tenantID],
    queryFn: async () => {
      const res = await client.get<Tenant>(`/tenants/${tenantID}`);
      return res.data;
    },
    enabled: !!tenantID,
  });

  const { data: projects } = useQuery({
    queryKey: ['tenant-projects', tenantID],
    queryFn: async () => {
      const res = await client.get<Project[]>(`/tenants/${tenantID}/projects`);
      return res.data;
    },
    enabled: !!tenantID,
  });

  const { data: users } = useQuery({
    queryKey: ['tenant-users', tenantID],
    queryFn: async () => {
      const res = await client.get<User[]>(`/tenants/${tenantID}/users`);
      return res.data;
    },
    enabled: !!tenantID,
  });

  const updateTenantMutation = useMutation({
    mutationFn: (body: UpdateTenantRequest) => client.put(`/tenants/${tenantID}`, body),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['tenant', tenantID] });
      queryClient.invalidateQueries({ queryKey: ['tenants'] });
      message.success('租户更新成功');
      setEditOpen(false);
      editForm.resetFields();
    },
    onError: (err: unknown) => {
      message.error(getApiErrorMessage(err, '更新失败'));
    },
  });

  const createProjectMutation = useMutation({
    mutationFn: (body: CreateProjectRequest) => client.post(`/tenants/${tenantID}/projects`, body),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['tenant-projects', tenantID] });
      message.success('项目创建成功');
      setProjectModalOpen(false);
      projectForm.resetFields();
    },
    onError: (err: unknown) => {
      message.error(getApiErrorMessage(err, '创建失败'));
    },
  });

  const updateProjectMutation = useMutation({
    mutationFn: ({ id, body }: { id: string; body: UpdateProjectRequest }) =>
      client.put(`/projects/${id}`, body),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['tenant-projects', tenantID] });
      message.success('项目更新成功');
      setEditingProject(null);
      projectForm.resetFields();
    },
    onError: (err: unknown) => {
      message.error(getApiErrorMessage(err, '更新失败'));
    },
  });

  const deleteProjectMutation = useMutation({
    mutationFn: (id: string) => client.delete(`/projects/${id}`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['tenant-projects', tenantID] });
      message.success('项目已删除');
    },
    onError: (err: unknown) => {
      message.error(getApiErrorMessage(err, '删除失败'));
    },
  });

  const updateUserMutation = useMutation({
    mutationFn: ({ id, body }: { id: string; body: { display_name?: string; roles?: RoleBinding[] } }) =>
      client.put(`/users/${id}`, body),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['tenant-users', tenantID] });
      message.success('用户更新成功');
      setEditingUser(null);
      userForm.resetFields();
    },
    onError: (err: unknown) => {
      message.error(getApiErrorMessage(err, '更新失败'));
    },
  });

  const deleteUserMutation = useMutation({
    mutationFn: (id: string) => client.delete(`/users/${id}`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['tenant-users', tenantID] });
      message.success('用户已删除');
    },
    onError: (err: unknown) => {
      message.error(getApiErrorMessage(err, '删除失败'));
    },
  });

  const handleEditTenant = () => {
    editForm.validateFields().then((values) => {
      updateTenantMutation.mutate({ name: values.name, description: values.description });
    });
  };

  const handleCreateProject = () => {
    projectForm.validateFields().then((values) => {
      createProjectMutation.mutate({ name: values.name, description: values.description });
    });
  };

  const handleEditProject = () => {
    projectForm.validateFields().then((values) => {
      if (!editingProject) return;
      updateProjectMutation.mutate({ id: editingProject.project_id, body: { name: values.name, description: values.description } });
    });
  };

  const handleEditUser = () => {
    userForm.validateFields().then((values) => {
      if (!editingUser) return;
      const roles: RoleBinding[] = (values.roles || []).map((role: UserRole) => ({
        role,
        tenant_id: tenantID!,
        project_id: '',
      }));
      updateUserMutation.mutate({ id: editingUser.user_id, body: { display_name: values.display_name, roles } });
    });
  };

  const openEditModal = () => {
    if (!tenant) return;
    editForm.setFieldsValue({ name: tenant.name, description: tenant.description });
    setEditOpen(true);
  };

  const openProjectEdit = (project: Project) => {
    setEditingProject(project);
    projectForm.setFieldsValue({ name: project.name, description: project.description });
  };

  const openUserEdit = (user: User) => {
    setEditingUser(user);
    userForm.setFieldsValue({ display_name: user.display_name, roles: user.roles.map((r) => r.role) });
  };

  const confirmDeleteProject = (project: Project) => {
    Modal.confirm({
      title: '删除项目',
      content: `确定要删除项目 "${project.name}" 吗？此操作不可撤销。`,
      okText: '确认删除',
      cancelText: '取消',
      okButtonProps: { danger: true },
      onOk: () => deleteProjectMutation.mutate(project.project_id),
    });
  };

  const confirmDeleteUser = (user: User) => {
    Modal.confirm({
      title: '删除用户',
      content: `确定要删除用户 "${user.email}" 吗？此操作不可撤销。`,
      okText: '确认删除',
      cancelText: '取消',
      okButtonProps: { danger: true },
      onOk: () => deleteUserMutation.mutate(user.user_id),
    });
  };

  if (isLoading || !tenant) {
    return (
      <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: 300 }}>
        <Spin size="large" />
      </div>
    );
  }

  const projectColumns = [
    { title: '名称', dataIndex: 'name', key: 'name', width: 200 },
    { title: '描述', dataIndex: 'description', key: 'description', width: 300, render: (v: string) => v || '暂无描述' },
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
      render: (_: unknown, record: Project) => (
        <Space>
          <Button type="link" size="small" icon={<EditOutlined />} onClick={() => openProjectEdit(record)}>
            编辑
          </Button>
          <Button type="link" danger size="small" icon={<DeleteOutlined />} onClick={() => confirmDeleteProject(record)}>
            删除
          </Button>
        </Space>
      ),
    },
  ];

  const userColumns = [
    { title: '邮箱', dataIndex: 'email', key: 'email', width: 220 },
    { title: '显示名', dataIndex: 'display_name', key: 'display_name', width: 140, render: (v: string) => v || '-' },
    {
      title: '角色',
      dataIndex: 'roles',
      key: 'roles',
      width: 280,
      render: (roles: RoleBinding[]) => (
        <Space size={[0, 4]} wrap>
          {roles.map((r, i) => (
            <Tag key={i} color={roleColorMap[r.role]}>{roleLabels[r.role]}</Tag>
          ))}
        </Space>
      ),
    },
    {
      title: '操作',
      key: 'actions',
      width: 160,
      render: (_: unknown, record: User) => (
        <Space>
          <Button type="link" size="small" icon={<EditOutlined />} onClick={() => openUserEdit(record)}>
            编辑
          </Button>
          <Button type="link" danger size="small" icon={<DeleteOutlined />} onClick={() => confirmDeleteUser(record)}>
            移除
          </Button>
        </Space>
      ),
    },
  ];

  const tabItems = [
    {
      key: 'overview',
      label: '概览',
      children: (
        <div>
          <Card
            title={
              <Space>
                <TeamOutlined style={{ color: '#58a6ff' }} />
                <span style={{ color: '#e6edf3' }}>租户信息</span>
              </Space>
            }
            extra={<Button type="primary" icon={<EditOutlined />} onClick={openEditModal}>编辑</Button>}
            style={{ marginBottom: 24 }}
          >
            <Descriptions column={2} bordered size="small">
              <Descriptions.Item label="名称">{tenant.name}</Descriptions.Item>
              <Descriptions.Item label="租户 ID">{tenant.tenant_id}</Descriptions.Item>
              <Descriptions.Item label="描述" span={2}>{tenant.description || '暂无描述'}</Descriptions.Item>
              <Descriptions.Item label="创建时间">{dayjs(tenant.created_at).format('YYYY-MM-DD HH:mm:ss')}</Descriptions.Item>
            </Descriptions>
          </Card>

          <Row gutter={[16, 16]}>
            <Col xs={12} sm={6}>
              <Card>
                <Statistic title="项目数" value={projects?.length ?? 0} prefix={<ProjectOutlined />} />
              </Card>
            </Col>
            <Col xs={12} sm={6}>
              <Card>
                <Statistic title="用户数" value={users?.length ?? 0} prefix={<UserOutlined />} />
              </Card>
            </Col>
          </Row>
        </div>
      ),
    },
    {
      key: 'projects',
      label: `项目${projects ? ` (${projects.length})` : ''}`,
      children: (
        <div>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
            <h3 style={{ color: '#e6edf3', margin: 0, fontSize: 16 }}>项目列表</h3>
            <Button type="primary" icon={<PlusOutlined />} onClick={() => setProjectModalOpen(true)}>
              新建项目
            </Button>
          </div>
          {!projects || projects.length === 0 ? (
            <Empty description="暂无项目" />
          ) : (
            <Table
              rowKey="project_id"
              dataSource={projects}
              columns={projectColumns}
              pagination={false}
              size="middle"
            />
          )}
        </div>
      ),
    },
    {
      key: 'users',
      label: `用户${users ? ` (${users.length})` : ''}`,
      children: (
        <div>
          <h3 style={{ color: '#e6edf3', margin: '0 0 16px', fontSize: 16 }}>用户列表</h3>
          {!users || users.length === 0 ? (
            <Empty description="暂无用户" />
          ) : (
            <Table
              rowKey="user_id"
              dataSource={users}
              columns={userColumns}
              pagination={false}
              size="middle"
            />
          )}
        </div>
      ),
    },
  ];

  return (
    <div>
      <h2 style={{ color: '#e6edf3', margin: '0 0 24px', fontSize: 20, fontWeight: 600 }}>
        {tenant.name}
      </h2>
      <Tabs items={tabItems} />

      <Modal
        title="编辑租户"
        open={editOpen}
        onCancel={() => { setEditOpen(false); editForm.resetFields(); }}
        onOk={handleEditTenant}
        confirmLoading={updateTenantMutation.isPending}
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
        title="新建项目"
        open={projectModalOpen}
        onCancel={() => { setProjectModalOpen(false); projectForm.resetFields(); }}
        onOk={handleCreateProject}
        confirmLoading={createProjectMutation.isPending}
        okText="创建"
        cancelText="取消"
      >
        <Form form={projectForm} layout="vertical" style={{ marginTop: 16 }}>
          <Form.Item name="name" label="名称" rules={[{ required: true, message: '请输入项目名称' }]}>
            <Input placeholder="请输入项目名称" />
          </Form.Item>
          <Form.Item name="description" label="描述">
            <Input.TextArea rows={3} placeholder="请输入项目描述（选填）" />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title="编辑项目"
        open={!!editingProject}
        onCancel={() => { setEditingProject(null); projectForm.resetFields(); }}
        onOk={handleEditProject}
        confirmLoading={updateProjectMutation.isPending}
        okText="保存"
        cancelText="取消"
      >
        <Form form={projectForm} layout="vertical" style={{ marginTop: 16 }}>
          <Form.Item name="name" label="名称" rules={[{ required: true, message: '请输入项目名称' }]}>
            <Input placeholder="请输入项目名称" />
          </Form.Item>
          <Form.Item name="description" label="描述">
            <Input.TextArea rows={3} placeholder="请输入项目描述（选填）" />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title="编辑用户"
        open={!!editingUser}
        onCancel={() => { setEditingUser(null); userForm.resetFields(); }}
        onOk={handleEditUser}
        confirmLoading={updateUserMutation.isPending}
        okText="保存"
        cancelText="取消"
      >
        <Form form={userForm} layout="vertical" style={{ marginTop: 16 }}>
          <Form.Item name="display_name" label="显示名">
            <Input placeholder="请输入显示名" />
          </Form.Item>
          <Form.Item name="roles" label="角色">
            <Form.Item noStyle name="roles">
              <TagSelect />
            </Form.Item>
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}

function TagSelect(props: { value?: UserRole[]; onChange?: (v: UserRole[]) => void }) {
  const roles: UserRole[] = ['tenant_owner', 'tenant_admin', 'project_operator', 'project_viewer'];
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
      {roles.map((role) => (
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
