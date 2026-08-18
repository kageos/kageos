export const CURRENT_LEGAL_POLICY_VERSION = '2026-08-18'

export type LegalDocumentKind = 'terms' | 'privacy'

export interface LegalSection {
  title: string
  paragraphs: string[]
  bullets?: string[]
}

export interface LegalDocument {
  title: string
  summary: string
  effectiveDate: string
  sections: LegalSection[]
}

const operator = '北京恰研智能科技有限公司'
const contactEmail = 'admin@kageos.ai'

const zhTerms: LegalDocument = {
  title: 'kageos 服务协议',
  summary: '本协议说明你使用 kageos 在线服务时与运营方之间的权利、义务和使用边界。',
  effectiveDate: '2026 年 8 月 18 日',
  sections: [
    {
      title: '一、协议主体与适用范围',
      paragraphs: [
        `kageos 在线服务由${operator}（以下称“我们”）运营。本协议适用于 app.kageos.com 及我们明确标注由本协议管辖的相关网页和功能。`,
        '你完成注册、授权注册或继续使用在线服务，即表示你已阅读并同意本协议及《隐私政策》。如果你不同意，请不要注册或使用在线服务。',
      ],
    },
    {
      title: '二、账户注册与安全',
      paragraphs: ['你应提供真实、准确且有权使用的信息，并妥善保管密码、第三方登录账号、访问令牌和设备。通过你的账户完成的操作原则上视为你的操作。'],
      bullets: [
        '发现账户被盗用、凭证泄露或异常访问时，请立即修改密码、撤销相关令牌并联系我们。',
        '企业或团队管理员创建的员工账户，还可能受该组织依法制定的管理规则约束。',
        '不得转让、出租、出售账户，或以虚假身份、冒用他人身份注册。',
      ],
    },
    {
      title: '三、服务内容与部署方式',
      paragraphs: [
        'kageos 提供工作空间、业务目录、表单、表格、文档、文件、定时任务、消息、连接器及数字员工等功能。具体功能以实际页面和当前版本为准。',
        '本协议主要适用于我们运营的在线实例。自行部署的 kageos 实例由部署者负责运营、用户管理、数据处理和合规义务；部署者应向其用户提供适用的服务条款和隐私规则。',
      ],
    },
    {
      title: '四、用户内容与必要授权',
      paragraphs: [
        '你保留对上传或创建的业务数据、文件、提示词、代码和其他内容依法享有的权利。为提供你请求的功能，你授予我们一项非独占、限于服务运行所必要范围的处理许可，包括存储、复制、转换、传输、展示、备份和恢复。',
        '你应确保有权处理和提交相关内容，并在向平台录入他人个人信息、客户资料、商业秘密或受保护数据前取得必要授权。未经授权不得上传违法、侵权或超出业务必要范围的数据。',
      ],
    },
    {
      title: '五、人工智能与第三方服务',
      paragraphs: [
        '数字员工和模型输出可能不准确、不完整或不适合特定用途。你应根据风险进行人工复核，不应仅依赖自动输出作出医疗、法律、金融、人事、安全等高影响决定。',
        '当你或管理员配置模型、消息、存储或其他第三方连接器并发起调用时，相关输入可能被发送给所选服务商。第三方服务的可用性、数据处理和输出同时受其自身条款约束。',
      ],
    },
    {
      title: '六、可接受使用规则',
      paragraphs: ['你不得利用服务实施违法行为、侵害他人合法权益、破坏平台安全或干扰其他用户。'],
      bullets: [
        '不得上传恶意程序，扫描、攻击、绕过权限、限额或其他安全控制。',
        '不得窃取凭证、冒充他人、传播欺诈信息或未经许可处理个人信息。',
        '不得利用服务生成、存储或传播法律禁止的内容。',
        '不得以影响平台稳定或其他用户正常使用的方式进行自动化调用、抓取或压力测试。',
      ],
    },
    {
      title: '七、知识产权与软件许可',
      paragraphs: [
        'kageos 名称、商标、界面设计、网站内容及我们提供的其他材料受适用法律保护。除另有说明外，本协议不转让相关知识产权。',
        'kageos 可自托管软件的使用以对应版本仓库中的许可证为准；服务协议不会扩大或缩小该软件许可证授予的权利。第三方组件仍适用其各自许可证。',
      ],
    },
    {
      title: '八、服务变更、中断与终止',
      paragraphs: [
        '我们可能因维护、升级、安全风险、法律要求或产品调整变更服务。对用户权益有重大影响的变更，我们会在合理可行范围内提前或及时通知。',
        '出现重大违约、攻击风险、违法要求、长期欠费（如未来提供付费服务）或为保护用户和平台所必需的情形时，我们可以限制、暂停或终止相关功能，并在适当情况下提供申诉或数据处理指引。',
      ],
    },
    {
      title: '九、责任说明',
      paragraphs: [
        '我们将采取合理措施维护服务安全和可用性，但互联网、云基础设施、第三方接口和人工智能模型可能发生延迟、中断、错误或不可抗力事件。你应根据业务重要程度自行建立备份、复核、权限和应急机制。',
        '在法律允许范围内，双方根据各自过错和实际因果关系承担责任；本条不排除法律规定不得限制或免除的责任。',
      ],
    },
    {
      title: '十、未成年人',
      paragraphs: ['服务主要面向具备完全民事行为能力的个人以及团队和企业用户。不满十四周岁的未成年人请勿自行注册；十四周岁以上未成年人应在监护人阅读并同意本协议和隐私政策后使用。'],
    },
    {
      title: '十一、协议更新、联系与争议解决',
      paragraphs: [
        `我们可能根据法律或服务变化更新本协议，并公布新的生效日期。重大变更会通过页面提示、站内消息或其他适当方式通知。问题、投诉或申诉可发送至 ${contactEmail}。`,
        '本协议适用中华人民共和国大陆地区法律。争议应先友好协商；协商不成的，可依法向有管辖权的人民法院提起诉讼。',
      ],
    },
  ],
}

