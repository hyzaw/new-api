package model

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"

	"github.com/bytedance/gopkg/util/gopool"
	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"
	"time"
)

type Log struct {
	Id               int    `json:"id" gorm:"index:idx_created_at_id,priority:1;index:idx_user_id_id,priority:2;index:idx_logs_user_type_id,priority:3;index:idx_logs_type_created_id,priority:3;index:idx_logs_created_at_id,priority:2;index:idx_logs_token_id_id,priority:2"`
	UserId           int    `json:"user_id" gorm:"index;index:idx_user_id_id,priority:1;index:idx_logs_user_type_id,priority:1"`
	CreatedAt        int64  `json:"created_at" gorm:"bigint;index:idx_created_at_id,priority:2;index:idx_created_at_type;index:idx_logs_type_created_id,priority:2;index:idx_logs_created_at_id,priority:1"`
	Type             int    `json:"type" gorm:"index:idx_created_at_type;index:idx_logs_user_type_id,priority:2;index:idx_logs_type_created_id,priority:1"`
	Content          string `json:"content"`
	Username         string `json:"username" gorm:"index;index:index_username_model_name,priority:2;default:''"`
	TokenName        string `json:"token_name" gorm:"index;default:''"`
	ModelName        string `json:"model_name" gorm:"index;index:index_username_model_name,priority:1;default:''"`
	Quota            int    `json:"quota" gorm:"default:0"`
	PromptTokens     int    `json:"prompt_tokens" gorm:"default:0"`
	CompletionTokens int    `json:"completion_tokens" gorm:"default:0"`
	UseTime          int    `json:"use_time" gorm:"default:0"`
	IsStream         bool   `json:"is_stream"`
	ChannelId        int    `json:"channel" gorm:"index"`
	ChannelName      string `json:"channel_name" gorm:"->"`
	TokenId          int    `json:"token_id" gorm:"default:0;index;index:idx_logs_token_id_id,priority:1"`
	Group            string `json:"group" gorm:"index"`
	Ip               string `json:"ip" gorm:"index;default:''"`
	RequestId        string `json:"request_id,omitempty" gorm:"type:varchar(64);index:idx_logs_request_id;default:''"`
	Other            string `json:"other"`
	HasDetail        bool   `json:"has_detail,omitempty" gorm:"-"`
	LogDetailStorage string `json:"log_detail_storage,omitempty" gorm:"-"`
}

type LogDetail struct {
	Id                   int    `json:"id"`
	LogId                int    `json:"log_id" gorm:"uniqueIndex"`
	RequestBodyEncoding  string `json:"request_body_encoding" gorm:"type:varchar(16);default:''"`
	RequestBody          string `json:"request_body" gorm:"type:text"`
	RequestBodyStorage   string `json:"request_body_storage,omitempty" gorm:"type:varchar(16);default:''"`
	RequestBodyRef       string `json:"request_body_ref,omitempty" gorm:"type:varchar(512);default:''"`
	RequestBodySize      int64  `json:"request_body_size,omitempty" gorm:"default:0"`
	RequestBodyHash      string `json:"request_body_hash,omitempty" gorm:"type:char(64);default:''"`
	ResponseBodyEncoding string `json:"response_body_encoding" gorm:"type:varchar(16);default:''"`
	ResponseBody         string `json:"response_body" gorm:"type:text"`
	ResponseBodyStorage  string `json:"response_body_storage,omitempty" gorm:"type:varchar(16);default:''"`
	ResponseBodyRef      string `json:"response_body_ref,omitempty" gorm:"type:varchar(512);default:''"`
	ResponseBodySize     int64  `json:"response_body_size,omitempty" gorm:"default:0"`
	ResponseBodyHash     string `json:"response_body_hash,omitempty" gorm:"type:char(64);default:''"`
}

const edgeOneClientIPCountryHeader = "EO-Client-IPCountry12"
const maxLoggedUserAgentLength = 1024

const (
	logDetailRedisKeyPrefix       = "log_detail:"
	logDetailSyncQueueKey         = "log_detail:pending_sync"
	defaultLogDetailCacheTTL      = 7 * 24 * time.Hour
	defaultLogDetailSyncInterval  = 5 * time.Second
	defaultLogDetailSyncBatchSize = 50
)

var (
	logDetailSyncOnce    sync.Once
	logDetailSyncRunning atomic.Bool
)

// don't use iota, avoid change log type value
const (
	LogTypeUnknown = 0
	LogTypeTopup   = 1
	LogTypeConsume = 2
	LogTypeManage  = 3
	LogTypeSystem  = 4
	LogTypeError   = 5
	LogTypeRefund  = 6
)

func normalizeClientIPCountryCode(raw string) string {
	code := strings.ToUpper(strings.TrimSpace(raw))
	if len(code) != 2 {
		return ""
	}
	for _, ch := range code {
		if ch < 'A' || ch > 'Z' {
			return ""
		}
	}
	return code
}

