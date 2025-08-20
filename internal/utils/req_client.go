package utils

import (
	"crypto/tls"
	"fmt"
	"monica-proxy/internal/config"
	"net"
	"net/http"
	"time"

	"github.com/go-resty/resty/v2"
)

var (
	// 全局客户端实例，将在初始化时设置
	RestySSEClient     *resty.Client
	RestyDefaultClient *resty.Client
)

// InitHTTPClients 初始化HTTP客户端
func InitHTTPClients(cfg *config.Config) {
	RestySSEClient = createSSEClient(cfg)
	RestyDefaultClient = createDefaultClient(cfg)
}

// createSSEClient 创建SSE专用客户端
func createSSEClient(cfg *config.Config) *resty.Client {
	// 创建自定义的Transport
	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:        cfg.HTTPClient.MaxIdleConns,
		MaxIdleConnsPerHost: cfg.HTTPClient.MaxIdleConnsPerHost,
		MaxConnsPerHost:     cfg.HTTPClient.MaxConnsPerHost,
		IdleConnTimeout:     90 * time.Second,
		TLSHandshakeTimeout: 10 * time.Second,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: cfg.Security.TLSSkipVerify,
			MinVersion:         tls.VersionTLS12, // 强制使用TLS 1.2+
		},
	}

	client := resty.NewWithClient(&http.Client{
		Transport: transport,
		Timeout:   cfg.HTTPClient.Timeout,
	}).
		SetRetryCount(cfg.HTTPClient.RetryCount).
		SetRetryWaitTime(cfg.HTTPClient.RetryWaitTime).
		SetRetryMaxWaitTime(cfg.HTTPClient.RetryMaxWaitTime).
		SetDoNotParseResponse(true). // SSE需要流式处理
		SetHeaders(map[string]string{
			"Content-Type":    "application/json",
			"User-Agent":      "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
			"x-client-locale": "zh_CN",
			"Accept":          "text/event-stream,application/json",
		}).
		OnAfterResponse(func(c *resty.Client, resp *resty.Response) error {
			if resp.StatusCode() >= 400 {
				return fmt.Errorf("monica API error: status %d, body: %s",
					resp.StatusCode(), resp.String())
			}
			return nil
		})

	// 添加重试条件
	client.AddRetryCondition(func(r *resty.Response, err error) bool {
		// 网络错误或5xx错误时重试
		return err != nil || r.StatusCode() >= 500
	})

	return client
}

// createDefaultClient 创建默认客户端
func createDefaultClient(cfg *config.Config) *resty.Client {
	// 创建自定义的Transport
	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:        cfg.HTTPClient.MaxIdleConns,
		MaxIdleConnsPerHost: cfg.HTTPClient.MaxIdleConnsPerHost,
		MaxConnsPerHost:     cfg.HTTPClient.MaxConnsPerHost,
		IdleConnTimeout:     90 * time.Second,
		TLSHandshakeTimeout: 10 * time.Second,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: cfg.Security.TLSSkipVerify,
			MinVersion:         tls.VersionTLS12, // 强制使用TLS 1.2+
		},
	}

	client := resty.NewWithClient(&http.Client{
		Transport: transport,
		Timeout:   cfg.Security.RequestTimeout,
	}).
		SetRetryCount(cfg.HTTPClient.RetryCount).
		SetRetryWaitTime(cfg.HTTPClient.RetryWaitTime).
		SetRetryMaxWaitTime(cfg.HTTPClient.RetryMaxWaitTime).
		SetHeaders(map[string]string{
			"Content-Type": "application/json",
			"User-Agent":   "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
		}).
		OnAfterResponse(func(c *resty.Client, resp *resty.Response) error {
			if resp.StatusCode() >= 400 {
				return fmt.Errorf("monica API error: status %d, body: %s",
					resp.StatusCode(), resp.String())
			}
			return nil
		})

	// 添加重试条件
	client.AddRetryCondition(func(r *resty.Response, err error) bool {
		// 网络错误或5xx错误时重试
		return err != nil || r.StatusCode() >= 500
	})

	return client
}
