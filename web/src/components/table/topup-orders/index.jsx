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

import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import {
  Badge,
  Button,
  Empty,
  Input,
  Modal,
  Tag,
  Typography,
} from '@douyinfe/semi-ui';
import { IconRefresh, IconSearch } from '@douyinfe/semi-icons';
import {
  IllustrationNoResult,
  IllustrationNoResultDark,
} from '@douyinfe/semi-illustrations';
import { Coins } from 'lucide-react';
import { useNavigate } from 'react-router-dom';
import CardPro from '../../common/ui/CardPro';
import CardTable from '../../common/ui/CardTable';
import { API, showError, showSuccess, timestamp2string } from '../../../helpers';
import { createCardProPagination } from '../../../helpers/utils';
import { useIsMobile } from '../../../hooks/common/useIsMobile';

const { Text, Title } = Typography;

const PAYMENT_METHOD_MAP = {
  stripe: 'Stripe',
  creem: 'Creem',
  waffo: 'Waffo',
  alipay: '支付宝',
  alipay_f2f: '支付宝当面付',
  wxpay: '微信',
};

const ORDER_STATUS_CONFIG = {
  success: { type: 'success', label: '成功' },
  pending: { type: 'warning', label: '待支付' },
  failed: { type: 'danger', label: '失败' },
  expired: { type: 'danger', label: '已过期' },
};

const REFUND_STATUS_CONFIG = {
  none: { color: 'grey', label: '未退款' },
  partial: { color: 'blue', label: '部分退款' },
  pending: { color: 'orange', label: '退款处理中' },
  full: { color: 'green', label: '已全额退款' },
};

const REFUND_RECORD_STATUS_CONFIG = {
  success: { color: 'green', label: '成功' },
  pending: { color: 'orange', label: '待确认' },
  failed: { color: 'red', label: '失败' },
};

const formatMoney = (value) => `¥${Number(value || 0).toFixed(2)}`;

const statCardStyle = {
  padding: '12px 14px',
  borderRadius: '12px',
  border: '1px solid var(--semi-color-border)',
  background: 'var(--semi-color-bg-1)',
};

const statLabelStyle = {
  display: 'block',
  marginBottom: '4px',
  fontSize: '12px',
  color: 'var(--semi-color-text-2)',
};

const statValueStyle = {
  fontSize: '18px',
  fontWeight: 600,
  color: 'var(--semi-color-text-0)',
};

const renderStatusBadge = (status, t) => {
  const config = ORDER_STATUS_CONFIG[status] || {
    type: 'primary',
    label: status || '-',
  };
  return (
    <span className='flex items-center gap-2'>
      <Badge dot type={config.type} />
      <span>{t(config.label)}</span>
    </span>
  );
};

const renderRefundStatusTag = (status, t) => {
  const config = REFUND_STATUS_CONFIG[status] || {
    color: 'grey',
    label: status || '-',
  };
  return (
    <Tag color={config.color} shape='circle' size='small'>
      {t(config.label)}
    </Tag>
  );
};

const renderRefundRecordStatus = (status, t) => {
  const config = REFUND_RECORD_STATUS_CONFIG[status] || {
    color: 'grey',
    label: status || '-',
  };
  return (
    <Tag color={config.color} shape='circle' size='small'>
      {t(config.label)}
    </Tag>
  );
};

const renderPaymentMethod = (value, t) => {
  const label = PAYMENT_METHOD_MAP[value];
  return <Text>{label ? t(label) : value || '-'}</Text>;
};