func getRequestClientIPCountry(c *gin.Context) string {
	if c == nil {
		return ""
	}
	return normalizeClientIPCountryCode(c.GetHeader(edgeOneClientIPCountryHeader))
}

func getRequestUserAgent(c *gin.Context) string {
	if c == nil || c.Request == nil {
		return ""
	}
	userAgent := strings.TrimSpace(c.Request.UserAgent())
	if userAgent == "" {
		return ""
	}
	if len(userAgent) > maxLoggedUserAgentLength {
		userAgent = userAgent[:maxLoggedUserAgentLength]
	}
	return userAgent
}

func cloneLogOther(other map[string]interface{}) map[string]interface{} {
	if len(other) == 0 {
		return map[string]interface{}{}
	}
	cloned := make(map[string]interface{}, len(other))
	for k, v := range other {
		cloned[k] = v
	}
	return cloned
}

func ensureAdminInfo(logOther map[string]interface{}) map[string]interface{} {
	if logOther == nil {
		return nil
	}
	if value, ok := logOther["admin_info"]; ok && value != nil {
		if adminInfo, ok := value.(map[string]interface{}); ok {
			return adminInfo
		}
	}
	adminInfo := make(map[string]interface{})
	logOther["admin_info"] = adminInfo
	return adminInfo
}

func buildLogDetailFromContext(c *gin.Context) *LogDetail {
	if c == nil {
		return nil
	}
	detail := &LogDetail{}
	if getLogDetailStoreRequestBody() {
		if storage, err := common.GetBodyStorage(c); err == nil && storage != nil {
			if body, err := storage.Bytes(); err == nil {
				if captured := buildCapturedLogDetailBody(body); captured != nil {
					detail.RequestBodyEncoding = captured.Encoding
					detail.RequestBody = captured.Body
				}
			}
		}
	}
	if getLogDetailStoreResponseBody() {
		if captured := buildCapturedLogDetailBody(common.GetCapturedResponseBody(c)); captured != nil {
			detail.ResponseBodyEncoding = captured.Encoding
			detail.ResponseBody = captured.Body
		}
	}
	if detail.RequestBody == "" && detail.ResponseBody == "" {
		return nil
	}
	return detail
}

func buildCapturedLogDetailBody(body []byte) *common.CapturedLogBody {
	if len(body) == 0 {
		return nil
	}
	if maxBytes := getLogDetailMaxBodyBytes(); maxBytes > 0 && len(body) > maxBytes {
		body = body[:maxBytes]
	}
	return common.BuildCapturedLogBody(body)
}

func logDetailRedisEnabled() bool {
	return common.RedisEnabled && common.RDB != nil
}

func getLogDetailCacheKey(logId int) string {
	return fmt.Sprintf("%s%d", logDetailRedisKeyPrefix, logId)
}

func getLogDetailCacheTTL() time.Duration {
	ttlSeconds := common.GetEnvOrDefault("LOG_DETAIL_CACHE_TTL_SECONDS", int(defaultLogDetailCacheTTL/time.Second))
	if ttlSeconds <= 0 {
		return 0
	}
	return time.Duration(ttlSeconds) * time.Second
}

func getLogDetailSyncInterval() time.Duration {
	intervalSeconds := common.GetEnvOrDefault("LOG_DETAIL_SYNC_INTERVAL_SECONDS", int(defaultLogDetailSyncInterval/time.Second))
	if intervalSeconds <= 0 {
		intervalSeconds = 1
	}
	return time.Duration(intervalSeconds) * time.Second
}

func getLogDetailSyncBatchSize() int {
	batchSize := common.GetEnvOrDefault("LOG_DETAIL_SYNC_BATCH_SIZE", defaultLogDetailSyncBatchSize)
	if batchSize <= 0 {
		return defaultLogDetailSyncBatchSize
	}
	return batchSize
}

func cacheLogDetail(detail *LogDetail, enqueueSync bool) error {
	if detail == nil {
		return nil
	}
	payload, err := common.Marshal(detail)
	if err != nil {
		return err
	}
	if err = common.RedisSet(getLogDetailCacheKey(detail.LogId), string(payload), getLogDetailCacheTTL()); err != nil {
		return err
	}
	if enqueueSync {
		return enqueueLogDetailSyncIDs([]int{detail.LogId})
	}
	return nil
}

func enqueueLogDetailSyncIDs(logIds []int) error {
	if !logDetailRedisEnabled() || len(logIds) == 0 {
		return nil
	}
	nowScore := float64(time.Now().UnixMilli())
	members := make([]*redis.Z, 0, len(logIds))
	for _, logId := range logIds {
		if logId <= 0 {
			continue
		}
		members = append(members, &redis.Z{
			Score:  nowScore,
			Member: strconv.Itoa(logId),
		})
	}
	if len(members) == 0 {
		return nil
	}
	return common.RDB.ZAdd(context.Background(), logDetailSyncQueueKey, members...).Err()
}

