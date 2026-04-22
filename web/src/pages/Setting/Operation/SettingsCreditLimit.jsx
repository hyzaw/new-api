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

import React, { useEffect, useState, useRef } from 'react';
import { Button, Col, Form, Row, Spin, Select } from '@douyinfe/semi-ui';
import { useTranslation } from 'react-i18next';
import {
  compareObjects,
  API,
  showError,
  showSuccess,
  showWarning,
} from '../../../helpers';

export default function SettingsCreditLimit(props) {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [inputs, setInputs] = useState({
    QuotaForNewUser: '',
    PreConsumedQuota: '',
    QuotaForInviter: '',
    QuotaForInvitee: '',
    TopUpAffRatio: '',
    'gift_quota_setting.rules': '[]',
    'quota_setting.enable_free_model_pre_consume': true,
  });
  const refForm = useRef();
  const [inputsRow, setInputsRow] = useState(inputs);
  const [giftQuotaRules, setGiftQuotaRules] = useState([]);
  const [groupOptions, setGroupOptions] = useState([]);
  const [modelOptions, setModelOptions] = useState([]);

  const normalizeSelectValue = (item) => {
    if (typeof item === 'string' || typeof item === 'number') {
      return String(item).trim();
    }
    if (item && typeof item === 'object') {
      return String(item.id || item.value || item.label || item.name || '').trim();
    }
    return '';
  };

  const buildSelectOptions = (items = []) => {
    const uniqueItems = new Set(['*']);
    for (const item of items) {
      const normalized = normalizeSelectValue(item);
      if (normalized) {
        uniqueItems.add(normalized);
      }
    }
    return Array.from(uniqueItems).map((item) => ({
      label: item,
      value: item,
    }));
  };

  const normalizeGiftQuotaRules = (rules = []) => {
    if (!Array.isArray(rules)) {
      return [];
    }
    return rules.map((rule) => ({
      group: String(rule?.group || '*').trim() || '*',
      model: String(rule?.model || '*').trim() || '*',
    }));
  };

  const parseGiftQuotaRules = (raw) => {
    if (!raw) {
      return [];
    }
    try {
      return normalizeGiftQuotaRules(JSON.parse(raw));
    } catch {
      return [];
    }
  };

  const serializeGiftQuotaRules = (rules = []) =>
    JSON.stringify(normalizeGiftQuotaRules(rules));

  const updateGiftQuotaRules = (rules) => {
    const normalizedRules = normalizeGiftQuotaRules(rules);
    setGiftQuotaRules(normalizedRules);
    setInputs((prev) => ({
      ...prev,
      'gift_quota_setting.rules': serializeGiftQuotaRules(normalizedRules),
    }));
  };

  const fetchOptions = async () => {
    try {
      const [groupsRes, modelsRes] = await Promise.all([
        API.get('/api/group/'),
        API.get('/api/channel/models'),
      ]);
      setGroupOptions(buildSelectOptions(groupsRes?.data?.data || []));
      setModelOptions(buildSelectOptions(modelsRes?.data?.data || []));
    } catch (error) {
      showError(t('加载赠送余额规则选项失败'));
    }
  };

  function onSubmit() {
    const updateArray = compareObjects(inputs, inputsRow);
    if (!updateArray.length) return showWarning(t('你似乎并没有修改什么'));
    const requestQueue = updateArray.map((item) => {
      let value = '';
      if (typeof inputs[item.key] === 'boolean') {
        value = String(inputs[item.key]);
      } else {
        value = inputs[item.key];
      }
      return API.put('/api/option/', {
        key: item.key,
        value,
      });
    });
    setLoading(true);
    Promise.all(requestQueue)
      .then((res) => {
        if (requestQueue.length === 1) {
          if (res.includes(undefined)) return;
        } else if (requestQueue.length > 1) {
          if (res.includes(undefined))
            return showError(t('部分保存失败，请重试'));
        }
        showSuccess(t('保存成功'));
        props.refresh();
      })
      .catch(() => {
        showError(t('保存失败，请重试'));
      })
      .finally(() => {
        setLoading(false);
      });
  }

  useEffect(() => {
    const currentInputs = {};
    for (let key in props.options) {
      if (Object.keys(inputs).includes(key)) {
        currentInputs[key] = props.options[key];
      }
    }
    currentInputs['gift_quota_setting.rules'] =
      props.options['gift_quota_setting.rules'] || '[]';
    const nextGiftQuotaRules = parseGiftQuotaRules(
      props.options['gift_quota_setting.rules'],
    );
    setInputs(currentInputs);
    setInputsRow(structuredClone(currentInputs));
    setGiftQuotaRules(nextGiftQuotaRules);
    refForm.current.setValues(currentInputs);
  }, [props.options]);

  useEffect(() => {
    fetchOptions();
  }, []);
  return (
    <>
      <Spin spinning={loading}>
        <Form
          values={inputs}
          getFormApi={(formAPI) => (refForm.current = formAPI)}
          style={{ marginBottom: 15 }}
        >
          <Form.Section text={t('额度设置')}>
            <Row gutter={16}>
              <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                <Form.InputNumber
                  label={t('新用户初始额度')}
                  field={'QuotaForNewUser'}
                  step={1}
                  min={0}
                  suffix={'Token'}
                  placeholder={''}
                  onChange={(value) =>
                    setInputs({
                      ...inputs,
                      QuotaForNewUser: String(value),
                    })
                  }
                />
              </Col>
              <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                <Form.InputNumber
                  label={t('请求预扣费额度')}
                  field={'PreConsumedQuota'}
                  step={1}
                  min={0}
                  suffix={'Token'}
                  extraText={t('请求结束后多退少补')}
                  placeholder={''}
                  onChange={(value) =>
                    setInputs({
                      ...inputs,
                      PreConsumedQuota: String(value),
                    })
                  }
                />
              </Col>
              <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                <Form.InputNumber
                  label={t('邀请新用户奖励额度')}
                  field={'QuotaForInviter'}
                  step={1}
                  min={0}
                  suffix={'Token'}
                  extraText={''}
                  placeholder={t('例如：2000')}
                  onChange={(value) =>
                    setInputs({
                      ...inputs,
                      QuotaForInviter: String(value),
                    })
                  }
                />
              </Col>
            </Row>
            <Row>
              <Col xs={24} sm={12} md={8} lg={8} xl={6}>
                <Form.InputNumber
                  label={t('新用户使用邀请码奖励额度')}
                  field={'QuotaForInvitee'}
                  step={1}
                  min={0}
                  suffix={'Token'}
                  extraText={''}
                  placeholder={t('例如：1000')}
                  onChange={(value) =>
                    setInputs({
                      ...inputs,
                      QuotaForInvitee: String(value),
                    })
                  }
                />
              </Col>
              <Col xs={24} sm={12} md={8} lg={8} xl={6}>
                <Form.InputNumber
                  label={t('邀请充值返利比例')}
                  field={'TopUpAffRatio'}
                  step={0.1}
                  min={0}
                  suffix={'%'}
                  extraText={t('被邀请用户充值成功后，按实际到账额度返利给邀请人')}
                  placeholder={t('例如：10')}
                  onChange={(value) =>
                    setInputs({
                      ...inputs,
                      TopUpAffRatio: String(value),
                    })
                  }
                />
              </Col>
            </Row>
            <Row gutter={16}>
              <Col span={24}>
                <Form.Slot
                  label={t('赠送余额可用范围')}
                  extraText={t(
                    '配置赠送余额可调用的分组与模型。支持手动输入，* 表示全部；留空则赠送余额不会参与扣费。',
                  )}
                >
                  <div className='space-y-3'>
                    {giftQuotaRules.map((rule, index) => (
                      <Row gutter={12} key={`gift-rule-${index}`}>
                        <Col xs={24} sm={10}>
                          <Select
                            value={rule.group}
                            optionList={groupOptions}
                            placeholder={t('分组，* 表示全部')}
                            allowCreate
                            filter
                            showSearch
                            style={{ width: '100%' }}
                            onChange={(value) => {
                              const nextRules = [...giftQuotaRules];
                              nextRules[index] = {
                                ...nextRules[index],
                                group: String(value || '*'),
                              };
                              updateGiftQuotaRules(nextRules);
                            }}
                          />
                        </Col>
                        <Col xs={24} sm={10}>
                          <Select
                            value={rule.model}
                            optionList={modelOptions}
                            placeholder={t('模型，* 表示全部')}
                            allowCreate
                            filter
                            showSearch
                            style={{ width: '100%' }}
                            onChange={(value) => {
                              const nextRules = [...giftQuotaRules];
                              nextRules[index] = {
                                ...nextRules[index],
                                model: String(value || '*'),
                              };
                              updateGiftQuotaRules(nextRules);
                            }}
                          />
                        </Col>
                        <Col xs={24} sm={4}>
                          <Button
                            theme='light'
                            type='danger'
                            style={{ width: '100%' }}
                            onClick={() =>
                              updateGiftQuotaRules(
                                giftQuotaRules.filter((_, i) => i !== index),
                              )
                            }
                          >
                            {t('删除')}
                          </Button>
                        </Col>
                      </Row>
                    ))}
                    <Button
                      theme='light'
                      onClick={() =>
                        updateGiftQuotaRules([
                          ...giftQuotaRules,
                          { group: '*', model: '*' },
                        ])
                      }
                    >
                      {t('新增赠送余额规则')}
                    </Button>
                  </div>
                </Form.Slot>
              </Col>
            </Row>
            <Row>
              <Col>
                <Form.Switch
                  label={t('对免费模型启用预消耗')}
                  field={'quota_setting.enable_free_model_pre_consume'}
                  extraText={t(
                    '开启后，对免费模型（倍率为0，或者价格为0）的模型也会预消耗额度',
                  )}
                  onChange={(value) =>
                    setInputs({
                      ...inputs,
                      'quota_setting.enable_free_model_pre_consume': value,
                    })
                  }
                />
              </Col>
            </Row>

            <Row>
              <Button size='default' onClick={onSubmit}>
                {t('保存额度设置')}
              </Button>
            </Row>
          </Form.Section>
        </Form>
      </Spin>
    </>
  );
}
