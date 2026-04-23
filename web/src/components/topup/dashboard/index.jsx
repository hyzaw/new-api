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
  Button,
  Empty,
  Skeleton,
  Tag,
  Typography,
} from '@douyinfe/semi-ui';
import { IconRefresh } from '@douyinfe/semi-icons';
import {
  IllustrationNoResult,
  IllustrationNoResultDark,
} from '@douyinfe/semi-illustrations';
import { VChart } from '@visactor/react-vchart';
import { API, renderQuota, showError, timestamp2string } from '../../../helpers';
import { useIsMobile } from '../../../hooks/common/useIsMobile';

const { Text, Title } = Typography;

const DAYS_OPTIONS = [7, 30, 90];

const PAYMENT_METHOD_LABELS = {
  stripe: 'Stripe',
  creem: 'Creem',
  waffo: 'Waffo',
  alipay: '支付宝',
  alipay_f2f: '支付宝当面付',
  wxpay: '微信',
};

const ORDER_STATUS_LABELS = {
  success: '成功',
  pending: '待支付',
  failed: '失败',
  expired: '已关闭',
};

const REFUND_STATUS_LABELS = {
  success: '退款成功',
  pending: '退款待确认',
  failed: '退款失败',
};

const formatMoney = (value) => `¥${Number(value || 0).toFixed(2)}`;

const pageStyle = {
  display: 'flex',
  flexDirection: 'column',
  gap: 16,
};

const headerCardStyle = {
  padding: '20px 22px',
  borderRadius: '24px',
  border: '1px solid var(--semi-color-border)',
  background:
    'linear-gradient(135deg, rgba(37,99,235,0.08), rgba(16,185,129,0.06) 58%, rgba(245,158,11,0.08))',
};

const statGridStyle = {
  display: 'grid',
  gridTemplateColumns: 'repeat(auto-fit, minmax(180px, 1fr))',
  gap: 12,
};

const statCardStyle = {
  padding: '16px 18px',
  borderRadius: '18px',
  border: '1px solid var(--semi-color-border)',
  background: 'var(--semi-color-bg-1)',
  boxShadow: '0 10px 30px rgba(15, 23, 42, 0.04)',
};

const statLabelStyle = {
  display: 'block',
  marginBottom: '6px',
  fontSize: '12px',
  color: 'var(--semi-color-text-2)',
};

const statValueStyle = {
  fontSize: '24px',
  fontWeight: 700,
  color: 'var(--semi-color-text-0)',
};

const threeColumnGridStyle = (isMobile) => ({
  display: 'grid',
  gridTemplateColumns: isMobile
    ? '1fr'
    : 'repeat(3, minmax(0, 1fr))',
  gap: 16,
});

const twoColumnGridStyle = (isMobile) => ({
  display: 'grid',
  gridTemplateColumns: isMobile
    ? '1fr'
    : 'repeat(2, minmax(0, 1fr))',
  gap: 16,
});

const chartCardStyle = {
  padding: '16px',
  borderRadius: '22px',
  border: '1px solid var(--semi-color-border)',
  background: 'var(--semi-color-bg-1)',
  boxShadow: '0 10px 30px rgba(15, 23, 42, 0.04)',
};

const chartTitleStyle = {
  marginBottom: '10px',
  fontSize: '16px',
  fontWeight: 700,
  color: 'var(--semi-color-text-0)',
};

const chartDescriptionStyle = {
  display: 'block',
  marginBottom: '14px',
  color: 'var(--semi-color-text-2)',
};

const valuableUsersListStyle = (isMobile) => ({
  display: 'grid',
  gridTemplateColumns: isMobile ? '1fr' : 'repeat(2, minmax(0, 1fr))',
  gap: 12,
});

const valuableUserCardStyle = {
  display: 'flex',
  alignItems: 'flex-start',
  gap: 12,
  padding: '14px 16px',
  borderRadius: '18px',
  border: '1px solid var(--semi-color-border)',
  background:
    'linear-gradient(135deg, rgba(37,99,235,0.06), rgba(16,185,129,0.04))',
};