func getLogDetailFromRedis(logId int) (*LogDetail, error) {
	payload, err := common.RedisGet(getLogDetailCacheKey(logId))
	if err != nil {
		return nil, err
	}
	var detail LogDetail
	if err = common.UnmarshalJsonStr(payload, &detail); err != nil {
		return nil, err
	}
	if detail.LogId == 0 {
		detail.LogId = logId
	}
	return &detail, nil
}

func loadLogDetailsBatchFromRedis(logIds []int) ([]*LogDetail, error) {
	if !logDetailRedisEnabled() || len(logIds) == 0 {
		return nil, nil
	}
	keys := make([]string, 0, len(logIds))
	for _, logId := range logIds {
		if logId > 0 {
			keys = append(keys, getLogDetailCacheKey(logId))
		}
	}
	if len(keys) == 0 {
		return nil, nil
	}
	values, err := common.RDB.MGet(context.Background(), keys...).Result()
	if err != nil {
		return nil, err
	}
	details := make([]*LogDetail, 0, len(values))
	for idx, value := range values {
		if value == nil {
			continue
		}
		var payload string
		switch typed := value.(type) {
		case string:
			payload = typed
		case []byte:
			payload = string(typed)
		default:
			payload = fmt.Sprint(typed)
		}
		var detail LogDetail
		if err = common.UnmarshalJsonStr(payload, &detail); err != nil {
			common.SysLog(fmt.Sprintf("failed to unmarshal cached log detail: log_id=%d err=%v", logIds[idx], err))
			continue
		}
		if detail.LogId == 0 {
			detail.LogId = logIds[idx]
		}
		details = append(details, &detail)
	}
	return details, nil
}

func parseLogDetailSyncMember(member interface{}) (int, error) {
	switch typed := member.(type) {
	case string:
		return strconv.Atoi(typed)
	case []byte:
		return strconv.Atoi(string(typed))
	case int:
		return typed, nil
	case int64:
		return int(typed), nil
	default:
		return 0, fmt.Errorf("unsupported log detail sync member type %T", member)
	}
}

func upsertLogDetails(details []*LogDetail) error {
	if len(details) == 0 {
		return nil
	}
	return LOG_DB.Transaction(func(tx *gorm.DB) error {
		for _, detail := range details {
			if detail == nil || detail.LogId <= 0 {
				continue
			}
			if err := prepareLogDetailForStorage(detail); err != nil {
				return err
			}
			updates := map[string]interface{}{
				"request_body_encoding":  detail.RequestBodyEncoding,
				"request_body":           detail.RequestBody,
				"request_body_storage":   detail.RequestBodyStorage,
				"request_body_ref":       detail.RequestBodyRef,
				"request_body_size":      detail.RequestBodySize,
				"request_body_hash":      detail.RequestBodyHash,
				"response_body_encoding": detail.ResponseBodyEncoding,
				"response_body":          detail.ResponseBody,
				"response_body_storage":  detail.ResponseBodyStorage,
				"response_body_ref":      detail.ResponseBodyRef,
				"response_body_size":     detail.ResponseBodySize,
				"response_body_hash":     detail.ResponseBodyHash,
			}

			var existing LogDetail
			err := tx.Where("log_id = ?", detail.LogId).Take(&existing).Error
			if err == nil {
				if updateErr := tx.Model(&existing).Updates(updates).Error; updateErr != nil {
					return updateErr
				}
				continue
			}
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
			if createErr := tx.Create(detail).Error; createErr != nil {
				return createErr
			}
		}
		return nil
	})
}

func createLogDetail(logId int, detail *LogDetail) {
	if logId == 0 || detail == nil {
		return
	}
	detail.LogId = logId
	if err := prepareLogDetailForStorage(detail); err != nil {
		common.SysLog("failed to prepare log detail storage: " + err.Error())
		return
	}
	if logDetailRedisEnabled() {
		if err := cacheLogDetail(detail, true); err != nil {
			common.SysLog("failed to cache log detail: " + err.Error())
		}
		return
	}
	if err := upsertLogDetails([]*LogDetail{detail}); err != nil {
		common.SysLog("failed to record log detail: " + err.Error())
	}
}

