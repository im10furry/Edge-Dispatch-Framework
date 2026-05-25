import { useMemo, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  Table, Input, Select, Button, Space, Row, Col, Tag, Card, Skeleton, Typography, App,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { ReloadOutlined, SearchOutlined } from '@ant-design/icons';
import dayjs from 'dayjs';
import relativeTime from 'dayjs/plugin/relativeTime';
import 'dayjs/locale/zh-cn';
import client from '../../api/client';
import type { Node, NodeStatus, PaginatedResponse } from '../../types/api';
import { NODE_STATUS_MAP } from './nodeStatus';

dayjs.extend(relativeTime);
dayjs.locale('zh-cn');

const STATUS_COLORS: Record<NodeStatus, string> = {
  REGISTERED: '#58a6ff',
  ACTIVE: '#3fb950',
  DEGRADED: '#d29922',
  QUARANTINED: '#fa541c',
  OFFLINE: '#f85149',
  DISABLED: '#8b949e',
  MAINTENANCE: '#d29922',
};

const STATUS_OPTIONS: NodeStatus[] = [
  'REGISTERED', 'ACTIVE', 'DEGRADED', 'QUARANTINED', 'OFFLINE', 'DISABLED', 'MAINTENANCE',
];

export default function NodeListPage() {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const { message } = App.useApp();

  const [searchName, setSearchName] = useState('');
  const [statusFilter, setStatusFilter] = useState<NodeStatus[]>([]);
  const [regionFilter, setRegionFilter] = useState<string>();
  const [ispFilter, setIspFilter] = useState<string>();
  const [tenantFilter, setTenantFilter] = useState<string>();
  const [projectFilter, setProjectFilter] = useState<string>();
  const [pagination, setPagination] = useState({ current: 1, pageSize: 25 });

  const queryParams = useMemo(() => {
    const p: Record<string, string | number> = {
      limit: pagination.pageSize,
      offset: (pagination.current - 1) * pagination.pageSize,
    };
    if (searchName) p.name = searchName;
    if (statusFilter.length > 0) p.status = statusFilter.join(',');
    if (regionFilter) p.region = regionFilter;
    if (ispFilter) p.isp = ispFilter;
    if (tenantFilter) p.tenant_id = tenantFilter;
    if (projectFilter) p.project_id = projectFilter;
    return p;
  }, [searchName, statusFilter, regionFilter, ispFilter, tenantFilter, projectFilter, pagination]);

  const { data, isLoading, refetch } = useQuery({
    queryKey: ['nodes', queryParams],
    queryFn: async () => {
      const res = await client.get<PaginatedResponse<Node>>('/nodes', { params: queryParams });
      return res.data;
    },
  });

  const toggleMutation = useMutation({
    mutationFn: async ({ nodeId, disable }: { nodeId: string; disable: boolean }) => {
      if (disable) {
        await client.post(`/nodes/${nodeId}:disable`);
      } else {
        await client.post(`/nodes/${nodeId}:enable`);
      }
    },
    onSuccess: (_data, variables) => {
      message.success(variables.disable ? '节点已禁用' : '节点已启用');
      queryClient.invalidateQueries({ queryKey: ['nodes'] });
    },
    onError: () => {
      message.error('操作失败');
    },
  });

  const columns: ColumnsType<Node> = useMemo(
    () => [
      {
        title: '状态',
        dataIndex: 'status',
        key: 'status',
        width: 110,
        render: (status: NodeStatus) => (
          <Space size={6}>
            <span
              style={{
                display: 'inline-block',
                width: 8,
                height: 8,
                borderRadius: '50%',
                backgroundColor: STATUS_COLORS[status] ?? '#8b949e',
              }}
            />
            <Typography.Text style={{ fontSize: 13 }}>
              {NODE_STATUS_MAP[status] ?? status}
            </Typography.Text>
          </Space>
        ),
      },
      {
        title: '节点',
        key: 'node',
        ellipsis: true,
        render: (_, record) => (
          <div>
            <Typography.Link onClick={() => navigate(`/nodes/${record.node_id}`)}>
              {record.name}
            </Typography.Link>
            <br />
            <Typography.Text type="secondary" style={{ fontSize: 12 }}>
              {record.node_id}
            </Typography.Text>
          </div>
        ),
      },
      {
        title: '区域/ISP',
        key: 'region_isp',
        width: 150,
        render: (_, record) => (
          <Space size={4} wrap>
            <Tag color="blue">{record.region || '-'}</Tag>
            <Tag>{record.isp || '-'}</Tag>
          </Space>
        ),
      },
      {
        title: '权重',
        dataIndex: 'weight',
        key: 'weight',
        width: 70,
      },
      {
        title: '最后上线',
        dataIndex: 'last_seen_at',
        key: 'last_seen_at',
        width: 140,
        render: (v: string) => (v ? dayjs(v).fromNow() : '-'),
      },
      {
        title: '端点',
        key: 'endpoints',
        width: 70,
        render: (_, record) => record.endpoints?.length ?? 0,
      },
      {
        title: '操作',
        key: 'actions',
        width: 150,
        render: (_, record) => (
          <Space size={4}>
            <Button type="link" size="small" onClick={() => navigate(`/nodes/${record.node_id}`)}>
              详情
            </Button>
            {record.status !== 'DISABLED' ? (
              <Button
                type="link"
                size="small"
                danger
                loading={toggleMutation.isPending}
                onClick={() => toggleMutation.mutate({ nodeId: record.node_id, disable: true })}
              >
                禁用
              </Button>
            ) : (
              <Button
                type="link"
                size="small"
                loading={toggleMutation.isPending}
                onClick={() => toggleMutation.mutate({ nodeId: record.node_id, disable: false })}
              >
                启用
              </Button>
            )}
          </Space>
        ),
      },
    ],
    [navigate, toggleMutation],
  );

  const handleReset = () => {
    setSearchName('');
    setStatusFilter([]);
    setRegionFilter(undefined);
    setIspFilter(undefined);
    setTenantFilter(undefined);
    setProjectFilter(undefined);
    setPagination({ current: 1, pageSize: 25 });
  };

  const handleFilterChange =
    <T,>(setter: React.Dispatch<React.SetStateAction<T>>) =>
    (value: T) => {
    setter(value);
    setPagination((prev) => ({ ...prev, current: 1 }));
  };

  return (
    <div style={{ padding: 24 }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
        <Typography.Title level={4} style={{ margin: 0 }}>
          节点列表
        </Typography.Title>
        <Button icon={<ReloadOutlined />} onClick={() => refetch()}>
          刷新
        </Button>
      </div>

      <Card style={{ marginBottom: 16 }}>
        <Row gutter={[12, 12]}>
          <Col xs={24} sm={12} md={6}>
            <Input
              placeholder="搜索节点名称"
              prefix={<SearchOutlined />}
              value={searchName}
              onChange={(e) => handleFilterChange(setSearchName)(e.target.value)}
              allowClear
            />
          </Col>
          <Col xs={24} sm={12} md={6}>
            <Select
              mode="multiple"
              placeholder="状态筛选"
              value={statusFilter}
              onChange={handleFilterChange(setStatusFilter)}
              style={{ width: '100%' }}
              allowClear
              maxTagCount={2}
            >
              {STATUS_OPTIONS.map((s) => (
                <Select.Option key={s} value={s}>
                  {NODE_STATUS_MAP[s]}
                </Select.Option>
              ))}
            </Select>
          </Col>
          <Col xs={24} sm={12} md={3}>
            <Input
              placeholder="区域"
              value={regionFilter}
              onChange={(e) => handleFilterChange(setRegionFilter)(e.target.value || undefined)}
              allowClear
            />
          </Col>
          <Col xs={24} sm={12} md={3}>
            <Input
              placeholder="ISP"
              value={ispFilter}
              onChange={(e) => handleFilterChange(setIspFilter)(e.target.value || undefined)}
              allowClear
            />
          </Col>
          <Col xs={24} sm={12} md={3}>
            <Input
              placeholder="租户"
              value={tenantFilter}
              onChange={(e) => handleFilterChange(setTenantFilter)(e.target.value || undefined)}
              allowClear
            />
          </Col>
          <Col xs={24} sm={12} md={3}>
            <Input
              placeholder="项目"
              value={projectFilter}
              onChange={(e) => handleFilterChange(setProjectFilter)(e.target.value || undefined)}
              allowClear
            />
          </Col>
        </Row>
        <div style={{ marginTop: 12 }}>
          <Button icon={<ReloadOutlined />} onClick={handleReset}>
            重置筛选
          </Button>
        </div>
      </Card>

      {isLoading ? (
        <Card>
          <Skeleton active paragraph={{ rows: 10 }} />
        </Card>
      ) : (
        <Table
          columns={columns}
          dataSource={data?.data ?? []}
          rowKey="node_id"
          locale={{ emptyText: '暂无节点数据' }}
          pagination={{
            current: pagination.current,
            pageSize: pagination.pageSize,
            total: data?.total ?? 0,
            showSizeChanger: true,
            pageSizeOptions: [10, 25, 50],
            showTotal: (total) => `共 ${total} 个节点`,
            onChange: (page, pageSize) => setPagination({ current: page, pageSize }),
          }}
          scroll={{ x: 900 }}
        />
      )}
    </div>
  );
}
