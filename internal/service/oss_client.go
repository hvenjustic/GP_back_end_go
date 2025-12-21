package service

import (
	"context"
	"fmt"
	"net/url"
	"path"
	"strings"

	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss"
	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss/credentials"

	"GP_back_end_go/pkg/config"
	"GP_back_end_go/pkg/log"
)

// OSSUploader 负责将字符串内容上传到阿里云 OSS。
type OSSUploader struct {
	client  *oss.Client
	bucket  string
	baseURL string
	prefix  string
}

// NewOSSUploaderFromConfig 基于全局配置初始化上传器。
func NewOSSUploaderFromConfig() (*OSSUploader, error) {
	ossCfg := config.Config.OSS
	if strings.TrimSpace(ossCfg.Endpoint) == "" || strings.TrimSpace(ossCfg.Bucket) == "" {
		return nil, fmt.Errorf("oss config missing endpoint/bucket")
	}
	region := strings.TrimSpace(ossCfg.Region)
	if region == "" {
		return nil, fmt.Errorf("oss config missing region")
	}
	cfg := &oss.Config{
		Endpoint:            oss.Ptr(ossCfg.Endpoint),
		CredentialsProvider: credentials.NewStaticCredentialsProvider(ossCfg.AccessKeyID, ossCfg.AccessKeySecret),
		Region:              oss.Ptr(region),
	}
	client := oss.NewClient(cfg)

	baseURL := buildBaseURL(ossCfg.Endpoint, ossCfg.Bucket, ossCfg.UseHTTPS)
	prefix := strings.Trim(ossCfg.Prefix, "/")
	return &OSSUploader{
		client:  client,
		bucket:  ossCfg.Bucket,
		baseURL: baseURL,
		prefix:  prefix,
	}, nil
}

// UploadString 将字符串内容以对象形式上传到 OSS，返回可公开访问的 URL（基于配置拼接）。
func (u *OSSUploader) UploadString(ctx context.Context, objectKey string, content string) (string, error) {
	if u == nil || u.client == nil {
		return "", fmt.Errorf("oss uploader not initialized")
	}
	key := objectKey
	if u.prefix != "" {
		key = path.Join(u.prefix, objectKey)
	}
	reader := strings.NewReader(content)
	req := &oss.PutObjectRequest{
		Bucket: oss.Ptr(u.bucket),
		Key:    oss.Ptr(key),
		Body:   reader,
	}
	if _, err := u.client.PutObject(ctx, req); err != nil {
		return "", err
	}
	url := strings.TrimRight(u.baseURL, "/") + "/" + strings.TrimLeft(key, "/")
	return url, nil
}

func buildBaseURL(endpoint string, bucket string, useHTTPS bool) string {
	scheme := "http"
	if useHTTPS {
		scheme = "https"
	}
	ep := strings.TrimSpace(endpoint)
	if u, err := url.Parse(ep); err == nil && u.Host != "" {
		ep = u.Host
	}
	return fmt.Sprintf("%s://%s.%s", scheme, bucket, ep)
}

// SafeObjectKeyFromURL 生成可用作对象键的基础名：域名-{id}，无法解析则使用 domain-{id}。
func SafeObjectKeyFromURL(rawURL string, id int) string {
	host := ""
	if u, err := url.Parse(strings.TrimSpace(rawURL)); err == nil && u.Host != "" {
		host = u.Hostname()
	}
	if host == "" {
		host = "domain"
	}
	replacer := strings.NewReplacer(
		":", "-",
		"/", "-",
		"\\", "-",
		"?", "-",
		"&", "-",
		"=", "-",
		"#", "-",
		" ", "-",
	)
	host = replacer.Replace(host)
	host = strings.Trim(host, "-")
	if host == "" {
		host = "domain"
	}
	return fmt.Sprintf("%s-%d", host, id)
}

// InitOSSUploader 在启动时尝试初始化，失败会记录错误但不 panic。
func InitOSSUploader() *OSSUploader {
	uploader, err := NewOSSUploaderFromConfig()
	if err != nil {
		log.Error("OSS", "init failed", err.Error())
		return nil
	}
	log.Info("OSS", "init success", "base", uploader.baseURL, "prefix", uploader.prefix)
	return uploader
}
