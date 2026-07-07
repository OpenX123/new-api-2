# 每用户独立倍率（Per-User Ratio）设计

日期：2026-07-06
状态：待评审

## 目标

允许管理员为单个用户设置一个独立的计费倍率。默认 1.0（不影响现状），与现有分组倍率**相乘叠加**：

```
最终倍率 = 模型倍率 × 分组倍率(或用户组特殊倍率) × 用户倍率
```

仅管理员可在后台用户编辑页调整；普通用户不可修改（可在日志计费详情中间接看到实际生效倍率）。

## 非目标

- 不做按用户×模型、按用户×分组的更细粒度倍率（现有 GroupGroupRatio 已覆盖组×组维度）。
- 不改动 web/classic 老前端（不加编辑入口，老前端管理员如需设置走默认前端或 API）。
- 用户倍率不参与「分组倍率为 0 则免费」之类的语义判断，只是纯乘数。

## 数据模型

`model/user.go` User 结构新增列：

```go
Ratio *float64 `json:"ratio,omitempty" gorm:"column:ratio"`
```

- 指针类型：`nil` 表示未设置，等效 1.0。不用 GORM default tag（遵守项目规则，避免 AutoMigrate 反复 ALTER）。
- AutoMigrate 自动 `ADD COLUMN`，SQLite/MySQL/PostgreSQL 均兼容。
- 语义归一：读取时 `nil` 或 `<= 0` 均按 1.0 处理（提供 `func (user *User) GetRatio() float64` / UserBase 同名方法作为唯一取值入口）。

## 数据流（计费路径）

1. **UserBase 缓存**（`model/user_cache.go`）：`UserBase` 增加 `Ratio *float64` 字段；`ToBaseUser()` 带上；`updateUserCache` 走 hash 序列化自动包含。
2. **Context 注入**：`UserBase.WriteContext` 新增 `common.SetContextKey(c, constant.ContextKeyUserRatio, user.GetRatio())`（float64，已归一）。`constant/context_key.go` 新增 `ContextKeyUserRatio`。
3. **RelayInfo**（`relay/common/relay_info.go`）：新增 `UserRatio float64` 字段，从 context 读取，读不到默认 1.0。
4. **计费汇入点**（`relay/helper/price.go` `HandleGroupRatio`）：在函数末尾把 `relayInfo.UserRatio` 乘进 `groupRatioInfo.GroupRatio`（两个分支——特殊倍率和普通分组倍率——之后统一乘）。由于普通计费、按次计费、分层表达式计费（`modelPriceHelperTiered`）都经过 `HandleGroupRatio` 返回的 `GroupRatioInfo`，一处修改全路径生效。
5. **日志展示**：不新增字段。日志 other 里的 `group_ratio` 即为乘过用户倍率后的实际生效值，用户在计费详情中看到真实倍率。

### 预扣费一致性

`ModelPriceHelper` 是预扣费与结算共同的定价入口，倍率在同一点生效，预扣与结算天然一致，无需额外处理。

## 管理端写入路径

`controller/user.go` `UpdateUser`（管理员编辑用户）+ `model/user.go` `Edit()`：

- `UpdateUser` 校验：若 `updatedUser.Ratio != nil`，要求 `*Ratio > 0`（上限 100，防误输入），否则报参数错误。
- `Edit()` 的 updates map 增加：仅当 `newUser.Ratio != nil` 时写入 `"ratio"`（前端总是提交该字段，默认 1；不提交则不动，兼容纯 API 调用方）。
- `Edit()` 已有 `updateUserCache` 调用，缓存自动刷新。
- 管理员创建用户（如有入口）不需要支持该字段，默认 nil 即 1.0。

### 越权防护

- 用户自助接口无越权风险（已核实）：`UpdateSelf` 最终只拷贝白名单字段（Username/Password/DisplayName）构造 `cleanUser` 再更新；`UpdateUserSetting` 只写 `UserSetting` JSON。ratio 均不可达。
- 该字段不在 `UserSetting` JSON 内，用户保存通知设置不会影响它。

## 前端（web/default）

`web/default/src/features/users/`：

- `types.ts`：User 类型加 `ratio?: number`。
- `user-form.ts` + `users-mutate-drawer.tsx`：编辑用户抽屉新增数字输入「用户倍率」，默认 1，step 0.01，min > 0；帮助文案说明与分组倍率相乘。
- `users-columns.tsx`：可选——列表展示 ratio ≠ 1 时的徽标（实现时若表格已拥挤则跳过）。
- i18n：新增文案 key（英文源串），补 zh 翻译，其余语言跑 `bun run i18n:sync`。

## 测试

- `relay/helper` 或就近包：表驱动测试 `HandleGroupRatio` —— (a) UserRatio=1 时行为与现状一致；(b) UserRatio=0.8 × 分组倍率 1.5 → 1.2；(c) 命中 GroupGroupRatio 特殊倍率时同样相乘；(d) context 缺失 UserRatio 时默认 1。
- `model` 层：`GetRatio()` 对 nil / 0 / 负数 / 正常值的归一。
- 使用 testify require/assert。

## 兼容性与风险

- 老数据 ratio 列为 NULL → 1.0，行为不变。
- Redis 用户缓存为 hash 字段序列化，新字段缺失时读出 nil → 1.0；无需清缓存。
- 上游合并冲突面：user.go / price.go / relay_info.go 均为上游活跃文件，改动保持最小（各 1-5 行）。
