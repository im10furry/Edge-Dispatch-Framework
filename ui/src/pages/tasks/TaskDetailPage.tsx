import {
  Card, Tag, Button, Typography, Descriptions, Spin, App, Progress, Timeline, Modal,
} from 'antd';
import { ArrowLeftOutlined, CloseCircleOutlined } from '@ant-design/icons';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useParams, useNavigate } from 'react-router-dom';
import { useState } from 'react';
import client from '../../api/client';
import type { Task, TaskType, TaskStatus } from '../../types/api';

const { Text, Title } = Typography;

const TYPE_LABELS: Record<TaskType, string> = { prewarm: '预取', purge: '清除', block: '封禁' };
const TYPE_COLORS: Record<TaskType, string> = { prewarm: 'blue', purge: 'orange', block: 'red' };
const STATUS_LABELS: Record<TaskStatus, string> = {
  pending: '等待中', running: '运行中', completed: '已完成', failed: '失败', cancelled: '已取消',
};
const STATUS_COLORS: Record<TaskStatus, string> = {
  pending: 'default', running: 'processing', completed: 'green', failed: 'red', cancelled: 'default',
};

export default function TaskDetailPage() {
  const { taskID } = useParams<{ taskID: string }>();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const { message } = App.useApp();
  const [cancelOpen, setCancelOpen] = useState(false);

  const { data: task, isLoading } = useQuery({
    queryKey: ['task', taskID],
    queryFn: async () => {
      const res = await client.get<Task>(`/tasks/${taskID}`);
      return res.data;
    },
    enabled: !!taskID,
    refetchInterval: (query) =>
      query.state.data?.status === 'pending' || query.state.data?.status === 'running'
        ? 3000
        : false,
  });

  const cancelMutation = useMutation({
    mutationFn: () => client.post(`/tasks/${taskID}:cancel`),
    onSuccess: () => {
      message.success('任务已取消');
      setCancelOpen(false);
      queryClient.invalidateQueries({ queryKey: ['task', taskID] });
      queryClient.invalidateQueries({ queryKey: ['tasks'] });
    },
    onError: () => message.error('取消失败'),
  });

  if (isLoading) {
    return <div style={{ display: 'flex', justifyContent: 'center', paddingTop: 120 }}><Spin size="large" /></div>;
  }

  if (!task) {
    return <Text type="secondary">任务不存在</Text>;
  }

  const percent = task.total_nodes > 0 ? Math.round((task.done_nodes / task.total_nodes) * 100) : 0;
  const isRunning = task.status === 'pending' || task.status === 'running';

  const paramsStr = task.params
    ? (typeof task.params === 'string' ? task.params : JSON.stringify(task.params, null, 2))
    : '-';

  const resultStr = task.result
    ? (typeof task.result === 'string' ? task.result : JSON.stringify(task.result, null, 2))
    : null;

  const timelineItems: { color: string; children: React.ReactNode }[] = [];

  timelineItems.push({
    color: 'blue',
    children: (
      <>
        <Text strong>任务创建</Text>
        <br />
        <Text type="secondary" style={{ fontSize: 12 }}>{new Date(task.created_at).toLocaleString('zh-CN')}</Text>
      </>
    ),
  });

  if (task.status !== 'pending') {
    timelineItems.push({
      color: 'blue',
      children: (
        <>
          <Text strong>开始执行</Text>
          <br />
          <Text type="secondary" style={{ fontSize: 12 }}>{new Date(task.updated_at).toLocaleString('zh-CN')}</Text>
        </>
      ),
    });
  }

  if (task.status === 'completed') {
    timelineItems.push({
      color: 'green',
      children: (
        <>
          <Text strong>任务完成</Text>
          <br />
          <Text type="secondary" style={{ fontSize: 12 }}>{new Date(task.updated_at).toLocaleString('zh-CN')}</Text>
        </>
      ),
    });
  } else if (task.status === 'failed') {
    timelineItems.push({
      color: 'red',
      children: (
        <>
          <Text strong>任务失败</Text>
          <br />
          <Text type="secondary" style={{ fontSize: 12 }}>{new Date(task.updated_at).toLocaleString('zh-CN')}</Text>
        </>
      ),
    });
  } else if (task.status === 'cancelled') {
    timelineItems.push({
      color: 'gray',
      children: (
        <>
          <Text strong>任务取消</Text>
          <br />
          <Text type="secondary" style={{ fontSize: 12 }}>{new Date(task.updated_at).toLocaleString('zh-CN')}</Text>
        </>
      ),
    });
  }

  return (
    <>
      <div style={{ marginBottom: 16 }}>
        <Button type="text" icon={<ArrowLeftOutlined />} onClick={() => navigate('/tasks')}>
          返回列表
        </Button>
      </div>

      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 20 }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
          <Title level={4} style={{ margin: 0, fontFamily: 'monospace' }}>{task.task_id.slice(0, 16)}...</Title>
          <Tag color={TYPE_COLORS[task.type]}>{TYPE_LABELS[task.type]}</Tag>
          <Tag color={STATUS_COLORS[task.status]}>{STATUS_LABELS[task.status]}</Tag>
        </div>
        {isRunning && (
          <Button
            danger
            icon={<CloseCircleOutlined />}
            onClick={() => setCancelOpen(true)}
          >
            取消任务
          </Button>
        )}
      </div>

      <Card style={{ marginBottom: 20 }}>
        <div style={{ marginBottom: 16 }}>
          <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 4 }}>
            <Text strong>执行进度</Text>
            <Text>{task.done_nodes} / {task.total_nodes} 节点</Text>
          </div>
          <Progress
            percent={percent}
            status={
              task.status === 'failed' ? 'exception'
              : task.status === 'completed' ? 'success'
              : isRunning ? 'active'
              : 'normal'
            }
          />
        </div>
      </Card>

      <Card title="任务信息" style={{ marginBottom: 20 }}>
        <Descriptions column={2} size="small">
          <Descriptions.Item label="任务 ID" span={2}>
            <Text copyable style={{ fontFamily: 'monospace' }}>{task.task_id}</Text>
          </Descriptions.Item>
          <Descriptions.Item label="类型">
            <Tag color={TYPE_COLORS[task.type]}>{TYPE_LABELS[task.type]}</Tag>
          </Descriptions.Item>
          <Descriptions.Item label="状态">
            <Tag color={STATUS_COLORS[task.status]}>{STATUS_LABELS[task.status]}</Tag>
          </Descriptions.Item>
          <Descriptions.Item label="租户 ID">{task.tenant_id}</Descriptions.Item>
          <Descriptions.Item label="项目 ID">{task.project_id}</Descriptions.Item>
          <Descriptions.Item label="创建者">{task.creator_id}</Descriptions.Item>
          <Descriptions.Item label="已完成节点">{task.done_nodes}</Descriptions.Item>
          <Descriptions.Item label="总节点数">{task.total_nodes}</Descriptions.Item>
          <Descriptions.Item label="创建时间">
            {new Date(task.created_at).toLocaleString('zh-CN')}
          </Descriptions.Item>
          <Descriptions.Item label="更新时间">
            {new Date(task.updated_at).toLocaleString('zh-CN')}
          </Descriptions.Item>
        </Descriptions>
      </Card>

      <Card title="任务参数" style={{ marginBottom: 20 }}>
        <pre style={{
          background: '#0a0e14',
          color: '#e6edf3',
          padding: 16,
          borderRadius: 6,
          fontFamily: '"Fira Code", "Cascadia Code", Consolas, monospace',
          fontSize: 13,
          lineHeight: 1.6,
          overflow: 'auto',
          maxHeight: 400,
          margin: 0,
          border: '1px solid #30363d',
        }}>
          <code>{paramsStr}</code>
        </pre>
      </Card>

      {resultStr && (
        <Card title="执行结果" style={{ marginBottom: 20 }}>
          <pre style={{
            background: '#0a0e14',
            color: '#e6edf3',
            padding: 16,
            borderRadius: 6,
            fontFamily: '"Fira Code", "Cascadia Code", Consolas, monospace',
            fontSize: 13,
            lineHeight: 1.6,
            overflow: 'auto',
            maxHeight: 400,
            margin: 0,
            border: '1px solid #30363d',
          }}>
            <code>{resultStr}</code>
          </pre>
        </Card>
      )}

      <Card title="事件时间线">
        <Timeline items={timelineItems} />
      </Card>

      {/* Cancel Modal */}
      <Modal
        title="确认取消任务"
        open={cancelOpen}
        onCancel={() => setCancelOpen(false)}
        onOk={() => cancelMutation.mutate()}
        confirmLoading={cancelMutation.isPending}
        okButtonProps={{ danger: true }}
        okText="确认取消"
      >
        <p>确认取消当前任务？取消后无法恢复。</p>
      </Modal>
    </>
  );
}
