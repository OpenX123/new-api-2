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
import { useTranslation } from 'react-i18next';
import DocumentRenderer from '../../components/common/DocumentRenderer';

const DEFAULT_PRIVACY_POLICY = `## 引言

本平台（基于 new-api 搭建）高度重视您的个人信息与隐私保护。本政策旨在说明我们会收集哪些信息、如何使用这些信息、如何保障信息安全，以及您享有的相关权利。

## 一、我们收集的信息

1. **账号信息**：注册与登录时您提供的用户名、邮箱、手机号（若启用）以及第三方登录返回的唯一标识。
2. **使用信息**：您在使用 API 过程中产生的请求模型、调用时间、Token 数量、返回状态等日志信息。
3. **设备与网络信息**：访问 IP、User-Agent 等基础网络信息，用于风控与排障。
4. **支付信息**：若您进行充值，我们会通过第三方支付服务处理交易，本平台不直接保存您的银行卡或支付凭证明细。

## 二、我们如何使用信息

1. 提供并维护 API 服务，包括计费、额度管理、接口路由与故障排查；
2. 保障账号与平台安全，识别并防范滥用、欺诈行为；
3. 根据法律法规要求配合监管或司法机关的合法请求；
4. 在您明确同意的情况下，向您发送服务通知、活动或产品更新。

## 三、信息共享与披露

1. 除以下情形外，我们不会向任何第三方出售或出租您的个人信息：
   - 为完成 API 转发，需要将请求转发至您所选择的上游模型服务商；
   - 法律法规、监管部门或司法机关要求披露；
   - 为保护平台、用户或公众的合法权益所必需。
2. 上游服务商对其接收的数据负有独立的数据处理责任，请您在使用对应模型前阅读并理解其隐私政策。

## 四、数据存储与安全

1. 我们采取合理的技术与管理措施，包括但不限于传输加密、访问控制、日志审计，防止信息被未授权访问、篡改或泄露；
2. 调用日志默认按平台配置的保留策略存储，超期后会被清理或脱敏；
3. 如发生个人信息泄露等安全事件，我们将按照法律要求及时向您与相关监管机构进行通知。

## 五、您的权利

您对自己账号所关联的个人信息依法享有访问、更正、删除、撤回同意等权利。您可以通过以下方式行使相关权利：

1. 在"个人设置"页面修改账号信息或删除账号；
2. 通过平台内的联系方式与管理员取得联系，我们会在合理期限内响应。

## 六、Cookie 与本地存储

本平台使用 Cookie 及浏览器本地存储用于保持登录状态、记忆偏好设置。您可以通过浏览器设置拒绝或清除 Cookie，但可能会影响部分功能的使用。

## 七、政策更新

我们可能会根据业务或法律法规变化更新本政策。更新后的政策将在本站公布，重大变更会以显著方式提醒您。若您在更新后继续使用本服务，即视为接受更新后的政策。

## 八、联系我们

如您对本隐私政策或个人信息处理有任何疑问、意见，请通过平台内的联系方式与我们联系，我们会尽快响应。`;

const PrivacyPolicy = () => {
  const { t } = useTranslation();

  return (
    <DocumentRenderer
      apiEndpoint='/api/privacy-policy'
      title={t('隐私政策')}
      cacheKey='privacy_policy'
      emptyMessage={t('加载隐私政策内容失败...')}
      fallbackContent={DEFAULT_PRIVACY_POLICY}
    />
  );
};

export default PrivacyPolicy;
