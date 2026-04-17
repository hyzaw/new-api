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
import { API, showError, showSuccess } from '../../../helpers';
import { useTranslation } from 'react-i18next';

export default function SettingsPaymentTopupNotifyFeishu(props) {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [inputs, setInputs] = useState({
    TopupNotifyFeishuEnabled: false,
    TopupNotifyFeishuAppID: '',
    TopupNotifyFeishuAppSecret: '',
    TopupNotifyFeishuChatID: '',
  });
  const formApiRef = useRef(null);

  useEffect(() => {
    if (props.options && formApiRef.current) {
      const currentInputs = {
        TopupNotifyFeishuEnabled:
          props.options.TopupNotifyFeishuEnabled === 'true' ||
          props.options.TopupNotifyFeishuEnabled === true,
        TopupNotifyFeishuAppID: props.options.TopupNotifyFeishuAppID || '',
        TopupNotifyFeishuAppSecret: '',
        TopupNotifyFeishuChatID: props.options.TopupNotifyFeishuChatID || '',
      };
      setInputs(currentInputs);
      formApiRef.current.setValues(currentInputs);
    }
  }, [props.options]);

  const submitFeishuSetting = async () => {
    if (
      inputs.TopupNotifyFeishuEnabled &&
      (!inputs.TopupNotifyFeishuAppID || !inputs.TopupNotifyFeishuChatID)
    ) {
      showError(t('启用飞书充值通知前，请先填写 App ID 和群 Chat ID'));
      return;
    }

    setLoading(true);
    try {
      const options = [
        {
          key: 'TopupNotifyFeishuAppID',
          value: inputs.TopupNotifyFeishuAppID || '',
        },
        {
          key: 'TopupNotifyFeishuChatID',
          value: inputs.TopupNotifyFeishuChatID || '',
        },
      ];

      if (inputs.TopupNotifyFeishuAppSecret) {
        options.push({
          key: 'TopupNotifyFeishuAppSecret',
          value: inputs.TopupNotifyFeishuAppSecret,
        });
      }

      options.push({
        key: 'TopupNotifyFeishuEnabled',
        value: inputs.TopupNotifyFeishuEnabled ? 'true' : 'false',
      });

      const results = [];
      for (const opt of options) {
        const result = await API.put('/api/option/', {
          key: opt.key,
          value: opt.value,
        });
        results.push(result);
      }

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
        <Form.Section text={t('充值飞书群通知')}>
          <Banner
            type='info'
            description={t(
              '当用户充值成功后，系统会使用飞书应用向指定群发送交互式卡片通知。请先将应用机器人加入目标群，并确保应用具备发送群消息权限。',
            )}
          />
          <Row gutter={{ xs: 8, sm: 16, md: 24, lg: 24, xl: 24, xxl: 24 }}>
            <Col xs={24} sm={24} md={8} lg={8} xl={8}>
              <Form.Switch
                field='TopupNotifyFeishuEnabled'
                label={t('启用飞书充值通知')}
                checkedText='｜'
                uncheckedText='〇'
              />
            </Col>
            <Col xs={24} sm={24} md={8} lg={8} xl={8}>
              <Form.Input
                field='TopupNotifyFeishuAppID'
                label={t('飞书 App ID')}
                placeholder='cli_xxxxxxxxxxxxx'
              />
            </Col>
            <Col xs={24} sm={24} md={8} lg={8} xl={8}>
              <Form.Input
                field='TopupNotifyFeishuChatID'
                label={t('目标群 Chat ID')}
                placeholder='oc_xxxxxxxxxxxxx'
              />
            </Col>
          </Row>
          <Row gutter={{ xs: 8, sm: 16, md: 24, lg: 24, xl: 24, xxl: 24 }}>
            <Col xs={24} sm={24} md={12} lg={12} xl={12}>
              <Form.Input
                field='TopupNotifyFeishuAppSecret'
                label={t('飞书 App Secret')}
                placeholder={t('敏感信息不会回显；留空表示保留当前已保存的 Secret')}
                type='password'
              />
            </Col>
          </Row>
          <Button onClick={submitFeishuSetting}>
            {t('更新飞书充值通知设置')}
          </Button>
        </Form.Section>
      </Form>
    </Spin>
  );
}