const RefundModalContent = ({
  currentOrder,
  refundAmount,
  refundReason,
  setRefundAmount,
  setRefundReason,
  t,
}) => {
  if (!currentOrder) {
    return null;
  }

  return (
    <div className='flex flex-col gap-4'>
      <div
        style={{
          display: 'grid',
          gridTemplateColumns: 'repeat(auto-fit, minmax(160px, 1fr))',
          gap: 12,
        }}
      >
        <div style={statCardStyle}>
          <span style={statLabelStyle}>{t('订单号')}</span>
          <Text copyable ellipsis={{ showTooltip: true }}>
            {currentOrder.trade_no}
          </Text>
        </div>
        <div style={statCardStyle}>
          <span style={statLabelStyle}>{t('支付金额')}</span>
          <span style={statValueStyle}>{formatMoney(currentOrder.money)}</span>
        </div>
        <div style={statCardStyle}>
          <span style={statLabelStyle}>{t('已退款')}</span>
          <span style={statValueStyle}>
            {formatMoney(currentOrder.successful_refund_amount)}
          </span>
        </div>
        <div style={statCardStyle}>
          <span style={statLabelStyle}>{t('可退款')}</span>
          <span style={statValueStyle}>
            {formatMoney(currentOrder.refundable_amount)}
          </span>
        </div>
      </div>

      <div className='flex flex-col gap-2'>
        <Text strong>{t('退款金额')}</Text>
        <Input
          value={refundAmount}
          onChange={setRefundAmount}
          placeholder={t('请输入退款金额，支持两位小数')}
          prefix='¥'
        />
      </div>

      <div className='flex flex-col gap-2'>
        <Text strong>{t('退款原因')}</Text>
        <Input
          value={refundReason}
          onChange={setRefundReason}
          maxLength={256}
          showClear
          placeholder={t('请输入退款原因')}
        />
      </div>

      <Text type='tertiary'>
        {t(
          '同一笔订单退款至少间隔 3 秒，且累计退款金额不能超过原支付金额。若接口返回待确认状态，请稍后查看退款记录。',
        )}
      </Text>
    </div>
  );
};

