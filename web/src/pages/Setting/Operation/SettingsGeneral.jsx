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

import React, { useEffect, useState, useRef, useMemo } from 'react';
import {
  Banner,
  Button,
  Col,
  Form,
  Select,
  InputNumber,
  Row,
  Spin,
  Modal,
  Input,
  Typography,
} from '@douyinfe/semi-ui';
import {
  compareObjects,
  API,
  showError,
  showSuccess,
  showWarning,
} from '../../../helpers';
import { useTranslation } from 'react-i18next';

const { Text } = Typography;

export default function GeneralSettings(props) {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [showQuotaWarning, setShowQuotaWarning] = useState(false);
  const [groupDelayRules, setGroupDelayRules] = useState([]);
  const [largePromptRPMRules, setLargePromptRPMRules] = useState([]);
  const [cacheHitMissMaskRules, setCacheHitMissMaskRules] = useState([]);
  const [groupOptions, setGroupOptions] = useState([]);
  const [groupMigrationSource, setGroupMigrationSource] = useState('');
  const [groupMigrationTarget, setGroupMigrationTarget] = useState('');
  const [inputs, setInputs] = useState({
    TopUpLink: '',
    'general_setting.docs_link': '',
    'general_setting.default_user_group': 'default',
    'general_setting.quota_display_type': 'USD',
    'general_setting.custom_currency_symbol': '¤',
    'general_setting.custom_currency_exchange_rate': '',
    'group_delay_setting.rules': '[]',
    'large_prompt_rpm_setting.rules': '[]',
    'cache_hit_miss_mask_setting.rules': '[]',
    QuotaPerUnit: '',
    RetryTimes: '',
    USDExchangeRate: '',
    DisplayTokenStatEnabled: false,
    DefaultCollapseSidebar: false,
    DemoSiteEnabled: false,
    SelfUseModeEnabled: false,
    'token_setting.max_user_tokens': 1000,
  });
  const refForm = useRef();
  const [inputsRow, setInputsRow] = useState(inputs);

  const parseRuleSeconds = (value) => {
    const numberValue = Number(value);
    if (!Number.isFinite(numberValue)) {
      return 0;
    }
    return Math.max(0, Math.trunc(numberValue));
  };

  const normalizeGroupDelayRules = (rules = []) => {
    if (!Array.isArray(rules)) {
      return [];
    }
    return rules.map((rule) => ({
      group: rule?.group ?? '',
      min_seconds: parseRuleSeconds(rule?.min_seconds),
      max_seconds: parseRuleSeconds(rule?.max_seconds),
    }));
  };

  const parseGroupDelayRules = (raw) => {
    if (!raw) {
      return [];
    }
    try {
      return normalizeGroupDelayRules(JSON.parse(raw));
    } catch {
      return [];
    }
  };

  const serializeGroupDelayRules = (rules = []) =>
    JSON.stringify(normalizeGroupDelayRules(rules));

  const updateGroupDelayRules = (rules) => {
    const normalizedRules = normalizeGroupDelayRules(rules);
    setGroupDelayRules(normalizedRules);
    setInputs((prev) => ({
      ...prev,
      'group_delay_setting.rules': serializeGroupDelayRules(normalizedRules),
    }));
  };

  const validateGroupDelayRules = () => {
    const seenGroups = new Set();
    for (let i = 0; i < groupDelayRules.length; i++) {
      const rule = groupDelayRules[i];
      const group = String(rule.group || '').trim();
      const minSeconds = Number(rule.min_seconds);
      const maxSeconds = Number(rule.max_seconds);
      if (!group) {
        return t(`第 ${i + 1} 条分组延迟规则的分组名不能为空`);
      }
      if (Number.isNaN(minSeconds) || Number.isNaN(maxSeconds)) {
        return t(`第 ${i + 1} 条分组延迟规则的秒数格式不正确`);
      }
      if (minSeconds < 0 || maxSeconds < 0) {
        return t(`第 ${i + 1} 条分组延迟规则的秒数不能小于 0`);
      }
      if (maxSeconds < minSeconds) {
        return t(`第 ${i + 1} 条分组延迟规则的最大延迟不能小于最小延迟`);
      }
      if (seenGroups.has(group)) {
        return t(`分组 ${group} 的延迟规则重复`);
      }
      seenGroups.add(group);
    }
    return '';
  };

  const parseRuleInt = (value) => {
    const numberValue = Number(value);
    if (!Number.isFinite(numberValue)) {
      return 0;
    }
    return Math.max(0, Math.trunc(numberValue));
  };

  const getDefaultLargePromptRPMDuration = () => {
    const fallback = parseRuleInt(
      props.options?.ModelRequestRateLimitDurationMinutes,
    );
    return fallback > 0 ? fallback : 1;
  };

  const normalizeLargePromptRPMRules = (rules = []) => {
    const defaultDurationMinutes = getDefaultLargePromptRPMDuration();
    if (!Array.isArray(rules)) {
      return [];
    }
    return rules.map((rule) => ({
      group: rule?.group ?? '',
      threshold_k: parseRuleInt(rule?.threshold_k),
      temporary_rpm: parseRuleInt(rule?.temporary_rpm),
      duration_minutes:
        parseRuleInt(rule?.duration_minutes) || defaultDurationMinutes,
    }));
  };

  const parseLargePromptRPMRules = (raw) => {
    if (!raw) {
      return [];
    }
    try {
      return normalizeLargePromptRPMRules(JSON.parse(raw));
    } catch {
      return [];
    }
  };

  const serializeLargePromptRPMRules = (rules = []) =>
    JSON.stringify(normalizeLargePromptRPMRules(rules));

  const updateLargePromptRPMRules = (rules) => {
    const normalizedRules = normalizeLargePromptRPMRules(rules);
    setLargePromptRPMRules(normalizedRules);
    setInputs((prev) => ({
      ...prev,
      'large_prompt_rpm_setting.rules':
        serializeLargePromptRPMRules(normalizedRules),
    }));
  };

  const validateLargePromptRPMRules = () => {
    const seenGroups = new Set();
    for (let i = 0; i < largePromptRPMRules.length; i++) {
      const rule = largePromptRPMRules[i];
      const group = String(rule.group || '').trim();
      const thresholdK = Number(rule.threshold_k);
      const temporaryRPM = Number(rule.temporary_rpm);
      const durationMinutes = Number(rule.duration_minutes);
      if (!group) {
        return t(`第 ${i + 1} 条大输入临时 RPM 规则的分组名不能为空`);
      }
      if (Number.isNaN(thresholdK) || thresholdK <= 0) {
        return t(`第 ${i + 1} 条大输入临时 RPM 规则的输入阈值必须大于 0`);
      }
      if (Number.isNaN(temporaryRPM) || temporaryRPM <= 0) {
        return t(`第 ${i + 1} 条大输入临时 RPM 规则的临时 RPM 必须大于 0`);
      }
      if (Number.isNaN(durationMinutes) || durationMinutes <= 0) {
        return t(`第 ${i + 1} 条大输入临时 RPM 规则的限制时长必须大于 0`);
      }
      if (seenGroups.has(group)) {
        return t(`分组 ${group} 的大输入临时 RPM 规则重复`);
      }
      seenGroups.add(group);
    }
    return '';
  };

  const normalizeCacheHitMissMaskRules = (rules = []) => {
    if (!Array.isArray(rules)) {
      return [];
    }
    return rules.map((rule) => ({
      group: rule?.group ?? '',
      min_percent: parseRuleInt(rule?.min_percent),
      max_percent: parseRuleInt(rule?.max_percent),
    }));
  };

  const parseCacheHitMissMaskRules = (raw) => {
    if (!raw) {
      return [];
    }
    try {
      return normalizeCacheHitMissMaskRules(JSON.parse(raw));
    } catch {
      return [];
    }
  };

  const serializeCacheHitMissMaskRules = (rules = []) =>
    JSON.stringify(normalizeCacheHitMissMaskRules(rules));

  const updateCacheHitMissMaskRules = (rules) => {
    const normalizedRules = normalizeCacheHitMissMaskRules(rules);
    setCacheHitMissMaskRules(normalizedRules);
    setInputs((prev) => ({
      ...prev,
      'cache_hit_miss_mask_setting.rules':
        serializeCacheHitMissMaskRules(normalizedRules),
    }));
  };

  const validateCacheHitMissMaskRules = () => {
    const seenGroups = new Set();
    for (let i = 0; i < cacheHitMissMaskRules.length; i++) {
      const rule = cacheHitMissMaskRules[i];
      const group = String(rule.group || '').trim();
      const minPercent = Number(rule.min_percent);
      const maxPercent = Number(rule.max_percent);
      if (!group) {
        return t(`第 ${i + 1} 条缓存命中转未命中规则的分组名不能为空`);
      }
      if (
        Number.isNaN(minPercent) ||
        Number.isNaN(maxPercent) ||
        minPercent <= 0 ||
        maxPercent <= 0
      ) {
        return t(`第 ${i + 1} 条缓存命中转未命中规则的比例必须大于 0`);
      }
      if (minPercent > 100 || maxPercent > 100) {
        return t(`第 ${i + 1} 条缓存命中转未命中规则的比例不能大于 100`);
      }
      if (maxPercent < minPercent) {
        return t(`第 ${i + 1} 条缓存命中转未命中规则的最大比例不能小于最小比例`);
      }
      if (seenGroups.has(group)) {
        return t(`分组 ${group} 的缓存命中转未命中规则重复`);
      }
      seenGroups.add(group);
    }
    return '';
  };

  const fetchGroups = async () => {
    try {
      const res = await API.get('/api/group/');
      const groups = Array.isArray(res?.data?.data) ? res.data.data : [];
      setGroupOptions(
        groups.map((group) => ({
          label: group,
          value: group,
        })),
      );
    } catch (error) {
      setGroupOptions([]);
    }
  };

  const handleMigrateGroupUsers = async () => {
    const sourceGroup = String(groupMigrationSource || '').trim();
    const targetGroup = String(groupMigrationTarget || '').trim();

    if (!sourceGroup || !targetGroup) {
      return showError(t('请选择来源分组和目标分组'));
    }
    if (sourceGroup === targetGroup) {
      return showError(t('来源分组和目标分组不能相同'));
    }

    Modal.confirm({
      title: t('确认批量切换分组'),
      content: t(
        '执行后，会将当前令牌分组属于来源分组的全部令牌切换到目标分组，不会修改用户默认分组。',
      ),
      onOk: async () => {
        setLoading(true);
        try {
          const res = await API.post('/api/group/migrate', {
            source_group: sourceGroup,
            target_group: targetGroup,
          });
          const { success, message, data } = res.data;
          if (!success) {
            return showError(message || t('批量切换失败'));
          }
          showSuccess(
            t(
              '批量切换完成：共切换 {{tokenCount}} 个令牌',
              {
                tokenCount: data?.token_count || 0,
              },
            ),
          );
        } catch (error) {
          showError(t('批量切换失败，请重试'));
        } finally {
          setLoading(false);
        }
      },
    });
  };

  function handleFieldChange(fieldName) {
    return (value) => {
      setInputs((inputs) => ({ ...inputs, [fieldName]: value }));
    };
  }

  function onSubmit() {
    const updateArray = compareObjects(inputs, inputsRow);
    if (!updateArray.length) return showWarning(t('你似乎并没有修改什么'));
    const defaultUserGroup = String(
      inputs['general_setting.default_user_group'] || '',
    ).trim();
    if (!defaultUserGroup) {
      return showError(t('默认用户分组不能为空'));
    }
    const groupDelayError = validateGroupDelayRules();
    if (groupDelayError) {
      return showError(groupDelayError);
    }
    const largePromptRPMError = validateLargePromptRPMRules();
    if (largePromptRPMError) {
      return showError(largePromptRPMError);
    }
    const cacheHitMissMaskError = validateCacheHitMissMaskRules();
    if (cacheHitMissMaskError) {
      return showError(cacheHitMissMaskError);
    }
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

  // 计算展示在输入框中的“1 USD = X <currency>”中的 X
  const combinedRate = useMemo(() => {
    const type = inputs['general_setting.quota_display_type'];
    if (type === 'USD') return '1';
    if (type === 'CNY') return String(inputs['USDExchangeRate'] || '');
    if (type === 'TOKENS') return String(inputs['QuotaPerUnit'] || '');
    if (type === 'CUSTOM')
      return String(
        inputs['general_setting.custom_currency_exchange_rate'] || '',
      );
    return '';
  }, [inputs]);

  const onCombinedRateChange = (val) => {
    const type = inputs['general_setting.quota_display_type'];
    if (type === 'CNY') {
      handleFieldChange('USDExchangeRate')(val);
    } else if (type === 'TOKENS') {
      handleFieldChange('QuotaPerUnit')(val);
    } else if (type === 'CUSTOM') {
      handleFieldChange('general_setting.custom_currency_exchange_rate')(val);
    }
  };

  const showTokensOption = useMemo(() => {
    const initialType = props.options?.['general_setting.quota_display_type'];
    const initialQuotaPerUnit = parseFloat(props.options?.QuotaPerUnit);
    const legacyTokensMode =
      initialType === undefined &&
      props.options?.DisplayInCurrencyEnabled !== undefined &&
      !props.options.DisplayInCurrencyEnabled;
    return (
      initialType === 'TOKENS' ||
      legacyTokensMode ||
      (!isNaN(initialQuotaPerUnit) && initialQuotaPerUnit !== 500000)
    );
  }, [props.options]);

  const quotaDisplayType = inputs['general_setting.quota_display_type'];

  const quotaDisplayTypeDesc = useMemo(() => {
    const descMap = {
      USD: t('站点所有额度将以美元 ($) 显示'),
      CNY: t('站点所有额度将按汇率换算为人民币 (¥) 显示'),
      TOKENS: t('站点所有额度将以原始 Token 数显示，不做货币换算'),
      CUSTOM: t('站点所有额度将按汇率换算为自定义货币显示'),
    };
    return descMap[quotaDisplayType] || '';
  }, [quotaDisplayType, t]);

  const rateLabel = useMemo(() => {
    if (quotaDisplayType === 'CNY') return t('汇率');
    if (quotaDisplayType === 'TOKENS') return t('每美元对应 Token 数');
    if (quotaDisplayType === 'CUSTOM') return t('汇率');
    return '';
  }, [quotaDisplayType, t]);

  const rateSuffix = useMemo(() => {
    if (quotaDisplayType === 'CNY') return 'CNY (¥)';
    if (quotaDisplayType === 'TOKENS') return 'Tokens';
    if (quotaDisplayType === 'CUSTOM')
      return inputs['general_setting.custom_currency_symbol'] || '¤';
    return '';
  }, [quotaDisplayType, inputs]);

  const rateExtraText = useMemo(() => {
    if (quotaDisplayType === 'CNY')
      return t(
        '系统内部以美元 (USD) 为基准计价。用户余额、充值金额、模型定价、用量日志等所有金额显示均按此汇率换算为人民币，不影响内部计费',
      );
    if (quotaDisplayType === 'TOKENS')
      return t(
        '系统内部计费精度，默认 500000，修改可能导致计费异常，请谨慎操作',
      );
    if (quotaDisplayType === 'CUSTOM')
      return t(
        '系统内部以美元 (USD) 为基准计价。用户余额、充值金额、模型定价、用量日志等所有金额显示均按此汇率换算为自定义货币，不影响内部计费',
      );
    return '';
  }, [quotaDisplayType, t]);

  const previewText = useMemo(() => {
    if (quotaDisplayType === 'USD') return '$1.00';
    const rate = parseFloat(combinedRate);
    if (!rate || isNaN(rate)) return t('请输入汇率');
    if (quotaDisplayType === 'CNY') return `$1.00 → ¥${rate.toFixed(2)}`;
    if (quotaDisplayType === 'TOKENS')
      return `$1.00 → ${Number(rate).toLocaleString()} Tokens`;
    if (quotaDisplayType === 'CUSTOM') {
      const symbol = inputs['general_setting.custom_currency_symbol'] || '¤';
      return `$1.00 → ${symbol}${rate.toFixed(2)}`;
    }
    return '';
  }, [quotaDisplayType, combinedRate, inputs, t]);

  const ruleDeleteColStyle = useMemo(
    () => ({
      display: 'flex',
      alignItems: 'flex-end',
    }),
    [],
  );

  useEffect(() => {
    fetchGroups().then();
  }, []);

  useEffect(() => {
    const currentInputs = {};
    for (let key in props.options) {
      if (Object.keys(inputs).includes(key)) {
        currentInputs[key] = props.options[key];
      }
    }
    // 若旧字段存在且新字段缺失，则做一次兜底映射
    if (
      currentInputs['general_setting.quota_display_type'] === undefined &&
      props.options?.DisplayInCurrencyEnabled !== undefined
    ) {
      currentInputs['general_setting.quota_display_type'] = props.options
        .DisplayInCurrencyEnabled
        ? 'USD'
        : 'TOKENS';
    }
    // 回填自定义货币相关字段（如果后端已存在）
    if (props.options['general_setting.custom_currency_symbol'] !== undefined) {
      currentInputs['general_setting.custom_currency_symbol'] =
        props.options['general_setting.custom_currency_symbol'];
    }
    if (
      props.options['general_setting.custom_currency_exchange_rate'] !==
      undefined
    ) {
      currentInputs['general_setting.custom_currency_exchange_rate'] =
        props.options['general_setting.custom_currency_exchange_rate'];
    }
    currentInputs['general_setting.default_user_group'] =
      props.options['general_setting.default_user_group'] || 'default';
    const currentGroupDelayRules = parseGroupDelayRules(
      props.options['group_delay_setting.rules'],
    );
    const currentLargePromptRPMRules = parseLargePromptRPMRules(
      props.options['large_prompt_rpm_setting.rules'],
    );
    const currentCacheHitMissMaskRules = parseCacheHitMissMaskRules(
      props.options['cache_hit_miss_mask_setting.rules'],
    );
    currentInputs['group_delay_setting.rules'] =
      props.options['group_delay_setting.rules'] ||
      serializeGroupDelayRules(currentGroupDelayRules);
    currentInputs['large_prompt_rpm_setting.rules'] =
      props.options['large_prompt_rpm_setting.rules'] ||
      serializeLargePromptRPMRules(currentLargePromptRPMRules);
    currentInputs['cache_hit_miss_mask_setting.rules'] =
      props.options['cache_hit_miss_mask_setting.rules'] ||
      serializeCacheHitMissMaskRules(currentCacheHitMissMaskRules);
    setInputs(currentInputs);
    setInputsRow(structuredClone(currentInputs));
    setGroupDelayRules(currentGroupDelayRules);
    setLargePromptRPMRules(currentLargePromptRPMRules);
    setCacheHitMissMaskRules(currentCacheHitMissMaskRules);
    refForm.current.setValues(currentInputs);
  }, [props.options]);

  return (
    <>
      <Spin spinning={loading}>
        <Form
          values={inputs}
          getFormApi={(formAPI) => (refForm.current = formAPI)}
          style={{ marginBottom: 15 }}
        >
          <Form.Section text={t('通用设置')}>
            <Row gutter={16}>
              <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                <Form.Input
                  field={'TopUpLink'}
                  label={t('充值链接')}
                  initValue={''}
                  placeholder={t('例如发卡网站的购买链接')}
                  onChange={handleFieldChange('TopUpLink')}
                  showClear
                />
              </Col>
              <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                <Form.Input
                  field={'general_setting.docs_link'}
                  label={t('文档地址')}
                  initValue={''}
                  placeholder={t('例如 https://docs.newapi.pro')}
                  onChange={handleFieldChange('general_setting.docs_link')}
                  showClear
                />
              </Col>
              <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                <Form.Select
                  field={'general_setting.default_user_group'}
                  label={t('默认用户分组')}
                  placeholder={t('请选择默认用户分组')}
                  optionList={groupOptions}
                  allowAdditions
                  showSearch
                  filter
                  onChange={handleFieldChange(
                    'general_setting.default_user_group',
                  )}
                  rules={[
                    { required: true, message: t('默认用户分组不能为空') },
                  ]}
                />
              </Col>
              {/* 单位美元额度已合入汇率组合控件（TOKENS 模式下编辑），不再单独展示 */}
              <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                <Form.Input
                  field={'RetryTimes'}
                  label={t('失败重试次数')}
                  initValue={''}
                  placeholder={t('失败重试次数')}
                  onChange={handleFieldChange('RetryTimes')}
                  showClear
                />
              </Col>
              <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                <Form.Select
                  field='general_setting.quota_display_type'
                  label={t('额度展示类型')}
                  extraText={quotaDisplayTypeDesc}
                  onChange={handleFieldChange(
                    'general_setting.quota_display_type',
                  )}
                >
                  <Form.Select.Option value='USD'>
                    USD ($)
                  </Form.Select.Option>
                  <Form.Select.Option value='CNY'>
                    CNY (¥)
                  </Form.Select.Option>
                  {showTokensOption && (
                    <Form.Select.Option value='TOKENS'>
                      Tokens
                    </Form.Select.Option>
                  )}
                  <Form.Select.Option value='CUSTOM'>
                    {t('自定义货币')}
                  </Form.Select.Option>
                </Form.Select>
              </Col>
              {quotaDisplayType !== 'USD' && (
                <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                  <Form.Slot label={rateLabel}>
                    <Input
                      prefix='1 USD = '
                      suffix={rateSuffix}
                      value={combinedRate}
                      onChange={onCombinedRateChange}
                    />
                    <Text
                      type='tertiary'
                      size='small'
                      style={{ marginTop: 4, display: 'block' }}
                    >
                      {rateExtraText}
                    </Text>
                  </Form.Slot>
                </Col>
              )}
              <Col
                xs={24}
                sm={12}
                md={8}
                lg={8}
                xl={8}
                style={
                  quotaDisplayType !== 'CUSTOM'
                    ? { display: 'none' }
                    : undefined
                }
              >
                <Form.Input
                  field='general_setting.custom_currency_symbol'
                  label={t('自定义货币符号')}
                  extraText={t(
                    '自定义货币符号将显示在所有额度数值前，例如 €1.50',
                  )}
                  placeholder={t('例如 €, £, Rp, ₩, ₹...')}
                  onChange={handleFieldChange(
                    'general_setting.custom_currency_symbol',
                  )}
                  showClear
                />
              </Col>
              <Col span={24}>
                <Text type='tertiary' size='small'>
                  {t('预览效果')}：{previewText}
                </Text>
              </Col>
            </Row>
            <Row gutter={16} style={{ marginTop: 8 }}>
              <Col span={24}>
                <Banner
                  type='info'
                  bordered
                  fullMode={false}
                  closeIcon={null}
                  description={t(
                    '分组延迟会在请求真正发往上游前，按规则随机等待一段时间。这样流式首字和普通首响应都会一起变慢，营造该分组较慢的体感。',
                  )}
                />
              </Col>
            </Row>
            <Row gutter={16} style={{ marginTop: 12 }}>
              <Col span={24}>
                <Text strong>{t('分组延迟')}</Text>
                <Text
                  type='tertiary'
                  size='small'
                  style={{ marginTop: 4, display: 'block' }}
                >
                  {t(
                    '仅对命中的分组生效。支持配置最小秒数和最大秒数，系统会在区间内随机延迟一次；重试不会重复叠加延迟。',
                  )}
                </Text>
              </Col>
            </Row>
            <Row gutter={16} style={{ marginTop: 12 }}>
              <Col span={24}>
                {groupDelayRules.length === 0 && (
                  <div
                    style={{
                      border: '1px dashed var(--semi-color-border)',
                      borderRadius: 12,
                      padding: 16,
                      background: 'var(--semi-color-fill-0)',
                    }}
                  >
                    <Text type='tertiary'>
                      {t('暂未配置分组延迟规则')}
                    </Text>
                  </div>
                )}
                {groupDelayRules.map((rule, index) => (
                  <div
                    key={`group-delay-rule-${index}`}
                    style={{
                      border: '1px solid var(--semi-color-border)',
                      borderRadius: 12,
                      padding: 16,
                      background: 'var(--semi-color-fill-0)',
                      marginBottom: 12,
                    }}
                  >
                    <Row gutter={16}>
                      <Col xs={24} sm={24} md={8} lg={8} xl={8}>
                        <Form.Slot label={t('分组名')}>
                          <Select
                            value={rule.group}
                            placeholder={t('请选择分组')}
                            optionList={groupOptions}
                            showSearch
                            filter
                            insetLabel={t('分组')}
                            style={{ width: '100%' }}
                            onChange={(value) => {
                              const nextRules = [...groupDelayRules];
                              nextRules[index] = {
                                ...nextRules[index],
                                group: String(value || ''),
                              };
                              updateGroupDelayRules(nextRules);
                            }}
                          />
                        </Form.Slot>
                      </Col>
                      <Col xs={24} sm={12} md={6} lg={6} xl={6}>
                        <Form.Slot label={t('最小延迟')}>
                          <InputNumber
                            value={rule.min_seconds ?? 0}
                            min={0}
                            step={1}
                            suffix={t('秒')}
                            onChange={(value) => {
                              const nextRules = [...groupDelayRules];
                              nextRules[index] = {
                                ...nextRules[index],
                                min_seconds: parseRuleSeconds(value),
                              };
                              updateGroupDelayRules(nextRules);
                            }}
                          />
                        </Form.Slot>
                      </Col>
                      <Col xs={24} sm={12} md={6} lg={6} xl={6}>
                        <Form.Slot label={t('最大延迟')}>
                          <InputNumber
                            value={rule.max_seconds ?? 0}
                            min={0}
                            step={1}
                            suffix={t('秒')}
                            onChange={(value) => {
                              const nextRules = [...groupDelayRules];
                              nextRules[index] = {
                                ...nextRules[index],
                                max_seconds: parseRuleSeconds(value),
                              };
                              updateGroupDelayRules(nextRules);
                            }}
                          />
                        </Form.Slot>
                      </Col>
                      <Col
                        xs={24}
                        sm={24}
                        md={4}
                        lg={4}
                        xl={4}
                        style={ruleDeleteColStyle}
                      >
                        <Form.Slot
                          label={
                            <span style={{ visibility: 'hidden' }}>
                              {t('操作')}
                            </span>
                          }
                        >
                          <Button
                            theme='light'
                            type='danger'
                            style={{ width: '100%' }}
                            onClick={() => {
                              updateGroupDelayRules(
                                groupDelayRules.filter((_, i) => i !== index),
                              );
                            }}
                          >
                            {t('删除规则')}
                          </Button>
                        </Form.Slot>
                      </Col>
                    </Row>
                  </div>
                ))}
                <Button
                  theme='light'
                  onClick={() =>
                    updateGroupDelayRules([
                      ...groupDelayRules,
                      {
                        group: '',
                        min_seconds: 0,
                        max_seconds: 0,
                      },
                    ])
                  }
                >
                  {t('新增分组延迟规则')}
                </Button>
              </Col>
            </Row>
            <Row gutter={16} style={{ marginTop: 24 }}>
              <Col span={24}>
                <Banner
                  type='warning'
                  bordered
                  fullMode={false}
                  closeIcon={null}
                  description={t(
                    '当某个分组的单次真实输入 token 超过阈值后，系统会在该用户后续请求中临时收紧 RPM，并持续指定分钟数。这里严格以返回包中的真实 usage 为准，不会使用本地预估 token。',
                  )}
                />
              </Col>
            </Row>
            <Row gutter={16} style={{ marginTop: 12 }}>
              <Col span={24}>
                <Text strong>{t('大输入临时 RPM')}</Text>
                <Text
                  type='tertiary'
                  size='small'
                  style={{ marginTop: 4, display: 'block' }}
                >
                  {t(
                    '命中规则后，会把该用户在该分组下的 RPM 临时调整为设定值，并持续限制指定分钟数。仅在上游返回了真实 input/prompt token 时才会触发。',
                  )}
                </Text>
              </Col>
            </Row>
            <Row gutter={16} style={{ marginTop: 12 }}>
              <Col span={24}>
                {largePromptRPMRules.length === 0 && (
                  <div
                    style={{
                      border: '1px dashed var(--semi-color-border)',
                      borderRadius: 12,
                      padding: 16,
                      background: 'var(--semi-color-fill-0)',
                    }}
                  >
                    <Text type='tertiary'>
                      {t('暂未配置大输入临时 RPM 规则')}
                    </Text>
                  </div>
                )}
                {largePromptRPMRules.map((rule, index) => (
                  <div
                    key={`large-prompt-rpm-rule-${index}`}
                    style={{
                      border: '1px solid var(--semi-color-border)',
                      borderRadius: 12,
                      padding: 16,
                      background: 'var(--semi-color-fill-0)',
                      marginBottom: 12,
                    }}
                  >
                    <Row gutter={16}>
                      <Col xs={24} sm={24} md={7} lg={7} xl={7}>
                        <Form.Slot label={t('分组名')}>
                          <Select
                            value={rule.group}
                            placeholder={t('请选择分组')}
                            optionList={groupOptions}
                            showSearch
                            filter
                            insetLabel={t('分组')}
                            style={{ width: '100%' }}
                            onChange={(value) => {
                              const nextRules = [...largePromptRPMRules];
                              nextRules[index] = {
                                ...nextRules[index],
                                group: String(value || ''),
                              };
                              updateLargePromptRPMRules(nextRules);
                            }}
                          />
                        </Form.Slot>
                      </Col>
                      <Col xs={24} sm={12} md={5} lg={5} xl={5}>
                        <Form.Slot label={t('输入阈值')}>
                          <InputNumber
                            value={rule.threshold_k ?? 0}
                            min={1}
                            step={1}
                            suffix='K'
                            onChange={(value) => {
                              const nextRules = [...largePromptRPMRules];
                              nextRules[index] = {
                                ...nextRules[index],
                                threshold_k: parseRuleInt(value),
                              };
                              updateLargePromptRPMRules(nextRules);
                            }}
                          />
                        </Form.Slot>
                      </Col>
                      <Col xs={24} sm={12} md={4} lg={4} xl={4}>
                        <Form.Slot label={t('临时 RPM')}>
                          <InputNumber
                            value={rule.temporary_rpm ?? 0}
                            min={1}
                            step={1}
                            suffix='RPM'
                            onChange={(value) => {
                              const nextRules = [...largePromptRPMRules];
                              nextRules[index] = {
                                ...nextRules[index],
                                temporary_rpm: parseRuleInt(value),
                              };
                              updateLargePromptRPMRules(nextRules);
                            }}
                          />
                        </Form.Slot>
                      </Col>
                      <Col xs={24} sm={12} md={5} lg={5} xl={5}>
                        <Form.Slot label={t('限制时长')}>
                          <InputNumber
                            value={rule.duration_minutes ?? 0}
                            min={1}
                            step={1}
                            suffix={t('分钟')}
                            onChange={(value) => {
                              const nextRules = [...largePromptRPMRules];
                              nextRules[index] = {
                                ...nextRules[index],
                                duration_minutes: parseRuleInt(value),
                              };
                              updateLargePromptRPMRules(nextRules);
                            }}
                          />
                        </Form.Slot>
                      </Col>
                      <Col
                        xs={24}
                        sm={24}
                        md={3}
                        lg={3}
                        xl={3}
                        style={ruleDeleteColStyle}
                      >
                        <Form.Slot
                          label={
                            <span style={{ visibility: 'hidden' }}>
                              {t('操作')}
                            </span>
                          }
                        >
                          <Button
                            theme='light'
                            type='danger'
                            style={{ width: '100%' }}
                            onClick={() => {
                              updateLargePromptRPMRules(
                                largePromptRPMRules.filter(
                                  (_, i) => i !== index,
                                ),
                              );
                            }}
                          >
                            {t('删除规则')}
                          </Button>
                        </Form.Slot>
                      </Col>
                    </Row>
                  </div>
                ))}
                <Button
                  theme='light'
                  onClick={() =>
                    updateLargePromptRPMRules([
                      ...largePromptRPMRules,
                      {
                        group: '',
                        threshold_k: 32,
                        temporary_rpm: 1,
                        duration_minutes: getDefaultLargePromptRPMDuration(),
                      },
                    ])
                  }
                >
                  {t('新增大输入临时 RPM 规则')}
                </Button>
              </Col>
            </Row>
            <Row gutter={16} style={{ marginTop: 24 }}>
              <Col span={24}>
                <Banner
                  type='info'
                  bordered
                  fullMode={false}
                  closeIcon={null}
                  description={t(
                    '可对指定分组按比例区间随机减少 cached tokens。系统会把这部分缓存命中视作未命中，返回给用户的 usage 与最终结算都会同步体现。',
                  )}
                />
              </Col>
            </Row>
            <Row gutter={16} style={{ marginTop: 12 }}>
              <Col span={24}>
                <Text strong>{t('缓存命中转未命中')}</Text>
                <Text
                  type='tertiary'
                  size='small'
                  style={{ marginTop: 4, display: 'block' }}
                >
                  {t(
                    '命中规则后，系统会在最小比例到最大比例之间随机选取一个百分比，只减少缓存命中 token，不修改总输入 token。',
                  )}
                </Text>
              </Col>
            </Row>
            <Row gutter={16} style={{ marginTop: 12 }}>
              <Col span={24}>
                {cacheHitMissMaskRules.length === 0 && (
                  <div
                    style={{
                      border: '1px dashed var(--semi-color-border)',
                      borderRadius: 12,
                      padding: 16,
                      background: 'var(--semi-color-fill-0)',
                    }}
                  >
                    <Text type='tertiary'>
                      {t('暂未配置缓存命中转未命中规则')}
                    </Text>
                  </div>
                )}
                {cacheHitMissMaskRules.map((rule, index) => (
                  <div
                    key={`cache-hit-miss-mask-rule-${index}`}
                    style={{
                      border: '1px solid var(--semi-color-border)',
                      borderRadius: 12,
                      padding: 16,
                      background: 'var(--semi-color-fill-0)',
                      marginBottom: 12,
                    }}
                  >
                    <Row gutter={16}>
                      <Col xs={24} sm={24} md={8} lg={8} xl={8}>
                        <Form.Slot label={t('分组名')}>
                          <Select
                            value={rule.group}
                            placeholder={t('请选择分组')}
                            optionList={groupOptions}
                            showSearch
                            filter
                            insetLabel={t('分组')}
                            style={{ width: '100%' }}
                            onChange={(value) => {
                              const nextRules = [...cacheHitMissMaskRules];
                              nextRules[index] = {
                                ...nextRules[index],
                                group: String(value || ''),
                              };
                              updateCacheHitMissMaskRules(nextRules);
                            }}
                          />
                        </Form.Slot>
                      </Col>
                      <Col xs={24} sm={12} md={6} lg={6} xl={6}>
                        <Form.Slot label={t('最小比例')}>
                          <InputNumber
                            value={rule.min_percent ?? 0}
                            min={1}
                            max={100}
                            step={1}
                            suffix='%'
                            onChange={(value) => {
                              const nextRules = [...cacheHitMissMaskRules];
                              nextRules[index] = {
                                ...nextRules[index],
                                min_percent: parseRuleInt(value),
                              };
                              updateCacheHitMissMaskRules(nextRules);
                            }}
                          />
                        </Form.Slot>
                      </Col>
                      <Col xs={24} sm={12} md={6} lg={6} xl={6}>
                        <Form.Slot label={t('最大比例')}>
                          <InputNumber
                            value={rule.max_percent ?? 0}
                            min={1}
                            max={100}
                            step={1}
                            suffix='%'
                            onChange={(value) => {
                              const nextRules = [...cacheHitMissMaskRules];
                              nextRules[index] = {
                                ...nextRules[index],
                                max_percent: parseRuleInt(value),
                              };
                              updateCacheHitMissMaskRules(nextRules);
                            }}
                          />
                        </Form.Slot>
                      </Col>
                      <Col
                        xs={24}
                        sm={24}
                        md={4}
                        lg={4}
                        xl={4}
                        style={ruleDeleteColStyle}
                      >
                        <Form.Slot
                          label={
                            <span style={{ visibility: 'hidden' }}>
                              {t('操作')}
                            </span>
                          }
                        >
                          <Button
                            theme='light'
                            type='danger'
                            style={{ width: '100%' }}
                            onClick={() => {
                              updateCacheHitMissMaskRules(
                                cacheHitMissMaskRules.filter(
                                  (_, i) => i !== index,
                                ),
                              );
                            }}
                          >
                            {t('删除规则')}
                          </Button>
                        </Form.Slot>
                      </Col>
                    </Row>
                  </div>
                ))}
                <Button
                  theme='light'
                  onClick={() =>
                    updateCacheHitMissMaskRules([
                      ...cacheHitMissMaskRules,
                      {
                        group: '',
                        min_percent: 10,
                        max_percent: 20,
                      },
                    ])
                  }
                >
                  {t('新增缓存命中转未命中规则')}
                </Button>
              </Col>
            </Row>
            <Row gutter={16}>
              <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                <Form.Switch
                  field={'DisplayTokenStatEnabled'}
                  label={t('额度查询接口返回令牌额度而非用户额度')}
                  size='default'
                  checkedText='｜'
                  uncheckedText='〇'
                  onChange={handleFieldChange('DisplayTokenStatEnabled')}
                />
              </Col>
              <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                <Form.Switch
                  field={'DefaultCollapseSidebar'}
                  label={t('默认折叠侧边栏')}
                  size='default'
                  checkedText='｜'
                  uncheckedText='〇'
                  onChange={handleFieldChange('DefaultCollapseSidebar')}
                />
              </Col>
              <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                <Form.Switch
                  field={'DemoSiteEnabled'}
                  label={t('演示站点模式')}
                  size='default'
                  checkedText='｜'
                  uncheckedText='〇'
                  onChange={handleFieldChange('DemoSiteEnabled')}
                />
              </Col>
              <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                <Form.Switch
                  field={'SelfUseModeEnabled'}
                  label={t('自用模式')}
                  extraText={t('开启后不限制：必须设置模型倍率')}
                  size='default'
                  checkedText='｜'
                  uncheckedText='〇'
                  onChange={handleFieldChange('SelfUseModeEnabled')}
                />
              </Col>
            </Row>
            <Row gutter={16}>
              <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                <Form.InputNumber
                  label={t('用户最大令牌数量')}
                  field={'token_setting.max_user_tokens'}
                  step={1}
                  min={1}
                  extraText={t('每个用户最多可创建的令牌数量，默认 1000，设置过大可能会影响性能')}
                  placeholder={'1000'}
                  onChange={handleFieldChange('token_setting.max_user_tokens')}
                />
              </Col>
            </Row>
            <Row gutter={16} style={{ marginTop: 24 }}>
              <Col span={24}>
                <Banner
                  type='warning'
                  bordered
                  fullMode={false}
                  closeIcon={null}
                  description={t(
                    '分组批量切换只会修改令牌分组，不会改动用户默认分组。该操作适合令牌迁移或分组合并，请确认后执行。',
                  )}
                />
              </Col>
            </Row>
            <Row gutter={16} style={{ marginTop: 12 }}>
              <Col span={24}>
                <Text strong>{t('分组批量切换')}</Text>
                <Text
                  type='tertiary'
                  size='small'
                  style={{ marginTop: 4, display: 'block' }}
                >
                  {t(
                    '按令牌当前所属分组筛选。来源分组支持手动输入旧分组名；命中后，仅会把这些令牌切换到目标分组，不会修改用户默认分组。',
                  )}
                </Text>
              </Col>
            </Row>
            <Row gutter={16} style={{ marginTop: 12 }}>
              <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                <Form.Slot label={t('来源分组')}>
                  <Select
                    value={groupMigrationSource}
                    placeholder={t('请选择或手动输入来源分组')}
                    optionList={groupOptions}
                    allowCreate
                    showSearch
                    filter
                    style={{ width: '100%' }}
                    onChange={(value) =>
                      setGroupMigrationSource(String(value || ''))
                    }
                  />
                </Form.Slot>
              </Col>
              <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                <Form.Slot label={t('目标分组')}>
                  <Select
                    value={groupMigrationTarget}
                    placeholder={t('请选择目标分组')}
                    optionList={groupOptions}
                    showSearch
                    filter
                    style={{ width: '100%' }}
                    onChange={(value) =>
                      setGroupMigrationTarget(String(value || ''))
                    }
                  />
                </Form.Slot>
              </Col>
              <Col
                xs={24}
                sm={24}
                md={8}
                lg={8}
                xl={8}
                style={ruleDeleteColStyle}
              >
                <Form.Slot
                  label={
                    <span style={{ visibility: 'hidden' }}>{t('操作')}</span>
                  }
                >
                  <Button
                    theme='solid'
                    type='warning'
                    style={{ width: '100%' }}
                    disabled={!groupMigrationSource || !groupMigrationTarget}
                    onClick={handleMigrateGroupUsers}
                  >
                    {t('一键切换令牌分组')}
                  </Button>
                </Form.Slot>
              </Col>
            </Row>
            <Row>
              <Button size='default' onClick={onSubmit}>
                {t('保存通用设置')}
              </Button>
            </Row>
          </Form.Section>
        </Form>
      </Spin>

      <Modal
        title={t('警告')}
        visible={showQuotaWarning}
        onOk={() => setShowQuotaWarning(false)}
        onCancel={() => setShowQuotaWarning(false)}
        closeOnEsc={true}
        width={500}
      >
        <Banner
          type='warning'
          description={t(
            '此设置用于系统内部计算，默认值500000是为了精确到6位小数点设计，不推荐修改。',
          )}
          bordered
          fullMode={false}
          closeIcon={null}
        />
      </Modal>
    </>
  );
}
