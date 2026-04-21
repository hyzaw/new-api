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
  Input,
  Modal,
  Select,
  Tag,
  TextArea,
  Typography,
} from '@douyinfe/semi-ui';
import { IconRefresh, IconSearch } from '@douyinfe/semi-icons';
import {
  IllustrationNoResult,
  IllustrationNoResultDark,
} from '@douyinfe/semi-illustrations';
import CardPro from '../../common/ui/CardPro';
import CardTable from '../../common/ui/CardTable';
import { createCardProPagination } from '../../../helpers/utils';
import {
  API,
  getCurrencyConfig,
  renderQuota,
  showError,
  showSuccess,
  timestamp2string,
} from '../../../helpers';
import { useIsMobile } from '../../../hooks/common/useIsMobile';

const { Text, Title } = Typography;

const STATUS_CONFIG = {
  pending: { color: 'orange', label: '待处理' },
  paid: { color: 'green', label: '已打款' },
  rejected: { color: 'red', label: '已驳回' },
};

const renderStatusTag = (status, t) => {
  const config = STATUS_CONFIG[status] || {
    color: 'grey',
    label: status || '-',
  };
  return (
    <Tag color={config.color} shape='circle' size='small'>
      {t(config.label)}
    </Tag>
  );
};

const InviteWithdrawalsPage = () => {
  const { t } = useTranslation();
  const isMobile = useIsMobile();
  const { symbol } = getCurrencyConfig();
  const [items, setItems] = useState([]);
  const [loading, setLoading] = useState(false);
  const [activePage, setActivePage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [total, setTotal] = useState(0);
  const [keywordInput, setKeywordInput] = useState('');
  const [keyword, setKeyword] = useState('');
  const [status, setStatus] = useState('');
  const [previewImage, setPreviewImage] = useState('');
  const [reviewVisible, setReviewVisible] = useState(false);
  const [reviewSubmitting, setReviewSubmitting] = useState(false);
  const [currentItem, setCurrentItem] = useState(null);
  const [reviewAction, setReviewAction] = useState('paid');
  const [adminRemark, setAdminRemark] = useState('');

  const loadItems = useCallback(async () => {
    setLoading(true);
    try {
      const res = await API.get('/api/user/aff_withdrawals', {
        params: {
          p: activePage,
          page_size: pageSize,
          keyword: keyword || undefined,
          status: status || undefined,
        },
      });
      const { success, message, data } = res.data;
      if (!success) {
        showError(message || t('加载提现申请失败'));
        return;
      }
      setItems(data?.items || []);
      setTotal(data?.total || 0);
    } catch (error) {
      showError(error);
    } finally {
      setLoading(false);
    }
  }, [activePage, keyword, pageSize, status, t]);

  useEffect(() => {
    loadItems();
  }, [loadItems]);

  const handleSearch = () => {
    setActivePage(1);
    setKeyword(keywordInput.trim());
  };

  const handleReset = () => {
    setKeywordInput('');
    setKeyword('');
    setStatus('');
    setActivePage(1);
  };

  const openReviewModal = (item, action) => {
    setCurrentItem(item);
    setReviewAction(action);
    setAdminRemark('');
    setReviewVisible(true);
  };

  const closeReviewModal = () => {
    if (reviewSubmitting) return;
    setReviewVisible(false);
    setCurrentItem(null);
    setAdminRemark('');
    setReviewAction('paid');
  };

  const handleReview = async () => {
    if (!currentItem?.id) return;
    setReviewSubmitting(true);
    try {
      const res = await API.post(
        `/api/user/aff_withdrawals/${currentItem.id}/review`,
        {
          action: reviewAction,
          admin_remark: adminRemark.trim(),
        },
      );
      const { success, message } = res.data;
      if (!success) {
        showError(message || t('处理提现申请失败'));
        return;
      }
      showSuccess(
        reviewAction === 'paid' ? t('已标记为完成打款') : t('已驳回提现申请'),
      );
      closeReviewModal();
      await loadItems();
    } catch (error) {
      showError(error);
    } finally {
      setReviewSubmitting(false);
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
        render: (value) => renderStatusTag(value, t),
      },
      {
        title: t('用户备注'),
        dataIndex: 'user_remark',
        key: 'user_remark',
        render: (value) => value || '-',
      },
      {
        title: t('管理员备注'),
        dataIndex: 'admin_remark',
        key: 'admin_remark',
        render: (value) => value || '-',
      },
      {
        title: t('申请时间'),
        dataIndex: 'created_at',
        key: 'created_at',
        render: (value) => timestamp2string(value),
      },
      {
        title: t('处理时间'),
        dataIndex: 'processed_at',
        key: 'processed_at',
        render: (value) => (value ? timestamp2string(value) : '-'),
      },
      {
        title: t('收款码'),
        key: 'receipt_code',
        render: (_, record) => (
          <Button size='small' theme='outline' onClick={() => setPreviewImage(record.receipt_code)}>
            {t('查看')}
          </Button>
        ),
      },
      {
        title: t('操作'),
        key: 'operate',
        fixed: 'right',
        width: 180,
        render: (_, record) => (
          <div className='flex flex-wrap justify-end gap-2'>
            {record.status === 'pending' ? (
              <>
                <Button
                  size='small'
                  type='primary'
                  theme='solid'
                  onClick={() => openReviewModal(record, 'paid')}
                >
                  {t('标记打款')}
                </Button>
                <Button
                  size='small'
                  type='danger'
                  theme='outline'
                  onClick={() => openReviewModal(record, 'rejected')}
                >
                  {t('驳回')}
                </Button>
              </>
            ) : (
              <Text type='tertiary'>-</Text>
            )}
          </div>
        ),
      },
    ];
  }, [symbol, t]);

  return (
    <>
      <CardPro
        type='type1'
        descriptionArea={
          <div>
            <Title heading={5} style={{ margin: 0 }}>
              {t('邀请提现管理')}
            </Title>
            <Text type='tertiary'>
              {t('查看用户提交的邀请提现申请，核对收款码后手动完成打款或驳回。')}
            </Text>
          </div>
        }
        actionsArea={
          <div className='flex flex-col md:flex-row justify-between items-center gap-2 w-full'>
            <div className='flex flex-col md:flex-row items-center gap-2 w-full md:w-auto'>
              <Input
                value={keywordInput}
                onChange={setKeywordInput}
                onEnterPress={handleSearch}
                prefix={<IconSearch />}
                showClear
                placeholder={t('搜索用户名或用户 ID')}
                style={{ width: isMobile ? '100%' : 240 }}
              />
              <Select
                value={status}
                onChange={(value) => {
                  setStatus(value || '');
                  setActivePage(1);
                }}
                placeholder={t('全部状态')}
                style={{ width: isMobile ? '100%' : 160 }}
                optionList={[
                  { label: t('全部状态'), value: '' },
                  { label: t('待处理'), value: 'pending' },
                  { label: t('已打款'), value: 'paid' },
                  { label: t('已驳回'), value: 'rejected' },
                ]}
              />
              <Button type='primary' onClick={handleSearch}>
                {t('搜索')}
              </Button>
              <Button theme='outline' onClick={handleReset}>
                {t('重置')}
              </Button>
            </div>
            <Button icon={<IconRefresh />} theme='outline' onClick={loadItems}>
              {t('刷新')}
            </Button>
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
          dataSource={items}
          rowKey='id'
          loading={loading}
          hidePagination={true}
          scroll={{ x: 'max-content' }}
          empty={
            <Empty
              image={<IllustrationNoResult style={{ width: 150, height: 150 }} />}
              darkModeImage={
                <IllustrationNoResultDark style={{ width: 150, height: 150 }} />
              }
              description={t('暂无提现申请')}
              style={{ padding: 30 }}
            />
          }
        />
      </CardPro>

      <Modal
        title={reviewAction === 'paid' ? t('确认已打款') : t('确认驳回')}
        visible={reviewVisible}
        onCancel={closeReviewModal}
        onOk={handleReview}
        confirmLoading={reviewSubmitting}
        okText={reviewAction === 'paid' ? t('确认打款') : t('确认驳回')}
        cancelText={t('取消')}
        size={isMobile ? 'full-width' : 'medium'}
      >
        {currentItem && (
          <div className='flex flex-col gap-4'>
            <div className='grid grid-cols-1 md:grid-cols-2 gap-3'>
              <div className='rounded-xl border border-[var(--semi-color-border)] p-3'>
                <Text type='tertiary'>{t('用户')}</Text>
                <div className='mt-1 font-medium'>{currentItem.username}</div>
              </div>
              <div className='rounded-xl border border-[var(--semi-color-border)] p-3'>
                <Text type='tertiary'>{t('提现金额')}</Text>
                <div className='mt-1 font-medium'>
                  {symbol}
                  {Number(currentItem.amount || 0).toFixed(2)}
                </div>
              </div>
            </div>

            <div className='rounded-xl border border-[var(--semi-color-border)] p-3'>
              <Text type='tertiary'>{t('收款码')}</Text>
              <img
                src={currentItem.receipt_code}
                alt='receipt-code'
                className='mt-3 max-h-72 w-full rounded-lg object-contain'
              />
            </div>

            <div>
              <div className='mb-1'>
                <Text strong>{t('管理员备注')}</Text>
              </div>
              <TextArea
                value={adminRemark}
                onChange={setAdminRemark}
                maxLength={255}
                autosize={{ minRows: 3, maxRows: 5 }}
                placeholder={t('可选，填写处理说明')}
              />
            </div>
          </div>
        )}
      </Modal>

      <Modal
        title={t('收款码预览')}
        visible={!!previewImage}
        footer={null}
        onCancel={() => setPreviewImage('')}
        size={isMobile ? 'full-width' : 'medium'}
      >
        {previewImage && (
          <img
            src={previewImage}
            alt='receipt-preview'
            className='max-h-[70vh] w-full rounded-lg object-contain'
          />
        )}
      </Modal>
    </>
  );
};

export default InviteWithdrawalsPage;
