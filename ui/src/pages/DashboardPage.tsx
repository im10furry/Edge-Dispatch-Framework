import { useEffect } from 'react';
import { useQuery } from '@tanstack/react-query';
import { Row, Col, Card, Statistic, Table, Tag, Typography, Skeleton, Empty, Alert, Button } from 'antd';
import {
  ThunderboltOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
  AimOutlined,
  CloudServerOutlined,
  ClockCircleOutlined,
  WarningOutlined,
  CloudOutlined,
  MinusCircleOutlined,
  ReloadOutlined,
} from '@ant-design/icons';
import { Line } from '@ant-design/charts';
import client from '../api/client';
import type { DashboardMetrics, DashboardAlert } from '../types/api';
import { useDashboardHistory } from '../hooks/useDashboardHistory';

const { Title, Text } = Typography;

const CHART_HEIGHT = 200;
const CHART_CONFIG = {
  smooth: true,
  animation: false as const,
  xField: 'time',
  yField: 'value',
  point: { size: 0 },
  tooltip: {
    channel: 'y' as const,
    valueFormatter: (v: number) => v?.toFixed(1) ?? '0',
  },
  axis: {
    x: { labelAutoHide: true, labelFormatter: (_: string, _idx: number, _total: number) => '' },
  },
} as const;

const ALERT_TYPE_CONFIG: Record<DashboardAlert['type'], { color: string; label: string }> = {
  node_offline: { color: 'error', label: '节点离线' },
  origin_spike: { color: 'warning', label: '回源激增' },
  error_spike: { color: 'error', label: '错误激增' },
};

function DashboardSkeleton() {
  return (
    <div>
      <Skeleton active paragraph={{ rows: 0 }} style={{ marginBottom: 24 }} />
      <Row gutter={[16, 16]} style={{ marginBottom: 24 }}>
        {Array.from({ length: 5 }).map((_, i) => (
          <Col key={i} xs={24} sm={12} md={8} lg={4}>
            <Card>
              <Skeleton active paragraph={{ rows: 1 }} />
            </Card>
          </Col>
        ))}
      </Row>
      <Row gutter={[16, 16]}>
        {Array.from({ length: 6 }).map((_, i) => (
          <Col key={i} xs={24} sm={12} md={8}>
            <Card style={{ height: 320 }}>
              <Skeleton active paragraph={{ rows: 6 }} />
            </Card>
          </Col>
        ))}
      </Row>
    </div>
  );
}

function formatPercent(v: number): number {
  return Number((v * 100).toFixed(1));
}

function statusColor(v: number, threshold = 0.95): string {
  if (v >= threshold) return '#3fb950';
  if (v >= 0.9) return '#d29922';
  return '#f85149';
}

function latencyColor(ms: number): string {
  if (ms < 50) return '#3fb950';
  if (ms < 200) return '#d29922';
  return '#f85149';
}