const rankBadgeStyle = {
  minWidth: 36,
  height: 36,
  borderRadius: 999,
  display: 'inline-flex',
  alignItems: 'center',
  justifyContent: 'center',
  fontWeight: 700,
  color: '#fff',
  background: 'linear-gradient(135deg, #2563eb, #0f766e)',
  boxShadow: '0 10px 24px rgba(37,99,235,0.2)',
};

const emptyNode = (t, text, size = 140) => (
  <Empty
    image={<IllustrationNoResult style={{ width: size, height: size }} />}
    darkModeImage={
      <IllustrationNoResultDark style={{ width: size, height: size }} />
    }
    description={t(text)}
    style={{ padding: 24 }}
  />
);

const TopUpDashboard = ({
  compact = false,
  showToolbar = true,
  autoRefresh = false,
}) => {
  const { t } = useTranslation();
  const isMobile = useIsMobile();
  const [statsLoading, setStatsLoading] = useState(false);
  const [statsDays, setStatsDays] = useState(30);
  const [stats, setStats] = useState(null);
  const [lastUpdatedAt, setLastUpdatedAt] = useState(0);

  const loadStats = useCallback(async () => {
    setStatsLoading(true);
    try {
      const res = await API.get('/api/user/topup/stats', {
        params: { days: statsDays },
      });
      const { success, message, data } = res.data;
      if (!success) {
        showError(message || t('加载充值统计失败'));
        return;
      }
      setStats(data || null);
      setLastUpdatedAt(Math.floor(Date.now() / 1000));
    } catch (error) {
      showError(error);
    } finally {
      setStatsLoading(false);
    }
  }, [statsDays, t]);

  useEffect(() => {
    loadStats();
  }, [loadStats]);

  useEffect(() => {
    if (!autoRefresh) {
      return undefined;
    }
    const timer = setInterval(() => {
      loadStats();
    }, 60000);
    return () => clearInterval(timer);
  }, [autoRefresh, loadStats]);

  const overviewCards = useMemo(() => {
    const overview = stats?.overview || {};
    const totalOrders = Number(overview.total_orders || 0);
    const successOrders = Number(overview.success_orders || 0);
    const successRate = totalOrders > 0 ? (successOrders / totalOrders) * 100 : 0;
    const refundRate =
      Number(overview.success_money || 0) > 0
        ? (Number(overview.refunded_money || 0) /
            Number(overview.success_money || 0)) *
          100
        : 0;

    return [
      { label: t('累计订单数'), value: totalOrders },
      { label: t('累计支付金额'), value: formatMoney(overview.total_money) },
      { label: t('成功支付金额'), value: formatMoney(overview.success_money) },
      { label: t('净入账金额'), value: formatMoney(overview.net_money) },
      { label: t('累计已退款'), value: formatMoney(overview.refunded_money) },
      {
        label: t('用户总剩余通用余额'),
        value: renderQuota(Number(overview.total_user_quota || 0)),
      },
      {
        label: t('用户总剩余赠送余额'),
        value: renderQuota(Number(overview.total_user_gift_quota || 0)),
      },
      {
        label: t('成功率'),
        value: `${successRate.toFixed(1)}%`,
      },
      {
        label: t('退款占比'),
        value: `${refundRate.toFixed(1)}%`,
      },
      {
        label: t('退款笔数'),
        value: Number(overview.refund_count || 0),
      },
    ];
  }, [stats, t]);

  const trendSpec = useMemo(() => {
    const values =
      stats?.daily_trend?.flatMap((item) => [
        {
          date: item.date,
          type: t('下单金额'),
          amount: Number(item.total_money || 0),
        },
        {
          date: item.date,
          type: t('成功金额'),
          amount: Number(item.success_money || 0),
        },
        {
          date: item.date,
          type: t('退款金额'),
          amount: Number(item.refunded_money || 0),
        },
      ]) || [];

    return {
      type: 'line',
      height: compact ? 300 : 380,
      data: [{ id: 'topup-dashboard-trend', values }],
      xField: 'date',
      yField: 'amount',
      seriesField: 'type',
      padding: {
        top: 20,
        right: 20,
        bottom: 36,
        left: 52,
      },
      legends: {
        orient: 'top',
      },
      point: {
        visible: true,
        style: {
          size: 6,
        },
      },
      line: {
        style: {
          lineWidth: 3,
        },
      },
      axes: [
        {
          orient: 'bottom',
          label: { visible: true },
        },
        {
          orient: 'left',
          label: { visible: true },
        },
      ],
      tooltip: {
        visible: true,
      },
      color: {
        specified: {
          [t('下单金额')]: '#2563eb',
          [t('成功金额')]: '#16a34a',
          [t('退款金额')]: '#dc2626',
        },
      },
    };
  }, [compact, stats, t]);

  const paymentMethodSpec = useMemo(() => {
    const values =
      stats?.payment_methods?.map((item) => ({
        type: t(PAYMENT_METHOD_LABELS[item.name] || item.name || '-'),
        value: Number(item.amount || 0),
      })) || [];

    return {
      type: 'bar',
      height: compact ? 280 : 320,
      data: [{ id: 'topup-dashboard-payment-methods', values }],
      xField: 'type',
      yField: 'value',
      padding: {
        top: 20,
        right: 20,
        bottom: 36,
        left: 52,
      },
      bar: {
        style: {
          cornerRadius: [8, 8, 0, 0],
        },
      },
      axes: [
        {
          orient: 'bottom',
          label: { visible: true },
        },
        {
          orient: 'left',
          label: { visible: true },
        },
      ],
      tooltip: {
        visible: true,
      },
      color: ['#16a34a'],
    };
  }, [compact, stats, t]);

  const statusSpec = useMemo(() => {
    const values =
      stats?.order_statuses?.map((item) => ({
        status: t(ORDER_STATUS_LABELS[item.name] || item.name || '-'),
        count: Number(item.count || 0),
      })) || [];

    return {
      type: 'bar',
      height: compact ? 280 : 320,
      data: [{ id: 'topup-dashboard-order-statuses', values }],
      xField: 'status',
      yField: 'count',
      padding: {
        top: 20,
        right: 20,
        bottom: 36,
        left: 52,
      },
      bar: {
        style: {
          cornerRadius: [8, 8, 0, 0],
        },
      },
      axes: [
        {
          orient: 'bottom',
          label: { visible: true },
        },
        {
          orient: 'left',
          label: { visible: true },
        },
      ],
      tooltip: {
        visible: true,
      },
      color: ['#f59e0b'],
    };
  }, [compact, stats, t]);

  const countTrendSpec = useMemo(() => {
    const values =
      stats?.daily_trend?.flatMap((item) => [
        {
          date: item.date,
          type: t('下单笔数'),
          count: Number(item.order_count || 0),
        },
        {
          date: item.date,
          type: t('成功笔数'),
          count: Number(item.success_count || 0),
        },
        {
          date: item.date,
          type: t('退款笔数'),
          count: Number(item.refund_count || 0),
        },
      ]) || [];

    return {
      type: 'bar',
      height: compact ? 280 : 340,
      data: [{ id: 'topup-dashboard-count-trend', values }],
      xField: 'date',
      yField: 'count',
      seriesField: 'type',
      padding: {
        top: 20,
        right: 20,
        bottom: 36,
        left: 52,
      },
      legends: {
        orient: 'top',
      },
      bar: {
        style: {
          cornerRadius: [6, 6, 0, 0],
        },
      },
      axes: [
        {
          orient: 'bottom',
          label: { visible: true },
        },
        {
          orient: 'left',
          label: { visible: true },
        },
      ],
      tooltip: {
        visible: true,
      },
      color: {
        specified: {
          [t('下单笔数')]: '#2563eb',
          [t('成功笔数')]: '#16a34a',
          [t('退款笔数')]: '#dc2626',
        },
      },
    };
  }, [compact, stats, t]);

  const paymentCountSpec = useMemo(() => {
    const values =
      stats?.payment_methods?.map((item) => ({
        type: t(PAYMENT_METHOD_LABELS[item.name] || item.name || '-'),
        count: Number(item.count || 0),
      })) || [];

    return {
      type: 'bar',
      height: compact ? 280 : 320,
      data: [{ id: 'topup-dashboard-payment-count', values }],
      xField: 'type',
      yField: 'count',
      padding: {
        top: 20,
        right: 20,
        bottom: 36,
        left: 52,
      },
      bar: {
        style: {
          cornerRadius: [8, 8, 0, 0],
        },
      },
      axes: [
        {
          orient: 'bottom',
          label: { visible: true },
        },
        {
          orient: 'left',
          label: { visible: true },
        },
      ],
      tooltip: {
        visible: true,
      },
      color: ['#2563eb'],
    };
  }, [compact, stats, t]);

  const refundStatusSpec = useMemo(() => {
    const values =
      stats?.refund_statuses?.map((item) => ({
        status: t(REFUND_STATUS_LABELS[item.name] || item.name || '-'),
        count: Number(item.count || 0),
      })) || [];

    return {
      type: 'bar',
      height: compact ? 280 : 320,
      data: [{ id: 'topup-dashboard-refund-status', values }],
      xField: 'status',
      yField: 'count',
      padding: {
        top: 20,
        right: 20,
        bottom: 36,
        left: 52,
      },
      bar: {
        style: {
          cornerRadius: [8, 8, 0, 0],
        },
      },
      axes: [
        {
          orient: 'bottom',
          label: { visible: true },
        },
        {
          orient: 'left',
          label: { visible: true },
        },
      ],
      tooltip: {
        visible: true,
      },
      color: ['#8b5cf6'],
    };
  }, [compact, stats, t]);

  const refundCompositionSpec = useMemo(() => {
    const overview = stats?.overview || {};
    const successMoney = Number(overview.success_money || 0);
    const refundedMoney = Number(overview.refunded_money || 0);
    const pendingRefundMoney = Number(overview.pending_refund_money || 0);
    const retainedMoney = Math.max(
      successMoney - refundedMoney - pendingRefundMoney,
      0,
    );

    const values = [
      {
        type: t('已入账保留'),
        value: retainedMoney,
      },
      {
        type: t('已退款金额'),
        value: refundedMoney,
      },
      {
        type: t('待确认退款'),
        value: pendingRefundMoney,
      },
    ].filter((item) => item.value > 0);

    return {
      type: 'bar',
      height: compact ? 280 : 320,
      data: [{ id: 'topup-dashboard-refund-composition', values }],
      xField: 'type',
      yField: 'value',
      padding: {
        top: 20,
        right: 20,
        bottom: 36,
        left: 52,
      },
      bar: {
        style: {
          cornerRadius: [8, 8, 0, 0],
        },
      },
      axes: [
        {
          orient: 'bottom',
          label: { visible: true },
        },
        {
          orient: 'left',
          label: { visible: true },
        },
      ],
      tooltip: {
        visible: true,
      },
      color: {
        specified: {
          [t('已入账保留')]: '#16a34a',
          [t('已退款金额')]: '#dc2626',
          [t('待确认退款')]: '#f59e0b',
        },
      },
    };
  }, [compact, stats, t]);

  const paymentAverageSpec = useMemo(() => {
    const values =
      stats?.payment_methods?.map((item) => ({
        type: t(PAYMENT_METHOD_LABELS[item.name] || item.name || '-'),
        amount:
          Number(item.count || 0) > 0
            ? Number(item.amount || 0) / Number(item.count || 0)
            : 0,
      })) || [];

    return {
      type: 'bar',
      height: compact ? 280 : 320,
      data: [{ id: 'topup-dashboard-payment-average', values }],
      xField: 'type',
      yField: 'amount',
      padding: {
        top: 20,
        right: 20,
        bottom: 36,
        left: 52,
      },
      bar: {
        style: {
          cornerRadius: [8, 8, 0, 0],
        },
      },
      axes: [
        {
          orient: 'bottom',
          label: { visible: true },
        },
        {
          orient: 'left',
          label: { visible: true },
        },
      ],
      tooltip: {
        visible: true,
      },
      color: ['#0f766e'],
    };
  }, [compact, stats, t]);

  const trendHasData = useMemo(
    () =>
      (stats?.daily_trend || []).some(
        (item) =>
          Number(item.total_money || 0) > 0 ||
          Number(item.success_money || 0) > 0 ||
          Number(item.refunded_money || 0) > 0,
      ),
    [stats],
  );

  const paymentMethodHasData = useMemo(
    () => (stats?.payment_methods || []).length > 0,
    [stats],
  );

  const statusHasData = useMemo(
    () => (stats?.order_statuses || []).length > 0,
    [stats],
  );

  const countTrendHasData = useMemo(
    () =>
      (stats?.daily_trend || []).some(
        (item) =>
          Number(item.order_count || 0) > 0 ||
          Number(item.success_count || 0) > 0 ||
          Number(item.refund_count || 0) > 0,
      ),
    [stats],
  );

  const paymentCountHasData = useMemo(
    () => (stats?.payment_methods || []).some((item) => Number(item.count || 0) > 0),
    [stats],
  );

  const refundStatusHasData = useMemo(
    () => (stats?.refund_statuses || []).length > 0,
    [stats],
  );

  const refundCompositionHasData = useMemo(() => {
    const overview = stats?.overview || {};
    return (
      Number(overview.success_money || 0) > 0 ||
      Number(overview.refunded_money || 0) > 0 ||
      Number(overview.pending_refund_money || 0) > 0
    );
  }, [stats]);

  const paymentAverageHasData = useMemo(
    () => (stats?.payment_methods || []).some((item) => Number(item.count || 0) > 0),
    [stats],
  );

  const valuableUsersHasData = useMemo(
    () => (stats?.valuable_users || []).length > 0,
    [stats],
  );

  const renderValuableUserName = useCallback((item) => {
    if (!item) {
      return '-';
    }
    return item.display_name || item.username || `#${item.user_id || '-'}`;
  }, []);

  return (
    <div style={pageStyle}>
      <div style={headerCardStyle}>
        <div className='flex flex-col gap-4'>
          <div className='flex flex-col md:flex-row md:items-start md:justify-between gap-3'>
            <div className='flex flex-col gap-2'>
              <Title heading={compact ? 4 : 3} style={{ margin: 0 }}>
                {t('充值金额大屏')}
              </Title>
              <Text type='tertiary'>
                {t(
                  '集中展示充值金额、退款金额、订单状态和支付方式分布，适合在后台监控场景持续查看。',
                )}
              </Text>
            </div>
            <div className='flex flex-wrap items-center gap-2'>
              {DAYS_OPTIONS.map((days) => (
                <Button
                  key={days}
                  theme={statsDays === days ? 'solid' : 'outline'}
                  type={statsDays === days ? 'primary' : 'tertiary'}
                  size='small'
                  onClick={() => setStatsDays(days)}
                >
                  {t('近 {{days}} 天', { days })}
                </Button>
              ))}
              {showToolbar && (
                <Button
                  icon={<IconRefresh />}
                  theme='outline'
                  onClick={loadStats}
                  loading={statsLoading}
                >
                  {t('刷新')}
                </Button>
              )}
            </div>
          </div>
          <div className='flex flex-wrap items-center gap-2'>
            <Tag color='blue' shape='circle'>
              {t('统计周期 {{days}} 天', { days: statsDays })}
            </Tag>
            {autoRefresh && (
              <Tag color='green' shape='circle'>
                {t('每 60 秒自动刷新')}
              </Tag>
            )}
            {lastUpdatedAt > 0 && (
              <Text type='tertiary'>
                {t('最近更新')} {timestamp2string(lastUpdatedAt)}
              </Text>
            )}
          </div>
        </div>
      </div>

      {statsLoading ? (
        <Skeleton
          placeholder={
            <div className='flex flex-col gap-3'>
              <Skeleton.Image
                style={{ width: '100%', height: 160, borderRadius: 18 }}
              />
              <Skeleton.Image
                style={{ width: '100%', height: 420, borderRadius: 18 }}
              />
            </div>
          }
          loading={true}
          active
        />
      ) : (
        <>
          <div style={statGridStyle}>
            {overviewCards.map((item) => (
              <div key={item.label} style={statCardStyle}>
                <span style={statLabelStyle}>{item.label}</span>
                <span style={statValueStyle}>{item.value}</span>
              </div>
            ))}
          </div>

          <div style={chartCardStyle}>
            <div className='flex flex-wrap items-center justify-between gap-2'>
              <div style={chartTitleStyle}>{t('高价值充值用户')}</div>
              <Tag color='green' shape='circle'>
                {t('按累计成功充值金额排序')}
              </Tag>
            </div>
            <Text style={chartDescriptionStyle}>
              {t('用于快速识别累计充值贡献最高的用户，榜单按全量历史成功充值统计。')}
            </Text>
            {valuableUsersHasData ? (
              <div style={valuableUsersListStyle(isMobile)}>
                {(stats?.valuable_users || []).map((item, index) => (
                  <div
                    key={`${item.user_id || 'user'}-${index}`}
                    style={valuableUserCardStyle}
                  >
                    <span style={rankBadgeStyle}>#{index + 1}</span>
                    <div className='flex flex-col gap-1' style={{ minWidth: 0, flex: 1 }}>
                      <div className='flex flex-wrap items-center gap-2'>
                        <Text strong>{renderValuableUserName(item)}</Text>
                        {item.username &&
                          item.display_name &&
                          item.display_name !== item.username && (
                            <Text type='tertiary'>@{item.username}</Text>
                          )}
                      </div>
                      <div className='flex flex-wrap items-center gap-3'>
                        <Text>
                          {t('累计成功充值')} {formatMoney(item.total_money)}
                        </Text>
                        <Text type='tertiary'>
                          {t('成功订单')} {Number(item.success_orders || 0)}
                        </Text>
                      </div>
                      <Text type='tertiary'>
                        {t('最近充值')}
                        {item.last_topup_time
                          ? ` ${timestamp2string(item.last_topup_time)}`
                          : ' -'}
                      </Text>
                    </div>
                  </div>
                ))}
              </div>
            ) : (
              emptyNode(t, '暂无高价值充值用户数据', 120)
            )}
          </div>

          <div style={chartCardStyle}>
            <div style={chartTitleStyle}>{t('充值与退款趋势')}</div>
            {trendHasData ? (
              <VChart spec={trendSpec} />
            ) : (
              emptyNode(t, '所选时间范围暂无充值趋势数据', 150)
            )}
          </div>

          <div style={twoColumnGridStyle(isMobile)}>
            <div style={chartCardStyle}>
              <div style={chartTitleStyle}>{t('成功支付方式分布')}</div>
              {paymentMethodHasData ? (
                <VChart spec={paymentMethodSpec} />
              ) : (
                emptyNode(t, '暂无支付方式分布数据', 120)
              )}
            </div>

            <div style={chartCardStyle}>
              <div style={chartTitleStyle}>{t('订单状态分布')}</div>
              {statusHasData ? (
                <VChart spec={statusSpec} />
              ) : (
                emptyNode(t, '暂无订单状态分布数据', 120)
              )}
            </div>
          </div>

          {!compact && (
            <div style={threeColumnGridStyle(isMobile)}>
              <div style={chartCardStyle}>
                <div style={chartTitleStyle}>{t('订单与退款笔数趋势')}</div>
                {countTrendHasData ? (
                  <VChart spec={countTrendSpec} />
                ) : (
                  emptyNode(t, '所选时间范围暂无订单笔数趋势数据', 120)
                )}
              </div>

              <div style={chartCardStyle}>
                <div style={chartTitleStyle}>{t('支付方式笔数分布')}</div>
                {paymentCountHasData ? (
                  <VChart spec={paymentCountSpec} />
                ) : (
                  emptyNode(t, '暂无支付方式笔数分布数据', 120)
                )}
              </div>

              <div style={chartCardStyle}>
                <div style={chartTitleStyle}>{t('退款资金结构')}</div>
                {refundCompositionHasData ? (
                  <VChart spec={refundCompositionSpec} />
                ) : (
                  emptyNode(t, '暂无退款资金结构数据', 120)
                )}
              </div>
            </div>
          )}

          {!compact && (
            <div style={twoColumnGridStyle(isMobile)}>
              <div style={chartCardStyle}>
                <div style={chartTitleStyle}>{t('退款状态分布')}</div>
                {refundStatusHasData ? (
                  <VChart spec={refundStatusSpec} />
                ) : (
                  emptyNode(t, '暂无退款状态分布数据', 120)
                )}
              </div>

              <div style={chartCardStyle}>
                <div style={chartTitleStyle}>{t('支付方式客单价')}</div>
                {paymentAverageHasData ? (
                  <VChart spec={paymentAverageSpec} />
                ) : (
                  emptyNode(t, '暂无支付方式客单价数据', 120)
                )}
              </div>
            </div>
          )}
        </>
      )}
    </div>
  );
};

export default TopUpDashboard;
