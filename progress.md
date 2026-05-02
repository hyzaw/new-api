# Progress

- 已确认主卡点是请求链路中同步写 `log_details`。
- 已完成 Redis 预写 + 后台 worker 批量落库，并同步调整读取、`has_detail` 标记和删除逻辑。
- 已通过 `go test ./model -count=1`。
- 已通过 `go test ./... -run TestDoesNotExist -count=1` 编译校验。

---

## MySQL Schema/Index Optimization

- 2026-04-26: 开始针对 MySQL 8.0 部署审查模型字段和高频查询索引。
- 2026-04-26: 已为日志、充值、邀请提现、邀请明细、邀请钱包流水、用户状态统计、渠道筛选添加组合索引 tag。
- 2026-04-26: 已通过 `go test ./model -count=1`。
- 2026-04-26: 已通过 `go test ./... -run TestDoesNotExist -count=1`。

---

## Log Detail COS Storage

- 2026-04-26: 已新增 `LogDetail` COS 引用和元数据字段。
- 2026-04-26: 已实现默认关闭的腾讯云 COS 存储层，支持 gzip 上传、透明读取解压。
- 2026-04-26: 已在 `createLogDetail` 入 Redis/落库前进行大字段外置，避免 Redis/MySQL 保存数 MB 正文。
- 2026-04-26: 已确认旧日志清理不删除 COS 对象。
- 2026-04-26: 已通过 `go test ./model -count=1`。
- 2026-04-26: 已通过 `go test ./... -run TestDoesNotExist -count=1`。

## Lottery Redemption

- 2026-04-26: 已确认用户兑换入口为 controller/user.go:TopUp -> model.Redeem。
- 2026-04-26: 设计为 redemptions 增加抽奖配置字段，并新增 redemption_records 通过 redemption_id + user_id 唯一索引限制每用户一次。
- 2026-04-26: 已实现抽奖兑换码，支持管理员自定义密钥、区间随机、固定额度按权重随机、最大领取数量和每用户限领一次。
- 2026-04-26: 已通过 `go test ./model -count=1`、`go test ./... -run TestDoesNotExist -count=1`、`web` 目录 `bun run build`。

## Selective Upstream Merge

- 2026-04-26: 已选择性合入模型定价、gpt-5.5 completion ratio、图像 n 计价、DeepSeek V4 reasoning suffix、支付网关 PaymentProvider 防串单修复。
- 2026-04-26: 已合入计费表达式 `len` 变量，前端保留本地 `p+cr` 条件兼容并新增 LLM 辅助提示词。
- 2026-04-26: 已通过 `go test ./... -run TestDoesNotExist -count=1`、`go test ./pkg/billingexpr -run "TestLen|TestImageAudio|TestMultimodal" -count=1`、`go test ./service -run "TestBuildTieredTokenParams|TestTryTiered|TestTiered" -count=1`、`go test ./model -run "Test.*Payment.*|Test.*Pricing.*|Test.*ModelList.*" -count=1`、`go test ./controller -run "Test.*ModelList|Test.*Topup|Test.*Payment" -count=1`、`web` 目录 `bun run build`。
- 2026-04-26: `go test ./pkg/billingexpr ./service ./model -count=1` 中完整 `service` 业务测试失败于既有测试库缺少 `invite_details` 表，编译级全量校验已通过。

## Claude Cache Creation Billing

- 2026-04-28: 开始排查截图中 `cache_creation` tokens 未计入特惠计费的问题；已读 `pkg/billingexpr/expr.md`。
- 2026-04-28: 已修复阶梯计费 token 参数构造，aggregate cache creation 会通过 `NormalizeCacheCreationSplit` 回落到 `cc`，并添加截图同类用例测试。
- 2026-04-28: 已补用户提供 SSE `message_delta` 的 Claude usage 解析测试。
- 2026-04-28: 已通过 `go test ./service -run "TestBuildTieredTokenParams_Claude|TestTryTiered" -count=1`。
- 2026-04-28: 已通过 `go test ./relay/channel/claude -run "TestFormatClaudeResponseInfo_MessageDelta_CacheCreation5mFromSSE|TestBuildOpenAIStyleUsage|TestBuildMessageDelta" -count=1`。

## GPT Image Force Image Interface

- 2026-04-28: 开始排查 `gpt-image*` 模型分发路径。
- 2026-04-28: 已实现 `gpt-image*` 请求强制转换为 `ImageRequest` 并分发到图片生成 relay mode。
- 2026-04-28: 已通过 controller/helper 定向单测。
- 2026-04-28: 已通过 `go test ./controller ./relay/helper ./relay -run TestDoesNotExist -count=1` 编译级校验。
- 2026-04-28: `go test ./controller ./relay/helper -count=1` 中 controller 通过，helper 包失败于既有 `TestStreamScannerHandler_StreamStatus_PreInitialized`，与本次 `gpt-image` 分发改动无关。