func markLogsHasDetail(logs []*Log) {
	if len(logs) == 0 {
		return
	}
	logIds := make([]int, 0, len(logs))
	logById := make(map[int]*Log, len(logs))
	for _, item := range logs {
		if item != nil && item.Id > 0 {
			logIds = append(logIds, item.Id)
			logById[item.Id] = item
		}
	}
	if len(logIds) == 0 {
		return
	}
	remainingLogIds := logIds
	if logDetailRedisEnabled() {
		cachedDetails, err := loadLogDetailsBatchFromRedis(logIds)
		cachedLogIds := make(map[int]struct{}, len(cachedDetails))
		if err != nil {
			common.SysLog("failed to query redis log detail flags: " + err.Error())
		} else {
			for _, detail := range cachedDetails {
				if detail == nil || detail.LogId <= 0 {
					continue
				}
				cachedLogIds[detail.LogId] = struct{}{}
				if item := logById[detail.LogId]; item != nil {
					item.HasDetail = true
					item.LogDetailStorage = getLogDetailStorageSummary(detail)
				}
			}
		}
		remainingLogIds = make([]int, 0, len(logIds))
		for _, logId := range logIds {
			if _, ok := cachedLogIds[logId]; !ok {
				remainingLogIds = append(remainingLogIds, logId)
			}
		}
	}
	if len(remainingLogIds) == 0 {
		return
	}
	var details []LogDetail
	if err := LOG_DB.Model(&LogDetail{}).
		Select("log_id", "request_body_storage", "response_body_storage").
		Where("log_id IN ?", remainingLogIds).
		Find(&details).Error; err != nil {
		common.SysLog("failed to query log detail flags: " + err.Error())
		return
	}
	for i := range details {
		if item := logById[details[i].LogId]; item != nil {
			item.HasDetail = true
			item.LogDetailStorage = getLogDetailStorageSummary(&details[i])
		}
	}
}

func GetLogDetail(logId int) (*LogDetail, error) {
	if logDetailRedisEnabled() {
		detail, err := getLogDetailFromRedis(logId)
		if err == nil {
			if err = hydrateLogDetailBodies(detail); err != nil {
				return nil, err
			}
			return detail, nil
		}
		if !errors.Is(err, redis.Nil) {
			common.SysLog("failed to get log detail from redis: " + err.Error())
		}
	}
	var detail LogDetail
	result := LOG_DB.Where("log_id = ?", logId).Limit(1).Find(&detail)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	if logDetailRedisEnabled() {
		if err := cacheLogDetail(&detail, false); err != nil {
			common.SysLog("failed to backfill log detail cache: " + err.Error())
		}
	}
	if err := hydrateLogDetailBodies(&detail); err != nil {
		return nil, err
	}
	return &detail, nil
}

func deleteLogDetailsCache(logIds []int) {
	if !logDetailRedisEnabled() || len(logIds) == 0 {
		return
	}
	keys := make([]string, 0, len(logIds))
	members := make([]interface{}, 0, len(logIds))
	for _, logId := range logIds {
		if logId <= 0 {
			continue
		}
		keys = append(keys, getLogDetailCacheKey(logId))
		members = append(members, strconv.Itoa(logId))
	}
	if len(keys) == 0 && len(members) == 0 {
		return
	}
	ctx := context.Background()
	pipeline := common.RDB.Pipeline()
	if len(keys) > 0 {
		pipeline.Del(ctx, keys...)
	}
	if len(members) > 0 {
		pipeline.ZRem(ctx, logDetailSyncQueueKey, members...)
	}
	if _, err := pipeline.Exec(ctx); err != nil && !errors.Is(err, redis.Nil) {
		common.SysLog("failed to delete log detail cache: " + err.Error())
	}
}

func StartLogDetailSyncTask() {
	logDetailSyncOnce.Do(func() {
		if !common.IsMasterNode || !logDetailRedisEnabled() {
			return
		}
		gopool.Go(func() {
			interval := getLogDetailSyncInterval()
			logger.LogInfo(context.Background(), fmt.Sprintf("log detail sync task started: tick=%s batch=%d", interval, getLogDetailSyncBatchSize()))
			ticker := time.NewTicker(interval)
			defer ticker.Stop()

			runLogDetailSyncOnce()
			for range ticker.C {
				runLogDetailSyncOnce()
			}
		})
	})
}

func runLogDetailSyncOnce() {
	if !logDetailRedisEnabled() {
		return
	}
	if !logDetailSyncRunning.CompareAndSwap(false, true) {
		return
	}
	defer logDetailSyncRunning.Store(false)

	batchSize := getLogDetailSyncBatchSize()
	if batchSize <= 0 {
		return
	}
	ctx := context.Background()
	entries, err := common.RDB.ZPopMin(ctx, logDetailSyncQueueKey, int64(batchSize)).Result()
	if err != nil {
		if !errors.Is(err, redis.Nil) {
			logger.LogWarn(ctx, fmt.Sprintf("log detail sync: pop queue failed: %v", err))
		}
		return
	}
	if len(entries) == 0 {
		return
	}

	logIds := make([]int, 0, len(entries))
	for _, entry := range entries {
		logId, parseErr := parseLogDetailSyncMember(entry.Member)
		if parseErr != nil {
			logger.LogWarn(ctx, fmt.Sprintf("log detail sync: invalid queue member=%v err=%v", entry.Member, parseErr))
			continue
		}
		if logId > 0 {
			logIds = append(logIds, logId)
		}
	}
	if len(logIds) == 0 {
		return
	}

	details, err := loadLogDetailsBatchFromRedis(logIds)
	if err != nil {
		logger.LogWarn(ctx, fmt.Sprintf("log detail sync: load cache failed: %v", err))
		_ = enqueueLogDetailSyncIDs(logIds)
		return
	}
	if len(details) == 0 {
		return
	}
	if err = upsertLogDetails(details); err != nil {
		logger.LogWarn(ctx, fmt.Sprintf("log detail sync: flush db failed: %v", err))
		_ = enqueueLogDetailSyncIDs(logIds)
		return
	}
	if common.DebugEnabled {
		logger.LogDebug(ctx, "log detail sync: flushed=%d", len(details))
	}
}

