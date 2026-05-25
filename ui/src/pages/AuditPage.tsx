import { useState, useCallback } from 'react';
import {
  Card,
  Table,
  Input,
  Select,
  DatePicker,
  Button,
  Space,
  Tag,
  Modal,
  Tooltip,
  App,
} from 'antd';
import {
  SearchOutlined,
  CopyOutlined,
  FileTextOutlined,
  ExportOutlined,
  ReloadOutlined,
} from '@ant-design/icons';
import { useQuery } from '@tanstack/react-query';
import type { ColumnsType } from 'antd/es/table';
import dayjs from 'dayjs';
import client from '../api/client';
import type { AuditEvent, PaginatedResponse } from '../types/api';

const { RangePicker } = DatePicker;

interface AuditFilters {
  actor_id: string;
  action: string;
  resource_type: string;
  result: string;
  dateRange: [dayjs.Dayjs, dayjs.Dayjs] | null;
  limit: number;
  offset: number;
}

function fetchAudit(filters: AuditFilters) {
  const params = new URLSearchParams();
  if (filters.actor_id) params.set('actor_id', filters.actor_id);
  if (filters.action) params.set('action', filters.action);
  if (filters.resource_type) params.set('resource_type', filters.resource_type);
  if (filters.result) params.set('result', filters.result);
  if (filters.dateRange) {
    params.set('from', filters.dateRange[0].toISOString());
    params.set('to', filters.dateRange[1].toISOString());
  }
  params.set('limit', String(filters.limit));
  params.set('offset', String(filters.offset));

  return client.get<PaginatedResponse<AuditEvent>>('/audit', { params }).then((r) => r.data);
}

const resultColorMap: Record<string, string> = {
  success: 'green',
  failure: 'red',
  error: 'red',
  denied: 'orange',
};

function formatJson(obj: unknown) {
  try {
    return JSON.stringify(obj, null, 2);
  } catch {
    return String(obj ?? '');
  }
}

