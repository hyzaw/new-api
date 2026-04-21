import React from 'react';
import { Card, Empty, Spin, Table, Tag, Typography } from '@douyinfe/semi-ui';
import {
  getCurrencyConfig,
  renderQuota,
  timestamp2string,
} from '../../helpers';

const { Text } = Typography;

const walletTypeLabelMap = {
  invite_reward: '邀请奖励',
  topup_rebate: '充值返利',
  topup_rebate_refund: '返利退款回退',
  transfer_out: '划转到余额',
  withdrawal_apply: '申请提现',
  withdrawal_reject_return: '提现驳回退回',
  admin_add: '管理员增加',
  admin_subtract: '管理员减少',
  admin_override: '管理员覆盖',
};

const withdrawalStatusMap = {
  pending: { color: 'orange', label: '待处理' },
  paid: { color: 'green', label: '已打款' },
  rejected: { color: 'red', label: '已驳回' },
};

const topupStatusMap = {
  success: { color: 'green', label: '成功' },
  pending: { color: 'orange', label: '待支付' },
  failed: { color: 'red', label: '失败' },
  expired: { color: 'grey', label: '已关闭' },
};

const renderStatusTag = (value, statusMap, t) => {
  const config = statusMap[value] || {
    color: 'grey',
    label: value || '-',
  };
  return (
    <Tag color={config.color} shape='circle' size='small'>
      {t(config.label)}
    </Tag>
  );
};

const renderDeltaTag = (value, t, color = 'green') => {
  if (!value) {
    return '-';
  }
  const positive = value > 0;
  return (
    <Tag color={positive ? color : 'red'} shape='circle' size='small'>
      {(positive ? '+' : '') + renderQuota(value)}
    </Tag>
  );
};