func formatUserLogs(logs []*Log, startIdx int) {
	for i := range logs {
		logs[i].ChannelName = ""
		var otherMap map[string]interface{}
		otherMap, _ = common.StrToMap(logs[i].Other)
		if otherMap == nil {
			otherMap = map[string]interface{}{}
		}
		// Remove admin-only debug fields and internal upstream model details.
		delete(otherMap, "admin_info")
		delete(otherMap, "stream_status")
		delete(otherMap, "upstream_model_name")
		delete(otherMap, "is_model_mapped")
		logs[i].Other = common.MapToJsonStr(otherMap)
		logs[i].Id = startIdx + i + 1
	}
}

func GetLogByTokenId(tokenId int) (logs []*Log, err error) {
	err = LOG_DB.Model(&Log{}).Where("token_id = ?", tokenId).Order("id desc").Limit(common.MaxRecentItems).Find(&logs).Error
	enrichLogsClientIPCountry(logs)
	markLogsHasDetail(logs)
	formatUserLogs(logs, 0)
	return logs, err
}

func RecordLog(userId int, logType int, content string) {
	if logType == LogTypeConsume && !common.LogConsumeEnabled {
		return
	}
	username, _ := GetUsernameById(userId, false)
	log := &Log{
		UserId:    userId,
		Username:  username,
		CreatedAt: common.GetTimestamp(),
		Type:      logType,
		Content:   content,
	}
	err := LOG_DB.Create(log).Error
	if err != nil {
		common.SysLog("failed to record log: " + err.Error())
	}
}

// RecordLogWithAdminInfo 记录操作日志，并将管理员相关信息存入 Other.admin_info，
func RecordLogWithAdminInfo(userId int, logType int, content string, adminInfo map[string]interface{}) {
	if logType == LogTypeConsume && !common.LogConsumeEnabled {
		return
	}
	username, _ := GetUsernameById(userId, false)
	log := &Log{
		UserId:    userId,
		Username:  username,
		CreatedAt: common.GetTimestamp(),
		Type:      logType,
		Content:   content,
	}
	if len(adminInfo) > 0 {
		other := map[string]interface{}{
			"admin_info": adminInfo,
		}
		log.Other = common.MapToJsonStr(other)
	}
	if err := LOG_DB.Create(log).Error; err != nil {
		common.SysLog("failed to record log: " + err.Error())
	}
}

func RecordTopupLog(userId int, content string, callerIp string, paymentMethod string, callbackPaymentMethod string) {
	username, _ := GetUsernameById(userId, false)
	adminInfo := map[string]interface{}{
		"server_ip":               common.GetIp(),
		"node_name":               common.NodeName,
		"caller_ip":               callerIp,
		"payment_method":          paymentMethod,
		"callback_payment_method": callbackPaymentMethod,
		"version":                 common.Version,
	}
	other := map[string]interface{}{
		"admin_info": adminInfo,
	}
	log := &Log{
		UserId:    userId,
		Username:  username,
		CreatedAt: common.GetTimestamp(),
		Type:      LogTypeTopup,
		Content:   content,
		Ip:        callerIp,
		Other:     common.MapToJsonStr(other),
	}
	err := LOG_DB.Create(log).Error
	if err != nil {
		common.SysLog("failed to record topup log: " + err.Error())
	}
}

func RecordRefundLog(userId int, content string, callerIp string, paymentMethod string, operatorId int) {
	username, _ := GetUsernameById(userId, false)
	adminInfo := map[string]interface{}{
		"server_ip":      common.GetIp(),
		"node_name":      common.NodeName,
		"caller_ip":      callerIp,
		"payment_method": paymentMethod,
		"operator_id":    operatorId,
		"version":        common.Version,
	}
	other := map[string]interface{}{
		"admin_info": adminInfo,
	}
	log := &Log{
		UserId:    userId,
		Username:  username,
		CreatedAt: common.GetTimestamp(),
		Type:      LogTypeRefund,
		Content:   content,
		Ip:        callerIp,
		Other:     common.MapToJsonStr(other),
	}
	if err := LOG_DB.Create(log).Error; err != nil {
		common.SysLog("failed to record refund log: " + err.Error())
	}
}

