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

const DEFAULT_USER_AGREEMENT = `## 一、服务说明

本服务是基于 new-api 搭建的 AI 接口聚合与分发平台（以下简称"本服务"）。在使用本服务前，请您仔细阅读并充分理解以下全部条款。您一旦注册、登录或使用本服务，即视为已阅读并同意本协议的全部内容。

## 二、账号与使用

1. 您应使用真实、准确的信息完成注册，并妥善保管账号与密钥（API Key / Token）。因您自身原因导致的密钥泄露或滥用，由您自行承担责任。
2. 您承诺不会利用本服务从事违反当地法律法规或公序良俗的活动，包括但不限于生成、传播违法违规内容、进行网络攻击、滥用接口等。
3. 您理解并同意，本服务仅作为上游 AI 服务商的中转与分发平台，不对上游模型输出的内容承担责任。

## 三、计费与额度

1. 本服务按模型的调用次数 / Token 数量及倍率计费，具体规则以平台内展示为准。
2. 充值完成后，额度将实时到账；非因平台原因导致的额度消耗，一经消费不予退还。
3. 如对计费结果有疑问，请及时通过站内渠道与管理员联系核实。

## 四、服务可用性

1. 本服务会尽最大努力保证服务的稳定性与连续性，但不保证 100% 可用；因上游服务商故障、网络异常、不可抗力等原因导致的中断，不视为违约。
2. 平台有权对服务进行升级、维护、调整模型列表与价格，相关变更会通过站内通知或公告方式进行告知。

## 五、内容与知识产权

1. 您通过本服务生成的内容版权归属依据上游模型服务商的条款执行；您应自行确认所生成内容可合法使用。
2. 本服务所涉及的界面、文案、商标、代码等知识产权归本平台或其权利人所有，未经授权不得复制、传播或用于商业用途。

## 六、违约与责任

1. 若您违反本协议，平台有权视情节轻重采取限速、暂停或终止服务、封禁账号等措施，且无需退还已使用的额度。
2. 在法律允许的最大范围内，因使用本服务造成的任何间接、附带、惩罚性损失，平台不承担责任。

## 七、协议变更

本协议可能会根据业务发展和法律法规要求适时更新，更新后的协议一经在本站公布即生效。若您继续使用本服务，即视为接受修订后的协议。

## 八、联系我们

如您对本协议有任何疑问、意见或建议，请通过平台内的联系方式与我们取得联系。`;

const UserAgreement = () => {
  const { t } = useTranslation();

  return (
    <DocumentRenderer
      apiEndpoint='/api/user-agreement'
      title={t('服务条款')}
      cacheKey='user_agreement'
      emptyMessage={t('加载用户协议内容失败...')}
      fallbackContent={DEFAULT_USER_AGREEMENT}
    />
  );
};

export default UserAgreement;
