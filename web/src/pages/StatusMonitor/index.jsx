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
import { API, showError } from '../../helpers';
import { Card, Empty, Spin, Tag, Tooltip, Typography } from '@douyinfe/semi-ui';
import { useTranslation } from 'react-i18next';

const { Title, Text } = Typography;

const statusColorMap = {
  healthy: '#16a34a',
  warning: '#f59e0b',
  error: '#ef4444',
  no_data: '#cbd5e1',
};

const statusLabelMap = {
  healthy: '绿色',
  warning: '黄色',
  error: '红色',
  no_data: '无数据',
};

const formatDateTime = (timestamp) => {
  if (!timestamp) return '--';
  return new Date(timestamp * 1000).toLocaleString();
};

const formatTime = (timestamp) => {
  if (!timestamp) return '--';
  const date = new Date(timestamp * 1000);
  return `${String(date.getHours()).padStart(2, '0')}:${String(date.getMinutes()).padStart(2, '0')}`;
};

const StatusMonitorPage = () => {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [monitor, setMonitor] = useState(null);

  const fetchMonitor = useCallback(async () => {
    setLoading(true);
    try {
      const res = await API.get('/api/status/monitor');
      const { success, message, data } = res.data;
      if (success) {
        setMonitor(data);
        return;
      }
      showError(message);
    } catch (error) {
      showError(t('加载状态监控失败'));
    } finally {
      setLoading(false);
    }
  }, [t]);

  useEffect(() => {
    fetchMonitor().then();
    const timer = setInterval(() => {
      fetchMonitor().then();
    }, 60 * 1000);
    return () => clearInterval(timer);
  }, [fetchMonitor]);

  const summaryCards = useMemo(() => {
    if (!monitor?.summary) return [];
    return [
      {
        title: t('总请求数'),
        value: monitor.summary.total_count ?? 0,
      },
      {
        title: t('总成功率'),
        value: `${(monitor.summary.success_rate ?? 0).toFixed(1)}%`,
      },
      {
        title: t('绿色点位'),
        value: monitor.summary.healthy_points ?? 0,
      },
      {
        title: t('黄色点位'),
        value: monitor.summary.warning_points ?? 0,
      },
      {
        title: t('红色点位'),
        value: monitor.summary.error_points ?? 0,
      },
    ];
  }, [monitor, t]);

  return (
    <div className='mt-[72px] px-2 md:px-4 pb-6'>
      <Card bordered={false}>
        <div className='flex flex-col gap-3'>
          <div className='flex flex-col gap-2 md:flex-row md:items-end md:justify-between'>
            <div>
              <Title heading={3} style={{ marginBottom: 8 }}>
                {t('状态监控')}
              </Title>
              <Text type='secondary'>
                {t('基于请求日志统计最近 24 小时状态，每个点代表 10 分钟')}
              </Text>
            </div>
            <div className='flex flex-wrap gap-2'>
              <Tag color='green'>{t('>=60% 成功率')}</Tag>
              <Tag color='yellow'>{t('>=30% 成功率')}</Tag>
              <Tag color='red'>{t('<30% 成功率')}</Tag>
              <Tag color='grey'>{t('无请求')}</Tag>
            </div>
          </div>

          <Text type='tertiary'>
            {t('数据快照时间')}：{formatDateTime(monitor?.window_end)}
          </Text>

          <Spin spinning={loading}>
            {!monitor?.points?.length ? (
              <Empty description={t('暂无状态数据')} />
            ) : (
              <div className='flex flex-col gap-4'>
                <div
                  className='grid gap-3'
                  style={{
                    gridTemplateColumns:
                      'repeat(auto-fit, minmax(140px, 1fr))',
                  }}
                >
                  {summaryCards.map((item) => (
                    <Card key={item.title} bodyStyle={{ padding: 16 }}>
                      <Text type='secondary'>{item.title}</Text>
                      <div
                        style={{
                          marginTop: 8,
                          fontSize: 28,
                          fontWeight: 700,
                          lineHeight: 1.1,
                        }}
                      >
                        {item.value}
                      </div>
                    </Card>
                  ))}
                </div>

                <Card
                  title={t('24 小时请求状态')}
                  bodyStyle={{ padding: 16 }}
                >
                  <div className='overflow-x-auto'>
                    <div
                      className='grid gap-2'
                      style={{
                        gridTemplateColumns: 'repeat(24, minmax(18px, 1fr))',
                        minWidth: 620,
                      }}
                    >
                      {monitor.points.map((point) => {
                        const color =
                          statusColorMap[point.status] || statusColorMap.no_data;
                        return (
                          <Tooltip
                            key={`${point.start_time}-${point.end_time}`}
                            content={
                              <div style={{ minWidth: 200 }}>
                                <div>
                                  {formatDateTime(point.start_time)} -{' '}
                                  {formatDateTime(point.end_time)}
                                </div>
                                <div>
                                  {t('成功率')}：{(point.success_rate ?? 0).toFixed(1)}%
                                </div>
                                <div>
                                  {t('成功')}：{point.success_count ?? 0}
                                </div>
                                <div>
                                  {t('失败')}：{point.error_count ?? 0}
                                </div>
                                <div>
                                  {t('状态')}：{t(statusLabelMap[point.status] || '无数据')}
                                </div>
                              </div>
                            }
                          >
                            <div
                              style={{
                                width: '100%',
                                aspectRatio: '1 / 1',
                                minHeight: 18,
                                borderRadius: 6,
                                background: color,
                                boxShadow:
                                  point.status === 'no_data'
                                    ? 'inset 0 0 0 1px rgba(148, 163, 184, 0.4)'
                                    : 'none',
                              }}
                            />
                          </Tooltip>
                        );
                      })}
                    </div>
                  </div>
                  <div className='mt-3 flex items-center justify-between'>
                    <Text type='tertiary'>
                      {formatTime(monitor.window_start)}
                    </Text>
                    <Text type='tertiary'>
                      {formatTime(monitor.window_end)}
                    </Text>
                  </div>
                </Card>
              </div>
            )}
          </Spin>
        </div>
      </Card>
    </div>
  );
};

export default StatusMonitorPage;