const zhPrivacy: LegalDocument = {
  title: 'kageos 隐私政策',
  summary: '本政策说明 kageos 在线服务如何收集、使用、保存、共享和保护个人信息，以及你如何行使相关权利。',
  effectiveDate: '2026 年 8 月 18 日',
  sections: [
    {
      title: '一、个人信息处理者与适用范围',
      paragraphs: [
        `${operator}是 app.kageos.com 在线服务的个人信息处理者。隐私问题、投诉和个人信息权利请求可发送至 ${contactEmail}。`,
        '本政策适用于我们运营的在线实例。自行部署的实例由实际部署和运营者决定处理目的、方式及保存期限，该运营者应单独向其用户履行告知义务。',
      ],
    },
    {
      title: '二、我们处理的信息',
      paragraphs: ['我们按照功能所必需的最小范围处理下列信息。具体是否发生取决于你使用的登录方式、功能以及管理员配置。'],
      bullets: [
        '账户信息：用户 code、邮箱、密码哈希、昵称、头像、个人简介、所属部门及账户状态。我们不保存明文密码。',
        '第三方登录信息：当你选择微信、GitHub、Google 等已启用方式时，处理第三方用户标识，以及第三方按授权范围返回的昵称、头像和邮箱等信息。',
        '使用与安全信息：登录会话、令牌状态、操作时间、访问日志、IP 地址、浏览器或设备信息、错误和安全事件。',
        '工作空间内容：你提交的表单数据、表格数据、文档、文件、代码、提示词、对话、函数输入输出、任务和操作记录。',
        '配置与连接信息：你或管理员配置的模型和连接器信息。API 密钥、OAuth 令牌等凭证采用加密存储，并仅在执行相应功能时使用。',
        '沟通信息：你提交的客服、反馈、投诉、安全报告及相关联系方式。',
      ],
    },
    {
      title: '三、处理目的和必要性',
      paragraphs: ['我们仅在具有明确、合理目的并与目的直接相关的范围内处理信息。'],
      bullets: [
        '创建和管理账户，完成邮箱验证或第三方授权登录。',
        '提供工作空间、目录、文件、消息、定时任务、连接器和数字员工功能。',
        '执行权限控制、身份验证、操作审计、故障排查、反滥用和安全防护。',
        '响应客服、申诉、个人信息权利请求及履行法定义务。',
        '在去标识化或汇总基础上分析可靠性和改进产品。未经另行告知和取得必要授权，我们不会使用你的私有业务内容训练面向其他用户的通用模型。',
      ],
    },
    {
      title: '四、不同功能所需的信息',
      paragraphs: [
        '邮箱注册需要用户 code、邮箱、验证码和密码；拒绝提供将无法使用邮箱注册，但可选择实例已启用的其他登录方式。微信等第三方登录需要第三方用户标识；拒绝授权不会影响使用其他可用登录方式。',
        '工作空间内容、模型调用和连接器数据由你主动选择提交；不提交会使对应功能无法完成，但不影响与其无关的基础功能。',
      ],
    },
    {
      title: '五、Cookie、本地存储和会话',
      paragraphs: [
        '我们使用登录和安全所必需的浏览器本地存储或类似技术保存访问令牌、刷新令牌、语言和界面偏好。拒绝必要存储会导致登录状态和相关功能无法正常工作。',
        '当前基础服务不以个性化广告为目的使用非必要 Cookie。未来如新增需要同意的统计或营销技术，我们会在启用前另行告知并提供选择。',
      ],
    },
    {
      title: '六、委托处理、共享和第三方服务清单',
      paragraphs: [
        '我们不会出售个人信息。仅在提供你选择的功能、履行法律义务或保护合法权益所必要的范围内向第三方提供信息。',
        '当前可能涉及的第三方包括：腾讯微信相关平台（扫码或微信授权登录，处理第三方标识、昵称和头像）；登录页实际展示并由你选择的 GitHub、Google 等身份服务；由你或管理员配置的模型服务商；以及飞书、企业微信、钉钉等通知或连接器服务。每次调用传输的内容取决于具体配置和操作。',
        '云主机、对象存储、邮件和网络防护等基础设施服务商可能受托处理运行服务所必需的数据。我们要求其按照约定目的、范围和安全要求处理信息。',
      ],
    },
    {
      title: '七、跨境处理',
      paragraphs: [
        '在线服务的核心账户和工作空间数据原则上存储在中华人民共和国境内。我们不会默认主动将个人信息提供至境外。',
        '如果你或管理员选择境外模型、身份认证或连接器服务，相关输入和账户信息可能被传输至境外。我们将在依法需要时提供单独告知并取得单独同意；企业管理员还应确认其有权为相关业务数据配置该服务。',
      ],
    },
    {
      title: '八、保存期限和删除方式',
      paragraphs: [
        '账户资料和工作空间内容通常保存至账户或对应内容被删除、账号注销，或服务终止后完成必要清理。临时微信登录二维码尝试通常在约 5 分钟后失效；登录会话按令牌有效期、退出登录或管理员撤销确定。操作和下载审计记录按照实例配置的保留周期处理，默认配置通常为 90 天。',
        '无法逐项确定固定期限时，我们按照实现处理目的所需的最短时间确定，并综合账户状态、功能必要性、安全调查、争议处理及法定义务定期审查。期限届满后删除或匿名化；法律要求继续保存或技术上暂时难以删除的，仅进行存储和必要安全保护。',
      ],
    },
    {
      title: '九、你的权利',
      paragraphs: [
        `你可以通过个人设置修改部分资料；也可以发送邮件至 ${contactEmail} 申请查阅、复制、更正、补充、删除、限制处理、撤回同意、解释处理规则或注销账号。为保护账户安全，我们可能需要验证你的身份。`,
        '我们通常会在十五个工作日内核验并答复；情况复杂或法律另有规定时会说明处理进度。撤回同意不影响撤回前基于同意已进行处理的效力，也不影响基于其他合法事由进行的必要处理。',
      ],
    },
    {
      title: '十、信息安全与事件处置',
      paragraphs: [
        '我们采取访问控制、最小权限、传输保护、凭证加密、日志、备份和安全审查等措施。任何系统都无法保证绝对安全；发生可能影响个人权益的安全事件时，我们将依法采取补救措施并履行通知或报告义务。',
      ],
    },
    {
      title: '十一、未成年人',
      paragraphs: ['服务不面向不满十四周岁的未成年人。我们不会在明知的情况下处理其个人信息；如发现相关信息，将依法删除或采取其他措施。十四周岁以上未成年人应在监护人指导下使用。'],
    },
    {
      title: '十二、政策更新与联系我们',
      paragraphs: [
        `我们会根据功能和法律变化更新本政策并标注新的生效日期。处理目的、方式、信息种类或第三方接收方发生重大变化时，我们会依法重新告知并在需要时重新取得同意。联系邮箱：${contactEmail}。`,
      ],
    },
  ],
}