export default function AuditPage() {
  const { message } = App.useApp();
  const [filters, setFilters] = useState<AuditFilters>({
    actor_id: '',
    action: '',
    resource_type: '',
    result: '',
    dateRange: null,
    limit: 20,
    offset: 0,
  });
  const [diffModalOpen, setDiffModalOpen] = useState(false);
  const [diffEntry, setDiffEntry] = useState<AuditEvent | null>(null);

  const { data, isLoading, refetch } = useQuery({
    queryKey: ['audit', filters],
    queryFn: () => fetchAudit(filters),
    placeholderData: (prev) => prev,
  });

  const handleCopy = useCallback(
    (text: string) => {
      navigator.clipboard.writeText(text).then(
        () => message.success('已复制'),
        () => message.error('复制失败'),
      );
    },
    [message],
  );

  const handleExport = useCallback(
    async (format: 'json' | 'csv') => {
      try {
        const params = new URLSearchParams();
        if (filters.actor_id) params.set('actor_id', filters.actor_id);
        if (filters.action) params.set('action', filters.action);
        if (filters.resource_type) params.set('resource_type', filters.resource_type);
        if (filters.result) params.set('result', filters.result);
        if (filters.dateRange) {
          params.set('from', filters.dateRange[0].toISOString());
          params.set('to', filters.dateRange[1].toISOString());
        }
        params.set('format', format);

        const res = await client.get('/audit/export', {
          params,
          responseType: 'blob',
        });

        const ext = format === 'csv' ? 'csv' : 'json';
        const mimeType = format === 'csv' ? 'text/csv' : 'application/json';
        const blob = new Blob([res.data], { type: mimeType });
        const url = URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = `audit_export_${dayjs().format('YYYYMMDD_HHmmss')}.${ext}`;
        a.click();
        URL.revokeObjectURL(url);
        message.success(`已导出 ${format.toUpperCase()}`);
      } catch {
        message.error('导出失败');
      }
    },
    [filters, message],
  );

  const columns: ColumnsType<AuditEvent> = [
    {
      title: '时间',
      dataIndex: 'created_at',
      key: 'created_at',
      width: 180,
      render: (v: string) => dayjs(v).format('YYYY-MM-DD HH:mm:ss'),
    },
    {
      title: '操作人',
      dataIndex: 'actor_email',
      key: 'actor_email',
      width: 200,
      ellipsis: true,
    },
    {
      title: '操作',
      key: 'action',
      width: 260,
      render: (_, record) => (
        <Space size={4} wrap>
          <Tag color="blue">{record.action}</Tag>
          <span style={{ color: '#8b949e', fontSize: 12 }}>
            {record.resource_type}/{record.resource_id}
          </span>
        </Space>
      ),
    },
    {
      title: '变更',
      key: 'diff',
      width: 100,
      render: (_, record) => (
        <Button
          type="link"
          size="small"
          icon={<FileTextOutlined />}
          onClick={() => {
            setDiffEntry(record);
            setDiffModalOpen(true);
          }}
        >
          查看变更
        </Button>
      ),
    },
    {
      title: '来源 IP',
      dataIndex: 'source_ip',
      key: 'source_ip',
      width: 140,
    },
    {
      title: '结果',
      dataIndex: 'result',
      key: 'result',
      width: 80,
      render: (v: string) => <Tag color={resultColorMap[v] || 'default'}>{v}</Tag>,
    },
    {
      title: '请求 ID',
      dataIndex: 'request_id',
      key: 'request_id',
      width: 180,
      render: (v: string) => (
        <Space>
          <Tooltip title={v}>
            <span style={{ fontFamily: 'monospace', fontSize: 12 }}>
              {v.slice(0, 8)}...
            </span>
          </Tooltip>
          <Button
            type="text"
            size="small"
            icon={<CopyOutlined />}
            onClick={() => handleCopy(v)}
          />
        </Space>
      ),
    },
  ];

  return (
    <div>
      <Card
        size="small"
        style={{ marginBottom: 16 }}
        styles={{ body: { paddingBottom: 0 } }}
      >
        <Space wrap size={[8, 8]}>
          <Input
            placeholder="操作人邮箱"
            prefix={<SearchOutlined />}
            allowClear
            style={{ width: 200 }}
            value={filters.actor_id}
            onChange={(e) =>
              setFilters((f) => ({ ...f, actor_id: e.target.value, offset: 0 }))
            }
          />
          <Select
            placeholder="操作类型"
            allowClear
            style={{ width: 140 }}
            value={filters.action || undefined}
            onChange={(v) =>
              setFilters((f) => ({ ...f, action: v || '', offset: 0 }))
            }
            options={[
              { value: 'create', label: '创建' },
              { value: 'update', label: '更新' },
              { value: 'delete', label: '删除' },
              { value: 'login', label: '登录' },
              { value: 'logout', label: '登出' },
              { value: 'disable', label: '禁用' },
              { value: 'enable', label: '启用' },
            ]}
          />
          <Select
            placeholder="资源类型"
            allowClear
            style={{ width: 140 }}
            value={filters.resource_type || undefined}
            onChange={(v) =>
              setFilters((f) => ({ ...f, resource_type: v || '', offset: 0 }))
            }
            options={[
              { value: 'node', label: '节点' },
              { value: 'tenant', label: '租户' },
              { value: 'user', label: '用户' },
              { value: 'policy', label: '策略' },
              { value: 'ingress', label: '入口' },
              { value: 'task', label: '任务' },
              { value: 'settings', label: '设置' },
            ]}
          />
          <Select
            placeholder="结果"
            allowClear
            style={{ width: 120 }}
            value={filters.result || undefined}
            onChange={(v) =>
              setFilters((f) => ({ ...f, result: v || '', offset: 0 }))
            }
            options={[
              { value: 'success', label: '成功' },
              { value: 'failure', label: '失败' },
            ]}
          />
          <RangePicker
            showTime
            value={filters.dateRange as [dayjs.Dayjs, dayjs.Dayjs] | null}
            onChange={(dates) =>
              setFilters((f) => ({
                ...f,
                dateRange: dates as [dayjs.Dayjs, dayjs.Dayjs] | null,
                offset: 0,
              }))
            }
            format="YYYY-MM-DD HH:mm"
            placeholder={['开始时间', '结束时间']}
          />
          <Button icon={<ReloadOutlined />} onClick={() => refetch()}>
            刷新
          </Button>
        </Space>
      </Card>

      <Card size="small" styles={{ body: { padding: 0 } }}>
        <Table<AuditEvent>
          rowKey="id"
          columns={columns}
          dataSource={data?.data ?? []}
          loading={isLoading}
          scroll={{ x: 1200 }}
          pagination={{
            current: Math.floor(filters.offset / filters.limit) + 1,
            pageSize: filters.limit,
            total: data?.total ?? 0,
            showSizeChanger: true,
            showTotal: (total) => `共 ${total} 条`,
            onChange: (page, pageSize) =>
              setFilters((f) => ({
                ...f,
                limit: pageSize,
                offset: (page - 1) * pageSize,
              })),
          }}
        />
      </Card>

      <div style={{ marginTop: 16, display: 'flex', gap: 12 }}>
        <Button icon={<ExportOutlined />} onClick={() => handleExport('json')}>
          导出 JSON
        </Button>
        <Button icon={<ExportOutlined />} onClick={() => handleExport('csv')}>
          导出 CSV
        </Button>
      </div>

      <Modal
        title="变更详情"
        open={diffModalOpen}
        onCancel={() => setDiffModalOpen(false)}
        footer={null}
        width={900}
        styles={{ body: { maxHeight: '60vh', overflow: 'auto' } }}
      >
        {diffEntry && (
          <div style={{ display: 'flex', gap: 16 }}>
            <div style={{ flex: 1, minWidth: 0 }}>
              <div style={{ color: '#f85149', fontWeight: 600, marginBottom: 8 }}>变更前</div>
              <pre
                style={{
                  background: '#0a0e14',
                  border: '1px solid #30363d',
                  borderRadius: 6,
                  padding: 12,
                  fontSize: 12,
                  color: '#e6edf3',
                  whiteSpace: 'pre-wrap',
                  wordBreak: 'break-all',
                  margin: 0,
                }}
              >
                {formatJson(diffEntry.before)}
              </pre>
            </div>
            <div style={{ flex: 1, minWidth: 0 }}>
              <div style={{ color: '#3fb950', fontWeight: 600, marginBottom: 8 }}>变更后</div>
              <pre
                style={{
                  background: '#0a0e14',
                  border: '1px solid #30363d',
                  borderRadius: 6,
                  padding: 12,
                  fontSize: 12,
                  color: '#e6edf3',
                  whiteSpace: 'pre-wrap',
                  wordBreak: 'break-all',
                  margin: 0,
                }}
              >
                {formatJson(diffEntry.after)}
              </pre>
            </div>
          </div>
        )}
      </Modal>
    </div>
  );
}
