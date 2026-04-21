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
  inviteDetailsLoading,
  withdrawalRecords,
  withdrawalRecordsLoading,
}) => {
  const { symbol } = getCurrencyConfig();
  const canWithdraw = quotaToDisplayAmount(userState?.user?.aff_quota || 0) >= 20;

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
          title={
            <div className='flex items-center justify-between'>
              <Text>{t('邀请记录')}</Text>
              <Tag color='green' shape='circle' size='small'>
                {inviteRecords?.length || 0}
              </Tag>
            </div>
          }
        >
          <Table
            columns={inviteColumns}
            dataSource={inviteRecords || []}
            loading={inviteDetailsLoading}
            pagination={false}
            rowKey='detail_key'
            size='small'
            scroll={{ y: 260 }}
            empty={
              <Empty
                image={Empty.PRESENTED_IMAGE_SIMPLE}
                description={t('暂无邀请记录')}
              />
            }
          />
        </Card>

        <Card
          className='!rounded-xl w-full'
          title={
            <div className='flex items-center justify-between'>
              <Text>{t('充值返利记录')}</Text>
              <Tag color='blue' shape='circle' size='small'>
                {rebateRecords?.length || 0}
              </Tag>
            </div>
          }
        >
          <Table
            columns={rebateColumns}
            dataSource={rebateRecords || []}
            loading={inviteDetailsLoading}
            pagination={false}
            rowKey='detail_key'
            size='small'
            scroll={{ y: 320, x: 860 }}
            empty={
              <Empty
                image={Empty.PRESENTED_IMAGE_SIMPLE}
                description={t('暂无返利记录')}
              />
            }
          />
        </Card>

        <Card
          className='!rounded-xl w-full'
          title={
            <div className='flex items-center justify-between'>
              <Text>{t('提现申请记录')}</Text>
              <Tag color='orange' shape='circle' size='small'>
                {withdrawalRecords?.length || 0}
              </Tag>
            </div>
          }
        >
          <Table
            columns={withdrawalColumns}
            dataSource={withdrawalRecords || []}
            loading={withdrawalRecordsLoading}
            pagination={false}
            rowKey='id'
            size='small'
            scroll={{ y: 280, x: 860 }}
            empty={
              <Empty
                image={Empty.PRESENTED_IMAGE_SIMPLE}
                description={t('暂无提现申请记录')}
              />
            }
          />
        </Card>
      </Space>
    </Card>
  );
};

export default InvitationCard;