const InviteOverviewDetails = ({ t, loading, overview }) => {
  const { symbol } = getCurrencyConfig();

  const inviteColumns = [
    {
      title: t('被邀请用户'),
      dataIndex: 'invitee_username',
      key: 'invitee_username',
      render: (_, record) => (
        <div className='flex flex-col'>
          <Text strong>{record.invitee_display_name || record.invitee_username}</Text>
          <Text type='tertiary' size='small'>
            {record.invitee_username || '-'}
          </Text>
        </div>
      ),
    },
    {
      title: t('邀请时间'),
      dataIndex: 'invite_time',
      key: 'invite_time',
      render: (value) => (value ? timestamp2string(value) : '-'),
    },
  ];

  const walletColumns = [
    {
      title: t('类型'),
      dataIndex: 'change_type',
      key: 'change_type',
      render: (value) => (
        <Tag color='blue' shape='circle' size='small'>
          {t(walletTypeLabelMap[value] || value || '-')}
        </Tag>
      ),
    },
    {
      title: t('关联用户'),
      dataIndex: 'invitee_username',
      key: 'invitee_username',
      render: (_, record) =>
        record.invitee_id ? (
          <div className='flex flex-col'>
            <Text strong>{record.invitee_display_name || record.invitee_username}</Text>
            <Text type='tertiary' size='small'>
              {record.invitee_username || '-'}
            </Text>
          </div>
        ) : (
          '-'
        ),
    },
    {
      title: t('邀请余额变动'),
      dataIndex: 'aff_quota_delta',
      key: 'aff_quota_delta',
      render: (value) => renderDeltaTag(value, t, 'green'),
    },
    {
      title: t('主余额变动'),
      dataIndex: 'quota_delta',
      key: 'quota_delta',
      render: (value) => renderDeltaTag(value, t, 'cyan'),
    },
    {
      title: t('关联单号'),
      key: 'related_key',
      render: (_, record) => {
        if (record.top_up_trade_no) {
          return (
            <Text copyable ellipsis={{ showTooltip: true }}>
              {record.top_up_trade_no}
            </Text>
          );
        }
        if (record.withdrawal_id) {
          return `#${record.withdrawal_id}`;
        }
        return '-';
      },
    },
    {
      title: t('操作人'),
      key: 'operator_name',
      render: (_, record) => record.operator_name || '-',
    },
    {
      title: t('备注'),
      dataIndex: 'remark',
      key: 'remark',
      render: (value) => value || '-',
    },
    {
      title: t('时间'),
      dataIndex: 'created_at',
      key: 'created_at',
      render: (value) => (value ? timestamp2string(value) : '-'),
    },
  ];

  const withdrawalColumns = [
    {
      title: t('提现金额'),
      dataIndex: 'amount',
      key: 'amount',
      render: (value) => `${symbol}${Number(value || 0).toFixed(2)}`,
    },
    {
      title: t('占用邀请额度'),
      dataIndex: 'quota',
      key: 'quota',
      render: (value) => renderQuota(value || 0),
    },
    {
      title: t('状态'),
      dataIndex: 'status',
      key: 'status',
      render: (value) => renderStatusTag(value, withdrawalStatusMap, t),
    },
    {
      title: t('处理人'),
      dataIndex: 'operator_name',
      key: 'operator_name',
      render: (value) => value || '-',
    },
    {
      title: t('申请时间'),
      dataIndex: 'created_at',
      key: 'created_at',
      render: (value) => (value ? timestamp2string(value) : '-'),
    },
    {
      title: t('处理时间'),
      dataIndex: 'processed_at',
      key: 'processed_at',
      render: (value) => (value ? timestamp2string(value) : '-'),
    },
    {
      title: t('备注'),
      key: 'remarks',
      render: (_, record) => record.admin_remark || record.user_remark || '-',
    },
  ];

  const topupColumns = [
    {
      title: t('充值用户'),
      dataIndex: 'username',
      key: 'username',
      render: (value, record) => (
        <div className='flex flex-col'>
          <Text strong>{value || '-'}</Text>
          <Text type='tertiary' size='small'>
            ID: {record.user_id}
          </Text>
        </div>
      ),
    },
    {
      title: t('订单号'),
      dataIndex: 'trade_no',
      key: 'trade_no',
      render: (value) => (
        <Text copyable ellipsis={{ showTooltip: true }}>
          {value}
        </Text>
      ),
    },
    {
      title: t('支付金额'),
      dataIndex: 'money',
      key: 'money',
      render: (value) => `${symbol}${Number(value || 0).toFixed(2)}`,
    },
    {
      title: t('充值额度'),
      dataIndex: 'granted_quota',
      key: 'granted_quota',
      render: (value) => renderQuota(value || 0),
    },
    {
      title: t('返利额度'),
      dataIndex: 'invite_rebate_quota',
      key: 'invite_rebate_quota',
      render: (value) => renderQuota(value || 0),
    },
    {
      title: t('已回退返利'),
      dataIndex: 'invite_rebate_refunded_quota',
      key: 'invite_rebate_refunded_quota',
      render: (value) =>
        value > 0 ? (
          <Tag color='orange' shape='circle' size='small'>
            {renderQuota(value)}
          </Tag>
        ) : (
          '-'
        ),
    },
    {
      title: t('状态'),
      dataIndex: 'status',
      key: 'status',
      render: (value) => renderStatusTag(value, topupStatusMap, t),
    },
    {
      title: t('返利时间'),
      dataIndex: 'invite_rebate_time',
      key: 'invite_rebate_time',
      render: (value) => (value ? timestamp2string(value) : '-'),
    },
  ];

  if (loading) {
    return (
      <div className='py-8 text-center'>
        <Spin size='large' />
      </div>
    );
  }

  if (!overview) {
    return (
      <Empty
        image={Empty.PRESENTED_IMAGE_SIMPLE}
        description={t('暂无邀请明细')}
      />
    );
  }

  return (
    <div className='space-y-4'>
      <div className='grid grid-cols-1 md:grid-cols-4 gap-3'>
        <Card className='!rounded-xl border-0 shadow-sm'>
          <Text type='tertiary'>{t('邀请用户')}</Text>
          <div className='mt-2 text-lg font-semibold'>
            {overview.summary?.aff_count || 0}
          </div>
        </Card>
        <Card className='!rounded-xl border-0 shadow-sm'>
          <Text type='tertiary'>{t('当前邀请余额')}</Text>
          <div className='mt-2 text-lg font-semibold'>
            {renderQuota(overview.summary?.aff_quota || 0)}
          </div>
        </Card>
        <Card className='!rounded-xl border-0 shadow-sm'>
          <Text type='tertiary'>{t('累计邀请收益')}</Text>
          <div className='mt-2 text-lg font-semibold'>
            {renderQuota(overview.summary?.aff_history_quota || 0)}
          </div>
        </Card>
        <Card className='!rounded-xl border-0 shadow-sm'>
          <Text type='tertiary'>{t('流水笔数')}</Text>
          <div className='mt-2 text-lg font-semibold'>
            {overview.wallet_records?.length || 0}
          </div>
        </Card>
      </div>

      <Card
        className='!rounded-xl border-0 shadow-sm'
        title={t('邀请用户')}
      >
        <Table
          columns={inviteColumns}
          dataSource={overview.invite_records || []}
          pagination={false}
          rowKey='detail_key'
          size='small'
          scroll={{ y: 220 }}
          empty={
            <Empty
              image={Empty.PRESENTED_IMAGE_SIMPLE}
              description={t('暂无邀请记录')}
            />
          }
        />
      </Card>

      <Card
        className='!rounded-xl border-0 shadow-sm'
        title={t('邀请余额流水')}
      >
        <Table
          columns={walletColumns}
          dataSource={overview.wallet_records || []}
          pagination={false}
          rowKey='record_key'
          size='small'
          scroll={{ y: 320, x: 1100 }}
          empty={
            <Empty
              image={Empty.PRESENTED_IMAGE_SIMPLE}
              description={t('暂无邀请余额流水')}
            />
          }
        />
      </Card>

      <Card
        className='!rounded-xl border-0 shadow-sm'
        title={t('提现记录')}
      >
        <Table
          columns={withdrawalColumns}
          dataSource={overview.withdrawals || []}
          pagination={false}
          rowKey='id'
          size='small'
          scroll={{ y: 240, x: 960 }}
          empty={
            <Empty
              image={Empty.PRESENTED_IMAGE_SIMPLE}
              description={t('暂无提现记录')}
            />
          }
        />
      </Card>

      <Card
        className='!rounded-xl border-0 shadow-sm'
        title={t('邀请相关充值订单')}
      >
        <Table
          columns={topupColumns}
          dataSource={overview.topup_orders || []}
          pagination={false}
          rowKey='id'
          size='small'
          scroll={{ y: 280, x: 1100 }}
          empty={
            <Empty
              image={Empty.PRESENTED_IMAGE_SIMPLE}
              description={t('暂无充值订单')}
            />
          }
        />
      </Card>
    </div>
  );
};

export default InviteOverviewDetails;
