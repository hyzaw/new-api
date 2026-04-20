package service

import (
	"fmt"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/pkg/cachex"
	"github.com/samber/hot"
)

const (
	requestStatusMonitorCacheNamespace = "new-api:request_status_monitor:v1"
	requestStatusMonitorCacheTTL       = 365 * 24 * time.Hour
)

var (
	requestStatusMonitorCacheOnce sync.Once
	requestStatusMonitorCache     *cachex.HybridCache[model.RequestStatusMonitor]
)

func getRequestStatusMonitorCache() *cachex.HybridCache[model.RequestStatusMonitor] {
	requestStatusMonitorCacheOnce.Do(func() {
		requestStatusMonitorCache = cachex.NewHybridCache[model.RequestStatusMonitor](cachex.HybridCacheConfig[model.RequestStatusMonitor]{
			Namespace: cachex.Namespace(requestStatusMonitorCacheNamespace),
			Redis:     common.RDB,
			RedisEnabled: func() bool {
				return common.RedisEnabled && common.RDB != nil
			},
			RedisCodec: cachex.JSONCodec[model.RequestStatusMonitor]{},
			Memory: func() *hot.HotCache[string, model.RequestStatusMonitor] {
				return hot.NewHotCache[string, model.RequestStatusMonitor](hot.LRU, 32).
					WithTTL(requestStatusMonitorCacheTTL).
					WithJanitor().
					Build()
			},
		})
	})
	return requestStatusMonitorCache
}

func alignRequestStatusWindowEnd(now int64) int64 {
	return now - now%model.RequestStatusIntervalSeconds
}

func requestStatusMonitorCacheKey(windowEnd int64) string {
	return fmt.Sprintf("snapshot:%d", windowEnd)
}

func GetRequestStatusMonitor() (*model.RequestStatusMonitor, error) {
	windowEnd := alignRequestStatusWindowEnd(time.Now().Unix())
	cache := getRequestStatusMonitorCache()

	cacheKey := requestStatusMonitorCacheKey(windowEnd)
	if cached, found, err := cache.Get(cacheKey); err == nil && found {
		return &cached, nil
	} else if err != nil {
		common.SysError(fmt.Sprintf("request status monitor cache get failed: %v", err))
	}

	monitor, err := model.GetRequestStatusMonitorSnapshot(windowEnd, model.RequestStatusPointCount, model.RequestStatusIntervalSeconds)
	if err != nil {
		return nil, err
	}

	if err := cache.SetWithTTL(cacheKey, *monitor, requestStatusMonitorCacheTTL); err != nil {
		common.SysError(fmt.Sprintf("request status monitor cache set failed: %v", err))
	}

	return monitor, nil
}