export default function DashboardPage() {
  const {
    data: metrics,
    isLoading,
    isError,
    error,
    refetch,
    isFetching,
  } = useQuery({
    queryKey: ['dashboard'],
    queryFn: () => client.get<DashboardMetrics>('/dashboard').then((r) => r.data),
    refetchInterval: 30000,
  });

  const { history, append } = useDashboardHistory();

  useEffect(() => {
    if (metrics) {
      append(metrics);
    }
  }, [metrics, append]);

  if (isLoading) return <DashboardSkeleton />;

  if (isError) {
    return (
      <div style={{ textAlign: 'center', paddingTop: 80 }}>
        <Alert
          type="error"
          message="加载失败"
          description={(error as Error)?.message || '无法获取仪表盘数据'}
          showIcon
          style={{ marginBottom: 16 }}
        />
        <Button icon={<ReloadOutlined />} onClick={() => refetch()}>
          重试
        </Button>
      </div>
    );
  }

  if (!metrics) {
    return <Empty description="暂无数据" style={{ paddingTop: 80 }} />;
  }

  const alertColumns = [
    {
      title: '时间',
      dataIndex: 'timestamp',
      key: 'timestamp',
      width: 140,
      render: (v: string) => (
        <Text type="secondary" style={{ fontSize: 12 }}>
          {new Date(v).toLocaleTimeString('zh-CN')}
        </Text>
      ),
    },
    {
      title: '类型',
      dataIndex: 'type',
      key: 'type',
      width: 90,
      render: (v: DashboardAlert['type']) => {
        const cfg = ALERT_TYPE_CONFIG[v];
        return <Tag color={cfg.color}>{cfg.label}</Tag>;
      },
    },
    {
      title: '内容',
      dataIndex: 'message',
      key: 'message',
      ellipsis: true,
    },
  ];

  const stats = [
    {
      title: 'QPS',
      value: metrics.qps,
      suffix: ' req/s',
      icon: <ThunderboltOutlined />,
      color: '#58a6ff',
    },
    {
      title: '成功率',
      value: formatPercent(metrics.success_rate),
      suffix: '%',
      icon: <CheckCircleOutlined />,
      color: statusColor(metrics.success_rate),
      precision: 1,
    },
    {
      title: '命中率',
      value: formatPercent(metrics.hit_rate),
      suffix: '%',
      icon: <AimOutlined />,
      color: statusColor(metrics.hit_rate, 0.7),
      precision: 1,
    },
    {
      title: '回源率',
      value: formatPercent(metrics.origin_ratio),
      suffix: '%',
      icon: <CloudServerOutlined />,
      color: '#58a6ff',
      precision: 1,
    },
    {
      title: 'P95 延迟',
      value: metrics.p95_latency_ms,
      suffix: ' ms',
      icon: <ClockCircleOutlined />,
      color: latencyColor(metrics.p95_latency_ms),
      precision: 1,
    },
  ];

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 24 }}>
        <Title level={4} style={{ margin: 0 }}>
          仪表盘
        </Title>
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
        {stats.map((s, i) => (
          <Col key={i} xs={24} sm={12} md={8} lg={4}>
            <Card>
              <Statistic
                title={
                  <span>
                    <span style={{ marginRight: 6, color: s.color }}>{s.icon}</span>
                    {s.title}
                  </span>
                }
                value={s.value}
                suffix={s.suffix}
                precision={'precision' in s ? s.precision : 0}
                valueStyle={{ color: s.color }}
              />
            </Card>
          </Col>
        ))}
      </Row>

      <Row gutter={[16, 16]}>
        <Col xs={24} lg={12}>
          <Card title="QPS 趋势" style={{ height: 320 }}>
            <Line
              height={CHART_HEIGHT}
              data={history.current.qps}
              {...CHART_CONFIG}
              color="#58a6ff"
              area={{
                style: {
                  fill: 'l(270) 0:#58a6ff10 1:#58a6ff01',
                },
              }}
            />
          </Card>
        </Col>

        <Col xs={24} lg={12}>
          <Card title="成功率" style={{ height: 320 }}>
            <Line
              height={CHART_HEIGHT}
              data={history.current.successRate}
              {...CHART_CONFIG}
              color="#3fb950"
              area={{
                style: {
                  fill: 'l(270) 0:#3fb95015 1:#3fb95001',
                },
              }}
            />
          </Card>
        </Col>

        <Col xs={24} lg={12}>
          <Card title="缓存命中率" style={{ height: 320 }}>
            <Line
              height={CHART_HEIGHT}
              data={history.current.hitRate}
              {...CHART_CONFIG}
              color="#58a6ff"
              area={{
                style: {
                  fill: 'l(270) 0:#58a6ff10 1:#58a6ff01',
                },
              }}
            />
          </Card>
        </Col>

        <Col xs={24} lg={12}>
          <Card title="回源率" style={{ height: 320 }}>
            <Line
              height={CHART_HEIGHT}
              data={history.current.originRatio}
              {...CHART_CONFIG}
              color="#d29922"
              area={{
                style: {
                  fill: 'l(270) 0:#d2992215 1:#d2992201',
                },
              }}
            />
          </Card>
        </Col>

        <Col xs={24} lg={12}>
          <Card title="P95 延迟趋势" style={{ height: 320 }}>
            <Line
              height={CHART_HEIGHT}
              data={history.current.latency}
              {...CHART_CONFIG}
              color={latencyColor(metrics.p95_latency_ms)}
              yAxis={{
                title: { text: 'ms' },
              }}
              area={{
                style: {
                  fill: 'l(270) 0:#f0883e10 1:#f0883e01',
                },
              }}
            />
          </Card>
        </Col>

        <Col xs={24} lg={12}>
          <Card title="节点在线率" style={{ height: 320 }}>
            <Line
              height={CHART_HEIGHT}
              data={history.current.onlineNodes}
              {...CHART_CONFIG}
              color="#3fb950"
              area={{
                style: {
                  fill: 'l(270) 0:#3fb95015 1:#3fb95001',
                },
              }}
              yAxis={{
                title: { text: 'nodes' },
              }}
            />
          </Card>
        </Col>

        <Col xs={24} lg={12}>
          <Card
            title={
              <span>
                <WarningOutlined style={{ marginRight: 6, color: '#d29922' }} />
                最近告警
              </span>
            }
            style={{ height: 320 }}
          >
            {metrics.recent_alerts.length === 0 ? (
              <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: 220 }}>
                <Text type="secondary">暂无告警</Text>
              </div>
            ) : (
              <Table
                columns={alertColumns}
                dataSource={metrics.recent_alerts}
                rowKey={(r) => r.timestamp + r.node_id}
                size="small"
                pagination={false}
                scroll={{ y: 220 }}
                showHeader={false}
              />
            )}
          </Card>
        </Col>

        <Col xs={24} lg={12}>
          <Card title="节点分布" style={{ height: 320 }}>
            <Row gutter={[16, 16]}>
              <Col span={12}>
                <Card size="small" style={{ textAlign: 'center', background: '#0a2e1a', borderColor: '#3fb95044' }}>
                  <CloudOutlined style={{ color: '#3fb950', fontSize: 28 }} />
                  <div style={{ fontSize: 32, fontWeight: 700, color: '#3fb950' }}>
                    {metrics.online_nodes}
                  </div>
                  <Text type="secondary" style={{ fontSize: 12 }}>在线</Text>
                </Card>
              </Col>
              <Col span={12}>
                <Card size="small" style={{ textAlign: 'center', background: '#2e0a0a', borderColor: '#f8514944' }}>
                  <MinusCircleOutlined style={{ color: '#f85149', fontSize: 28 }} />
                  <div style={{ fontSize: 32, fontWeight: 700, color: '#f85149' }}>
                    {metrics.offline_nodes}
                  </div>
                  <Text type="secondary" style={{ fontSize: 12 }}>离线</Text>
                </Card>
              </Col>
            </Row>
          </Card>
        </Col>
      </Row>
    </div>
  );
}
