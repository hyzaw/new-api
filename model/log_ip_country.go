package model

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
)

const (
	defaultLogIPCountryLookupURL       = "https://api.dwo.cc/api/ip-lookup"
	logIPCountryCacheKeyPrefix         = "log:ip-country:v1:"
	logIPCountryCacheTTL               = 7 * 24 * time.Hour
	logIPCountryLookupTimeout          = 3 * time.Second
	maxLogIPCountryLookupsPerQuery int = 50
)

var (
	logIPCountryLookupURL  = common.GetEnvOrDefaultString("LOG_IP_COUNTRY_LOOKUP_URL", defaultLogIPCountryLookupURL)
	logIPCountryHTTPClient = &http.Client{
		Timeout: logIPCountryLookupTimeout,
	}
)

type logIPCountryLookupRequest struct {
	IP     string `json:"ip"`
	IsIPv6 bool   `json:"isIPv6"`
}

type logIPCountryLookupResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		Status string `json:"状态"`
		Data   struct {
			Data struct {
				CountryCode string `json:"country_code"`
				CountryName string `json:"country_name"`
			} `json:"data"`
		} `json:"data"`
	} `json:"data"`
}

type logIPCountryCacheValue struct {
	CountryCode string `json:"country_code"`
	CountryName string `json:"country_name"`
}

func getLogIPCountryCacheKey(ip string) string {
	return logIPCountryCacheKeyPrefix + strings.TrimSpace(ip)
}

func getCachedLogIPCountry(ip string) (logIPCountryCacheValue, bool) {
	if !common.RedisEnabled || common.RDB == nil {
		return logIPCountryCacheValue{}, false
	}
	raw, err := common.RedisGet(getLogIPCountryCacheKey(ip))
	if err != nil || strings.TrimSpace(raw) == "" {
		return logIPCountryCacheValue{}, false
	}
	var cached logIPCountryCacheValue
	if err = common.UnmarshalJsonStr(raw, &cached); err != nil {
		return logIPCountryCacheValue{}, false
	}
	cached.CountryCode = normalizeClientIPCountryCode(cached.CountryCode)
	cached.CountryName = strings.TrimSpace(cached.CountryName)
	return cached, true
}

func setCachedLogIPCountry(ip string, value logIPCountryCacheValue) {
	if !common.RedisEnabled || common.RDB == nil {
		return
	}
	value.CountryCode = normalizeClientIPCountryCode(value.CountryCode)
	value.CountryName = strings.TrimSpace(value.CountryName)
	raw, err := common.Marshal(value)
	if err != nil {
		return
	}
	if err = common.RedisSet(getLogIPCountryCacheKey(ip), string(raw), logIPCountryCacheTTL); err != nil {
		common.SysLog(fmt.Sprintf("failed to cache log ip country for %s: %s", ip, err.Error()))
	}
}

func shouldSkipLogIPCountryLookup(ip string) bool {
	parsed := net.ParseIP(strings.TrimSpace(ip))
	if parsed == nil {
		return true
	}
	return parsed.IsLoopback() ||
		parsed.IsPrivate() ||
		parsed.IsMulticast() ||
		parsed.IsLinkLocalMulticast() ||
		parsed.IsLinkLocalUnicast() ||
		parsed.IsUnspecified()
}