const TopUpOrdersPage = () => {
  const { t } = useTranslation();
  const isMobile = useIsMobile();
  const navigate = useNavigate();
  const [orders, setOrders] = useState([]);
  const [loading, setLoading] = useState(false);
  const [activePage, setActivePage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [total, setTotal] = useState(0);
  const [keywordInput, setKeywordInput] = useState('');
  const [keyword, setKeyword] = useState('');
  const [actionTradeNo, setActionTradeNo] = useState('');
  const [refundModalVisible, setRefundModalVisible] = useState(false);
  const [refundSubmitting, setRefundSubmitting] = useState(false);
  const [currentOrder, setCurrentOrder] = useState(null);
  const [refundAmount, setRefundAmount] = useState('');
  const [refundReason, setRefundReason] = useState('');
  const [refundRecordsVisible, setRefundRecordsVisible] = useState(false);
  const [refundRecordsLoading, setRefundRecordsLoading] = useState(false);
  const [refundRecords, setRefundRecords] = useState([]);
  const [refundRecordsOrder, setRefundRecordsOrder] = useState(null);

  const loadOrders = useCallback(async () => {
    setLoading(true);
    try {
      const res = await API.get('/api/user/topup', {
        params: {
          p: activePage,
          page_size: pageSize,
          keyword: keyword || undefined,
        },
      });
      const { success, message, data } = res.data;
      if (!success) {
        showError(message || t('加载充值订单失败'));
        return;
      }
      setOrders(data?.items || []);
      setTotal(data?.total || 0);
    } catch (error) {
      showError(error);
    } finally {
      setLoading(false);
    }
  }, [activePage, pageSize, keyword, t]);

  const loadRefundRecords = useCallback(
    async (order) => {
      if (!order?.id) {
        return;
      }

      setRefundRecordsVisible(true);
      setRefundRecordsOrder(order);
      setRefundRecordsLoading(true);
      try {
        const res = await API.get(`/api/user/topup/${order.id}/refunds`);
        const { success, message, data } = res.data;
        if (!success) {
          showError(message || t('加载退款记录失败'));
          return;
        }
        setRefundRecords(data?.items || []);
      } catch (error) {
        showError(error);
      } finally {
        setRefundRecordsLoading(false);
      }
    },
    [t],
  );

  useEffect(() => {
    loadOrders();
  }, [loadOrders]);

  const handleSearch = () => {
    setActivePage(1);
    setKeyword(keywordInput.trim());
  };

  const handleResetSearch = () => {
    setKeywordInput('');
    setKeyword('');
    setActivePage(1);
  };

  const handleRefresh = async () => {
    await loadOrders();
    if (refundRecordsVisible && refundRecordsOrder) {
      await loadRefundRecords(refundRecordsOrder);
    }
  };

  const handleComplete = (order) => {
    Modal.confirm({
      title: t('确认补单'),
      content: t('是否将该订单标记为成功并为用户入账？'),
      onOk: async () => {
        setActionTradeNo(order.trade_no);
        try {
          const res = await API.post('/api/user/topup/complete', {
            trade_no: order.trade_no,
          });
          const { success, message } = res.data;
          if (!success) {
            showError(message || t('补单失败'));
            return;
          }
          showSuccess(t('补单成功'));
          await loadOrders();
        } catch (error) {
          showError(error);
        } finally {
          setActionTradeNo('');
        }
      },
    });
  };

  const openRefundModal = (order) => {
    setCurrentOrder(order);
    setRefundAmount('');
    setRefundReason('');
    setRefundModalVisible(true);
  };

  const closeRefundModal = () => {
    if (refundSubmitting) {
      return;
    }
    setRefundModalVisible(false);
    setCurrentOrder(null);
    setRefundAmount('');
    setRefundReason('');
  };

  const handleSubmitRefund = async () => {
    if (!currentOrder) {
      return;
    }

    const amount = Number(refundAmount);
    if (!Number.isFinite(amount) || amount <= 0) {
      showError(t('请输入正确的退款金额'));
      return;
    }
    if (amount - Number(currentOrder.refundable_amount || 0) > 0.000001) {
      showError(t('退款金额不能超过当前可退款金额'));
      return;
    }

    setRefundSubmitting(true);
    setActionTradeNo(currentOrder.trade_no);
    try {
      const res = await API.post('/api/user/topup/refund', {
        top_up_id: currentOrder.id,
        refund_amount: amount.toFixed(2),
        refund_reason: refundReason.trim(),
      });
      const { success, message, data } = res.data;
      if (!success) {
        showError(message || t('退款失败'));
        return;
      }

      showSuccess(data?.message || message || t('退款请求已提交'));
      setRefundModalVisible(false);
      await loadOrders();
      if (refundRecordsVisible && refundRecordsOrder?.id === currentOrder.id) {
        await loadRefundRecords(currentOrder);
      }
    } catch (error) {
      showError(error);
    } finally {
      setRefundSubmitting(false);
      setActionTradeNo('');
    }
  };

  const columns = useMemo(() => {
    return [
      {
        title: t('用户'),
        dataIndex: 'username',
        key: 'username',
        render: (value, record) => (
          <div className='flex flex-col gap-1'>
            <Text strong>{value || '-'}</Text>
            <Text type='tertiary'>ID: {record.user_id}</Text>
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
        title: t('支付方式'),
        dataIndex: 'payment_method',
        key: 'payment_method',
        render: (value) => renderPaymentMethod(value, t),
      },
      {
        title: t('充值额度'),
        dataIndex: 'amount',
        key: 'amount',
        render: (value) => (
          <span className='flex items-center justify-end gap-1'>
            <Coins size={16} />
            <Text>{value}</Text>
          </span>
        ),
      },
      {
        title: t('支付金额'),
        dataIndex: 'money',
        key: 'money',
        render: (value) => <Text type='danger'>{formatMoney(value)}</Text>,
      },
      {
        title: t('订单状态'),
        dataIndex: 'status',
        key: 'status',
        render: (value) => renderStatusBadge(value, t),
      },
      {
        title: t('退款状态'),
        dataIndex: 'refund_status',
        key: 'refund_status',
        render: (value) => renderRefundStatusTag(value, t),
      },
      {
        title: t('已退款'),
        dataIndex: 'successful_refund_amount',
        key: 'successful_refund_amount',
        render: (value) => formatMoney(value),
      },
      {
        title: t('待确认退款'),
        dataIndex: 'pending_refund_amount',
        key: 'pending_refund_amount',
        render: (value) => formatMoney(value),
      },
      {
        title: t('可退款'),
        dataIndex: 'refundable_amount',
        key: 'refundable_amount',
        render: (value, record) => (
          <div className='flex flex-col gap-1'>
            <Text>{formatMoney(value)}</Text>
            <Text type='tertiary'>
              {t('累计 {{count}} 笔', { count: record.refund_count || 0 })}
            </Text>
          </div>
        ),
      },
      {
        title: t('创建时间'),
        dataIndex: 'create_time',
        key: 'create_time',
        render: (value) => timestamp2string(value),
      },
      {
        title: t('完成时间'),
        dataIndex: 'complete_time',
        key: 'complete_time',
        render: (value) => (value ? timestamp2string(value) : '-'),
      },
      {
        title: t('操作'),
        key: 'operate',
        fixed: 'right',
        width: 240,
        render: (_, record) => (
          <div className='flex flex-wrap justify-end gap-2'>
            {record.status === 'pending' && (
              <Button
                size='small'
                theme='outline'
                type='primary'
                loading={actionTradeNo === record.trade_no && !refundSubmitting}
                onClick={() => handleComplete(record)}
              >
                {t('补单')}
              </Button>
            )}
            <Button
              size='small'
              theme='outline'
              onClick={() => loadRefundRecords(record)}
            >
              {t('退款记录')}
            </Button>
            {record.can_refund && (
              <Button
                size='small'
                type='danger'
                theme='solid'
                loading={actionTradeNo === record.trade_no && refundSubmitting}
                onClick={() => openRefundModal(record)}
              >
                {t('退款')}
              </Button>
            )}
          </div>
        ),
      },
    ];
  }, [
    actionTradeNo,
    loadRefundRecords,
    refundSubmitting,
    t,
  ]);

  const refundRecordColumns = useMemo(() => {
    return [
      {
        title: t('退款单号'),
        dataIndex: 'refund_no',
        key: 'refund_no',
        render: (value) => <Text copyable>{value}</Text>,
      },
      {
        title: t('退款金额'),
        dataIndex: 'refund_amount',
        key: 'refund_amount',
        render: (value) => formatMoney(value),
      },
      {
        title: t('退款状态'),
        dataIndex: 'status',
        key: 'status',
        render: (value) => renderRefundRecordStatus(value, t),
      },
      {
        title: t('退款原因'),
        dataIndex: 'refund_reason',
        key: 'refund_reason',
        render: (value) => value || '-',
      },
      {
        title: t('资金变动'),
        dataIndex: 'fund_change',
        key: 'fund_change',
        render: (value) => {
          if (!value) return '-';
          return value === 'Y' ? t('已退回') : value;
        },
      },
      {
        title: t('返回信息'),
        key: 'response',
        render: (_, record) =>
          record.response_sub_msg || record.response_msg || '-',
      },
      {
        title: t('创建时间'),
        dataIndex: 'create_time',
        key: 'create_time',
        render: (value) => timestamp2string(value),
      },
      {
        title: t('完成时间'),
        dataIndex: 'complete_time',
        key: 'complete_time',
        render: (value) => (value ? timestamp2string(value) : '-'),
      },
    ];
  }, [t]);

  return (
    <>
      <CardPro
        type='type1'
        descriptionArea={
          <div className='flex flex-col gap-3'>
            <div>
              <Title heading={5} style={{ margin: 0 }}>
                {t('充值订单管理')}
              </Title>
              <Text type='tertiary'>
                {t(
                  '管理员可在这里查看全平台充值订单、处理补单，并对支付宝当面付订单发起部分退款或多次退款。',
                )}
              </Text>
            </div>
          </div>
        }
        actionsArea={
          <div className='flex flex-col md:flex-row justify-between items-center gap-2 w-full'>
            <div className='flex items-center gap-2 w-full md:w-auto'>
              <Input
                value={keywordInput}
                onChange={setKeywordInput}
                onEnterPress={handleSearch}
                prefix={<IconSearch />}
                showClear
                placeholder={t('搜索订单号、用户名或用户 ID')}
                style={{ width: isMobile ? '100%' : 260 }}
              />
              <Button type='primary' onClick={handleSearch}>
                {t('搜索')}
              </Button>
              <Button theme='outline' onClick={handleResetSearch}>
                {t('重置')}
              </Button>
            </div>
            <div className='flex items-center gap-2'>
              <Button
                theme='outline'
                type='primary'
                onClick={() => navigate('/console/topup-dashboard')}
              >
                {t('打开金额大屏')}
              </Button>
              <Button
                icon={<IconRefresh />}
                theme='outline'
                onClick={handleRefresh}
              >
                {t('刷新')}
              </Button>
            </div>
          </div>
        }
        paginationArea={createCardProPagination({
          currentPage: activePage,
          pageSize: pageSize,
          total: total,
          onPageChange: setActivePage,
          onPageSizeChange: (size) => {
            setPageSize(size);
            setActivePage(1);
          },
          isMobile: isMobile,
          t: t,
        })}
        t={t}
      >
        <CardTable
          columns={columns}
          dataSource={orders}
          rowKey='id'
          loading={loading}
          scroll={{ x: 'max-content' }}
          hidePagination={true}
          empty={
            <Empty
              image={<IllustrationNoResult style={{ width: 150, height: 150 }} />}
              darkModeImage={
                <IllustrationNoResultDark style={{ width: 150, height: 150 }} />
              }
              description={t('暂无充值订单')}
              style={{ padding: 30 }}
            />
          }
        />
      </CardPro>

      <Modal
        title={t('发起退款')}
        visible={refundModalVisible}
        onCancel={closeRefundModal}
        footer={null}
        size={isMobile ? 'full-width' : 'medium'}
      >
        <RefundModalContent
          currentOrder={currentOrder}
          refundAmount={refundAmount}
          refundReason={refundReason}
          setRefundAmount={setRefundAmount}
          setRefundReason={setRefundReason}
          t={t}
        />
        <div className='flex justify-end gap-2 mt-5'>
          <Button onClick={closeRefundModal} disabled={refundSubmitting}>
            {t('取消')}
          </Button>
          <Button
            type='danger'
            theme='solid'
            loading={refundSubmitting}
            onClick={handleSubmitRefund}
          >
            {t('确认退款')}
          </Button>
        </div>
      </Modal>

      <Modal
        title={t('退款记录')}
        visible={refundRecordsVisible}
        onCancel={() => setRefundRecordsVisible(false)}
        footer={null}
        size={isMobile ? 'full-width' : 'large'}
      >
        <div className='flex flex-col gap-3'>
          {refundRecordsOrder && (
            <div
              className='flex flex-col md:flex-row md:items-center md:justify-between gap-2'
              style={{
                padding: '12px 14px',
                borderRadius: '12px',
                background: 'var(--semi-color-fill-0)',
              }}
            >
              <Text>
                {t('订单号')}:
                {' '}
                <Text copyable>{refundRecordsOrder.trade_no}</Text>
              </Text>
              <Text type='tertiary'>
                {t('已退款 {{amount}} / 可退款 {{refundable}}', {
                  amount: formatMoney(refundRecordsOrder.successful_refund_amount),
                  refundable: formatMoney(refundRecordsOrder.refundable_amount),
                })}
              </Text>
            </div>
          )}

          <CardTable
            columns={refundRecordColumns}
            dataSource={refundRecords}
            rowKey='id'
            loading={refundRecordsLoading}
            hidePagination={true}
            empty={
              <Empty
                image={<IllustrationNoResult style={{ width: 150, height: 150 }} />}
                darkModeImage={
                  <IllustrationNoResultDark style={{ width: 150, height: 150 }} />
                }
                description={t('暂无退款记录')}
                style={{ padding: 30 }}
              />
            }
          />
        </div>
      </Modal>
    </>
  );
};

export default TopUpOrdersPage;
