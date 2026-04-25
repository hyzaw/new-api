package model

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"
	_ "time/tzdata"

	"github.com/QuantumNous/new-api/common"
	"github.com/tencentyun/cos-go-sdk-v5"
)

const (
	logDetailStorageInline  = "inline"
	logDetailStorageCOS     = "cos"
	logDetailStorageCOSGzip = "cos-gzip"

	defaultLogDetailInlineMaxBytes = 64 * 1024
	defaultLogDetailCOSTimeout     = 10 * time.Second
	defaultLogDetailCOSBasePath    = "log-details"
)

type logDetailCOSConfig struct {
	SecretID  string
	SecretKey string
	Bucket    string
	Region    string
	Scheme    string
	BasePath  string
	Timeout   time.Duration
	Compress  bool
}

var (
	logDetailCOSClientOnce sync.Once
	logDetailCOSClient     *cos.Client
	logDetailCOSClientErr  error
	logDetailCOSLocation   = loadLogDetailCOSLocation()
)

func loadLogDetailCOSLocation() *time.Location {
	location, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		return time.FixedZone("Asia/Shanghai", 8*60*60)
	}
	return location
}

func logDetailExternalStorageEnabled() bool {
	return strings.EqualFold(strings.TrimSpace(os.Getenv("LOG_DETAIL_STORAGE_TYPE")), logDetailStorageCOS)
}

func getLogDetailInlineMaxBytes() int {
	inlineMax := common.GetEnvOrDefault("LOG_DETAIL_BODY_INLINE_MAX_BYTES", defaultLogDetailInlineMaxBytes)
	if inlineMax < 0 {
		return defaultLogDetailInlineMaxBytes
	}
	return inlineMax
}

func getLogDetailCOSConfig() (logDetailCOSConfig, error) {
	cfg := logDetailCOSConfig{
		SecretID:  strings.TrimSpace(os.Getenv("COS_SECRET_ID")),
		SecretKey: strings.TrimSpace(os.Getenv("COS_SECRET_KEY")),
		Bucket:    strings.TrimSpace(os.Getenv("COS_BUCKET")),
		Region:    strings.TrimSpace(os.Getenv("COS_REGION")),
		Scheme:    strings.TrimSpace(common.GetEnvOrDefaultString("COS_SCHEME", "https")),
		BasePath:  strings.Trim(strings.TrimSpace(common.GetEnvOrDefaultString("COS_BASE_PATH", defaultLogDetailCOSBasePath)), "/"),
		Timeout:   time.Duration(common.GetEnvOrDefault("LOG_DETAIL_COS_TIMEOUT_SECONDS", int(defaultLogDetailCOSTimeout/time.Second))) * time.Second,
		Compress:  common.GetEnvOrDefaultBool("LOG_DETAIL_COMPRESS", true),
	}
	if cfg.BasePath == "" {
		cfg.BasePath = defaultLogDetailCOSBasePath
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = defaultLogDetailCOSTimeout
	}
	if cfg.Scheme == "" {
		cfg.Scheme = "https"
	}
	if cfg.SecretID == "" || cfg.SecretKey == "" || cfg.Bucket == "" || cfg.Region == "" {
		return cfg, fmt.Errorf("COS_SECRET_ID, COS_SECRET_KEY, COS_BUCKET and COS_REGION are required when LOG_DETAIL_STORAGE_TYPE=cos")
	}
	return cfg, nil
}

func getLogDetailCOSClient() (*cos.Client, logDetailCOSConfig, error) {
	cfg, cfgErr := getLogDetailCOSConfig()
	if cfgErr != nil {
		return nil, cfg, cfgErr
	}
	logDetailCOSClientOnce.Do(func() {
		endpoint := fmt.Sprintf("%s://%s.cos.%s.myqcloud.com", cfg.Scheme, cfg.Bucket, cfg.Region)
		bucketURL, err := url.Parse(endpoint)
		if err != nil {
			logDetailCOSClientErr = err
			return
		}
		baseURL := &cos.BaseURL{BucketURL: bucketURL}
		logDetailCOSClient = cos.NewClient(baseURL, &http.Client{
			Timeout: cfg.Timeout,
			Transport: &cos.AuthorizationTransport{
				SecretID:  cfg.SecretID,
				SecretKey: cfg.SecretKey,
			},
		})
	})
	return logDetailCOSClient, cfg, logDetailCOSClientErr
}

func prepareLogDetailForStorage(detail *LogDetail) error {
	if detail == nil {
		return nil
	}
	if err := prepareLogDetailBodyForStorage(
		detail,
		"request",
		&detail.RequestBody,
		&detail.RequestBodyStorage,
		&detail.RequestBodyRef,
		&detail.RequestBodySize,
		&detail.RequestBodyHash,
	); err != nil {
		return err
	}
	return prepareLogDetailBodyForStorage(
		detail,
		"response",
		&detail.ResponseBody,
		&detail.ResponseBodyStorage,
		&detail.ResponseBodyRef,
		&detail.ResponseBodySize,
		&detail.ResponseBodyHash,
	)
}

