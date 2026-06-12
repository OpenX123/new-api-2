# 发票管理功能设计（Invoice Management）

日期：2026-06-12
状态：已与用户逐节确认

## 1. 背景与目标

new-api 目前有充值订单（`TopUp`）与订阅订单（`SubscriptionOrder`），但没有发票能力。本功能允许：

- 用户在钱包页对自己**支付成功**的订单（充值 + 订阅）勾选多个、合并申请开票；
- 管理员在后台查看申请，上传发票文件（置为已开票）或驳回（填原因）；
- 开票完成/驳回时向用户接收邮箱发送邮件（已开票邮件附发票文件）；
- 发票文件存数据库（BLOB），用户可随时在页面重新下载。

## 2. 需求决策记录

| 决策点 | 结论 |
|---|---|
| 开票对象 | 充值订单 + 订阅订单 |
| 合并开票 | 支持多选合并，金额累加（服务端计算） |
| 抬头管理 | 一份默认抬头，存 `dto.UserSetting` JSON（`invoice_title`），申请时带出可改，可"设为默认" |
| 抬头字段 | 简化版：抬头类型（个人/企业）、抬头名称、税号（个人可空）、接收邮箱 |
| 文件存储 | 数据库 BLOB（独立表），限 PDF/PNG/JPG ≤ 10MB |
| 状态流转 | 待开票 → 已开票 / 已驳回；用户可撤销待开票申请 |
| 用户端入口 | 钱包页新增「发票」标签页 |
| 通知 | 开票完成/驳回发邮件，已开票附文件 |

## 3. 数据模型（方案 B：规范化三表）

### 3.1 `invoices` 申请主表

| 字段 | 类型 | 说明 |
|---|---|---|
| id | int 自增 | |
| user_id | int, index | 申请人 |
| invoice_no | varchar(64), unique | 申请单号，如 `INV202606121234` |
| title_type | int | 1=个人 2=企业 |
| title_name | varchar(255) | 抬头名称（快照） |
| tax_no | varchar(64) | 税号，个人可空 |
| email | varchar(255) | 接收邮箱（快照） |
| money | float64 | 合计金额，服务端按关联订单累加，不信任前端 |
| status | int | 1=待开票 2=已开票 3=已驳回 |
| reject_reason | text | 驳回原因 |
| remark | text | 用户备注（可选） |
| create_time / complete_time | int64 | 秒级 Unix 时间戳 |

### 3.2 `invoice_orders` 关联表

`id, invoice_id(index), order_type(varchar32: topup/subscription), order_id(int), trade_no(varchar255), money(float64)`

- 唯一索引 `(order_type, order_id)`：同一订单同时只能挂在一张有效申请上，数据库层防重复开票。
- 驳回/撤销时删除关联行即释放订单。

### 3.3 `invoice_files` 文件表

`id, invoice_id(unique index), filename(varchar255), mime_type(varchar64), size(int64), data(BLOB / []byte), upload_time(int64)`

- BLOB 独立成表，业务列表查询永不触碰文件数据。
- 重新上传覆盖原记录（开错换票）。

### 3.4 迁移与兼容

- 三表加入 `model/main.go` AutoMigrate；纯 GORM 标准类型，SQLite/MySQL/PostgreSQL 兼容（`[]byte` → longblob/bytea/blob）。
- 默认抬头：`dto.UserSetting` 新增 `invoice_title` 子对象，无表结构变更。

## 4. 状态机与业务规则

```
用户提交 ──► 待开票 ──管理员上传文件──► 已开票（终态，留档不可删；可重新上传文件）
                │
                ├─管理员驳回（必填原因）──► 已驳回（释放订单）
                └─用户撤销──► 删除申请与关联行（释放订单）
```

- 可开票订单条件：`status = success` 且不在任何有效申请的关联表中；**不设时间窗**（不复用充值记录 30 天限制）。
- 提交申请使用事务：插 `invoices` → 批量插 `invoice_orders`；撞唯一索引回滚，报"订单已在其他申请中"，并发安全。
- 服务端校验：订单归属本人、状态 success、未被占用。

