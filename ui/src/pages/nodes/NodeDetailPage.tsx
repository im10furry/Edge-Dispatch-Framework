import { useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  Card, Row, Col, Tag, Button, Space, Skeleton, Typography, Progress, Modal,
  Input, DatePicker, App, Descriptions, Empty,
} from 'antd';
import {
  ArrowLeftOutlined, EditOutlined, StopOutlined, CheckCircleOutlined,
  ExclamationCircleOutlined, KeyOutlined, EyeOutlined, EyeInvisibleOutlined,
} from '@ant-design/icons';
import dayjs from 'dayjs';
import relativeTime from 'dayjs/plugin/relativeTime';
import 'dayjs/locale/zh-cn';
import client from '../../api/client';
import type { Node, DisableNodeRequest, NodeAdminPatch, NodeCredential } from '../../types/api';
import { NODE_STATUS_MAP as StatusMap } from './nodeStatus';

dayjs.extend(relativeTime);
dayjs.locale('zh-cn');

const STATUS_COLORS: Record<string, string> = {
  REGISTERED: '#58a6ff',
  ACTIVE: '#3fb950',
  DEGRADED: '#d29922',
  QUARANTINED: '#fa541c',
  OFFLINE: '#f85149',
  DISABLED: '#8b949e',
  MAINTENANCE: '#d29922',
};

function statusTag(status: string) {
  return (
    <Tag
      color={STATUS_COLORS[status] ?? 'default'}
      style={{ display: 'inline-flex', alignItems: 'center', gap: 6 }}
    >
      <span
        style={{
          width: 7,
          height: 7,
          borderRadius: '50%',
          display: 'inline-block',
          backgroundColor: STATUS_COLORS[status] ?? '#8b949e',
        }}
      />
      {StatusMap[status as keyof typeof StatusMap] ?? status}
    </Tag>
  );
}