func prepareLogDetailBodyForStorage(detail *LogDetail, part string, body *string, storage *string, ref *string, size *int64, hash *string) error {
	if body == nil || storage == nil || ref == nil || size == nil || hash == nil {
		return nil
	}
	if *body == "" {
		return nil
	}
	if isLogDetailCOSStorage(*storage) && *ref != "" {
		return nil
	}
	bodyBytes := []byte(*body)
	*size = int64(len(bodyBytes))
	*hash = logDetailBodyHash(bodyBytes)

	if !logDetailExternalStorageEnabled() || len(bodyBytes) <= getLogDetailInlineMaxBytes() {
		*storage = logDetailStorageInline
		*ref = ""
		return nil
	}

	objectKey, objectStorage, err := uploadLogDetailBodyToCOS(detail.LogId, part, bodyBytes, *hash)
	if err != nil {
		return err
	}
	*storage = objectStorage
	*ref = objectKey
	*body = ""
	return nil
}

func hydrateLogDetailBodies(detail *LogDetail) error {
	if detail == nil {
		return nil
	}
	if err := hydrateLogDetailBody(&detail.RequestBody, detail.RequestBodyStorage, detail.RequestBodyRef); err != nil {
		return err
	}
	return hydrateLogDetailBody(&detail.ResponseBody, detail.ResponseBodyStorage, detail.ResponseBodyRef)
}

func hydrateLogDetailBody(body *string, storage string, ref string) error {
	if body == nil || *body != "" || !isLogDetailCOSStorage(storage) || ref == "" {
		return nil
	}
	bodyBytes, err := downloadLogDetailBodyFromCOS(ref, storage)
	if err != nil {
		return err
	}
	*body = string(bodyBytes)
	return nil
}

func uploadLogDetailBodyToCOS(logId int, part string, body []byte, hash string) (string, string, error) {
	client, cfg, err := getLogDetailCOSClient()
	if err != nil {
		return "", "", err
	}

	payload := body
	storage := logDetailStorageCOS
	extension := ".txt"
	if cfg.Compress {
		payload, err = gzipLogDetailBody(body)
		if err != nil {
			return "", "", err
		}
		storage = logDetailStorageCOSGzip
		extension = ".gz"
	}

	objectKey := buildLogDetailCOSObjectKey(cfg.BasePath, logId, part, hash, extension)
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()

	_, err = client.Object.Put(ctx, objectKey, bytes.NewReader(payload), &cos.ObjectPutOptions{
		ObjectPutHeaderOptions: &cos.ObjectPutHeaderOptions{
			ContentType:   "application/octet-stream",
			ContentLength: int64(len(payload)),
		},
	})
	if err != nil {
		return "", "", err
	}
	return objectKey, storage, nil
}

func downloadLogDetailBodyFromCOS(objectKey string, storage string) ([]byte, error) {
	client, cfg, err := getLogDetailCOSClient()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()

	resp, err := client.Object.Get(ctx, objectKey, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if storage == logDetailStorageCOSGzip {
		return gunzipLogDetailBody(body)
	}
	return body, nil
}

func isLogDetailCOSStorage(storage string) bool {
	return storage == logDetailStorageCOS || storage == logDetailStorageCOSGzip
}

func getLogDetailStorageSummary(detail *LogDetail) string {
	if detail == nil {
		return ""
	}
	if isLogDetailCOSStorage(detail.RequestBodyStorage) || isLogDetailCOSStorage(detail.ResponseBodyStorage) {
		return logDetailStorageCOS
	}
	return logDetailStorageInline
}

func buildLogDetailCOSObjectKey(basePath string, logId int, part string, hash string, extension string) string {
	if part != "request" && part != "response" {
		part = "body"
	}
	if len(hash) > 16 {
		hash = hash[:16]
	}
	now := time.Now().In(logDetailCOSLocation)
	fileName := fmt.Sprintf("%s-%s-%d%s", part, hash, now.UnixNano(), extension)
	return path.Join(basePath, now.Format("2006/01/02"), strconv.Itoa(logId), fileName)
}

func logDetailBodyHash(body []byte) string {
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}

func gzipLogDetailBody(body []byte) ([]byte, error) {
	var buf bytes.Buffer
	writer := gzip.NewWriter(&buf)
	if _, err := writer.Write(body); err != nil {
		_ = writer.Close()
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func gunzipLogDetailBody(body []byte) ([]byte, error) {
	reader, err := gzip.NewReader(bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	return io.ReadAll(reader)
}
