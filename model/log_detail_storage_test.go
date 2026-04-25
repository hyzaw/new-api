package model

import "testing"

func TestPrepareLogDetailForStorageKeepsInlineWhenCOSDisabled(t *testing.T) {
	t.Setenv("LOG_DETAIL_STORAGE_TYPE", "")

	detail := &LogDetail{
		LogId:        123,
		RequestBody:  "request body",
		ResponseBody: "response body",
	}

	if err := prepareLogDetailForStorage(detail); err != nil {
		t.Fatalf("prepareLogDetailForStorage() error = %v", err)
	}
	if detail.RequestBody != "request body" || detail.ResponseBody != "response body" {
		t.Fatalf("body should stay inline, got request=%q response=%q", detail.RequestBody, detail.ResponseBody)
	}
	if detail.RequestBodyStorage != logDetailStorageInline || detail.ResponseBodyStorage != logDetailStorageInline {
		t.Fatalf("unexpected storage flags: request=%q response=%q", detail.RequestBodyStorage, detail.ResponseBodyStorage)
	}
	if detail.RequestBodyRef != "" || detail.ResponseBodyRef != "" {
		t.Fatalf("COS refs should be empty, got request=%q response=%q", detail.RequestBodyRef, detail.ResponseBodyRef)
	}
	if detail.RequestBodySize != int64(len("request body")) || detail.ResponseBodySize != int64(len("response body")) {
		t.Fatalf("unexpected body sizes: request=%d response=%d", detail.RequestBodySize, detail.ResponseBodySize)
	}
	if detail.RequestBodyHash == "" || detail.ResponseBodyHash == "" {
		t.Fatal("body hashes should be populated")
	}
}

func TestLogDetailGzipRoundTrip(t *testing.T) {
	original := []byte("large captured log body\nline 2")

	compressed, err := gzipLogDetailBody(original)
	if err != nil {
		t.Fatalf("gzipLogDetailBody() error = %v", err)
	}
	if string(compressed) == string(original) {
		t.Fatal("compressed body should differ from original")
	}

	got, err := gunzipLogDetailBody(compressed)
	if err != nil {
		t.Fatalf("gunzipLogDetailBody() error = %v", err)
	}
	if string(got) != string(original) {
		t.Fatalf("round trip mismatch: got %q want %q", got, original)
	}
}
