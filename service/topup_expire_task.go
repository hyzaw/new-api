package service

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"

	"github.com/bytedance/gopkg/util/gopool"
)

const (
	topupExpireTickInterval   = 5 * time.Minute
	topupExpireBatchSize      = 300
	topupExpirePendingSeconds = 2 * 60 * 60
)

var (
	topupExpireOnce    sync.Once
	topupExpireRunning atomic.Bool
)

func StartTopUpExpireTask() {
	topupExpireOnce.Do(func() {
		if !common.IsMasterNode {
			return
		}
		gopool.Go(func() {
			logger.LogInfo(context.Background(), fmt.Sprintf(
				"topup expire task started: tick=%s pending_timeout=%s",
				topupExpireTickInterval,
				time.Duration(topupExpirePendingSeconds)*time.Second,
			))
			ticker := time.NewTicker(topupExpireTickInterval)
			defer ticker.Stop()

			runTopUpExpireOnce()
			for range ticker.C {
				runTopUpExpireOnce()
			}
		})
	})
}

func runTopUpExpireOnce() {
	if !topupExpireRunning.CompareAndSwap(false, true) {
		return
	}
	defer topupExpireRunning.Store(false)

	ctx := context.Background()
	totalExpired := 0
	for {
		n, err := model.ExpirePendingTopUps(topupExpireBatchSize, topupExpirePendingSeconds)
		if err != nil {
			logger.LogWarn(ctx, fmt.Sprintf("topup expire task failed: %v", err))
			return
		}
		if n == 0 {
			break
		}
		totalExpired += n
		if n < topupExpireBatchSize {
			break
		}
	}
	if common.DebugEnabled && totalExpired > 0 {
		logger.LogDebug(ctx, "topup expire maintenance: expired_count=%d", totalExpired)
	}
}
