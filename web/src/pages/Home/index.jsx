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

import React, { useContext, useEffect, useState } from 'react';
import { Button, Typography } from '@douyinfe/semi-ui';
import {
  API,
  showError,
  copy,
  showSuccess,
  getSystemName,
} from '../../helpers';
import { useIsMobile } from '../../hooks/common/useIsMobile';
import { API_ENDPOINTS } from '../../constants/common.constant';
import { StatusContext } from '../../context/Status';
import { useActualTheme } from '../../context/Theme';
import { marked } from 'marked';
import { useTranslation } from 'react-i18next';
import {
  IconGithubLogo,
  IconPlay,
  IconFile,
  IconCopy,
} from '@douyinfe/semi-icons';
import { Link } from 'react-router-dom';
import NoticeModal from '../../components/layout/NoticeModal';
import {
  Moonshot,
  OpenAI,
  XAI,
  Zhipu,
  Volcengine,
  Claude,
  Gemini,
  DeepSeek,
  Qwen,
  AzureAI,
  Hunyuan,
  Xinference,
} from '@lobehub/icons';

const { Text } = Typography;

const PROVIDER_ICONS = [
  ({ size }) => <OpenAI size={size} />,
  ({ size }) => <Claude.Color size={size} />,
  ({ size }) => <Gemini.Color size={size} />,
  ({ size }) => <XAI size={size} />,
  ({ size }) => <DeepSeek.Color size={size} />,
  ({ size }) => <Qwen.Color size={size} />,
  ({ size }) => <Moonshot size={size} />,
  ({ size }) => <Zhipu.Color size={size} />,
  ({ size }) => <Volcengine.Color size={size} />,
  ({ size }) => <AzureAI.Color size={size} />,
  ({ size }) => <Hunyuan.Color size={size} />,
  ({ size }) => <Xinference.Color size={size} />,
];

