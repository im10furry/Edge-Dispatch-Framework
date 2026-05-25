import { useState, useMemo } from 'react';
import {
  Table, Button, Space, Tag, Modal, Form, Input, Select, App, Typography, Tooltip, Progress, Dropdown,
} from 'antd';
import {
  PlusOutlined, CopyOutlined, CloseCircleOutlined, InfoCircleOutlined, FireOutlined, DeleteOutlined, StopOutlined,
} from '@ant-design/icons';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useNavigate } from 'react-router-dom';
import type { ColumnsType } from 'antd/es/table';
import type { MenuProps } from 'antd';
import client from '../../api/client';
import type { Task, TaskType, TaskStatus, PaginatedResponse, Node } from '../../types/api';

const { Text, Paragraph } = Typography;

const TYPE_LABELS: Record<TaskType, string> = { prewarm: '预取', purge: '清除', block: '封禁' };
const TYPE_COLORS: Record<TaskType, string> = { prewarm: 'blue', purge: 'orange', block: 'red' };
const STATUS_LABELS: Record<TaskStatus, string> = {
  pending: '等待中', running: '运行中', completed: '已完成', failed: '失败', cancelled: '已取消',
};
const STATUS_COLORS: Record<TaskStatus, string> = {
  pending: 'default', running: 'processing', completed: 'green', failed: 'red', cancelled: 'default',
};

const SCOPE_OPTIONS = [
  { label: '全部节点', value: 'all' },
  { label: '按区域', value: 'region' },
  { label: '指定节点', value: 'node_list' },
];