export default function NodeDetailPage() {
  const navigate = useNavigate();
  const { nodeID } = useParams<{ nodeID: string }>();
  const queryClient = useQueryClient();
  const { message } = App.useApp();

  const [disableOpen, setDisableOpen] = useState(false);
  const [disableReason, setDisableReason] = useState('');
  const [disableUntil, setDisableUntil] = useState<dayjs.Dayjs | null>(null);
  const [revokeOpen, setRevokeOpen] = useState(false);
  const [revokeConfirm, setRevokeConfirm] = useState('');
  const [labelsOpen, setLabelsOpen] = useState(false);
  const [editLabels, setEditLabels] = useState<Array<{ key: string; value: string }>>([]);
  const [credModalOpen, setCredModalOpen] = useState(false);
  const [credToken, setCredToken] = useState('');
  const [credLoading, setCredLoading] = useState(false);
  const [tokenVisible, setTokenVisible] = useState(false);

  const { data: node, isLoading } = useQuery({
    queryKey: ['node', nodeID],
    queryFn: async () => {
      const res = await client.get<Node>(`/nodes/${nodeID}`);
      return res.data;
    },
    enabled: !!nodeID,
  });

  const enableMutation = useMutation({
    mutationFn: async () => {
      await client.post(`/nodes/${nodeID}:enable`);
    },
    onSuccess: () => {
      message.success('节点已启用');
      queryClient.invalidateQueries({ queryKey: ['node', nodeID] });
      queryClient.invalidateQueries({ queryKey: ['nodes'] });
    },
    onError: () => message.error('启用失败'),
  });

  const disableMutation = useMutation({
    mutationFn: async (payload: DisableNodeRequest) => {
      await client.post(`/nodes/${nodeID}:disable`, payload);
    },
    onSuccess: () => {
      message.success('节点已禁用');
      setDisableOpen(false);
      setDisableReason('');
      setDisableUntil(null);
      queryClient.invalidateQueries({ queryKey: ['node', nodeID] });
      queryClient.invalidateQueries({ queryKey: ['nodes'] });
    },
    onError: () => message.error('禁用失败'),
  });

  const revokeMutation = useMutation({
    mutationFn: async () => {
      await client.post(`/nodes/${nodeID}:revoke`);
    },
    onSuccess: () => {
      message.success('节点已撤销');
      queryClient.invalidateQueries({ queryKey: ['nodes'] });
      navigate('/nodes');
    },
    onError: () => message.error('撤销失败'),
  });

  const labelsMutation = useMutation({
    mutationFn: async (labels: Record<string, string>) => {
      const body: NodeAdminPatch = { labels };
      await client.patch(`/nodes/${nodeID}`, body);
    },
    onSuccess: () => {
      message.success('标签已更新');
      setLabelsOpen(false);
      queryClient.invalidateQueries({ queryKey: ['node', nodeID] });
      queryClient.invalidateQueries({ queryKey: ['nodes'] });
    },
    onError: () => message.error('更新失败'),
  });

  const handleDisable = () => {
    const payload: DisableNodeRequest = { reason: disableReason };
    if (disableUntil) {
      payload.until = disableUntil.toISOString();
    }
    payload.message = disableReason;
    disableMutation.mutate(payload);
  };

  const handleLabelsOpen = () => {
    if (node?.labels) {
      setEditLabels(Object.entries(node.labels).map(([k, v]) => ({ key: k, value: v })));
    } else {
      setEditLabels([]);
    }
    setLabelsOpen(true);
  };

  const handleLabelsSave = () => {
    const labels: Record<string, string> = {};
    for (const item of editLabels) {
      if (item.key.trim()) {
        labels[item.key.trim()] = item.value;
      }
    }
    labelsMutation.mutate(labels);
  };

  const handleShowCredentials = async () => {
    setCredLoading(true);
    try {
      const res = await client.get<NodeCredential>(`/nodes/${nodeID}/credentials`);
      setCredToken(res.data.token);
      setTokenVisible(false);
      setCredModalOpen(true);
    } catch {
      message.error('获取凭证失败');
    } finally {
      setCredLoading(false);
    }
  };

  const handleAddLabelRow = () => {
    setEditLabels((prev) => [...prev, { key: '', value: '' }]);
  };

  const handleRemoveLabelRow = (index: number) => {
    setEditLabels((prev) => prev.filter((_, i) => i !== index));
  };

  if (isLoading || !node) {
    return (
      <div style={{ padding: 24 }}>
        <Skeleton active paragraph={{ rows: 12 }} />
      </div>
    );
  }

  return (
    <div style={{ padding: 24 }}>
      {/* Header */}
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 20 }}>
        <Space size={12}>
          <Button
            type="text"
            icon={<ArrowLeftOutlined />}
            onClick={() => navigate('/nodes')}
            style={{ padding: 0 }}
          >
            返回
          </Button>
          <Typography.Title level={4} style={{ margin: 0 }}>
            {node.name}
          </Typography.Title>
          {statusTag(node.status)}
        </Space>
        <Space>
          {node.status === 'DISABLED' ? (
            <Button
              type="primary"
              icon={<CheckCircleOutlined />}
              loading={enableMutation.isPending}
              onClick={() => enableMutation.mutate()}
            >
              启用节点
            </Button>
          ) : (
            <Button
              icon={<StopOutlined />}
              loading={disableMutation.isPending}
              onClick={() => {
                setDisableReason('');
                setDisableUntil(null);
                setDisableOpen(true);
              }}
            >
              禁用节点
            </Button>
          )}
          <Button
            icon={<KeyOutlined />}
            loading={credLoading}
            onClick={handleShowCredentials}
          >
            凭证
          </Button>
          <Button
            danger
            icon={<ExclamationCircleOutlined />}
            onClick={() => {
              setRevokeConfirm('');
              setRevokeOpen(true);
            }}
          >
            撤销节点
          </Button>
        </Space>
      </div>

      {/* Overview Cards Row */}
      <Row gutter={[16, 16]} style={{ marginBottom: 20 }}>
        {/* Basic Info */}
        <Col xs={24} sm={12} lg={6}>
          <Card title="基本信息" size="small">
            <Descriptions column={1} size="small" colon={false}>
              <Descriptions.Item label="节点 ID">
                <Typography.Text copyable style={{ fontSize: 13 }}>
                  {node.node_id}
                </Typography.Text>
              </Descriptions.Item>
              <Descriptions.Item label="区域">
                <Tag color="blue">{node.region || '-'}</Tag>
              </Descriptions.Item>
              <Descriptions.Item label="ISP">
                <Tag>{node.isp || '-'}</Tag>
              </Descriptions.Item>
              <Descriptions.Item label="ASN">{node.asn || '-'}</Descriptions.Item>
              <Descriptions.Item label="租户">{node.tenant_id}</Descriptions.Item>
              <Descriptions.Item label="项目">{node.project_id}</Descriptions.Item>
            </Descriptions>
          </Card>
        </Col>

        {/* Timestamps */}
        <Col xs={24} sm={12} lg={6}>
          <Card title="时间信息" size="small">
            <Descriptions column={1} size="small" colon={false}>
              <Descriptions.Item label="创建时间">
                {node.created_at ? dayjs(node.created_at).format('YYYY-MM-DD HH:mm:ss') : '-'}
              </Descriptions.Item>
              <Descriptions.Item label="最后上线">
                {node.last_seen_at ? dayjs(node.last_seen_at).fromNow() : '-'}
              </Descriptions.Item>
              <Descriptions.Item label="更新时间">
                {node.updated_at ? dayjs(node.updated_at).format('YYYY-MM-DD HH:mm:ss') : '-'}
              </Descriptions.Item>
              {node.disable_reason && (
                <Descriptions.Item label="禁用原因">
                  <Typography.Text type="danger">{node.disable_reason}</Typography.Text>
                </Descriptions.Item>
              )}
              {node.maintain_until && (
                <Descriptions.Item label="维护至">
                  {dayjs(node.maintain_until).format('YYYY-MM-DD HH:mm')}
                </Descriptions.Item>
              )}
            </Descriptions>
          </Card>
        </Col>

        {/* Endpoints */}
        <Col xs={24} sm={12} lg={6}>
          <Card title="端点列表" size="small">
            {node.endpoints?.length ? (
              <Space direction="vertical" style={{ width: '100%' }}>
                {node.endpoints.map((ep, idx) => (
                  <Card key={idx} size="small" style={{ background: 'rgba(255,255,255,0.03)' }}>
                    <Typography.Text code style={{ fontSize: 12 }}>
                      {ep.scheme}://{ep.host}:{ep.port}
                    </Typography.Text>
                  </Card>
                ))}
              </Space>
            ) : (
              <Empty description="无端点" image={Empty.PRESENTED_IMAGE_SIMPLE} />
            )}
          </Card>
        </Col>

        {/* Capabilities */}
        <Col xs={24} sm={12} lg={6}>
          <Card title="能力标签" size="small">
            {node.capabilities ? (
              <Space size={[4, 8]} wrap>
                <Tag
                  color={node.capabilities.inbound_reachable ? 'green' : 'red'}
                >
                  入站{node.capabilities.inbound_reachable ? '可达' : '不可达'}
                </Tag>
                <Tag color="blue">缓存 {node.capabilities.cache_disk_gb} GB</Tag>
                <Tag color="purple">上行 {node.capabilities.max_uplink_mbps} Mbps</Tag>
                {node.capabilities.supports_https !== undefined && (
                  <Tag color={node.capabilities.supports_https ? 'green' : 'default'}>
                    HTTPS{node.capabilities.supports_https ? '' : ' 不支持'}
                  </Tag>
                )}
                {node.capabilities.nat_type && (
                  <Tag>{node.capabilities.nat_type}</Tag>
                )}
                {node.capabilities.tunnel_required !== undefined && (
                  <Tag color={node.capabilities.tunnel_required ? 'orange' : 'default'}>
                    隧道{node.capabilities.tunnel_required ? '需要' : '不需要'}
                  </Tag>
                )}
              </Space>
            ) : (
              <Empty description="无能力数据" image={Empty.PRESENTED_IMAGE_SIMPLE} />
            )}
          </Card>
        </Col>
      </Row>

      {/* Runtime Metrics */}
      {node.runtime && (
        <Card title="运行时指标" size="small" style={{ marginBottom: 20 }}>
          <Row gutter={[16, 16]}>
            <Col xs={24} sm={12} md={6}>
              <Card size="small" style={{ textAlign: 'center' }}>
                <Typography.Text type="secondary">CPU 使用率</Typography.Text>
                <Progress
                  type="dashboard"
                  percent={Math.min(node.runtime.cpu, 100)}
                  size={100}
                  strokeColor={node.runtime.cpu > 80 ? '#f85149' : '#3fb950'}
                />
              </Card>
            </Col>
            <Col xs={24} sm={12} md={6}>
              <Card size="small" style={{ textAlign: 'center' }}>
                <Typography.Text type="secondary">内存</Typography.Text>
                <div style={{ fontSize: 28, fontWeight: 600, margin: '16px 0' }}>
                  {node.runtime.mem_mb >= 1024
                    ? `${(node.runtime.mem_mb / 1024).toFixed(1)} GB`
                    : `${node.runtime.mem_mb} MB`}
                </div>
                <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                  已用内存
                </Typography.Text>
              </Card>
            </Col>
            <Col xs={24} sm={12} md={6}>
              <Card size="small" style={{ textAlign: 'center' }}>
                <Typography.Text type="secondary">磁盘剩余</Typography.Text>
                <div style={{ fontSize: 28, fontWeight: 600, margin: '16px 0' }}>
                  {node.runtime.disk_free_gb >= 1024
                    ? `${(node.runtime.disk_free_gb / 1024).toFixed(1)} TB`
                    : `${node.runtime.disk_free_gb.toFixed(1)} GB`}
                </div>
                <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                  可用空间
                </Typography.Text>
              </Card>
            </Col>
            <Col xs={24} sm={12} md={6}>
              <Card size="small" style={{ textAlign: 'center' }}>
                <Typography.Text type="secondary">连接数</Typography.Text>
                <div style={{ fontSize: 28, fontWeight: 600, margin: '16px 0' }}>
                  {node.runtime.conn.toLocaleString()}
                </div>
                <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                  当前连接
                </Typography.Text>
              </Card>
            </Col>
          </Row>
        </Card>
      )}

      {/* Traffic & Cache */}
      <Row gutter={[16, 16]} style={{ marginBottom: 20 }}>
        {node.traffic && (
          <Col xs={24} md={12}>
            <Card title="流量指标" size="small">
              <Row gutter={[16, 16]}>
                <Col span={12}>
                  <Card size="small" style={{ textAlign: 'center' }}>
                    <Typography.Text type="secondary">入站</Typography.Text>
                    <div style={{ fontSize: 24, fontWeight: 600, margin: '8px 0' }}>
                      {node.traffic.ingress_mbps.toFixed(1)}
                    </div>
                    <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                      Mbps
                    </Typography.Text>
                  </Card>
                </Col>
                <Col span={12}>
                  <Card size="small" style={{ textAlign: 'center' }}>
                    <Typography.Text type="secondary">出站</Typography.Text>
                    <div style={{ fontSize: 24, fontWeight: 600, margin: '8px 0' }}>
                      {node.traffic.egress_mbps.toFixed(1)}
                    </div>
                    <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                      Mbps
                    </Typography.Text>
                  </Card>
                </Col>
              </Row>
              <div style={{ marginTop: 16 }}>
                <Typography.Text type="secondary">5xx 错误率</Typography.Text>
                <Progress
                  percent={Math.min(node.traffic.err_5xx_rate * 100, 100)}
                  size="small"
                  strokeColor={node.traffic.err_5xx_rate > 0.05 ? '#f85149' : '#3fb950'}
                  format={(p) => `${p?.toFixed(2)}%`}
                />
              </div>
            </Card>
          </Col>
        )}

        {node.cache && (
          <Col xs={24} md={node.traffic ? 12 : 24}>
            <Card title="缓存命中率" size="small">
              <div style={{ textAlign: 'center' }}>
                <Progress
                  type="dashboard"
                  percent={Math.min(node.cache.hit_ratio * 100, 100)}
                  size={180}
                  strokeColor={{
                    '0%': '#58a6ff',
                    '100%': '#3fb950',
                  }}
                  format={(p) => `${p?.toFixed(1)}%`}
                />
              </div>
            </Card>
          </Col>
        )}
      </Row>

      {/* Scores */}
      <Card title="评分" size="small" style={{ marginBottom: 20 }}>
        <Row gutter={[16, 16]}>
          <Col xs={24} sm={8}>
            <div style={{ textAlign: 'center' }}>
              <Typography.Text type="secondary">可达性</Typography.Text>
              <Progress
                percent={Math.min(node.scores.reachable_score * 100, 100)}
                size="small"
                strokeColor="#3fb950"
                format={(p) => `${p?.toFixed(0)}%`}
              />
            </div>
          </Col>
          <Col xs={24} sm={8}>
            <div style={{ textAlign: 'center' }}>
              <Typography.Text type="secondary">健康度</Typography.Text>
              <Progress
                percent={Math.min(node.scores.health_score * 100, 100)}
                size="small"
                strokeColor="#58a6ff"
                format={(p) => `${p?.toFixed(0)}%`}
              />
            </div>
          </Col>
          <Col xs={24} sm={8}>
            <div style={{ textAlign: 'center' }}>
              <Typography.Text type="secondary">风险值</Typography.Text>
              <Progress
                percent={Math.min(node.scores.risk_score * 100, 100)}
                size="small"
                strokeColor={node.scores.risk_score > 0.5 ? '#f85149' : '#d29922'}
                format={(p) => `${p?.toFixed(0)}%`}
              />
            </div>
          </Col>
        </Row>
      </Card>

      {/* Labels */}
      <Card
        title="标签"
        size="small"
        style={{ marginBottom: 20 }}
        extra={
          <Button type="link" icon={<EditOutlined />} onClick={handleLabelsOpen}>
            编辑
          </Button>
        }
      >
        {node.labels && Object.keys(node.labels).length > 0 ? (
          <Space size={[4, 8]} wrap>
            {Object.entries(node.labels).map(([k, v]) => (
              <Tag key={k} color="blue">
                {k}: {v}
              </Tag>
            ))}
          </Space>
        ) : (
          <Typography.Text type="secondary">暂无标签</Typography.Text>
        )}
      </Card>

      {/* Credentials Modal */}
      <Modal
        title={
          <Space>
            <KeyOutlined style={{ color: '#d29922' }} />
            <span>节点凭证</span>
          </Space>
        }
        open={credModalOpen}
        onCancel={() => { setCredModalOpen(false); setTokenVisible(false); }}
        footer={
          <Button onClick={() => { setCredModalOpen(false); setTokenVisible(false); }}>关闭</Button>
        }
        destroyOnClose
      >
        <Card
          size="small"
          style={{
            background: 'rgba(210,153,34,0.08)',
            border: '1px solid rgba(210,153,34,0.2)',
            marginBottom: 16,
          }}
        >
          <Typography.Text type="warning" strong>
            <ExclamationCircleOutlined style={{ marginRight: 6 }} />
            请妥善保管此凭证
          </Typography.Text>
          <br />
          <Typography.Text type="secondary" style={{ fontSize: 13 }}>
            此 Token 用于边缘节点与调度中心通信，请勿泄露。
          </Typography.Text>
        </Card>
        <div>
          <Typography.Text strong>节点 ID</Typography.Text>
          <Typography.Paragraph copyable style={{ marginTop: 4 }}>
            {node?.node_id}
          </Typography.Paragraph>
        </div>
        <div style={{ marginTop: 16 }}>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
            <Typography.Text strong>Token</Typography.Text>
            <Button
              type="text"
              size="small"
              icon={tokenVisible ? <EyeInvisibleOutlined /> : <EyeOutlined />}
              onClick={() => setTokenVisible(!tokenVisible)}
            >
              {tokenVisible ? '隐藏' : '显示'}
            </Button>
          </div>
          <div
            style={{
              background: 'rgba(0,0,0,0.2)',
              padding: 12,
              borderRadius: 6,
              marginTop: 8,
              wordBreak: 'break-all',
              fontFamily: 'monospace',
              fontSize: 13,
            }}
          >
            {credToken ? (
              tokenVisible ? (
                <Typography.Text copyable>{credToken}</Typography.Text>
              ) : (
                <Typography.Text type="secondary">
                  {'*'.repeat(Math.min(credToken.length, 36))}
                </Typography.Text>
              )
            ) : (
              <Typography.Text type="secondary">加载中...</Typography.Text>
            )}
          </div>
        </div>
      </Modal>

      {/* Disable Modal */}
      <Modal
        title="禁用节点"
        open={disableOpen}
        onOk={handleDisable}
        onCancel={() => setDisableOpen(false)}
        okText="确认禁用"
        cancelText="取消"
        confirmLoading={disableMutation.isPending}
        okButtonProps={{ danger: true }}
        destroyOnClose
      >
        <Space direction="vertical" style={{ width: '100%' }} size={12}>
          <div>
            <Typography.Text strong>
              {node.name}
            </Typography.Text>
            <Typography.Text type="secondary" style={{ marginLeft: 8 }}>
              ({node.node_id})
            </Typography.Text>
          </div>

          <div>
            <Typography.Text>禁用原因</Typography.Text>
            <Typography.Text type="danger" style={{ marginLeft: 4 }}>*</Typography.Text>
            <Input.TextArea
              rows={3}
              placeholder="请输入禁用原因"
              value={disableReason}
              onChange={(e) => setDisableReason(e.target.value)}
              style={{ marginTop: 4 }}
            />
          </div>

          <div>
            <Typography.Text>禁用截止时间</Typography.Text>
            <Typography.Text type="secondary" style={{ marginLeft: 4 }}>
              （可选）
            </Typography.Text>
            <br />
            <DatePicker
              showTime
              placeholder="选择截止时间，不选则永久禁用"
              value={disableUntil}
              onChange={(v) => setDisableUntil(v)}
              style={{ width: '100%', marginTop: 4 }}
            />
          </div>

          <Card size="small" style={{ background: 'rgba(248,81,73,0.08)', border: '1px solid rgba(248,81,73,0.2)' }}>
            <Typography.Text type="danger">
              禁用后该节点将不再接收调度流量，确认继续？
            </Typography.Text>
          </Card>
        </Space>
      </Modal>

      {/* Revoke Modal */}
      <Modal
        title={
          <Space>
            <ExclamationCircleOutlined style={{ color: '#f85149' }} />
            <span>撤销节点</span>
          </Space>
        }
        open={revokeOpen}
        onOk={() => revokeMutation.mutate()}
        onCancel={() => setRevokeOpen(false)}
        okText="确认撤销"
        cancelText="取消"
        confirmLoading={revokeMutation.isPending}
        okButtonProps={{ danger: true }}
        destroyOnClose
      >
        <Space direction="vertical" style={{ width: '100%' }} size={12}>
          <Card
            size="small"
            style={{
              background: 'rgba(248,81,73,0.08)',
              border: '1px solid rgba(248,81,73,0.3)',
            }}
          >
            <Typography.Text type="danger" strong>
              此操作不可撤销
            </Typography.Text>
            <br />
            <Typography.Text type="secondary" style={{ fontSize: 13 }}>
              撤销将永久删除该节点及其所有关联数据，包括端点、标签和指标历史。
            </Typography.Text>
          </Card>

          <div>
            <Typography.Text>
              请输入节点名称
              <Typography.Text strong style={{ marginLeft: 4 }}>
                {node.name}
              </Typography.Text>
              以确认：
            </Typography.Text>
            <Input
              placeholder={node.name}
              value={revokeConfirm}
              onChange={(e) => setRevokeConfirm(e.target.value)}
              style={{ marginTop: 4 }}
            />
          </div>
        </Space>
      </Modal>

      {/* Labels Edit Modal */}
      <Modal
        title="编辑标签"
        open={labelsOpen}
        onOk={handleLabelsSave}
        onCancel={() => setLabelsOpen(false)}
        okText="保存"
        cancelText="取消"
        confirmLoading={labelsMutation.isPending}
        destroyOnClose
        width={560}
      >
        <Space direction="vertical" style={{ width: '100%' }} size={8}>
          {editLabels.map((item, idx) => (
            <Row key={idx} gutter={8} align="middle">
              <Col flex="1">
                <Input
                  placeholder="键"
                  value={item.key}
                  onChange={(e) => {
                    const next = [...editLabels];
                    next[idx] = { ...next[idx], key: e.target.value };
                    setEditLabels(next);
                  }}
                />
              </Col>
              <Col flex="1">
                <Input
                  placeholder="值"
                  value={item.value}
                  onChange={(e) => {
                    const next = [...editLabels];
                    next[idx] = { ...next[idx], value: e.target.value };
                    setEditLabels(next);
                  }}
                />
              </Col>
              <Col>
                <Button
                  type="text"
                  danger
                  size="small"
                  onClick={() => handleRemoveLabelRow(idx)}
                >
                  删除
                </Button>
              </Col>
            </Row>
          ))}
          <Button type="dashed" block onClick={handleAddLabelRow} style={{ marginTop: 4 }}>
            + 添加标签
          </Button>
        </Space>
      </Modal>
    </div>
  );
}
