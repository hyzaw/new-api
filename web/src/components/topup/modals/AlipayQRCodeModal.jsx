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
import { Modal, Space, Typography } from '@douyinfe/semi-ui';
import { QRCodeSVG } from 'qrcode.react';

const { Text } = Typography;

const AlipayQRCodeModal = ({
  t,
  visible,
  onCancel,
  qrCode,
  tradeNo,
  amount,
  polling,
}) => {
  return (
    <Modal
      title={t('支付宝扫码支付')}
      visible={visible}
      onCancel={onCancel}
      footer={null}
      maskClosable={false}
      centered
    >
      <Space
        vertical
        align='center'
        spacing={16}
        style={{ width: '100%', textAlign: 'center' }}
      >
        {qrCode ? <QRCodeSVG value={qrCode} size={220} /> : null}
        <Text strong>{t('请使用支付宝扫码完成支付')}</Text>
        <Text type='secondary'>
          {t('支付金额')}：{amount ? `¥${amount}` : '-'}
        </Text>
        <Text type='secondary'>
          {t('订单号')}：{tradeNo}
        </Text>
        <Text type='tertiary'>
          {t('二维码默认有效期约 2 小时，页面会自动检查支付结果。')}
        </Text>
        {polling ? (
          <Text type='warning'>{t('正在等待支付结果...')}</Text>
        ) : null}
      </Space>
    </Modal>
  );
};

export default AlipayQRCodeModal;
