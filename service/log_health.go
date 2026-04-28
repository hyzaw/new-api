package service

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/pkg/cachex"
	"github.com/samber/hot"
)

const (
	requestStatusMonitorCacheNamespace = "new-api:request_status_monitor:v1"
	requestStatusMonitorCacheTTL       = 30 * time.Minute
	requestStatusMonitorCacheRetention = 2 * time.Hour
	requestStatusMonitorPruneInterval  = 1 * time.Hour
)

var (
	requestStatusMonitorCacheOnce sync.Once
	requestStatusMonitorCache     *cachex.HybridCache[model.RequestStatusMonitor]
	requestStatusMonitorPruneAt   atomic.Int64
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

func maybePruneRequestStatusMonitorCache(cache *cachex.HybridCache[model.RequestStatusMonitor], now int64) {
	if cache == nil || now <= 0 {
		return
	}
	last := requestStatusMonitorPruneAt.Load()
	if now-last < int64(requestStatusMonitorPruneInterval/time.Second) {
		return
	}
	if !requestStatusMonitorPruneAt.CompareAndSwap(last, now) {
		return
	}
	go pruneRequestStatusMonitorCache(cache, now)
}

func pruneRequestStatusMonitorCache(cache *cachex.HybridCache[model.RequestStatusMonitor], now int64) {
	keys, err := cache.Keys()
	if err != nil {
		common.SysError(fmt.Sprintf("request status monitor cache prune scan failed: %v", err))
		return
	}
	if len(keys) == 0 {
		return
	}

	prefix := cache.FullKey("snapshot:")
	cutoff := now - int64(requestStatusMonitorCacheRetention/time.Second)
	staleKeys := make([]string, 0)
	for _, key := range keys {
		if !strings.HasPrefix(key, prefix) {
			continue
		}
		windowEnd, err := strconv.ParseInt(strings.TrimPrefix(key, prefix), 10, 64)
		if err != nil {
			continue
		}
		if windowEnd < cutoff {
			staleKeys = append(staleKeys, key)
		}
	}
	if len(staleKeys) == 0 {
		return
	}
	if _, err := cache.DeleteMany(staleKeys); err != nil {
		common.SysError(fmt.Sprintf("request status monitor cache prune delete failed: %v", err))
	}
}

func GetRequestStatusMonitor() (*model.RequestStatusMonitor, error) {
	now := time.Now().Unix()
	windowEnd := alignRequestStatusWindowEnd(now)
	cache := getRequestStatusMonitorCache()
	maybePruneRequestStatusMonitorCache(cache, now)

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
