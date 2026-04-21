/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/

import React from 'react';
import {
  Avatar,
  Typography,
  Card,
  Button,
  Input,
  Badge,
  Space,
  Empty,
  Table,
  Tag,
  Modal,
} from '@douyinfe/semi-ui';
import { Copy, Users, BarChart2, TrendingUp, Gift, Zap } from 'lucide-react';
import { getCurrencyConfig, timestamp2string } from '../../helpers';
import { quotaToDisplayAmount } from '../../helpers/quota';

const { Text } = Typography;

const InvitationCard = ({
  t,
  userState,
  renderQuota,
  setOpenTransfer,
  setOpenWithdraw,
  affLink,
  handleAffLinkClick,
  inviteRecords,
  rebateRecords,
  walletRecords,
  inviteDetailsLoading,
  inviteDetailsFetched,
  onOpenInviteDetails,
  withdrawalRecords,
  withdrawalRecordsLoading,
  withdrawalRecordsFetched,
  onOpenWithdrawalRecords,
}) => {
  const { symbol } = getCurrencyConfig();
  const canWithdraw = quotaToDisplayAmount(userState?.user?.aff_quota || 0) >= 20;
  const [activeDetailModal, setActiveDetailModal] = React.useState('');

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

  const rebateColumns = [
    {
      title: t('充值用户'),
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
      title: t('返利时间'),
      dataIndex: 'rebate_time',
      key: 'rebate_time',
      render: (value) => (value ? timestamp2string(value) : '-'),
    },
    {
      title: t('支付金额'),
      dataIndex: 'top_up_money',
      key: 'top_up_money',
      render: (value) => `¥${Number(value || 0).toFixed(2)}`,
    },
    {
      title: t('充值额度'),
      dataIndex: 'granted_quota',
      key: 'granted_quota',
      render: (value) => renderQuota(value || 0),
    },
    {
      title: t('返利额度'),
      dataIndex: 'rebate_quota',
      key: 'rebate_quota',
      render: (value) => renderQuota(value || 0),
    },
    {
      title: t('已回退款'),
      dataIndex: 'rebate_refunded_quota',
      key: 'rebate_refunded_quota',
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
      title: t('订单号'),
      dataIndex: 'top_up_trade_no',
      key: 'top_up_trade_no',
      render: (value) => (
        <Text copyable={!!value} ellipsis={{ showTooltip: true }}>
          {value || '-'}
        </Text>
      ),
    },
  ];

  const walletChangeTypeLabelMap = {
    invite_reward: t('邀请奖励'),
    topup_rebate: t('充值返利'),
    topup_rebate_refund: t('返利退款回退'),
    transfer_out: t('划转到余额'),
    withdrawal_apply: t('申请提现'),
    withdrawal_reject_return: t('提现驳回退回'),
    admin_add: t('管理员增加'),
    admin_subtract: t('管理员减少'),
    admin_override: t('管理员覆盖'),
  };

  const renderDeltaTag = (value, positiveColor = 'green') => {
    if (!value) {
      return '-';
    }
    const isPositive = value > 0;
    return (
      <Tag color={isPositive ? positiveColor : 'red'} shape='circle' size='small'>
        {(isPositive ? '+' : '') + renderQuota(value)}
      </Tag>
    );
  };

  const walletColumns = [
    {
      title: t('类型'),
      dataIndex: 'change_type',
      key: 'change_type',
      render: (value) => (
        <Tag color='blue' shape='circle' size='small'>
          {walletChangeTypeLabelMap[value] || value || '-'}
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
      render: (value) => renderDeltaTag(value, 'green'),
    },
    {
      title: t('主余额变动'),
      dataIndex: 'quota_delta',
      key: 'quota_delta',
      render: (value) => renderDeltaTag(value, 'cyan'),
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
          return <Text>#{record.withdrawal_id}</Text>;
        }
        return '-';
      },
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
      render: (value) => {
        const statusConfig = {
          pending: { color: 'orange', label: t('待处理') },
          paid: { color: 'green', label: t('已打款') },
          rejected: { color: 'red', label: t('已驳回') },
        };
        const config = statusConfig[value] || {
          color: 'grey',
          label: value || '-',
        };
        return (
          <Tag color={config.color} shape='circle' size='small'>
            {config.label}
          </Tag>
        );
      },
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
      title: t('管理员备注'),
      dataIndex: 'admin_remark',
      key: 'admin_remark',
      render: (value) => value || '-',
    },
  ];

  const detailConfigs = {
    wallet: {
      title: t('邀请余额流水'),
      description: t('查看邀请余额的每次增加、减少和划转明细'),
      tagColor: 'cyan',
      countLabel: inviteDetailsFetched
        ? String(walletRecords?.length || 0)
        : t('按需加载'),
      columns: walletColumns,
      dataSource: walletRecords || [],
      loading: inviteDetailsLoading,
      rowKey: 'record_key',
      scroll: { y: 420, x: 980 },
      emptyText: t('暂无邀请余额流水'),
    },
    invites: {
      title: t('邀请记录'),
      description: t('查看什么时候邀请了哪些用户'),
      tagColor: 'green',
      countLabel: inviteDetailsFetched
        ? String(inviteRecords?.length || 0)
        : t('按需加载'),
      columns: inviteColumns,
      dataSource: inviteRecords || [],
      loading: inviteDetailsLoading,
      rowKey: 'detail_key',
      scroll: { y: 420 },
      emptyText: t('暂无邀请记录'),
    },
    rebates: {
      title: t('充值返利记录'),
      description: t('查看被邀请用户充值后产生的返利明细'),
      tagColor: 'blue',
      countLabel: inviteDetailsFetched
        ? String(rebateRecords?.length || 0)
        : t('按需加载'),
      columns: rebateColumns,
      dataSource: rebateRecords || [],
      loading: inviteDetailsLoading,
      rowKey: 'detail_key',
      scroll: { y: 420, x: 860 },
      emptyText: t('暂无返利记录'),
    },
    withdrawals: {
      title: t('提现申请记录'),
      description: t('查看每次提现申请状态和处理结果'),
      tagColor: 'orange',
      countLabel: withdrawalRecordsFetched
        ? String(withdrawalRecords?.length || 0)
        : t('按需加载'),
      columns: withdrawalColumns,
      dataSource: withdrawalRecords || [],
      loading: withdrawalRecordsLoading,
      rowKey: 'id',
      scroll: { y: 420, x: 860 },
      emptyText: t('暂无提现申请记录'),
    },
  };

  const detailEntries = [
    detailConfigs.wallet,
    detailConfigs.invites,
    detailConfigs.rebates,
    detailConfigs.withdrawals,
  ].map((item, index) => ({
    key: ['wallet', 'invites', 'rebates', 'withdrawals'][index],
    ...item,
  }));

  const currentDetailConfig = activeDetailModal
    ? detailConfigs[activeDetailModal]
    : null;

  const handleOpenDetailModal = async (key) => {
    setActiveDetailModal(key);
    if (key === 'withdrawals') {
      await onOpenWithdrawalRecords?.();
      return;
    }
    await onOpenInviteDetails?.();
  };

  return (
    <Card className='!rounded-2xl shadow-sm border-0'>
      {/* 卡片头部 */}
      <div className='flex items-center mb-4'>
        <Avatar size='small' color='green' className='mr-3 shadow-md'>
          <Gift size={16} />
        </Avatar>
        <div>
          <Typography.Text className='text-lg font-medium'>
            {t('邀请奖励')}
          </Typography.Text>
          <div className='text-xs'>{t('邀请好友获得额外奖励')}</div>
        </div>
      </div>

      {/* 收益展示区域 */}
      <Space vertical style={{ width: '100%' }}>
        {/* 统计数据统一卡片 */}
        <Card
          className='!rounded-xl w-full'
          cover={
            <div
              className='relative h-30'
              style={{
                '--palette-primary-darkerChannel': '0 75 80',
                backgroundImage: `linear-gradient(0deg, rgba(var(--palette-primary-darkerChannel) / 80%), rgba(var(--palette-primary-darkerChannel) / 80%)), url('/cover-4.webp')`,
                backgroundSize: 'cover',
                backgroundPosition: 'center',
                backgroundRepeat: 'no-repeat',
              }}
            >
              {/* 标题和按钮 */}
              <div className='relative z-10 h-full flex flex-col justify-between p-4'>
                <div className='flex justify-between items-center'>
                  <Text strong style={{ color: 'white', fontSize: '16px' }}>
                    {t('收益统计')}
                  </Text>
                  <div className='flex items-center gap-2'>
                    <Button
                      theme='solid'
                      type='warning'
                      size='small'
                      disabled={!canWithdraw}
                      onClick={() => setOpenWithdraw(true)}
                      className='!rounded-lg'
                    >
                      {t('申请提现')}
                    </Button>
                    <Button
                      type='primary'
                      theme='solid'
                      size='small'
                      disabled={
                        !userState?.user?.aff_quota ||
                        userState?.user?.aff_quota <= 0
                      }
                      onClick={() => setOpenTransfer(true)}
                      className='!rounded-lg'
                    >
                      <Zap size={12} className='mr-1' />
                      {t('划转到余额')}
                    </Button>
                  </div>
                </div>

                {/* 统计数据 */}
                <div className='grid grid-cols-3 gap-6 mt-4'>
                  {/* 待使用收益 */}
                  <div className='text-center'>
                    <div
                      className='text-base sm:text-2xl font-bold mb-2'
                      style={{ color: 'white' }}
                    >
                      {renderQuota(userState?.user?.aff_quota || 0)}
                    </div>
                    <div className='flex items-center justify-center text-sm'>
                      <TrendingUp
                        size={14}
                        className='mr-1'
                        style={{ color: 'rgba(255,255,255,0.8)' }}
                      />
                      <Text
                        style={{
                          color: 'rgba(255,255,255,0.8)',
                          fontSize: '12px',
                        }}
                      >
                        {t('待使用收益')}
                      </Text>
                    </div>
                  </div>

                  {/* 总收益 */}
                  <div className='text-center'>
                    <div
                      className='text-base sm:text-2xl font-bold mb-2'
                      style={{ color: 'white' }}
                    >
                      {renderQuota(userState?.user?.aff_history_quota || 0)}
                    </div>
                    <div className='flex items-center justify-center text-sm'>
                      <BarChart2
                        size={14}
                        className='mr-1'
                        style={{ color: 'rgba(255,255,255,0.8)' }}
                      />
                      <Text
                        style={{
                          color: 'rgba(255,255,255,0.8)',
                          fontSize: '12px',
                        }}
                      >
                        {t('总收益')}
                      </Text>
                    </div>
                  </div>

                  {/* 邀请人数 */}
                  <div className='text-center'>
                    <div
                      className='text-base sm:text-2xl font-bold mb-2'
                      style={{ color: 'white' }}
                    >
                      {userState?.user?.aff_count || 0}
                    </div>
                    <div className='flex items-center justify-center text-sm'>
                      <Users
                        size={14}
                        className='mr-1'
                        style={{ color: 'rgba(255,255,255,0.8)' }}
                      />
                      <Text
                        style={{
                          color: 'rgba(255,255,255,0.8)',
                          fontSize: '12px',
                        }}
                      >
                        {t('邀请人数')}
                      </Text>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          }
        >
          {/* 邀请链接部分 */}
          <Input
            value={affLink}
            readonly
            className='!rounded-lg'
            prefix={t('邀请链接')}
            suffix={
              <Button
                type='primary'
                theme='solid'
                onClick={handleAffLinkClick}
                icon={<Copy size={14} />}
                className='!rounded-lg'
              >
                {t('复制')}
              </Button>
            }
          />
        </Card>

        {/* 奖励说明 */}
        <Card
          className='!rounded-xl w-full'
          title={<Text type='tertiary'>{t('奖励说明')}</Text>}
        >
          <div className='space-y-3'>
            <div className='flex items-start gap-2'>
              <Badge dot type='success' />
              <Text type='tertiary' className='text-sm'>
                {t('邀请好友注册，好友充值后您可获得相应奖励')}
              </Text>
            </div>

            <div className='flex items-start gap-2'>
              <Badge dot type='success' />
              <Text type='tertiary' className='text-sm'>
                {t('通过划转功能将奖励额度转入到您的账户余额中')}
              </Text>
            </div>

            <div className='flex items-start gap-2'>
              <Badge dot type='success' />
              <Text type='tertiary' className='text-sm'>
                {t('邀请的好友越多，获得的奖励越多')}
              </Text>
            </div>
          </div>
        </Card>

        <Card
          className='!rounded-xl w-full'
          title={<Text type='tertiary'>{t('邀请明细')}</Text>}
        >
          <div className='grid grid-cols-1 md:grid-cols-2 gap-4'>
            {detailEntries.map((item) => (
              <div
                key={item.key}
                className='rounded-2xl border border-[var(--semi-color-border)] bg-[var(--semi-color-fill-0)] p-4'
              >
                <div className='flex items-start justify-between gap-3'>
                  <div>
                    <Text strong>{item.title}</Text>
                    <div className='mt-1 text-xs text-[var(--semi-color-text-2)]'>
                      {item.description}
                    </div>
                  </div>
                  <Tag color={item.tagColor} shape='circle' size='small'>
                    {item.countLabel}
                  </Tag>
                </div>
                <Button
                  theme='light'
                  type='primary'
                  className='!rounded-lg mt-4'
                  onClick={() => handleOpenDetailModal(item.key)}
                >
                  {t('点击查看')}
                </Button>
              </div>
            ))}
          </div>
        </Card>
      </Space>

      <Modal
        title={currentDetailConfig?.title || t('邀请明细')}
        visible={!!activeDetailModal}
        onCancel={() => setActiveDetailModal('')}
        footer={null}
        size='large'
        centered
      >
        <Table
          columns={currentDetailConfig?.columns || []}
          dataSource={currentDetailConfig?.dataSource || []}
          loading={currentDetailConfig?.loading}
          pagination={false}
          rowKey={currentDetailConfig?.rowKey || 'id'}
          size='small'
          scroll={currentDetailConfig?.scroll}
          empty={
            <Empty
              image={Empty.PRESENTED_IMAGE_SIMPLE}
              description={currentDetailConfig?.emptyText || t('暂无数据')}
            />
          }
        />
      </Modal>
    </Card>
  );
};

export default InvitationCard;