func RecordErrorLog(c *gin.Context, userId int, channelId int, modelName string, tokenName string, content string, tokenId int, useTimeSeconds int,
	isStream bool, group string, other map[string]interface{}) {
	logger.LogInfo(c, fmt.Sprintf("record error log: userId=%d, channelId=%d, modelName=%s, tokenName=%s, content=%s", userId, channelId, modelName, tokenName, content))
	username := c.GetString("username")
	requestId := c.GetString(common.RequestIdKey)
	logOther := cloneLogOther(other)
	if countryCode := getRequestClientIPCountry(c); countryCode != "" {
		logOther["client_ip_country"] = countryCode
	}
	if userAgent := getRequestUserAgent(c); userAgent != "" {
		logOther["user_agent"] = userAgent
	}
	detail := buildLogDetailFromContext(c)
	otherStr := common.MapToJsonStr(logOther)
	log := &Log{
		UserId:           userId,
		Username:         username,
		CreatedAt:        common.GetTimestamp(),
		Type:             LogTypeError,
		Content:          content,
		PromptTokens:     0,
		CompletionTokens: 0,
		TokenName:        tokenName,
		ModelName:        modelName,
		Quota:            0,
		ChannelId:        channelId,
		TokenId:          tokenId,
		UseTime:          useTimeSeconds,
		IsStream:         isStream,
		Group:            group,
		Ip:               c.ClientIP(),
		RequestId:        requestId,
		Other:            otherStr,
	}
	err := LOG_DB.Create(log).Error
	if err != nil {
		logger.LogError(c, "failed to record log: "+err.Error())
	}
	createLogDetail(log.Id, detail)
}

type RecordConsumeLogParams struct {
	ChannelId        int                    `json:"channel_id"`
	PromptTokens     int                    `json:"prompt_tokens"`
	CompletionTokens int                    `json:"completion_tokens"`
	ModelName        string                 `json:"model_name"`
	TokenName        string                 `json:"token_name"`
	Quota            int                    `json:"quota"`
	Content          string                 `json:"content"`
	TokenId          int                    `json:"token_id"`
	UseTimeSeconds   int                    `json:"use_time_seconds"`
	IsStream         bool                   `json:"is_stream"`
	Group            string                 `json:"group"`
	Other            map[string]interface{} `json:"other"`
}

func RecordConsumeLog(c *gin.Context, userId int, params RecordConsumeLogParams) {
	if !common.LogConsumeEnabled {
		return
	}
	logger.LogInfo(c, fmt.Sprintf("record consume log: userId=%d, params=%s", userId, common.GetJsonString(params)))
	username := c.GetString("username")
	requestId := c.GetString(common.RequestIdKey)
	logOther := cloneLogOther(params.Other)
	if countryCode := getRequestClientIPCountry(c); countryCode != "" {
		logOther["client_ip_country"] = countryCode
	}
	if userAgent := getRequestUserAgent(c); userAgent != "" {
		logOther["user_agent"] = userAgent
	}
	detail := buildLogDetailFromContext(c)
	otherStr := common.MapToJsonStr(logOther)
	log := &Log{
		UserId:           userId,
		Username:         username,
		CreatedAt:        common.GetTimestamp(),
		Type:             LogTypeConsume,
		Content:          params.Content,
		PromptTokens:     params.PromptTokens,
		CompletionTokens: params.CompletionTokens,
		TokenName:        params.TokenName,
		ModelName:        params.ModelName,
		Quota:            params.Quota,
		ChannelId:        params.ChannelId,
		TokenId:          params.TokenId,
		UseTime:          params.UseTimeSeconds,
		IsStream:         params.IsStream,
		Group:            params.Group,
		Ip:               c.ClientIP(),
		RequestId:        requestId,
		Other:            otherStr,
	}
	err := LOG_DB.Create(log).Error
	if err != nil {
		logger.LogError(c, "failed to record log: "+err.Error())
	}
	createLogDetail(log.Id, detail)
	if common.DataExportEnabled {
		gopool.Go(func() {
			LogQuotaData(userId, username, params.ModelName, params.Quota, common.GetTimestamp(), params.PromptTokens+params.CompletionTokens)
		})
	}
}

type RecordTaskBillingLogParams struct {
	UserId    int
	LogType   int
	Content   string
	ChannelId int
	ModelName string
	Quota     int
	TokenId   int
	Group     string
	Other     map[string]interface{}
}

