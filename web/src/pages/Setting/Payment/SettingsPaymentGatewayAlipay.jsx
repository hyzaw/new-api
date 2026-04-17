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

import React, { useEffect, useRef, useState } from 'react';
import { Banner, Button, Col, Form, Row, Spin } from '@douyinfe/semi-ui';
import {
  API,
  removeTrailingSlash,
  showError,
  showSuccess,
} from '../../../helpers';
import { useTranslation } from 'react-i18next';

export default function SettingsPaymentGatewayAlipay(props) {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [inputs, setInputs] = useState({
    AlipayF2FEnabled: false,
    AlipaySandbox: false,
    AlipayAppID: '',
    AlipayPrivateKey: '',
    AlipayPublicKey: '',
    AlipayAppAuthToken: '',
    AlipaySellerID: '',
    AlipayNotifyURL: '',
    AlipayProductCode: 'FACE_TO_FACE_PAYMENT',
  });
  const formApiRef = useRef(null);

  useEffect(() => {
    if (props.options && formApiRef.current) {
      const currentInputs = {
        AlipayF2FEnabled:
          props.options.AlipayF2FEnabled === 'true' ||
          props.options.AlipayF2FEnabled === true,
        AlipaySandbox:
          props.options.AlipaySandbox === 'true' ||
          props.options.AlipaySandbox === true,
        AlipayAppID: props.options.AlipayAppID || '',
        AlipayPrivateKey: '',
        AlipayPublicKey: '',
        AlipayAppAuthToken: '',
        AlipaySellerID: props.options.AlipaySellerID || '',
        AlipayNotifyURL: props.options.AlipayNotifyURL || '',
        AlipayProductCode:
          props.options.AlipayProductCode || 'FACE_TO_FACE_PAYMENT',
      };
      setInputs(currentInputs);
      formApiRef.current.setValues(currentInputs);
    }
  }, [props.options]);

  const submitAlipaySetting = async () => {
    setLoading(true);
    try {
      const options = [
        {
          key: 'AlipayF2FEnabled',
          value: inputs.AlipayF2FEnabled ? 'true' : 'false',
        },
        {
          key: 'AlipaySandbox',
          value: inputs.AlipaySandbox ? 'true' : 'false',
        },
        {
          key: 'AlipayAppID',
          value: inputs.AlipayAppID || '',
        },
        {
          key: 'AlipaySellerID',
          value: inputs.AlipaySellerID || '',
        },
        {
          key: 'AlipayNotifyURL',
          value: removeTrailingSlash(inputs.AlipayNotifyURL || ''),
        },
        {
          key: 'AlipayProductCode',
          value: inputs.AlipayProductCode || 'FACE_TO_FACE_PAYMENT',
        },
      ];

      if (inputs.AlipayPrivateKey) {
        options.push({
          key: 'AlipayPrivateKey',
          value: inputs.AlipayPrivateKey,
        });
      }
      if (inputs.AlipayPublicKey) {
        options.push({
          key: 'AlipayPublicKey',
          value: inputs.AlipayPublicKey,
        });
      }
      if (inputs.AlipayAppAuthToken) {
        options.push({
          key: 'AlipayAppAuthToken',
          value: inputs.AlipayAppAuthToken,
        });
      }

      const results = await Promise.all(
        options.map((opt) =>
          API.put('/api/option/', {
            key: opt.key,
            value: opt.value,
          }),
        ),
      );

      const failed = results.filter((res) => !res.data.success);
      if (failed.length > 0) {
        failed.forEach((res) => showError(res.data.message));
      } else {
        showSuccess(t('更新成功'));
        props.refresh?.();
      }
    } catch (error) {
      showError(t('更新失败'));
    } finally {
      setLoading(false);
    }
  };

  return (
    <Spin spinning={loading}>
      <Form
        initValues={inputs}
        onValueChange={setInputs}
        getFormApi={(api) => (formApiRef.current = api)}
      >
        <Form.Section text={t('支付宝当面付设置')}>
          <Banner
            type='info'
            description={t(
              '该配置会启用 alipay.trade.precreate 扫码支付。后端会通过异步通知确认到账，并使用交易查询作为辅助兜底。',
            )}
          />
          <Banner
            type='warning'
            description={`notify_url: ${
              props.options.AlipayNotifyURL
                ? removeTrailingSlash(props.options.AlipayNotifyURL)
                : props.options.ServerAddress
                  ? `${removeTrailingSlash(props.options.ServerAddress)}/api/alipay/f2f/notify`
                  : '/api/alipay/f2f/notify'
            }`}
          />
          <Row gutter={{ xs: 8, sm: 16, md: 24, lg: 24, xl: 24, xxl: 24 }}>
            <Col xs={24} sm={24} md={8} lg={8} xl={8}>
              <Form.Switch
                field='AlipayF2FEnabled'
                label={t('启用支付宝当面付')}
                checkedText='｜'
                uncheckedText='〇'
              />
            </Col>
            <Col xs={24} sm={24} md={8} lg={8} xl={8}>
              <Form.Switch
                field='AlipaySandbox'
                label={t('沙盒模式')}
                checkedText='｜'
                uncheckedText='〇'
              />
            </Col>
            <Col xs={24} sm={24} md={8} lg={8} xl={8}>
              <Form.Input
                field='AlipayAppID'
                label={t('App ID')}
                placeholder='2021001135664944'
              />
            </Col>
          </Row>
          <Row gutter={{ xs: 8, sm: 16, md: 24, lg: 24, xl: 24, xxl: 24 }}>
            <Col xs={24} sm={24} md={12} lg={12} xl={12}>
              <Form.TextArea
                field='AlipayPrivateKey'
                label={t('应用私钥')}
                placeholder={t('支持 PEM 或 Base64 PKCS8，敏感信息不会回显')}
                autosize={{ minRows: 3, maxRows: 6 }}
              />
            </Col>
            <Col xs={24} sm={24} md={12} lg={12} xl={12}>
              <Form.TextArea
                field='AlipayPublicKey'
                label={t('支付宝公钥')}
                placeholder={t('支持 PEM、证书公钥或 Base64 公钥')}
                autosize={{ minRows: 3, maxRows: 6 }}
              />
            </Col>
          </Row>
          <Row gutter={{ xs: 8, sm: 16, md: 24, lg: 24, xl: 24, xxl: 24 }}>
            <Col xs={24} sm={24} md={8} lg={8} xl={8}>
              <Form.Input
                field='AlipayAppAuthToken'
                label={t('App Auth Token')}
                placeholder={t('第三方代理调用可选')}
              />
            </Col>
            <Col xs={24} sm={24} md={8} lg={8} xl={8}>
              <Form.Input
                field='AlipaySellerID'
                label={t('Seller ID')}
                placeholder={t('收款支付宝用户 ID，可选')}
              />
            </Col>
            <Col xs={24} sm={24} md={8} lg={8} xl={8}>
              <Form.Input
                field='AlipayProductCode'
                label={t('Product Code')}
                placeholder='FACE_TO_FACE_PAYMENT'
              />
            </Col>
          </Row>
          <Row gutter={{ xs: 8, sm: 16, md: 24, lg: 24, xl: 24, xxl: 24 }}>
            <Col xs={24} sm={24} md={12} lg={12} xl={12}>
              <Form.Input
                field='AlipayNotifyURL'
                label={t('异步通知地址')}
                placeholder={t(
                  '留空则自动使用服务器地址 + /api/alipay/f2f/notify',
                )}
              />
            </Col>
          </Row>
          <Button onClick={submitAlipaySetting}>
            {t('更新支付宝当面付设置')}
          </Button>
        </Form.Section>
      </Form>
    </Spin>
  );
}
