package model

import "time"

// RefreshPricing 强制立即重新计算与定价相关的缓存。
// 该方法用于需要最新数据的内部管理 API，
// 因此会绕过默认的 1 分钟延迟刷新。
func RefreshPricing() {
	updatePricingLock.Lock()
	defer updatePricingLock.Unlock()

	modelSupportEndpointsLock.Lock()
	defer modelSupportEndpointsLock.Unlock()

	updatePricing()
}

// InvalidatePricingCache 仅使定价缓存失效，避免批量更新定价相关 option 时
// 重复触发完整重建。下一次 GetPricing 会自动按最新配置重算。
func InvalidatePricingCache() {
	updatePricingLock.Lock()
	defer updatePricingLock.Unlock()

	pricingMap = nil
	vendorsList = nil
	supportedEndpointMap = nil
	lastGetPricingTime = time.Time{}
}