const Home = () => {
  const { t, i18n } = useTranslation();
  const [statusState] = useContext(StatusContext);
  const actualTheme = useActualTheme();
  const [homePageContentLoaded, setHomePageContentLoaded] = useState(false);
  const [homePageContent, setHomePageContent] = useState('');
  const [noticeVisible, setNoticeVisible] = useState(false);
  const isMobile = useIsMobile();
  const isDemoSiteMode = statusState?.status?.demo_site_enabled || false;
  const docsLink = statusState?.status?.docs_link || '';
  const serverAddress =
    statusState?.status?.server_address || `${window.location.origin}`;
  const endpointItems = API_ENDPOINTS.map((value) => ({ value }));
  const [endpointIndex, setEndpointIndex] = useState(0);
  const systemName = getSystemName();
  const currentEndpoint = endpointItems[endpointIndex]?.value || '/v1';

  const businessHighlights = [
    {
      eyebrow: t('兼容协议'),
      title: t('兼容 OpenAI 协议的调用方式'),
      description: t('客户替换基址后即可继续使用现有 SDK、脚本与第三方客户端。'),
    },
    {
      eyebrow: t('供应商接入'),
      title: t('热门模型与多供应商 Token'),
      description: t('按需售卖公有云、国产模型与自托管模型额度。'),
    },
    {
      eyebrow: t('销售能力'),
      title: t('充值、分组、倍率、日志'),
      description: t('把客户充值、令牌发放、分组定价与调用审计集中在一个后台。'),
    },
  ];

  const displayHomePageContent = async () => {
    setHomePageContent(localStorage.getItem('home_page_content') || '');
    const res = await API.get('/api/home_page_content');
    const { success, message, data } = res.data;
    if (success) {
      let content = data;
      if (!data.startsWith('https://')) {
        content = marked.parse(data);
      }
      setHomePageContent(content);
      localStorage.setItem('home_page_content', content);

      if (data.startsWith('https://')) {
        const iframe = document.querySelector('iframe');
        if (iframe) {
          iframe.onload = () => {
            iframe.contentWindow.postMessage({ themeMode: actualTheme }, '*');
            iframe.contentWindow.postMessage({ lang: i18n.language }, '*');
          };
        }
      }
    } else {
      showError(message);
      setHomePageContent('加载首页内容失败...');
    }
    setHomePageContentLoaded(true);
  };

  const handleCopyBaseURL = async () => {
    const ok = await copy(serverAddress);
    if (ok) {
      showSuccess(t('已复制到剪切板'));
    }
  };

  useEffect(() => {
    const checkNoticeAndShow = async () => {
      const lastCloseDate = localStorage.getItem('notice_close_date');
      const today = new Date().toDateString();
      if (lastCloseDate !== today) {
        try {
          const res = await API.get('/api/notice');
          const { success, data } = res.data;
          if (success && data && data.trim() !== '') {
            setNoticeVisible(true);
          }
        } catch (error) {
          console.error('获取公告失败:', error);
        }
      }
    };

    checkNoticeAndShow();
  }, []);

  useEffect(() => {
    displayHomePageContent().then();
  }, []);

  useEffect(() => {
    const timer = setInterval(() => {
      setEndpointIndex((prev) => (prev + 1) % endpointItems.length);
    }, 3000);
    return () => clearInterval(timer);
  }, [endpointItems.length]);

  return (
    <div className='w-full overflow-x-hidden'>
      <NoticeModal
        visible={noticeVisible}
        onClose={() => setNoticeVisible(false)}
        isMobile={isMobile}
      />
      {homePageContentLoaded && homePageContent === '' ? (
        <div className='home-business-shell w-full overflow-hidden border-b border-semi-color-border'>
          <div className='mx-auto max-w-7xl px-5 pb-16 pt-24 md:px-8 md:pb-24 md:pt-28 lg:pt-32'>
            <div className='grid gap-6 lg:grid-cols-[minmax(0,1.15fr)_380px] lg:gap-10'>
              <div className='flex flex-col gap-6 lg:gap-8'>
                <div className='home-business-badge'>
                  <span className='home-business-badge-dot' />
                  <span>{t('模型 Token 销售平台')}</span>
                </div>

                <div className='max-w-4xl'>
                  <Text className='!text-xs !font-semibold uppercase tracking-[0.28em] !text-semi-color-text-2'>
                    {systemName}
                  </Text>
                  <h1 className='mt-4 text-4xl font-semibold leading-tight tracking-[-0.03em] text-semi-color-text-0 md:text-5xl lg:text-6xl'>
                    {t('统一售卖 OpenAI、Claude、Gemini 等模型 Token')}
                  </h1>
                  <p className='mt-5 max-w-2xl text-base leading-7 text-semi-color-text-1 md:text-lg'>
                    {t('为客户提供充值、额度分发、API 调用与账单管理的一站式入口。')}
                  </p>
                </div>

                <div className='flex flex-wrap items-center gap-3'>
                  <Link to='/console'>
                    <Button
                      theme='solid'
                      type='primary'
                      size={isMobile ? 'default' : 'large'}
                      className='!rounded-full !px-7'
                      icon={<IconPlay />}
                    >
                      {t('获取 Token')}
                    </Button>
                  </Link>
                  {isDemoSiteMode && statusState?.status?.version ? (
                    <Button
                      size={isMobile ? 'default' : 'large'}
                      className='!rounded-full !px-6'
                      icon={<IconGithubLogo />}
                      onClick={() =>
                        window.open(
                          'https://github.com/QuantumNous/new-api',
                          '_blank',
                        )
                      }
                    >
                      {statusState.status.version}
                    </Button>
                  ) : (
                    docsLink && (
                      <Button
                        size={isMobile ? 'default' : 'large'}
                        className='!rounded-full !px-6'
                        icon={<IconFile />}
                        onClick={() => window.open(docsLink, '_blank')}
                      >
                        {t('文档')}
                      </Button>
                    )
                  )}
                </div>

                <div className='home-business-panel max-w-3xl p-5 md:p-6'>
                  <div className='flex flex-col gap-4 md:flex-row md:items-start md:justify-between'>
                    <div className='min-w-0 flex-1'>
                      <p className='text-xs font-medium uppercase tracking-[0.24em] text-semi-color-text-2'>
                        {t('默认 API 地址')}
                      </p>
                      <div className='mt-3 break-all font-mono text-sm text-semi-color-text-0 md:text-base'>
                        {serverAddress}
                      </div>
                    </div>
                    <Button
                      type='tertiary'
                      icon={<IconCopy />}
                      className='!rounded-full !px-5'
                      onClick={handleCopyBaseURL}
                    >
                      {t('复制地址')}
                    </Button>
                  </div>
                  <div className='mt-5 flex flex-col gap-3 border-t border-semi-color-border pt-5 md:flex-row md:items-center md:justify-between'>
                    <div className='flex items-center gap-3'>
                      <span className='text-xs font-medium uppercase tracking-[0.24em] text-semi-color-text-2'>
                        {t('当前端点')}
                      </span>
                      <span className='rounded-full border border-semi-color-border bg-semi-color-bg-1 px-3 py-1 font-mono text-xs text-semi-color-text-0 md:text-sm'>
                        {currentEndpoint}
                      </span>
                    </div>
                    <Text type='tertiary'>
                      {t('客户只需替换基址即可开始调用模型')}
                    </Text>
                  </div>
                </div>
              </div>

              <div className='grid gap-4'>
                {businessHighlights.map((item) => (
                  <div className='home-business-panel p-5 md:p-6' key={item.eyebrow}>
                    <p className='text-xs font-medium uppercase tracking-[0.24em] text-semi-color-text-2'>
                      {item.eyebrow}
                    </p>
                    <h2 className='mt-3 text-2xl font-semibold tracking-[-0.03em] text-semi-color-text-0'>
                      {item.title}
                    </h2>
                    <p className='mt-3 text-sm leading-6 text-semi-color-text-1 md:text-base'>
                      {item.description}
                    </p>
                  </div>
                ))}
              </div>
            </div>

            <div className='home-business-panel mt-10 p-5 md:mt-14 md:p-8'>
              <div className='flex flex-col gap-3 md:flex-row md:items-end md:justify-between'>
                <div>
                  <p className='text-xs font-medium uppercase tracking-[0.24em] text-semi-color-text-2'>
                    {t('可售卖的模型生态')}
                  </p>
                  <h2 className='mt-3 text-2xl font-semibold tracking-[-0.03em] text-semi-color-text-0 md:text-3xl'>
                    {t('覆盖主流供应商与客户接入场景')}
                  </h2>
                </div>
                <Text type='tertiary'>
                  {t('支持 OpenAI、Claude、Gemini、DeepSeek、Qwen 等热门模型')}
                </Text>
              </div>
              <div className='mt-6 flex flex-wrap items-center gap-3 md:gap-4'>
                {PROVIDER_ICONS.map((ProviderIcon, index) => (
                  <div className='home-provider-icon' key={index}>
                    <ProviderIcon size={28} />
                  </div>
                ))}
                <div className='home-provider-count'>30+</div>
              </div>
            </div>
          </div>
        </div>
      ) : (
        <div className='overflow-x-hidden w-full'>
          {homePageContent.startsWith('https://') ? (
            <iframe
              src={homePageContent}
              className='w-full h-screen border-none'
            />
          ) : (
            <div
              className='mt-[60px]'
              dangerouslySetInnerHTML={{ __html: homePageContent }}
            />
          )}
        </div>
      )}
    </div>
  );
};

export default Home;
