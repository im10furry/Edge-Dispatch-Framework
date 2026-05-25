import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  Card,
  Switch,
  Slider,
  InputNumber,
  Button,
  Typography,
  Row,
  Col,
  Form,
  Divider,
  Skeleton,
  Alert,
  message,
  Space,
  Input,
} from 'antd';
import {
  SettingOutlined,
  ThunderboltOutlined,
  ShareAltOutlined,
  ForwardOutlined,
  CloudServerOutlined,
  ReloadOutlined,
  SaveOutlined,
} from '@ant-design/icons';
import { useEffect } from 'react';
import client from '../api/client';
import type { GlobalConfig } from '../types/api';

const { Title, Text } = Typography;

export default function GlobalConfigPage() {
  const qc = useQueryClient();
  const [form] = Form.useForm<GlobalConfig>();

  const {
    data: config,
    isLoading,
    isError,
    error,
    refetch,
    isFetching,
  } = useQuery({
    queryKey: ['global-config'],
    queryFn: () => client.get<GlobalConfig>('/config').then((r) => r.data),
  });

  useEffect(() => {
    if (config) {
      form.setFieldsValue(config);
    }
  }, [config, form]);

  const saveMutation = useMutation({
    mutationFn: (values: GlobalConfig) => client.put('/config', values),
    onSuccess: () => {
      message.success('配置已保存');
      qc.invalidateQueries({ queryKey: ['global-config'] });
    },
    onError: (err: Error) => {
      message.error(`保存失败: ${err.message}`);
    },
  });

  if (isLoading) {
    return (
      <div>
        <Skeleton active paragraph={{ rows: 0 }} style={{ marginBottom: 24 }} />
        <Row gutter={[16, 16]}>
          {Array.from({ length: 4 }).map((_, i) => (
            <Col key={i} xs={24} lg={12}>
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
          description={(error as Error)?.message || '无法获取全局配置'}
          showIcon
          style={{ marginBottom: 16 }}
        />
        <Button icon={<ReloadOutlined />} onClick={() => refetch()}>重试</Button>
      </div>
    );
  }

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 24 }}>
        <Title level={4} style={{ margin: 0 }}>
          <SettingOutlined style={{ marginRight: 8 }} />
          全局配置
        </Title>
        <Space>
          <Button
            icon={<ReloadOutlined spin={isFetching} />}
            onClick={() => refetch()}
            loading={isFetching}
            size="small"
          >
            刷新
          </Button>
        </Space>
      </div>

      <Form
        form={form}
        layout="vertical"
        onFinish={(values) => saveMutation.mutate(values)}
      >
        <Row gutter={[16, 16]}>
          <Col xs={24} lg={12}>
            <Card
              title={<span><ThunderboltOutlined style={{ marginRight: 6, color: '#d29922' }} />小带宽优化</span>}
            >
              <Form.Item name={['small_bandwidth', 'enabled']} label="启用" valuePropName="checked">
                <Switch />
              </Form.Item>
              <Form.Item name={['small_bandwidth', 'threshold_mbps']} label="带宽阈值 (Mbps)">
                <InputNumber min={1} max={10000} style={{ width: '100%' }} addonAfter="Mbps" />
              </Form.Item>
              <Text type="secondary">
                低于此阈值的节点将被标记为小带宽节点，仅接收少量低频请求
              </Text>
            </Card>
          </Col>

          <Col xs={24} lg={12}>
            <Card
              title={<span><ShareAltOutlined style={{ marginRight: 6, color: '#58a6ff' }} />P2P 共享</span>}
            >
              <Form.Item name={['p2p', 'enabled']} label="启用" valuePropName="checked">
                <Switch />
              </Form.Item>
              <Form.Item name={['p2p', 'max_peers']} label="最大对等节点数">
                <Slider min={1} max={50} marks={{ 1: '1', 10: '10', 25: '25', 50: '50' }} />
              </Form.Item>
              <Form.Item name={['p2p', 'bandwidth_limit_mbps']} label="带宽限制 (Mbps)">
                <InputNumber min={1} max={10000} style={{ width: '100%' }} addonAfter="Mbps" />
              </Form.Item>
              <Form.Item name={['p2p', 'discovery_interval_sec']} label="发现间隔 (秒)">
                <InputNumber min={10} max={600} style={{ width: '100%' }} addonAfter="秒" />
              </Form.Item>
              <Text type="secondary">
                节点间通过P2P共享内容以降低回源压力
              </Text>
            </Card>
          </Col>

          <Col xs={24} lg={12}>
            <Card
              title={<span><ForwardOutlined style={{ marginRight: 6, color: '#3fb950' }} />智能预取</span>}
            >
              <Form.Item name={['prefetch', 'enabled']} label="启用" valuePropName="checked">
                <Switch />
              </Form.Item>
              <Divider orientation="left" plain style={{ fontSize: 13 }}>日间模式</Divider>
              <Form.Item name={['prefetch', 'day_mode', 'bandwidth_limit_mbps']} label="带宽限制 (Mbps)">
                <InputNumber min={1} max={10000} style={{ width: '100%' }} addonAfter="Mbps" />
              </Form.Item>
              <Form.Item name={['prefetch', 'day_mode', 'min_priority']} label="最小优先级">
                <Slider min={1} max={10} marks={{ 1: '低', 5: '中', 10: '高' }} />
              </Form.Item>
              <Divider orientation="left" plain style={{ fontSize: 13 }}>夜间模式</Divider>
              <Form.Item name={['prefetch', 'night_mode', 'start']} label="开始时间">
                <Input placeholder="HH:MM" />
              </Form.Item>
              <Form.Item name={['prefetch', 'night_mode', 'end']} label="结束时间">
                <Input placeholder="HH:MM" />
              </Form.Item>
              <Form.Item name={['prefetch', 'night_mode', 'bandwidth_limit_mbps']} label="带宽限制 (Mbps)">
                <InputNumber min={1} max={10000} style={{ width: '100%' }} addonAfter="Mbps" />
              </Form.Item>
              <Text type="secondary">
                夜间模式带宽设置更大以支持离线内容预取
              </Text>
            </Card>
          </Col>

          <Col xs={24} lg={12}>
            <Card
              title={<span><CloudServerOutlined style={{ marginRight: 6, color: '#f0883e' }} />源站回源</span>}
            >
              <Form.Item name={['origin_fetch', 'bandwidth_percent']} label="带宽占比 (%)">
                <Slider min={10} max={100} marks={{ 10: '10%', 50: '50%', 80: '80%', 100: '100%' }} />
              </Form.Item>
              <Form.Item name={['origin_fetch', 'max_concurrent']} label="最大并发">
                <InputNumber min={1} max={50} style={{ width: '100%' }} />
              </Form.Item>
              <Form.Item name={['origin_fetch', 'timeout_sec']} label="超时 (秒)">
                <InputNumber min={5} max={120} style={{ width: '100%' }} addonAfter="秒" />
              </Form.Item>
              <Form.Item name={['origin_fetch', 'priority']} label="回源优先级">
                <Input placeholder="按逗号分隔, 例如: p2p,origin" />
              </Form.Item>
              <Text type="secondary">
                控制边缘节点从源站拉取内容的行为参数
              </Text>
            </Card>
          </Col>
        </Row>

        <div style={{ marginTop: 24, textAlign: 'right' }}>
          <Button
            type="primary"
            htmlType="submit"
            icon={<SaveOutlined />}
            loading={saveMutation.isPending}
            size="large"
          >
            保存配置
          </Button>
        </div>
      </Form>
    </div>
  );
}
