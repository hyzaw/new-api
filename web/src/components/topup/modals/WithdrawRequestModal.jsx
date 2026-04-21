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

import React, { useRef } from 'react';
import {
  Button,
  Input,
  InputNumber,
  Modal,
  TextArea,
  Typography,
} from '@douyinfe/semi-ui';
import { showError } from '../../../helpers';

const MAX_RECEIPT_SIZE = 5 * 1024 * 1024;
const { Text } = Typography;

const WithdrawRequestModal = ({
  t,
  visible,
  onCancel,
  onSubmit,
  loading,
  amount,
  setAmount,
  receiptCode,
  setReceiptCode,
  userRemark,
  setUserRemark,
  currencySymbol,
  availableAmountText,
}) => {
  const fileInputRef = useRef(null);

  const handleReceiptFileChange = (e) => {
    const file = e.target.files?.[0];
    if (!file) return;
    if (!file.type?.startsWith('image/')) {
      showError(t('请上传图片格式的收款码'));
      e.target.value = '';
      return;
    }
    if (file.size > MAX_RECEIPT_SIZE) {
      showError(t('收款码图片不能超过 5MB'));
      e.target.value = '';
      return;
    }
    const reader = new FileReader();
    reader.onload = (event) => {
      setReceiptCode(event.target?.result || '');
    };
    reader.readAsDataURL(file);
    e.target.value = '';
  };

  return (
    <Modal
      title={t('申请提现')}
      visible={visible}
      onOk={onSubmit}
      onCancel={onCancel}
      confirmLoading={loading}
      centered
      okText={t('提交申请')}
      cancelText={t('取消')}
    >
      <div className='flex flex-col gap-4'>
        <div className='rounded-xl border border-[var(--semi-color-border)] bg-[var(--semi-color-fill-0)] p-3'>
          <Text type='tertiary'>
            {t('当前可提现邀请余额')}：{availableAmountText}
          </Text>
        </div>

        <div>
          <div className='mb-1'>
            <Text strong>{t('提现金额')}</Text>
          </div>
          <InputNumber
            value={amount}
            onChange={setAmount}
            min={20}
            precision={2}
            step={1}
            prefix={currencySymbol}
            style={{ width: '100%' }}
            placeholder={t('最低提现金额为 20')}
          />
        </div>

        <div>
          <div className='mb-1'>
            <Text strong>{t('收款码')}</Text>
          </div>
          <input
            ref={fileInputRef}
            type='file'
            accept='image/*'
            style={{ display: 'none' }}
            onChange={handleReceiptFileChange}
          />
          <div className='flex items-center gap-2'>
            <Button onClick={() => fileInputRef.current?.click()}>
              {receiptCode ? t('重新上传') : t('上传收款码')}
            </Button>
            {receiptCode && (
              <Button type='danger' theme='outline' onClick={() => setReceiptCode('')}>
                {t('清除')}
              </Button>
            )}
          </div>
          <div className='mt-2'>
            <Text type='tertiary' size='small'>
              {t('支持 PNG / JPG / WebP，单张不超过 5MB')}
            </Text>
          </div>
          {receiptCode && (
            <div className='mt-3 rounded-xl border border-[var(--semi-color-border)] p-3'>
              <img
                src={receiptCode}
                alt='receipt-code'
                className='max-h-56 mx-auto rounded-lg object-contain'
              />
            </div>
          )}
        </div>

        <div>
          <div className='mb-1'>
            <Text strong>{t('备注')}</Text>
          </div>
          <TextArea
            value={userRemark}
            onChange={setUserRemark}
            maxLength={255}
            autosize={{ minRows: 3, maxRows: 5 }}
            placeholder={t('可选，填写收款说明或备注')}
          />
        </div>

        <Input
          readonly
          value={t('管理员审核后会线下处理打款，驳回时会自动退回邀请余额')}
        />
      </div>
    </Modal>
  );
};

export default WithdrawRequestModal;
