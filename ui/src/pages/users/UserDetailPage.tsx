import { useState } from 'react';
import { useParams } from 'react-router-dom';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Card, Descriptions, Button, Tag, Space, Modal, Form, Input, Spin, App } from 'antd';
import { EditOutlined, MailOutlined, ClockCircleOutlined } from '@ant-design/icons';
import client from '../../api/client';
import { getApiErrorMessage } from '../../api/errors';
import type { User, UserRole, RoleBinding } from '../../types/api';
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

export default function UserDetailPage() {
  const { userID } = useParams<{ userID: string }>();
  const queryClient = useQueryClient();
  const { message } = App.useApp();
  const [editOpen, setEditOpen] = useState(false);
  const [form] = Form.useForm();

  const { data: user, isLoading } = useQuery({
    queryKey: ['user', userID],
    queryFn: async () => {
      const res = await client.get<User>(`/users/${userID}`);
      return res.data;
    },
    enabled: !!userID,
  });

  const updateMutation = useMutation({
    mutationFn: ({ id, body }: { id: string; body: { display_name?: string; roles?: RoleBinding[] } }) =>
      client.put(`/users/${id}`, body),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['user', userID] });
      queryClient.invalidateQueries({ queryKey: ['users'] });
      message.success('用户更新成功');
      setEditOpen(false);
      form.resetFields();
    },
    onError: (err: unknown) => {
      message.error(getApiErrorMessage(err, '更新失败'));
    },
  });

  const handleEdit = () => {
    form.validateFields().then((values) => {
      if (!user) return;
      const roles: RoleBinding[] = (values.roles || []).map((role: UserRole) => ({
        role,
        tenant_id: user.tenant_id || '',
        project_id: '',
      }));
      const body: { display_name?: string; roles?: RoleBinding[] } = { display_name: values.display_name, roles };
      updateMutation.mutate({ id: user.user_id, body });
    });
  };

  const openEdit = () => {
    if (!user) return;
    form.setFieldsValue({
      display_name: user.display_name,
      roles: user.roles?.map((r) => r.role) || [],
    });
    setEditOpen(true);
  };

  if (isLoading || !user) {
    return (
      <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: 300 }}>
        <Spin size="large" />
      </div>
    );
  }

  return (
    <div>
      <h2 style={{ color: '#e6edf3', margin: '0 0 24px', fontSize: 20, fontWeight: 600 }}>
        用户详情
      </h2>

      <Card
        title={
          <Space>
            <MailOutlined style={{ color: '#58a6ff' }} />
            <span style={{ color: '#e6edf3' }}>用户信息</span>
          </Space>
        }
        extra={<Button type="primary" icon={<EditOutlined />} onClick={openEdit}>编辑</Button>}
        style={{ marginBottom: 24 }}
      >
        <Descriptions column={2} bordered size="small">
          <Descriptions.Item label="邮箱">{user.email}</Descriptions.Item>
          <Descriptions.Item label="显示名">{user.display_name || '-'}</Descriptions.Item>
          <Descriptions.Item label="用户 ID">{user.user_id}</Descriptions.Item>
          <Descriptions.Item label="所属租户">{user.tenant_id || '-'}</Descriptions.Item>
          <Descriptions.Item label="创建时间">
            <Space>
              <ClockCircleOutlined style={{ color: '#8b949e' }} />
              {dayjs(user.created_at).format('YYYY-MM-DD HH:mm:ss')}
            </Space>
          </Descriptions.Item>
          <Descriptions.Item label="更新时间">
            {dayjs(user.updated_at).format('YYYY-MM-DD HH:mm:ss')}
          </Descriptions.Item>
        </Descriptions>
      </Card>

      <Card
        title={
          <Space>
            <span style={{ color: '#e6edf3' }}>角色绑定</span>
          </Space>
        }
      >
        {!user.roles || user.roles.length === 0 ? (
          <span style={{ color: '#8b949e' }}>暂无角色</span>
        ) : (
          <Space size={[0, 8]} wrap>
            {user.roles.map((r, i) => (
              <Tag key={i} color={roleColorMap[r.role]} style={{ fontSize: 13, padding: '2px 10px' }}>
                {roleLabels[r.role]}
              </Tag>
            ))}
          </Space>
        )}
      </Card>

      <Modal
        title="编辑用户"
        open={editOpen}
        onCancel={() => { setEditOpen(false); form.resetFields(); }}
        onOk={handleEdit}
        confirmLoading={updateMutation.isPending}
        okText="保存"
        cancelText="取消"
        width={520}
      >
        <Form form={form} layout="vertical" style={{ marginTop: 16 }}>
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