func RecordTaskBillingLog(params RecordTaskBillingLogParams) {
	if params.LogType == LogTypeConsume && !common.LogConsumeEnabled {
		return
	}
	username, _ := GetUsernameById(params.UserId, false)
	tokenName := ""
	if params.TokenId > 0 {
		if token, err := GetTokenById(params.TokenId); err == nil {
			tokenName = token.Name
		}
	}
	log := &Log{
		UserId:    params.UserId,
		Username:  username,
		CreatedAt: common.GetTimestamp(),
		Type:      params.LogType,
		Content:   params.Content,
		TokenName: tokenName,
		ModelName: params.ModelName,
		Quota:     params.Quota,
		ChannelId: params.ChannelId,
		TokenId:   params.TokenId,
		Group:     params.Group,
		Other:     common.MapToJsonStr(params.Other),
	}
	err := LOG_DB.Create(log).Error
	if err != nil {
		common.SysLog("failed to record task billing log: " + err.Error())
	}
}

func GetAllLogs(logType int, startTimestamp int64, endTimestamp int64, modelName string, username string, tokenName string, startIdx int, num int, channel int, group string, requestId string) (logs []*Log, total int64, err error) {
	var tx *gorm.DB
	if logType == LogTypeUnknown {
		tx = LOG_DB
	} else {
		tx = LOG_DB.Where("logs.type = ?", logType)
	}

	if modelName != "" {
		tx = tx.Where("logs.model_name like ?", modelName)
	}
	if username != "" {
		tx = tx.Where("logs.username = ?", username)
	}
	if tokenName != "" {
		tx = tx.Where("logs.token_name = ?", tokenName)
	}
	if requestId != "" {
		tx = tx.Where("logs.request_id = ?", requestId)
	}
	if startTimestamp != 0 {
		tx = tx.Where("logs.created_at >= ?", startTimestamp)
	}
	if endTimestamp != 0 {
		tx = tx.Where("logs.created_at <= ?", endTimestamp)
	}
	if channel != 0 {
		tx = tx.Where("logs.channel_id = ?", channel)
	}
	if group != "" {
		tx = tx.Where("logs."+logGroupCol+" = ?", group)
	}
	err = tx.Model(&Log{}).Count(&total).Error
	if err != nil {
		return nil, 0, err
	}
	err = tx.Order("logs.id desc").Limit(num).Offset(startIdx).Find(&logs).Error
	if err != nil {
		return nil, 0, err
	}

	channelIds := types.NewSet[int]()
	for _, log := range logs {
		if log.ChannelId != 0 {
			channelIds.Add(log.ChannelId)
		}
	}

	if channelIds.Len() > 0 {
		var channels []struct {
			Id   int    `gorm:"column:id"`
			Name string `gorm:"column:name"`
		}
		if common.MemoryCacheEnabled {
			// Cache get channel
			for _, channelId := range channelIds.Items() {
				if cacheChannel, err := CacheGetChannel(channelId); err == nil {
					channels = append(channels, struct {
						Id   int    `gorm:"column:id"`
						Name string `gorm:"column:name"`
					}{
						Id:   channelId,
						Name: cacheChannel.Name,
					})
				}
			}
		} else {
			// Bulk query channels from DB
			if err = DB.Table("channels").Select("id, name").Where("id IN ?", channelIds.Items()).Find(&channels).Error; err != nil {
				return logs, total, err
			}
		}
		channelMap := make(map[int]string, len(channels))
		for _, channel := range channels {
			channelMap[channel.Id] = channel.Name
		}
		for i := range logs {
			logs[i].ChannelName = channelMap[logs[i].ChannelId]
		}
	}

	enrichLogsClientIPCountry(logs)
	markLogsHasDetail(logs)

	return logs, total, err
}

const logSearchCountLimit = 10000

func GetUserLogs(userId int, logType int, startTimestamp int64, endTimestamp int64, modelName string, tokenName string, startIdx int, num int, group string, requestId string) (logs []*Log, total int64, err error) {
	var tx *gorm.DB
	if logType == LogTypeUnknown {
		tx = LOG_DB.Where("logs.user_id = ?", userId)
	} else {
		tx = LOG_DB.Where("logs.user_id = ? and logs.type = ?", userId, logType)
	}

	if modelName != "" {
		modelNamePattern, err := sanitizeLikePattern(modelName)
		if err != nil {
			return nil, 0, err
		}
		tx = tx.Where("logs.model_name LIKE ? ESCAPE '!'", modelNamePattern)
	}
	if tokenName != "" {
		tx = tx.Where("logs.token_name = ?", tokenName)
	}
	if requestId != "" {
		tx = tx.Where("logs.request_id = ?", requestId)
	}
	if startTimestamp != 0 {
		tx = tx.Where("logs.created_at >= ?", startTimestamp)
	}
	if endTimestamp != 0 {
		tx = tx.Where("logs.created_at <= ?", endTimestamp)
	}
	if group != "" {
		tx = tx.Where("logs."+logGroupCol+" = ?", group)
	}
	err = tx.Model(&Log{}).Limit(logSearchCountLimit).Count(&total).Error
	if err != nil {
		common.SysError("failed to count user logs: " + err.Error())
		return nil, 0, errors.New("查询日志失败")
	}
	err = tx.Order("logs.id desc").Limit(num).Offset(startIdx).Find(&logs).Error
	if err != nil {
		common.SysError("failed to search user logs: " + err.Error())
		return nil, 0, errors.New("查询日志失败")
	}

	enrichLogsClientIPCountry(logs)
	markLogsHasDetail(logs)
	formatUserLogs(logs, startIdx)
	return logs, total, err
}

