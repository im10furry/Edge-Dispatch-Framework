import { useQuery } from '@tanstack/react-query';
import { Row, Col, Card, Statistic, Progress, Table, Tag, Typography, Skeleton, Empty, Alert, Button } from 'antd';
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
import client from '../api/client';
import type { DashboardMetrics, DashboardAlert } from '../types/api';

const { Title, Text } = Typography;

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
        <Col xs={24} sm={12} md={8}>
          <Card title="请求量" style={{ height: 320 }}>
            <div style={{ textAlign: 'center', paddingTop: 16 }}>
              <div style={{ fontSize: 48, fontWeight: 700, color: '#58a6ff', lineHeight: 1.2 }}>
                {metrics.qps.toLocaleString()}
              </div>
              <Text type="secondary">QPS</Text>
              <div style={{ marginTop: 24 }}>
                <Text type="secondary">累计请求</Text>
                <div style={{ fontSize: 28, fontWeight: 600 }}>
                  {metrics.total_requests.toLocaleString()}
                </div>
              </div>
            </div>
          </Card>
        </Col>

        <Col xs={24} sm={12} md={8}>
          <Card title="成功率 / 错误率" style={{ height: 320 }}>
            <div style={{ paddingTop: 16 }}>
              <div style={{ marginBottom: 32 }}>
                <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 8 }}>
                  <span><CheckCircleOutlined style={{ color: '#3fb950', marginRight: 6 }} />成功</span>
                  <span style={{ color: '#3fb950', fontWeight: 600 }}>{formatPercent(metrics.success_rate)}%</span>
                </div>
                <Progress percent={formatPercent(metrics.success_rate)} strokeColor="#3fb950" showInfo={false} />
              </div>
              <div>
                <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 8 }}>
                  <span><CloseCircleOutlined style={{ color: '#f85149', marginRight: 6 }} />错误</span>
                  <span style={{ color: '#f85149', fontWeight: 600 }}>{formatPercent(metrics.error_rate)}%</span>
                </div>
                <Progress percent={formatPercent(metrics.error_rate)} strokeColor="#f85149" showInfo={false} />
              </div>
            </div>
          </Card>
        </Col>

        <Col xs={24} sm={12} md={8}>
          <Card title="命中率 vs 回源率" style={{ height: 320 }}>
            <div style={{ paddingTop: 16 }}>
              <div style={{ marginBottom: 32 }}>
                <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 8 }}>
                  <span><AimOutlined style={{ color: '#58a6ff', marginRight: 6 }} />缓存命中</span>
                  <span style={{ color: '#58a6ff', fontWeight: 600 }}>{formatPercent(metrics.hit_rate)}%</span>
                </div>
                <Progress percent={formatPercent(metrics.hit_rate)} strokeColor="#58a6ff" showInfo={false} />
              </div>
              <div>
                <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 8 }}>
                  <span><CloudServerOutlined style={{ color: '#d29922', marginRight: 6 }} />回源</span>
                  <span style={{ color: '#d29922', fontWeight: 600 }}>{formatPercent(metrics.origin_ratio)}%</span>
                </div>
                <Progress percent={formatPercent(metrics.origin_ratio)} strokeColor="#d29922" showInfo={false} />
              </div>
            </div>
          </Card>
        </Col>

        <Col xs={24} sm={12} md={8}>
          <Card title="节点分布" style={{ height: 320 }}>
            <div style={{ paddingTop: 8 }}>
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
              <div style={{ marginTop: 24, textAlign: 'center' }}>
                <Text type="secondary">
                  共 {metrics.total_nodes} 个节点
                </Text>
                <Progress
                  percent={metrics.total_nodes > 0
                    ? Math.round((metrics.online_nodes / metrics.total_nodes) * 100)
                    : 0}
                  strokeColor="#3fb950"
                  trailColor="#f8514933"
                  size="small"
                  style={{ marginTop: 8 }}
                />
              </div>
            </div>
          </Card>
        </Col>

        <Col xs={24} sm={12} md={8}>
          <Card title="延迟" style={{ height: 320 }}>
            <div style={{ paddingTop: 16 }}>
              <div style={{ textAlign: 'center', marginBottom: 32 }}>
                <div style={{ fontSize: 48, fontWeight: 700, color: latencyColor(metrics.p95_latency_ms), lineHeight: 1.2 }}>
                  {metrics.p95_latency_ms}
                </div>
                <Text type="secondary">P95 延迟 (ms)</Text>
              </div>
              <div>
                <Text type="secondary">延迟等级</Text>
                <div style={{ marginTop: 8 }}>
                  <div
                    style={{
                      display: 'flex',
                      justifyContent: 'space-between',
                      marginBottom: 4,
                      fontSize: 12,
                    }}
                  >
                    <span style={{ color: metrics.p95_latency_ms < 50 ? '#3fb950' : undefined }}>低延迟 &lt;50ms</span>
                    <span style={{ color: metrics.p95_latency_ms >= 50 && metrics.p95_latency_ms < 200 ? '#d29922' : undefined }}>中等 &lt;200ms</span>
                    <span style={{ color: metrics.p95_latency_ms >= 200 ? '#f85149' : undefined }}>高延迟 &ge;200ms</span>
                  </div>
                  <Progress
                    percent={Math.min(100, (metrics.p95_latency_ms / 500) * 100)}
                    strokeColor={latencyColor(metrics.p95_latency_ms)}
                    showInfo={false}
                  />
                </div>
              </div>
            </div>
          </Card>
        </Col>

        <Col xs={24} sm={12} md={8}>
          <Card
            title={
              <span>
                <WarningOutlined style={{ marginRight: 6, color: '#d29922' }} />
                最近告警
              </span>
            }
            style={{ height: 320 }}
            styles={{ body: { padding: '12px 16px' } }}
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
      </Row>
    </div>
  );
}