## 5. 后端 API

### 用户端（`middleware.UserAuth()`）

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/api/user/invoice/orders` | 可开票订单列表（充值+订阅合并） |
| GET | `/api/user/invoice/self` | 我的申请列表（分页） |
| POST | `/api/user/invoice` | 提交申请 `{order_keys:[{type,id}], title_type, title_name, tax_no, email, remark, save_as_default}` |
| DELETE | `/api/user/invoice/:id` | 撤销（仅本人、仅待开票） |
| GET | `/api/user/invoice/:id/file` | 下载发票（仅本人、仅已开票） |

### 管理端（`middleware.AdminAuth()`）

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/api/invoice/` | 全部申请（分页、状态/关键字筛选，待开票置顶） |
| GET | `/api/invoice/:id` | 详情（含关联订单明细） |
| POST | `/api/invoice/:id/file` | 上传发票（multipart）→ 已开票 + 邮件 |
| POST | `/api/invoice/:id/reject` | 驳回 `{reason}` → 释放订单 + 邮件 |
| GET | `/api/invoice/:id/file` | 管理员下载已传文件 |

### 实现要点

- 文件上传为项目首个 multipart 接口：`c.FormFile` + MIME 白名单（application/pdf、image/png、image/jpeg）+ 10MB 上限，读入内存存 BLOB。
- 邮件复用现有邮件基础设施；如不支持附件则扩展 `SendEmailWithAttachment`，正文含下载指引兜底。
- JSON 全部走 `common.Marshal/Unmarshal`（项目 Rule 1）；后端提示语补 `i18n/` en/zh。
- 文件组织：`model/invoice.go`、`controller/invoice.go`，路由挂 `router/api-router.go`。

## 6. 前端

### 用户端 — 钱包页「发票」标签页（扩展 `features/wallet/`）

1. **申请开票卡片**：可开票订单表格（类型/订单号/时间/金额，多选），底部合计 +「申请开票」→ 弹窗填抬头（带出默认值，可勾"设为默认"）+ 备注 → 提交。
2. **我的申请列表**：申请单号、金额、抬头、状态徽标、时间；行操作：已开票→下载，待开票→撤销，已驳回→悬浮显示原因。

新增 `components/invoice-tab.tsx`、`components/invoice-apply-dialog.tsx`；API/类型扩展进 `api.ts`、`types.ts`。

### 管理端 — 「发票管理」页面（仿 `redemption-codes`）

- 新目录 `features/invoices/`（api.ts / types.ts / index.tsx / components/）。
- 列表：申请单号、用户名、抬头、金额、状态、时间；状态/关键字筛选，待开票置顶。
- 详情抽屉：抬头信息、关联订单明细、上传发票（文件选择）/ 驳回（必填原因）操作；已开票可重新上传。
- 路由 `routes/_authenticated/invoices/`，菜单仅管理员可见（沿用现有 admin 菜单权限模式）。

### i18n

新文案 `t('English key')`，补 `zh.json`，其余语言 `bun run i18n:sync`。

## 7. 测试

后端 Go 测试（跟随 `model/*_test.go` 模式）：

- 订单占用唯一性（并发重复提交只成功一单）；
- 状态流转合法性（不能驳回已开票、不能撤销已开票等）;
- 撤销/驳回释放订单后可再次申请；
- 金额服务端累加正确；
- 越权防护：用户 A 操作 B 的申请/文件 → 403。

前端不新增测试基建（与项目现状一致），类型检查 + 手动验证。本地仅运行 test/vet 类命令，不执行构建。

## 8. 不在本期范围

- 增值税专票完整字段（注册地址、电话、开户行、账号）
- 多抬头管理
- 对接税务系统自动开票
- 部分金额开票 / 一单拆多票
