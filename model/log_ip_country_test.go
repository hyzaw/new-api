package model

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/QuantumNous/new-api/common"
)

func TestLookupLogIPCountry(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"code": 200,
			"msg": "查询成功",
			"data": {
				"状态": "成功",
				"data": {
					"data": {
						"country_code": "SG",
						"country_name": "新加坡"
					},
					"query_time_ms": 4.48
				}
			}
		}`))
	}))
	defer server.Close()

	prevURL := logIPCountryLookupURL
	prevClient := logIPCountryHTTPClient
	logIPCountryLookupURL = server.URL
	logIPCountryHTTPClient = server.Client()
	defer func() {
		logIPCountryLookupURL = prevURL
		logIPCountryHTTPClient = prevClient
	}()

	value, err := lookupLogIPCountry(context.Background(), "45.129.228.101")
	if err != nil {
		t.Fatalf("lookupLogIPCountry returned error: %v", err)
	}
	if value.CountryCode != "SG" {
		t.Fatalf("unexpected country code: got %q", value.CountryCode)
	}
	if value.CountryName != "新加坡" {
		t.Fatalf("unexpected country name: got %q", value.CountryName)
	}
}

func TestEnrichLogsClientIPCountryBackfillsOther(t *testing.T) {
	var requestCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"code": 200,
			"msg": "查询成功",
			"data": {
				"状态": "成功",
				"data": {
					"data": {
						"country_code": "SG",
						"country_name": "新加坡"
					}
				}
			}
		}`))
	}))
	defer server.Close()

	prevURL := logIPCountryLookupURL
	prevClient := logIPCountryHTTPClient
	prevRedisEnabled := common.RedisEnabled
	prevRDB := common.RDB
	logIPCountryLookupURL = server.URL
	logIPCountryHTTPClient = server.Client()
	common.RedisEnabled = false
	common.RDB = nil
	defer func() {
		logIPCountryLookupURL = prevURL
		logIPCountryHTTPClient = prevClient
		common.RedisEnabled = prevRedisEnabled
		common.RDB = prevRDB
	}()

	logs := []*Log{
		{
			Ip:    "45.129.228.101",
			Other: "",
		},
		{
			Ip:    "45.129.228.101",
			Other: "",
		},
		{
			Ip:    "8.8.8.8",
			Other: `{"client_ip_country":"US","client_ip_country_name":"美国"}`,
		},
	}

	enrichLogsClientIPCountry(logs)

	if atomic.LoadInt32(&requestCount) != 1 {
		t.Fatalf("expected 1 upstream lookup, got %d", requestCount)
	}

	otherMap, err := common.StrToMap(logs[0].Other)
	if err != nil {
		t.Fatalf("failed to parse enriched other: %v", err)
	}
	if got := otherMap["client_ip_country"]; got != "SG" {
		t.Fatalf("unexpected enriched country code: got %v", got)
	}
	if got := otherMap["client_ip_country_name"]; got != "新加坡" {
		t.Fatalf("unexpected enriched country name: got %v", got)
	}

	existingMap, err := common.StrToMap(logs[2].Other)
	if err != nil {
		t.Fatalf("failed to parse existing other: %v", err)
	}
	if got := existingMap["client_ip_country"]; got != "US" {
		t.Fatalf("unexpected existing country code: got %v", got)
	}
}
