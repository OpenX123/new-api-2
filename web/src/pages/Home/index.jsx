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
import { API, showError, copy, showSuccess } from '../../helpers';
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
  Cohere,
  Claude,
  Gemini,
  Suno,
  Minimax,
  Wenxin,
  Spark,
  Qingyan,
  DeepSeek,
  Qwen,
  Midjourney,
  Grok,
  AzureAI,
  Hunyuan,
  Xinference,
} from '@lobehub/icons';

const Home = () => {
  const { t, i18n } = useTranslation();
  const [statusState] = useContext(StatusContext);
  const actualTheme = useActualTheme();
  const [homePageContentLoaded, setHomePageContentLoaded] = useState(false);
  const [homePageContent, setHomePageContent] = useState('');
  const [noticeVisible, setNoticeVisible] = useState(false);
  const isMobile = useIsMobile();
  const isDemoSiteMode = statusState?.status?.demo_site_enabled || false;
  const docsLink =
    statusState?.status?.docs_link || 'https://docs.newapi.pro';
  const serverAddress =
    statusState?.status?.server_address || `${window.location.origin}`;
  const isChinese = i18n.language.startsWith('zh');

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

  // Doubled endpoints for seamless CSS scroll loop
  const scrollEndpoints = [
    '/v1/chat/completions',
    '/v1/responses',
    '/v1/messages',
    '/v1/embeddings',
    '/v1/images/generations',
    '/v1/audio/speech',
    '/v1/chat/completions',
    '/v1/responses',
    '/v1/messages',
    '/v1/embeddings',
    '/v1/images/generations',
    '/v1/audio/speech',
  ];

  const iconItems = [
    { Icon: OpenAI, name: 'OpenAI', color: false },
    { Icon: Claude, name: 'Claude', color: true },
    { Icon: Gemini, name: 'Gemini', color: true },
    { Icon: DeepSeek, name: 'DeepSeek', color: true },
    { Icon: Grok, name: 'Grok', color: false },
    { Icon: XAI, name: 'xAI', color: false },
    { Icon: Qwen, name: 'Qwen', color: true },
    { Icon: Zhipu, name: 'Zhipu', color: true },
    { Icon: Moonshot, name: 'Moonshot', color: false },
    { Icon: Volcengine, name: 'Volcengine', color: true },
    { Icon: Wenxin, name: 'Wenxin', color: true },
    { Icon: Spark, name: 'Spark', color: true },
    { Icon: Minimax, name: 'Minimax', color: true },
    { Icon: Cohere, name: 'Cohere', color: true },
    { Icon: Hunyuan, name: 'Hunyuan', color: true },
    { Icon: AzureAI, name: 'Azure', color: true },
    { Icon: Midjourney, name: 'Midjourney', color: false },
    { Icon: Suno, name: 'Suno', color: false },
    { Icon: Xinference, name: 'Xinference', color: true },
    { Icon: Qingyan, name: 'Qingyan', color: true },
  ];

  const renderDefaultHome = () => (
    <div className='home-root w-full overflow-x-hidden'>
      {/* ===== Hero Section ===== */}
      <section
        className='home-section-bg-0 relative w-full min-h-screen flex items-center overflow-hidden'
        style={{ background: 'var(--semi-color-bg-0)', color: 'var(--semi-color-text-0)' }}
      >
        <div className='hero-glow' />
        <div className='w-full max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-20 md:py-24 lg:py-0'>
          <div className='flex flex-col lg:flex-row items-center gap-12 lg:gap-16'>
            {/* Left: Text content */}
            <div className='flex-1 text-center lg:text-left fade-up'>
              <h1
                className={`text-4xl md:text-5xl lg:text-6xl xl:text-7xl font-bold text-semi-color-text-0 leading-tight ${isChinese ? 'tracking-wide' : ''}`}
              >
                {t('统一的')}
                <br />
                <span className='shine-text'>{t('大模型接口网关')}</span>
              </h1>
              <p className='text-lg md:text-xl text-semi-color-text-2 mt-6 max-w-lg mx-auto lg:mx-0'>
                {t('更好的价格，更好的稳定性，只需要将模型基址替换为：')}
              </p>

              {/* Base URL Card */}
              <div
                className='mt-6 rounded-2xl p-4 max-w-md mx-auto lg:mx-0 fade-up fade-up-d2'
                style={{
                  background: 'var(--semi-color-bg-1)',
                  border: '1px solid var(--semi-color-border)',
                }}
              >
                <div className='flex items-center gap-2'>
                  <code
                    className='flex-1 text-sm font-mono truncate'
                    style={{ color: 'var(--semi-color-text-0)' }}
                  >
                    {serverAddress}
                  </code>
                  <div className='endpoint-scroller flex-shrink-0'>
                    <ul>
                      {scrollEndpoints.map((ep, i) => (
                        <li
                          key={i}
                          className='text-xs'
                          style={{ color: 'var(--semi-color-primary)' }}
                        >
                          {ep}
                        </li>
                      ))}
                    </ul>
                  </div>
                  <Button
                    type='primary'
                    theme='solid'
                    size='small'
                    icon={<IconCopy />}
                    className='!rounded-full flex-shrink-0'
                    onClick={handleCopyBaseURL}
                  />
                </div>
              </div>

              {/* CTA Buttons */}
              <div className='flex flex-row gap-4 justify-center lg:justify-start items-center mt-8 fade-up fade-up-d3'>
                <Link to='/console/token'>
                  <Button
                    theme='solid'
                    type='primary'
                    size={isMobile ? 'default' : 'large'}
                    className='!rounded-full px-8 py-2'
                    icon={<IconPlay />}
                  >
                    {t('获取密钥')}
                  </Button>
                </Link>
                {isDemoSiteMode && statusState?.status?.version ? (
                  <Button
                    size={isMobile ? 'default' : 'large'}
                    className='flex items-center !rounded-full px-6 py-2'
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
                      className='flex items-center !rounded-full px-6 py-2'
                      icon={<IconFile />}
                      onClick={() => window.open(docsLink, '_blank')}
                    >
                      {t('文档')}
                    </Button>
                  )
                )}
              </div>
            </div>

            {/* Right: Icon grid */}
            <div className='flex-1 relative fade-up fade-up-d2'>
              {/* Network SVG lines */}
              <svg
                className='absolute inset-0 w-full h-full pointer-events-none'
                viewBox='0 0 100 100'
                preserveAspectRatio='none'
                style={{ zIndex: 0 }}
              >
                <line
                  x1='20' y1='30' x2='50' y2='50'
                  stroke='var(--semi-color-border)'
                  strokeWidth='0.3'
                  strokeDasharray='2 1.5'
                  className='network-line'
                />
                <line
                  x1='80' y1='25' x2='50' y2='50'
                  stroke='var(--semi-color-border)'
                  strokeWidth='0.3'
                  strokeDasharray='2 1.5'
                  className='network-line'
                  style={{ animationDelay: '1s' }}
                />
                <line
                  x1='30' y1='75' x2='50' y2='50'
                  stroke='var(--semi-color-border)'
                  strokeWidth='0.3'
                  strokeDasharray='2 1.5'
                  className='network-line'
                  style={{ animationDelay: '2s' }}
                />
                <line
                  x1='75' y1='70' x2='50' y2='50'
                  stroke='var(--semi-color-border)'
                  strokeWidth='0.3'
                  strokeDasharray='2 1.5'
                  className='network-line'
                  style={{ animationDelay: '0.5s' }}
                />
              </svg>
              <div
                className='grid grid-cols-4 sm:grid-cols-5 gap-3 md:gap-4 relative'
                style={{ zIndex: 1 }}
              >
                {iconItems.map(({ Icon, name, color }, idx) => (
                  <div
                    key={name}
                    className={`icon-card float-node-${idx % 4}`}
                    title={name}
                  >
                    {color ? <Icon.Color size={32} /> : <Icon size={32} />}
                  </div>
                ))}
                <div className='icon-card float-node-0'>
                  <Typography.Text className='!text-xl font-bold'>
                    30+
                  </Typography.Text>
                </div>
              </div>
            </div>
          </div>
        </div>
      </section>

      {/* ===== Features Section ===== */}
      <section
        className='home-section-bg-1 w-full py-20 md:py-28'
        style={{ background: 'var(--semi-color-bg-1)' }}
      >
        <div className='max-w-7xl mx-auto px-4 sm:px-6 lg:px-8'>
          <div className='text-center mb-16 fade-up'>
            <h2 className='text-3xl md:text-4xl lg:text-5xl font-bold'>
              <span className='text-gradient'>
                {t('强悍内核，轻盈体验。')}
              </span>
            </h2>
            <p className='text-lg text-semi-color-text-2 mt-4 max-w-2xl mx-auto'>
              {t('为开发者与企业精心打造的 AI 网关')}
            </p>
          </div>
          <div className='grid grid-cols-1 md:grid-cols-3 gap-6 lg:gap-8'>
            {/* Feature 1 */}
            <div className='apple-card fade-up fade-up-d1'>
              <div
                className='w-14 h-14 rounded-2xl flex items-center justify-center mb-6'
                style={{ background: 'rgba(0, 122, 255, 0.1)' }}
              >
                <svg width='28' height='28' viewBox='0 0 24 24' fill='none' stroke='#007AFF' strokeWidth='2' strokeLinecap='round' strokeLinejoin='round'>
                  <path d='M13 2L3 14h9l-1 8 10-12h-9l1-8z' />
                </svg>
              </div>
              <h3 className='text-xl font-semibold text-semi-color-text-0 mb-3'>
                {t('超低延迟')}
              </h3>
              <p className='text-semi-color-text-2 leading-relaxed'>
                {t('边缘节点加速，智能路由调度，毫秒级响应，让每一次 API 调用都快人一步。')}
              </p>
            </div>
            {/* Feature 2 */}
            <div className='apple-card fade-up fade-up-d2'>
              <div
                className='w-14 h-14 rounded-2xl flex items-center justify-center mb-6'
                style={{ background: 'rgba(175, 82, 222, 0.1)' }}
              >
                <svg width='28' height='28' viewBox='0 0 24 24' fill='none' stroke='#AF52DE' strokeWidth='2' strokeLinecap='round' strokeLinejoin='round'>
                  <path d='M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z' />
                  <path d='M9 12l2 2 4-4' />
                </svg>
              </div>
              <h3 className='text-xl font-semibold text-semi-color-text-0 mb-3'>
                {t('坚如磐石')}
              </h3>
              <p className='text-semi-color-text-2 leading-relaxed'>
                {t('企业级高可用架构，并发管控，自动熔断降级，稳定性高达 99.9%。')}
              </p>
            </div>
            {/* Feature 3 */}
            <div className='apple-card fade-up fade-up-d3'>
              <div
                className='w-14 h-14 rounded-2xl flex items-center justify-center mb-6'
                style={{ background: 'rgba(255, 149, 0, 0.1)' }}
              >
                <svg width='28' height='28' viewBox='0 0 24 24' fill='none' stroke='#FF9500' strokeWidth='2' strokeLinecap='round' strokeLinejoin='round'>
                  <rect x='2' y='2' width='20' height='8' rx='2' />
                  <rect x='2' y='14' width='20' height='8' rx='2' />
                  <line x1='6' y1='6' x2='6.01' y2='6' />
                  <line x1='6' y1='18' x2='6.01' y2='18' />
                </svg>
              </div>
              <h3 className='text-xl font-semibold text-semi-color-text-0 mb-3'>
                {t('全模型覆盖')}
              </h3>
              <p className='text-semi-color-text-2 leading-relaxed'>
                {t('原生协议兼容，支持 40+ 主流 AI 模型供应商，无缝切换模型。')}
              </p>
            </div>
          </div>
        </div>
      </section>

      {/* ===== Integration Section ===== */}
      <section
        className='w-full py-20 md:py-28'
        style={{ background: '#1d1d1f' }}
      >
        <div className='max-w-7xl mx-auto px-4 sm:px-6 lg:px-8'>
          <div className='flex flex-col lg:flex-row items-center gap-12 lg:gap-16'>
            {/* Left text */}
            <div className='flex-1 fade-up'>
              <h2 className='text-3xl md:text-4xl lg:text-5xl font-bold text-white leading-tight'>
                {t('一行代码，拥抱整个')}
                <br />
                <span style={{ color: '#007AFF' }}>{t('AI 世界。')}</span>
              </h2>
              <p className='text-lg mt-6' style={{ color: '#a1a1a6' }}>
                {t('完全兼容 OpenAI SDK，迁移无需任何代码改动。')}
              </p>
              <div className='mt-8 space-y-4'>
                {[
                  t('无需修改业务逻辑'),
                  t('完整支持流式传输'),
                  t('内置计费与日志'),
                ].map((item, i) => (
                  <div key={i} className='flex items-center gap-3'>
                    <svg width='20' height='20' viewBox='0 0 20 20' fill='none'>
                      <circle cx='10' cy='10' r='10' fill='#007AFF' fillOpacity='0.2' />
                      <path d='M6 10l3 3 5-5' stroke='#007AFF' strokeWidth='2' strokeLinecap='round' strokeLinejoin='round' />
                    </svg>
                    <span style={{ color: '#d1d1d6' }}>{item}</span>
                  </div>
                ))}
              </div>
            </div>

            {/* Right code window */}
            <div className='flex-1 w-full fade-up fade-up-d2'>
              <div className='code-window'>
                <div className='code-header'>
                  <div className='code-dot' style={{ background: '#ff5f57' }} />
                  <div className='code-dot' style={{ background: '#febc2e' }} />
                  <div className='code-dot' style={{ background: '#28c840' }} />
                  <span className='ml-3 text-sm' style={{ color: '#808080' }}>
                    main.py
                  </span>
                </div>
                <pre className='p-5 text-sm leading-relaxed overflow-x-auto' style={{ margin: 0 }}>
                  <code>
                    <span style={{ color: '#c586c0' }}>import</span>
                    <span style={{ color: '#d4d4d4' }}> openai</span>
                    {'\n\n'}
                    <span style={{ color: '#d4d4d4' }}>client = openai.</span>
                    <span style={{ color: '#4ec9b0' }}>OpenAI</span>
                    <span style={{ color: '#d4d4d4' }}>(</span>
                    {'\n'}
                    <span style={{ color: '#d4d4d4' }}>{'    '}api_key=</span>
                    <span style={{ color: '#ce9178' }}>"sk-your-api-key"</span>
                    <span style={{ color: '#d4d4d4' }}>,</span>
                    {'\n'}
                    <span style={{ color: '#6a9955' }}>{'    '}# {t('将目标地址指向网关')}</span>
                    {'\n'}
                    <span style={{ color: '#d4d4d4' }}>{'    '}base_url=</span>
                    <span style={{ color: '#ce9178' }}>"{serverAddress}/v1"</span>
                    {'\n'}
                    <span style={{ color: '#d4d4d4' }}>)</span>
                    {'\n\n'}
                    <span style={{ color: '#d4d4d4' }}>response = client.chat.completions.</span>
                    <span style={{ color: '#dcdcaa' }}>create</span>
                    <span style={{ color: '#d4d4d4' }}>(</span>
                    {'\n'}
                    <span style={{ color: '#d4d4d4' }}>{'    '}model=</span>
                    <span style={{ color: '#ce9178' }}>"gpt-4-turbo"</span>
                    <span style={{ color: '#d4d4d4' }}>,</span>
                    {'\n'}
                    <span style={{ color: '#d4d4d4' }}>{'    '}messages=[</span>
                    {'\n'}
                    <span style={{ color: '#d4d4d4' }}>{'        '}</span>
                    <span style={{ color: '#d4d4d4' }}>{'{'}</span>
                    <span style={{ color: '#9cdcfe' }}>"role"</span>
                    <span style={{ color: '#d4d4d4' }}>: </span>
                    <span style={{ color: '#ce9178' }}>"user"</span>
                    <span style={{ color: '#d4d4d4' }}>, </span>
                    <span style={{ color: '#9cdcfe' }}>"content"</span>
                    <span style={{ color: '#d4d4d4' }}>: </span>
                    <span style={{ color: '#ce9178' }}>"Hello!"</span>
                    <span style={{ color: '#d4d4d4' }}>{'}'}</span>
                    {'\n'}
                    <span style={{ color: '#d4d4d4' }}>{'    '}]</span>
                    {'\n'}
                    <span style={{ color: '#d4d4d4' }}>)</span>
                    {'\n\n'}
                    <span style={{ color: '#c586c0' }}>print</span>
                    <span style={{ color: '#d4d4d4' }}>(response.choices[</span>
                    <span style={{ color: '#b5cea8' }}>0</span>
                    <span style={{ color: '#d4d4d4' }}>].message.content)</span>
                  </code>
                </pre>
              </div>
            </div>
          </div>
        </div>
      </section>

      {/* ===== CTA Section ===== */}
      <section
        className='home-section-bg-0 w-full py-20 md:py-28'
        style={{ background: 'var(--semi-color-bg-0)', color: 'var(--semi-color-text-0)' }}
      >
        <div className='max-w-4xl mx-auto px-4 text-center fade-up'>
          <h2 className='text-3xl md:text-4xl lg:text-5xl font-bold text-semi-color-text-0'>
            {t('准备好加速您的')}
            <br />
            <span style={{ color: 'var(--semi-color-primary)' }}>
              {t('AI 应用了吗？')}
            </span>
          </h2>
          <p className='text-lg text-semi-color-text-2 mt-6'>
            {t('注册即赠送测试额度，极速体验。')}
          </p>
          <div className='mt-8'>
            <Link to='/register'>
              <Button
                theme='solid'
                type='primary'
                size='large'
                className='!rounded-full px-10 py-3 text-lg'
              >
                {t('立即创建账号')}
              </Button>
            </Link>
          </div>
        </div>
      </section>

      {/* ===== Footer ===== */}
      <footer
        className='home-section-bg-0 w-full py-8'
        style={{
          background: 'var(--semi-color-bg-0)',
          borderTop: '1px solid var(--semi-color-border)',
        }}
      >
        <div className='max-w-7xl mx-auto px-4 sm:px-6 lg:px-8'>
          <div className='flex flex-col md:flex-row items-center justify-between gap-4'>
            <div className='flex items-center gap-3'>
              <div
                className='home-brand-square w-8 h-8 rounded-lg flex items-center justify-center text-white font-bold text-sm'
                style={{ background: '#1d1d1f' }}
              >
                N
              </div>
              <span className='font-semibold text-semi-color-text-0'>
                New API
              </span>
            </div>
            <div className='flex items-center gap-6 text-sm text-semi-color-text-2'>
              <Link
                to='/console'
                className='hover:text-semi-color-text-0 transition-colors'
                style={{ color: 'var(--semi-color-text-2)', textDecoration: 'none' }}
              >
                {t('控制台')}
              </Link>
              {docsLink && (
                <a
                  href={docsLink}
                  target='_blank'
                  rel='noopener noreferrer'
                  className='hover:text-semi-color-text-0 transition-colors'
                  style={{ color: 'var(--semi-color-text-2)', textDecoration: 'none' }}
                >
                  {t('文档')}
                </a>
              )}
              <Link
                to='/about'
                className='hover:text-semi-color-text-0 transition-colors'
                style={{ color: 'var(--semi-color-text-2)', textDecoration: 'none' }}
              >
                {t('关于')}
              </Link>
              <Link
                to='/user-agreement'
                className='hover:text-semi-color-text-0 transition-colors'
                style={{ color: 'var(--semi-color-text-2)', textDecoration: 'none' }}
              >
                {t('服务条款')}
              </Link>
              <Link
                to='/privacy-policy'
                className='hover:text-semi-color-text-0 transition-colors'
                style={{ color: 'var(--semi-color-text-2)', textDecoration: 'none' }}
              >
                {t('隐私政策')}
              </Link>
            </div>
          </div>
        </div>
      </footer>

    </div>
  );

  return (
    <div className='w-full overflow-x-hidden'>
      <NoticeModal
        visible={noticeVisible}
        onClose={() => setNoticeVisible(false)}
        isMobile={isMobile}
      />
      {homePageContentLoaded && homePageContent === '' ? (
        renderDefaultHome()
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
