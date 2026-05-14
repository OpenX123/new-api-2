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

import React, { useEffect, useState } from 'react';
import { API, showError } from '../../helpers';
import { marked } from 'marked';
import { useTranslation } from 'react-i18next';

const DEFAULT_ABOUT_MD = `# 关于我们

本站是基于开源项目 **new-api** 搭建的 AI 接口聚合与分发平台，致力于为开发者与企业提供稳定、统一、可观测的 AI 能力入口。

## 我们提供什么

- **统一接入**：一次接入，即可调用 OpenAI、Claude、Gemini、Azure、Bedrock 等 40+ 家主流 AI 服务。
- **兼容协议**：原生兼容 OpenAI / Claude / Gemini 等多种协议，迁移成本近乎为零。
- **灵活计费**：按 Token 精确计费，支持模型倍率、分组倍率、额度与令牌多维度管理。
- **安全可控**：完善的鉴权、审计日志、请求限流与风控能力，保障服务使用安全。
- **运营完善**：自带管理后台、用量看板、渠道负载均衡与自动切换，方便企业化部署。

## 为什么选择我们

- 🚀 **稳定高效**：多渠道冗余与自动故障转移，将上游抖动对业务的影响降到最低。
- 🔐 **企业友好**：支持私有部署、模型白名单、团队与子账号管理，适配多种合规场景。
- 📊 **可观测**：全链路日志与用量统计，帮助您精细化管理成本与用量。
- 🤝 **长期迭代**：紧跟上游模型更新，持续同步最新模型与能力。

## 技术栈

- 后端：Go 1.22+、Gin、GORM
- 前端：React 18、Vite、Semi Design
- 存储：MySQL / PostgreSQL / SQLite + Redis

## 联系我们

如有合作、反馈或问题，欢迎通过站内"联系方式"或底部链接与我们取得联系。

> 管理员可在"系统设置 → 关于"中自定义本页内容（支持 Markdown 与 HTML）。
`;

const About = () => {
  const { t } = useTranslation();
  const [about, setAbout] = useState('');
  const [aboutLoaded, setAboutLoaded] = useState(false);
  const currentYear = new Date().getFullYear();

  const displayAbout = async () => {
    setAbout(localStorage.getItem('about') || '');
    try {
      const res = await API.get('/api/about');
      const { success, message, data } = res.data;
      if (success) {
        let aboutContent = data || '';
        if (aboutContent && !aboutContent.startsWith('https://')) {
          aboutContent = marked.parse(aboutContent);
        }
        setAbout(aboutContent);
        localStorage.setItem('about', aboutContent);
      } else {
        showError(message);
      }
    } catch (err) {
      // 静默失败，使用本地缓存或默认内容
    }
    setAboutLoaded(true);
  };

  useEffect(() => {
    displayAbout().then();
  }, []);

  const defaultAboutHtml = marked.parse(DEFAULT_ABOUT_MD);

  const customDescription = (
    <div style={{ textAlign: 'center' }}>
      <p>{t('可在设置页面设置关于内容，支持 HTML & Markdown')}</p>
      {t('New API项目仓库地址：')}
      <a
        href='https://github.com/QuantumNous/new-api'
        target='_blank'
        rel='noopener noreferrer'
        className='!text-semi-color-primary'
      >
        https://github.com/QuantumNous/new-api
      </a>
      <p>
        <a
          href='https://github.com/QuantumNous/new-api'
          target='_blank'
          rel='noopener noreferrer'
          className='!text-semi-color-primary'
        >
          NewAPI
        </a>{' '}
        {t('© {{currentYear}}', { currentYear })}{' '}
        <a
          href='https://github.com/QuantumNous'
          target='_blank'
          rel='noopener noreferrer'
          className='!text-semi-color-primary'
        >
          QuantumNous
        </a>{' '}
        {t('| 基于')}{' '}
        <a
          href='https://github.com/songquanpeng/one-api/releases/tag/v0.5.4'
          target='_blank'
          rel='noopener noreferrer'
          className='!text-semi-color-primary'
        >
          One API v0.5.4
        </a>{' '}
        © 2023{' '}
        <a
          href='https://github.com/songquanpeng'
          target='_blank'
          rel='noopener noreferrer'
          className='!text-semi-color-primary'
        >
          JustSong
        </a>
      </p>
      <p>
        {t('本项目根据')}
        <a
          href='https://github.com/songquanpeng/one-api/blob/v0.5.4/LICENSE'
          target='_blank'
          rel='noopener noreferrer'
          className='!text-semi-color-primary'
        >
          {t('MIT许可证')}
        </a>
        {t('授权，需在遵守')}
        <a
          href='https://www.gnu.org/licenses/agpl-3.0.html'
          target='_blank'
          rel='noopener noreferrer'
          className='!text-semi-color-primary'
        >
          {t('AGPL v3.0协议')}
        </a>
        {t('的前提下使用。')}
      </p>
    </div>
  );

  return (
    <div className='mt-[60px] px-2'>
      {aboutLoaded && about === '' ? (
        <div className='max-w-4xl mx-auto py-8 px-4 sm:px-6 lg:px-8'>
          <div
            className='prose prose-lg max-w-none bg-white rounded-lg shadow-sm p-8'
            style={{ fontSize: 'larger' }}
            dangerouslySetInnerHTML={{ __html: defaultAboutHtml }}
          />
          <div className='mt-6 text-center text-semi-color-text-2 text-sm'>
            {customDescription}
          </div>
        </div>
      ) : (
        <>
          {about.startsWith('https://') ? (
            <iframe
              src={about}
              style={{ width: '100%', height: '100vh', border: 'none' }}
            />
          ) : (
            <div
              style={{ fontSize: 'larger' }}
              dangerouslySetInnerHTML={{ __html: about }}
            ></div>
          )}
        </>
      )}
    </div>
  );
};

export default About;