export default function TaskListPage() {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const { message } = App.useApp();
  const [modalType, setModalType] = useState<TaskType | null>(null);
  const [form] = Form.useForm();

  const { data, isLoading } = useQuery({
    queryKey: ['tasks'],
    queryFn: async () => {
      const res = await client.get<PaginatedResponse<Task>>('/tasks', {
        params: { limit: 100, offset: 0 },
      });
      return res.data;
    },
    refetchInterval: 5000,
  });

  const { data: nodesData } = useQuery({
    queryKey: ['nodesForTask'],
    queryFn: async () => {
      const res = await client.get<PaginatedResponse<Node>>('/nodes', {
        params: { limit: 200, offset: 0 },
      });
      return res.data;
    },
  });

  const hasRunningTasks = useMemo(
    () => (data?.data || []).some((t) => t.status === 'running' || t.status === 'pending'),
    [data],
  );

  const createMutation = useMutation({
    mutationFn: ({ type, body }: { type: TaskType; body: Record<string, unknown> }) => {
      const endpoints: Record<TaskType, string> = {
        prewarm: '/cache/prewarm',
        purge: '/cache/purge',
        block: '/objects/block',
      };
      return client.post(endpoints[type], body);
    },
    onSuccess: () => {
      message.success('任务创建成功');
      setModalType(null);
      form.resetFields();
      queryClient.invalidateQueries({ queryKey: ['tasks'] });
    },
    onError: () => message.error('任务创建失败'),
  });

  const cancelMutation = useMutation({
    mutationFn: (id: string) => client.post(`/tasks/${id}:cancel`),
    onSuccess: () => {
      message.success('任务已取消');
      queryClient.invalidateQueries({ queryKey: ['tasks'] });
    },
    onError: () => message.error('取消失败'),
  });

  const handleCopy = (text: string) => {
    navigator.clipboard.writeText(text).then(
      () => message.success('已复制到剪贴板'),
      () => message.error('复制失败'),
    );
  };

  const columns: ColumnsType<Task> = [
    {
      title: '任务 ID',
      dataIndex: 'task_id',
      width: 160,
      render: (v: string) => (
        <Space size={4}>
          <Text style={{ fontSize: 12, fontFamily: 'monospace' }}>{v.slice(0, 12)}...</Text>
          <Tooltip title="复制">
            <Button type="text" size="small" icon={<CopyOutlined />} onClick={(e) => { e.stopPropagation(); handleCopy(v); }} />
          </Tooltip>
        </Space>
      ),
    },
    {
      title: '类型',
      dataIndex: 'type',
      width: 80,
      render: (t: TaskType) => <Tag color={TYPE_COLORS[t]}>{TYPE_LABELS[t]}</Tag>,
    },
    {
      title: '对象',
      key: 'object',
      width: 200,
      ellipsis: true,
      render: (_, r) => {
        const p = r.params as { object_key?: string; key_prefix?: string } | null;
        return p?.object_key || p?.key_prefix || '-';
      },
    },
    {
      title: '进度',
      key: 'progress',
      width: 160,
      render: (_, r) => (
        <Space direction="vertical" size={0} style={{ width: '100%' }}>
          <Progress
            percent={r.total_nodes > 0 ? Math.round((r.done_nodes / r.total_nodes) * 100) : 0}
            size="small"
            status={r.status === 'failed' ? 'exception' : r.status === 'completed' ? 'success' : 'active'}
          />
          <Text type="secondary" style={{ fontSize: 11 }}>{r.done_nodes} / {r.total_nodes}</Text>
        </Space>
      ),
    },
    {
      title: '状态',
      dataIndex: 'status',
      width: 90,
      align: 'center',
      render: (s: TaskStatus) => <Tag color={STATUS_COLORS[s]}>{STATUS_LABELS[s]}</Tag>,
    },
    {
      title: '创建者',
      dataIndex: 'creator_id',
      width: 120,
      ellipsis: true,
    },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      width: 160,
      render: (v: string) => new Date(v).toLocaleString('zh-CN'),
    },
    {
      title: '操作',
      key: 'actions',
      width: 140,
      render: (_, r) => (
        <Space size={4}>
          <Tooltip title="详情">
            <Button type="link" size="small" icon={<InfoCircleOutlined />} onClick={(e) => { e.stopPropagation(); navigate(`/tasks/${r.task_id}`); }}>
              详情
            </Button>
          </Tooltip>
          {(r.status === 'pending' || r.status === 'running') && (
            <Tooltip title="取消">
              <Button
                type="link"
                size="small"
                danger
                icon={<CloseCircleOutlined />}
                onClick={(e) => { e.stopPropagation(); cancelMutation.mutate(r.task_id); }}
              >
                取消
              </Button>
            </Tooltip>
          )}
        </Space>
      ),
    },
  ];

  const newTaskItems: MenuProps['items'] = [
    { key: 'prewarm', icon: <FireOutlined />, label: '预取任务' },
    { key: 'purge', icon: <DeleteOutlined />, label: '清除任务' },
    { key: 'block', icon: <StopOutlined />, label: '封禁任务' },
  ];

  const handleNewTask: MenuProps['onClick'] = ({ key }) => {
    form.resetFields();
    setModalType(key as TaskType);
  };

  const renderTaskForm = () => (
    <Form form={form} layout="vertical">
      <Form.Item name="target_scope" label="目标范围" initialValue="all">
        <Select options={SCOPE_OPTIONS} />
      </Form.Item>

      <Form.Item
        noStyle
        shouldUpdate={(prev, cur) => prev.target_scope !== cur.target_scope}
      >
        {({ getFieldValue }) => {
          const scope = getFieldValue('target_scope');
          return (
            <>
              {scope === 'node_list' && (
                <Form.Item name="target_nodes" label="目标节点" rules={[{ required: true, message: '请选择目标节点' }]}>
                  <Select
                    mode="multiple"
                    placeholder="选择节点"
                    options={(nodesData?.data || []).map((n) => ({ label: n.name, value: n.node_id }))}
                    filterOption={(input, option) =>
                      (option?.label as string)?.toLowerCase().includes(input.toLowerCase())
                    }
                    style={{ width: '100%' }}
                  />
                </Form.Item>
              )}

              {scope === 'region' && (
                <Form.Item name="region_key_prefix" label="目标区域" rules={[{ required: true, message: '请输入区域' }]}>
                  <Input placeholder="输入区域标识" />
                </Form.Item>
              )}

              {(modalType === 'prewarm' || modalType === 'purge') && (
                <>
                  <Form.Item name="object_key" label="对象 Key">
                    <Input placeholder="指定具体对象 Key" />
                  </Form.Item>
                  <Form.Item name="key_prefix" label="Key 前缀">
                    <Input placeholder="按前缀匹配" />
                  </Form.Item>
                  <Form.Item name="concurrency" label="并发数">
                    <Input type="number" placeholder="默认自动" style={{ width: 160 }} />
                  </Form.Item>
                  <Form.Item name="rate_limit" label="速率限制 (req/s)">
                    <Input type="number" placeholder="默认自动" style={{ width: 160 }} />
                  </Form.Item>
                </>
              )}

              {modalType === 'block' && (
                <Form.Item name="object_key" label="对象 Key" rules={[{ required: true, message: '请输入对象 Key' }]}>
                  <Input placeholder="要封禁的对象 Key" />
                </Form.Item>
              )}
            </>
          );
        }}
      </Form.Item>
    </Form>
  );

  return (
    <>
      <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 16 }}>
        <Typography.Title level={4} style={{ margin: 0 }}>缓存任务</Typography.Title>
        <Dropdown menu={{ items: newTaskItems, onClick: handleNewTask }}>
          <Button type="primary" icon={<PlusOutlined />}>
            新建任务
          </Button>
        </Dropdown>
      </div>

      {hasRunningTasks && (
        <Paragraph type="secondary" style={{ marginBottom: 12, fontSize: 12 }}>
          检测到运行中的任务，数据每 5 秒自动刷新。
        </Paragraph>
      )}

      <Table
        rowKey="task_id"
        columns={columns}
        dataSource={data?.data}
        loading={isLoading}
        pagination={{ pageSize: 20, showSizeChanger: true }}
        onRow={(r) => ({ onClick: () => navigate(`/tasks/${r.task_id}`), style: { cursor: 'pointer' } })}
      />

      {/* Create Task Modal */}
      <Modal
        title={
          modalType === 'prewarm' ? '新建预取任务'
          : modalType === 'purge' ? '新建清除任务'
          : '新建封禁任务'
        }
        open={!!modalType}
        onCancel={() => setModalType(null)}
        onOk={async () => {
        if (!modalType) return;
        try {
          const vals = await form.validateFields();
          const body: Record<string, unknown> = { target_scope: vals.target_scope };

          if (vals.target_scope === 'node_list') {
            body.target_nodes = vals.target_nodes;
          }
          if (vals.object_key) body.object_key = vals.object_key;
          if (vals.key_prefix) body.key_prefix = vals.key_prefix;
          if (vals.region_key_prefix) body.key_prefix = vals.region_key_prefix;
          if (vals.concurrency !== undefined && vals.concurrency !== '') {
            body.concurrency = Number(vals.concurrency);
          }
          if (vals.rate_limit !== undefined && vals.rate_limit !== '') {
            body.rate_limit = Number(vals.rate_limit);
          }

          createMutation.mutate({ type: modalType, body });
        } catch {
          // validation failed
        }
      }}
        confirmLoading={createMutation.isPending}
        width={600}
        destroyOnClose
      >
        {renderTaskForm()}
      </Modal>
    </>
  );
}