const enTerms: LegalDocument = {
  title: 'kageos Terms of Service',
  summary: 'These terms govern your use of the hosted kageos service.',
  effectiveDate: 'August 18, 2026',
  sections: [
    { title: '1. Operator and scope', paragraphs: [`The hosted service at app.kageos.com is operated by ${operator}. By registering or continuing to use it, you agree to these Terms and the Privacy Policy.`] },
    { title: '2. Accounts and security', paragraphs: ['Provide accurate information, protect your password, third-party account, tokens, and devices, and notify us promptly of suspected unauthorized access. You may not sell, rent, transfer, or impersonate another person through an account.'] },
    { title: '3. Service and deployment models', paragraphs: ['kageos provides workspaces, business directories, forms, tables, documents, files, schedules, messages, connectors, and agents. Operators of self-hosted instances are responsible for their own users, data handling, and legal notices.'] },
    { title: '4. Customer content', paragraphs: ['You retain rights you hold in content you submit. You grant us a non-exclusive permission to store, copy, transform, transmit, display, back up, and restore that content only as needed to provide the requested service. You must have the right to submit it.'] },
    { title: '5. AI and third-party services', paragraphs: ['AI outputs may be incomplete or inaccurate and require human review, especially for high-impact decisions. Inputs sent through a model, identity, messaging, storage, or connector provider are also governed by that provider’s terms.'] },
    { title: '6. Acceptable use', paragraphs: ['Do not violate law or third-party rights, upload malware, evade permissions or limits, steal credentials, impersonate others, or interfere with service availability or security.'] },
    { title: '7. Intellectual property and licenses', paragraphs: ['kageos branding, interface designs, and service materials are protected by applicable law. Self-hosted software and third-party components remain governed by the licenses shipped with the relevant version.'] },
    { title: '8. Changes, suspension, and termination', paragraphs: ['We may change, maintain, suspend, or discontinue features for security, legal, operational, or product reasons. Material changes will be communicated where reasonably practicable.'] },
    { title: '9. Responsibility', paragraphs: ['We use reasonable safeguards but cannot guarantee that internet infrastructure, third-party services, or AI outputs are uninterrupted or error-free. Each party remains responsible according to fault and causation, subject to non-waivable law.'] },
    { title: '10. Minors', paragraphs: ['The service is intended for adults and organizational users. Children under 14 must not register. Users aged 14 to 17 should use the service only with guidance and agreement from a guardian.'] },
    { title: '11. Updates, contact, and disputes', paragraphs: [`Updated terms will show a new effective date. Questions and complaints may be sent to ${contactEmail}. These Terms are governed by the laws of mainland China, without limiting mandatory protections that apply to you.`] },
  ],
}