type Stat struct {
	Quota int `json:"quota"`
	Rpm   int `json:"rpm"`
	Tpm   int `json:"tpm"`
}

func SumUsedQuota(logType int, startTimestamp int64, endTimestamp int64, modelName string, username string, tokenName string, channel int, group string) (stat Stat, err error) {
	tx := LOG_DB.Table("logs").Select("sum(quota) quota")

	// 为rpm和tpm创建单独的查询
	rpmTpmQuery := LOG_DB.Table("logs").Select("count(*) rpm, sum(prompt_tokens) + sum(completion_tokens) tpm")

	if username != "" {
		tx = tx.Where("username = ?", username)
		rpmTpmQuery = rpmTpmQuery.Where("username = ?", username)
	}
	if tokenName != "" {
		tx = tx.Where("token_name = ?", tokenName)
		rpmTpmQuery = rpmTpmQuery.Where("token_name = ?", tokenName)
	}
	if startTimestamp != 0 {
		tx = tx.Where("created_at >= ?", startTimestamp)
	}
	if endTimestamp != 0 {
		tx = tx.Where("created_at <= ?", endTimestamp)
	}
	if modelName != "" {
		modelNamePattern, err := sanitizeLikePattern(modelName)
		if err != nil {
			return stat, err
		}
		tx = tx.Where("model_name LIKE ? ESCAPE '!'", modelNamePattern)
		rpmTpmQuery = rpmTpmQuery.Where("model_name LIKE ? ESCAPE '!'", modelNamePattern)
	}
	if channel != 0 {
		tx = tx.Where("channel_id = ?", channel)
		rpmTpmQuery = rpmTpmQuery.Where("channel_id = ?", channel)
	}
	if group != "" {
		tx = tx.Where(logGroupCol+" = ?", group)
		rpmTpmQuery = rpmTpmQuery.Where(logGroupCol+" = ?", group)
	}

	tx = tx.Where("type = ?", LogTypeConsume)
	rpmTpmQuery = rpmTpmQuery.Where("type = ?", LogTypeConsume)

	// 只统计最近60秒的rpm和tpm
	rpmTpmQuery = rpmTpmQuery.Where("created_at >= ?", time.Now().Add(-60*time.Second).Unix())

	// 执行查询
	if err := tx.Scan(&stat).Error; err != nil {
		common.SysError("failed to query log stat: " + err.Error())
		return stat, errors.New("查询统计数据失败")
	}
	if err := rpmTpmQuery.Scan(&stat).Error; err != nil {
		common.SysError("failed to query rpm/tpm stat: " + err.Error())
		return stat, errors.New("查询统计数据失败")
	}

	return stat, nil
}

func SumUsedToken(logType int, startTimestamp int64, endTimestamp int64, modelName string, username string, tokenName string) (token int) {
	tx := LOG_DB.Table("logs").Select("ifnull(sum(prompt_tokens),0) + ifnull(sum(completion_tokens),0)")
	if username != "" {
		tx = tx.Where("username = ?", username)
	}
	if tokenName != "" {
		tx = tx.Where("token_name = ?", tokenName)
	}
	if startTimestamp != 0 {
		tx = tx.Where("created_at >= ?", startTimestamp)
	}
	if endTimestamp != 0 {
		tx = tx.Where("created_at <= ?", endTimestamp)
	}
	if modelName != "" {
		tx = tx.Where("model_name = ?", modelName)
	}
	tx.Where("type = ?", LogTypeConsume).Scan(&token)
	return token
}

func DeleteOldLog(ctx context.Context, targetTimestamp int64, limit int) (int64, error) {
	var total int64 = 0

	for {
		if nil != ctx.Err() {
			return total, ctx.Err()
		}

		var logIds []int
		if err := LOG_DB.Model(&Log{}).
			Where("created_at < ?", targetTimestamp).
			Order("id asc").
			Limit(limit).
			Pluck("id", &logIds).Error; err != nil {
			return total, err
		}
		if len(logIds) == 0 {
			break
		}
		deleteLogDetailsCache(logIds)
		if err := LOG_DB.Where("log_id IN ?", logIds).Delete(&LogDetail{}).Error; err != nil {
			return total, err
		}
		result := LOG_DB.Where("id IN ?", logIds).Delete(&Log{})
		if nil != result.Error {
			return total, result.Error
		}

		total += result.RowsAffected

		if result.RowsAffected < int64(limit) {
			break
		}
	}

	return total, nil
}