func lookupLogIPCountry(ctx context.Context, ip string) (logIPCountryCacheValue, error) {
	ip = strings.TrimSpace(ip)
	if ip == "" {
		return logIPCountryCacheValue{}, nil
	}
	if logIPCountryLookupURL == "" {
		return logIPCountryCacheValue{}, nil
	}
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return logIPCountryCacheValue{}, nil
	}
	payload, err := common.Marshal(logIPCountryLookupRequest{
		IP:     ip,
		IsIPv6: parsed.To4() == nil,
	})
	if err != nil {
		return logIPCountryCacheValue{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, logIPCountryLookupURL, bytes.NewReader(payload))
	if err != nil {
		return logIPCountryCacheValue{}, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "new-api/"+strings.TrimSpace(common.Version))

	resp, err := logIPCountryHTTPClient.Do(req)
	if err != nil {
		return logIPCountryCacheValue{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return logIPCountryCacheValue{}, fmt.Errorf("unexpected status code %d", resp.StatusCode)
	}

	var result logIPCountryLookupResponse
	if err = common.DecodeJson(resp.Body, &result); err != nil {
		return logIPCountryCacheValue{}, err
	}

	value := logIPCountryCacheValue{
		CountryCode: normalizeClientIPCountryCode(result.Data.Data.Data.CountryCode),
		CountryName: strings.TrimSpace(result.Data.Data.Data.CountryName),
	}
	return value, nil
}

func getLogOtherClientIPCountry(otherMap map[string]interface{}) (string, string) {
	if len(otherMap) == 0 {
		return "", ""
	}
	countryCode := normalizeClientIPCountryCode(common.Interface2String(otherMap["client_ip_country"]))
	countryName := strings.TrimSpace(common.Interface2String(otherMap["client_ip_country_name"]))
	if countryCode != "" || countryName != "" {
		return countryCode, countryName
	}
	adminInfo, ok := otherMap["admin_info"].(map[string]interface{})
	if !ok || len(adminInfo) == 0 {
		return "", ""
	}
	return normalizeClientIPCountryCode(common.Interface2String(adminInfo["client_ip_country"])),
		strings.TrimSpace(common.Interface2String(adminInfo["client_ip_country_name"]))
}

func setLogOtherClientIPCountry(otherMap map[string]interface{}, value logIPCountryCacheValue) map[string]interface{} {
	if otherMap == nil {
		otherMap = make(map[string]interface{})
	}
	if value.CountryCode != "" {
		otherMap["client_ip_country"] = value.CountryCode
	}
	if value.CountryName != "" {
		otherMap["client_ip_country_name"] = value.CountryName
	}
	return otherMap
}

func enrichLogsClientIPCountry(logs []*Log) {
	if len(logs) == 0 {
		return
	}

	lookups := make(map[string]logIPCountryCacheValue)
	pendingIPs := make([]string, 0)
	pendingSet := make(map[string]struct{})

	for _, logItem := range logs {
		if logItem == nil {
			continue
		}
		ip := strings.TrimSpace(logItem.Ip)
		if ip == "" || shouldSkipLogIPCountryLookup(ip) {
			continue
		}

		otherMap, _ := common.StrToMap(logItem.Other)
		if countryCode, countryName := getLogOtherClientIPCountry(otherMap); countryCode != "" || countryName != "" {
			lookups[ip] = logIPCountryCacheValue{
				CountryCode: countryCode,
				CountryName: countryName,
			}
			continue
		}

		if cached, found := getCachedLogIPCountry(ip); found {
			lookups[ip] = cached
			continue
		}

		if len(pendingIPs) >= maxLogIPCountryLookupsPerQuery {
			continue
		}
		if _, exists := pendingSet[ip]; exists {
			continue
		}
		pendingSet[ip] = struct{}{}
		pendingIPs = append(pendingIPs, ip)
	}

	for _, ip := range pendingIPs {
		ctx, cancel := context.WithTimeout(context.Background(), logIPCountryLookupTimeout)
		value, err := lookupLogIPCountry(ctx, ip)
		cancel()
		if err != nil {
			common.SysLog(fmt.Sprintf("failed to lookup log ip country for %s: %s", ip, err.Error()))
			continue
		}
		lookups[ip] = value
		setCachedLogIPCountry(ip, value)
	}

	for _, logItem := range logs {
		if logItem == nil {
			continue
		}
		ip := strings.TrimSpace(logItem.Ip)
		if ip == "" {
			continue
		}
		value, ok := lookups[ip]
		if !ok || (value.CountryCode == "" && value.CountryName == "") {
			continue
		}
		otherMap, _ := common.StrToMap(logItem.Other)
		logItem.Other = common.MapToJsonStr(setLogOtherClientIPCountry(otherMap, value))
	}
}