const enPrivacy: LegalDocument = {
  title: 'kageos Privacy Policy',
  summary: 'How the hosted kageos service handles personal information.',
  effectiveDate: 'August 18, 2026',
  sections: [
    {
      title: '1. Controller and scope',
      paragraphs: [`${operator} operates app.kageos.com. Privacy and rights requests may be sent to ${contactEmail}. Self-hosted deployments are controlled by their respective operators.`],
    },
    {
      title: '2. Information and purposes',
      paragraphs: ['Depending on the features you choose, we process account details, authentication and security records, workspace content, files, prompts, operation logs, and encrypted model or connector credentials to provide, secure, and support the service.'],
    },
    {
      title: '3. Providers and transfers',
      paragraphs: ['Chosen sign-in, model, messaging, storage, and connector providers receive only the data needed for the requested function. An overseas provider may process selected inputs outside mainland China; separate notice and consent will be provided where required.'],
    },
    {
      title: '4. Retention and rights',
      paragraphs: [`We retain information for the shortest period needed for the relevant account, feature, security, dispute, and legal purpose. You may request access, copy, correction, deletion, restriction, withdrawal of consent, or account closure at ${contactEmail}.`],
    },
    {
      title: '5. Security, children, and updates',
      paragraphs: ['We use access controls, encryption, logging, backups, and other proportionate safeguards. The service is not directed to children under 14. Material policy changes will be clearly notified and will show a new effective date.'],
    },
  ],
}

export function getLegalDocument(kind: LegalDocumentKind, locale: string): LegalDocument {
  const isChinese = locale.toLowerCase().startsWith('zh')
  if (kind === 'privacy') return isChinese ? zhPrivacy : enPrivacy
  return isChinese ? zhTerms : enTerms
}
