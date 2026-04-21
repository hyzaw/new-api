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
import { Card, Empty, Spin, Tooltip, Typography } from '@douyinfe/semi-ui';
import { useTranslation } from 'react-i18next';

const { Title } = Typography;

const statusColorMap = {
  healthy: '#22c55e',
  warning: '#facc15',
  error: '#ef4444',
  no_data: '#e5e7eb',
};

const formatDateTime = (timestamp) => {
  if (!timestamp) return '--';
  return new Date(timestamp * 1000).toLocaleString();
};

const parseMonitorModelName = (value) => {
  const raw = String(value || '').trim();
  if (!raw) {
    return { groupName: 'default', modelName: '(unknown)' };
  }

  const separatorIndex = raw.indexOf('-');
  if (separatorIndex <= 0) {
    return { groupName: 'default', modelName: raw };
  }

  return {
    groupName: raw.slice(0, separatorIndex) || 'default',
    modelName: raw.slice(separatorIndex + 1) || '(unknown)',
  };
};

const groupMonitorModels = (models) => {
  const groups = [];
  const groupMap = new Map();

  (models || []).forEach((item) => {
    const fullName = item?.model_name || '';
    const { groupName, modelName } = parseMonitorModelName(fullName);
    let group = groupMap.get(groupName);

    if (!group) {
      group = {
        groupName,
        items: [],
      };
      groupMap.set(groupName, group);
      groups.push(group);
    }

    group.items.push({
      ...item,
      groupName,
      fullName,
      modelName,
    });
  });

  groups.sort((a, b) => {
    if (a.groupName === 'default') return -1;
    if (b.groupName === 'default') return 1;
    return a.groupName.localeCompare(b.groupName);
  });

  return groups;
};

const StatusDotsRow = ({ title, points }) => {
  return (
    <div className='grid grid-cols-1 items-center gap-3 md:grid-cols-[180px_minmax(0,1fr)] md:gap-4'>
      <div style={{ minWidth: 0 }}>
        <div
          style={{
            fontSize: 15,
            fontWeight: 700,
            color: 'var(--semi-color-text-0)',
            whiteSpace: 'nowrap',
            overflow: 'hidden',
            textOverflow: 'ellipsis',
          }}
        >
          {title}
        </div>
      </div>

      <div style={{ minWidth: 0, overflow: 'hidden' }}>
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'flex-end',
            gap: 3,
            width: 'max-content',
            maxWidth: '100%',
            marginLeft: 'auto',
          }}
        >
          {points.map((point) => {
            const color = statusColorMap[point.status] || statusColorMap.no_data;
            return (
              <Tooltip
                key={`${title}-${point.start_time}`}
                content={
                  <div style={{ minWidth: 220 }}>
                    <div>{title}</div>
                    <div>{formatDateTime(point.start_time)}</div>
                    <div>{formatDateTime(point.end_time)}</div>
                  </div>
                }
              >
                <div
                  style={{
                    flex: '0 0 auto',
                    width: 6,
                    minHeight: 18,
                    height: 18,
                    borderRadius: 999,
                    background: color,
                    boxShadow:
                      point.status === 'no_data'
                        ? 'inset 0 0 0 1px rgba(148, 163, 184, 0.35)'
                        : 'none',
                  }}
                />
              </Tooltip>
            );
          })}
        </div>
      </div>
    </div>
  );
};

const StatusMonitorPage = () => {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [monitor, setMonitor] = useState(null);
  const groupedModels = useMemo(
    () => groupMonitorModels(monitor?.models),
    [monitor?.models],
  );

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

  return (
    <div className='mt-[72px] px-2 md:px-4 pb-6'>
      <Card bordered={false}>
        <div className='flex flex-col gap-4'>
          <div className='flex flex-col gap-2 md:flex-row md:items-end md:justify-between'>
            <div>
              <Title heading={3} style={{ marginBottom: 8 }}>
                {t('状态监控')}
              </Title>
            </div>
          </div>

          <Spin spinning={loading}>
            {!monitor?.models?.length ? (
              <Empty description={t('暂无状态数据')} />
            ) : (
              <div className='flex flex-col gap-4'>
                <Card bodyStyle={{ padding: 16 }}>
                  <StatusDotsRow
                    title={t('整体状态')}
                    points={monitor.points || []}
                  />
                </Card>

                {groupedModels.map((group) => (
                  <Card
                    key={group.groupName}
                    bodyStyle={{ padding: 16 }}
                    title={
                      <div
                        style={{
                          fontSize: 18,
                          fontWeight: 700,
                          color: 'var(--semi-color-text-0)',
                        }}
                      >
                        {group.groupName}
                      </div>
                    }
                  >
                    <div className='flex flex-col gap-4'>
                      {group.items.map((item) => (
                        <StatusDotsRow
                          key={item.fullName}
                          title={item.modelName}
                          points={item.points || []}
                        />
                      ))}
                    </div>
                  </Card>
                ))}
              </div>
            )}
          </Spin>
        </div>
      </Card>
    </div>
  );
};

export default StatusMonitorPage;
