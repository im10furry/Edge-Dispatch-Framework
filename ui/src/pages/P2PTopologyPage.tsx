import { useQuery } from '@tanstack/react-query';
import { Row, Col, Card, Table, Tag, Typography, Skeleton, Empty, Alert, Button, Statistic } from 'antd';
import {
  ClusterOutlined,
  LinkOutlined,
  ThunderboltOutlined,
  CheckCircleOutlined,
  ReloadOutlined,
} from '@ant-design/icons';
import client from '../api/client';
import type { P2PTopology } from '../types/api';

const { Title, Text } = Typography;

export default function P2PTopologyPage() {
  const {
    data: topo,
    isLoading,
    isError,
    error,
    refetch,
    isFetching,
  } = useQuery({
    queryKey: ['p2p-topology'],
    queryFn: () => client.get<P2PTopology>('/p2p/topology').then((r) => r.data),
    refetchInterval: 30000,
  });

  if (isLoading) {
    return (
      <div>
        <Skeleton active paragraph={{ rows: 0 }} style={{ marginBottom: 24 }} />
        <Row gutter={[16, 16]}>
          {Array.from({ length: 4 }).map((_, i) => (
            <Col key={i} xs={24} sm={12}>
              <Card>
                <Skeleton active paragraph={{ rows: 4 }} />
              </Card>
            </Col>
          ))}
        </Row>
      </div>
    );
  }

  if (isError) {
    return (
      <div style={{ textAlign: 'center', paddingTop: 80 }}>
        <Alert
          type="error"
          message="加载失败"
          description={(error as Error)?.message || '无法获取P2P拓扑数据'}
          showIcon
          style={{ marginBottom: 16 }}
        />
        <Button icon={<ReloadOutlined />} onClick={() => refetch()}>重试</Button>
      </div>
    );
  }

  if (!topo || topo.nodes.length === 0) {
    return (
      <div>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 24 }}>
          <Title level={4} style={{ margin: 0 }}>P2P 拓扑</Title>
          <Button icon={<ReloadOutlined />} onClick={() => refetch()} size="small">刷新</Button>
        </div>
        <Empty description="暂无P2P节点数据" style={{ paddingTop: 80 }} />
      </div>
    );
  }

  const activeLinks = topo.links.filter((l) => l.success_rate > 0);
  const smallNodes = topo.nodes.filter((n) => n.is_small_bandwidth);
  const activeNodes = topo.nodes.filter((n) => n.status === 'ACTIVE');
  const avgHitRatio = topo.nodes.length > 0
    ? topo.nodes.reduce((sum, n) => sum + n.cache_hit_ratio, 0) / topo.nodes.length
    : 0;

  const nodeColumns = [
    {
      title: '节点',
      dataIndex: 'node_id',
      key: 'node_id',
      width: 200,
      ellipsis: true,
      render: (_: string, r: typeof topo.nodes[number]) => (
        <span>
          <ClusterOutlined style={{ marginRight: 6, color: r.is_small_bandwidth ? '#d29922' : '#58a6ff' }} />
          {r.name || r.node_id}
        </span>
      ),
    },
    {
      title: '带宽 (Mbps)',
      dataIndex: 'bandwidth_mbps',
      key: 'bandwidth_mbps',
      width: 110,
      render: (v: number, r: typeof topo.nodes[number]) => (
        <span style={{ color: r.is_small_bandwidth ? '#d29922' : '#3fb950' }}>
          {v > 0 ? v.toLocaleString() : '-'}
        </span>
      ),
    },
    {
      title: '类型',
      dataIndex: 'is_small_bandwidth',
      key: 'is_small_bandwidth',
      width: 80,
      render: (v: boolean) => (
        <Tag color={v ? 'orange' : 'blue'}>{v ? '小带宽' : '正常'}</Tag>
      ),
    },
    {
      title: '命中率',
      dataIndex: 'cache_hit_ratio',
      key: 'cache_hit_ratio',
      width: 90,
      render: (v: number) => (
        <span style={{ color: v > 0.7 ? '#3fb950' : v > 0.3 ? '#d29922' : '#f85149' }}>
          {(v * 100).toFixed(1)}%
        </span>
      ),
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 80,
      render: (v: string) => {
        const isActive = v === 'ACTIVE';
        return <Tag color={isActive ? 'green' : 'default'}>{isActive ? '在线' : v}</Tag>;
      },
    },
  ];

  const linkColumns = [
    {
      title: '源节点',
      dataIndex: 'source',
      key: 'source',
      width: 200,
      ellipsis: true,
      render: (v: string) => (
        <span>
          <LinkOutlined style={{ marginRight: 6, color: '#58a6ff' }} />
          <Text code style={{ fontSize: 11 }}>{v}</Text>
        </span>
      ),
    },
    {
      title: '目标节点',
      dataIndex: 'target',
      key: 'target',
      width: 200,
      ellipsis: true,
      render: (v: string) => <Text code style={{ fontSize: 11 }}>{v}</Text>,
    },
    {
      title: '延迟 (ms)',
      dataIndex: 'latency_ms',
      key: 'latency_ms',
      width: 100,
      render: (v: number) => (
        <span style={{ color: v < 50 ? '#3fb950' : v < 200 ? '#d29922' : '#f85149' }}>
          {v > 0 ? v : '-'}
        </span>
      ),
    },
    {
      title: '成功率',
      dataIndex: 'success_rate',
      key: 'success_rate',
      width: 100,
      render: (v: number) => (
        <span style={{ color: v > 0.9 ? '#3fb950' : v > 0.5 ? '#d29922' : '#f85149' }}>
          {(v * 100).toFixed(1)}%
        </span>
      ),
    },
  ];

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 24 }}>
        <Title level={4} style={{ margin: 0 }}>P2P 拓扑</Title>
        <Button
          icon={<ReloadOutlined spin={isFetching} />}
          onClick={() => refetch()}
          loading={isFetching}
          size="small"
        >
          刷新
        </Button>
      </div>

      <Row gutter={[16, 16]} style={{ marginBottom: 24 }}>
        <Col xs={24} sm={12} md={6}>
          <Card>
            <Statistic
              title="P2P 节点"
              value={topo.nodes.length}
              prefix={<ClusterOutlined style={{ color: '#58a6ff' }} />}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} md={6}>
          <Card>
            <Statistic
              title="活跃链路"
              value={activeLinks.length}
              prefix={<LinkOutlined style={{ color: '#3fb950' }} />}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} md={6}>
          <Card>
            <Statistic
              title="小带宽节点"
              value={smallNodes.length}
              prefix={<ThunderboltOutlined style={{ color: '#d29922' }} />}
              suffix={`/ ${topo.nodes.length}`}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} md={6}>
          <Card>
            <Statistic
              title="平均命中率"
              value={(avgHitRatio * 100).toFixed(1)}
              prefix={<CheckCircleOutlined style={{ color: '#3fb950' }} />}
              suffix="%"
            />
          </Card>
        </Col>
      </Row>

      <Row gutter={[16, 16]}>
        <Col xs={24} lg={14}>
          <Card title="P2P 节点" style={{ marginBottom: 16 }}>
            <Table
              columns={nodeColumns}
              dataSource={topo.nodes}
              rowKey="node_id"
              size="small"
              pagination={false}
              scroll={{ y: 400 }}
            />
          </Card>
        </Col>
        <Col xs={24} lg={10}>
          <Card title="P2P 链路">
            {topo.links.length === 0 ? (
              <Empty description="暂无活跃链路" />
            ) : (
              <Table
                columns={linkColumns}
                dataSource={topo.links}
                rowKey={(r) => `${r.source}-${r.target}`}
                size="small"
                pagination={false}
                scroll={{ y: 400 }}
              />
            )}
          </Card>
        </Col>
      </Row>
    </div>
  );
}
