# Findings

- 2026-05-05: `responses/compact` 在 `relay/helper/model_mapped.go` 中会把 `OriginModelName` 改成“映射后的上游模型名 + `-openai-compact` 后缀”。这会污染后续计费查价键。
- 2026-05-05: 因此当渠道模型映射把 `gpt-5.5` 映射到某个上游变体（例如带日期版本）时，计费层实际查的是映射后的 compact 名，而不是用户配置过价格的基础模型名，导致仍报“价格未配置”。
- 2026-05-05: 用户看到的第二段 `new_api_panic` 不是首个错误根因，而是响应已写出后，后续清理/记录逻辑再次 panic，被 `middleware/RelayPanicRecover` 追加写回客户端形成双 JSON。

- 现状：`model/log.go` 中 `createLogDetail` 直接 `LOG_DB.Create(detail)`。
- `GetLogDetail`、`markLogsHasDetail`、`DeleteOldLog` 都依赖 `log_details` 表。
- `RecordConsumeLog` / `RecordErrorLog` 都在写日志后同步调用 `createLogDetail`。
- 测试环境默认 `common.RedisEnabled = false`，需要保留无 Redis 兼容逻辑。
- 已实现 `log_detail:{id}` Redis 缓存和 `log_detail:pending_sync` 有序集合队列。
- 已实现后台 worker 按批次从 Redis 队列取出详情，再写入 `log_details` 主表。
- SQLite 测试环境未可靠提供 `log_id` 唯一约束，因此主表补写改为显式“查到则更新，未查到则插入”。

---

## MySQL Schema/Index Optimization Findings

- `invite_withdrawals` 管理列表按 `status`、`username LIKE`、`user_id` 过滤，并按 `id DESC` 分页；列表已不应返回 `receipt_code` 大字段。
- `invite_withdrawals.receipt_code` 业务允许最大 5MB base64 图片，MySQL 必须使用 `LONGTEXT`，否则旧镜像 AutoMigrate 会尝试改回 `TEXT` 并失败。
- `logs` 查询路径包含管理员列表、用户列表、统计和按 token 最近日志，主要过滤字段包括 `user_id`、`type`、`created_at`、`token_id`、`request_id`、`username`、`token_name`、`channel_id`、`group`。
- `top_ups` 查询路径包含用户充值列表、后台充值列表、交易号查询、状态统计和时间范围统计，需要关注 `user_id + create_time/id`、`status`、`trade_no` / 支付参考号。
- 已添加日志组合索引：用户+类型+ID、类型+时间+ID、时间+ID、令牌+ID。
- 已添加充值组合索引：用户+创建时间+ID、状态+创建时间+ID，并补充 `status` / `create_time` 类型与单列索引。
- 已添加邀请相关组合索引：提现用户+ID、提现状态+ID、邀请明细 inviter+type+time+id、钱包流水 user+created_at+id。
- 已添加用户状态统计索引 `status+role` 和渠道筛选索引 `type+status`。

---

## Log Detail COS Storage Findings

- `log_details` 当前写入会先进入 Redis 缓存/队列，再后台落库；无 Redis 时直接落库。
- 大字段必须在进入 Redis 前完成 COS 外置，否则 Redis 仍会持有数 MB 请求体/响应体。
- COS 存储需要读写权限：写入 PutObject，后台日志详情展示需要 GetObject；不需要 DeleteObject。
- 用户明确要求 COS 数据永远不要删除，因此旧日志清理路径保持只删 Redis 和数据库记录，不调用 COS 删除。
- 新增字段只通过 GORM AutoMigrate 增加列，保留 SQLite/MySQL/PostgreSQL 兼容和旧内联行读取兼容。

---

## Claude Cache Creation Billing Findings

- 截图费用可复算为 `(cache_read 25329 * 0.5 + completion 282 * 25 + input 6 * 5) / 1e6 * 0.3 = 0.00592335`，与显示 `¥0.005924` 一致。
- 同一截图显示 `cache_creation 6846` 且价格 `¥6.25 / 1M tokens`，但该项未进入上述实际金额，疑点集中在结算时没有把 cache creation token 传入表达式变量。
- 根因：`BuildTieredTokenParams` 在 `usage.UsageSemantic == "anthropic"` 时只读 `ClaudeCacheCreation5mTokens/1hTokens`，没有把只有 aggregate `PromptTokensDetails.CachedCreationTokens` 的 Claude usage 回落到 `cc`。
- 用户提供的 SSE `message_delta` 包含 `cache_creation.ephemeral_5m_input_tokens=6846`；已补解析回归测试确认该字段会进入 `ClaudeCacheCreation5mTokens`。

---

## GPT Image Force Image Interface Findings

- 待排查：当前 `gpt-image` 模型可能在 chat/completions 或 responses 路径被当成文本模型分发。
- `controller.Relay` 在读取并校验请求后生成 `RelayInfo`，实际 handler 根据 `RelayMode` 分发；图片 handler 需要 `*dto.ImageRequest`。
- 仅改 `RelayMode` 不够，必须同时把通用 OpenAI/Responses 请求转换成 `ImageRequest`，并把上游 `RequestURLPath` 固定到 `/v1/images/generations`。
- `relay/helper.GetAndValidateRequest` 现在对 OpenAI format 先尝试按 image request 解析 `gpt-image*` + `prompt` 请求，避免 image 形态请求走 chat 校验时报 `messages` 缺失。
