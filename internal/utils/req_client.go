package utils

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"monica-proxy/internal/config"
	"net"
	"net/http"
	"net/url"
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
		Proxy: http.ProxyFromEnvironment, // 使用环境变量中的代理设置
	}
	
	// 如果配置中有代理设置，则使用配置的代理
	if cfg.Proxy.HTTPProxy != "" || cfg.Proxy.HTTPSProxy != "" {
		proxyURL := cfg.Proxy.HTTPProxy
		if proxyURL == "" {
			proxyURL = cfg.Proxy.HTTPSProxy
		}
		if parsedProxyURL, err := url.Parse(proxyURL); err == nil {
			transport.Proxy = http.ProxyURL(parsedProxyURL)
		}
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
		Proxy: http.ProxyFromEnvironment, // 使用环境变量中的代理设置
	}
	
	// 如果配置中有代理设置，则使用配置的代理
	if cfg.Proxy.HTTPProxy != "" || cfg.Proxy.HTTPSProxy != "" {
		proxyURL := cfg.Proxy.HTTPProxy
		if proxyURL == "" {
			proxyURL = cfg.Proxy.HTTPSProxy
		}
		if parsedProxyURL, err := url.Parse(proxyURL); err == nil {
			transport.Proxy = http.ProxyURL(parsedProxyURL)
		}
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

// MonicaQuotaResponse Monica额度查询响应结构
type MonicaQuotaResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		ModuleQuotas []struct {
			Module  string `json:"module"`
			Quotas  []struct {
				Scene          string `json:"scene"`
				ResetFrequency string `json:"reset_frequency"`
				DefaultQuota   int    `json:"default_quota"`
				CurrentQuota   int    `json:"current_quota"`
				LastResetTime  string `json:"last_reset_time"`
			} `json:"quotas"`
		} `json:"module_quotas"`
	} `json:"data"`
}

// GetMonicaQuota 获取Monica额度信息
func GetMonicaQuota(cfg *config.Config) (*MonicaQuotaResponse, error) {
	// 创建专用的HTTP客户端
	client := resty.New().
		SetTimeout(30 * time.Second).
		SetHeaders(map[string]string{
			"accept":           "*/*",
			"accept-language":  "zh-CN,zh;q=0.9,en;q=0.8",
			"cache-control":    "no-cache",
			"content-type":     "application/json",
			"origin":           "https://monica.im",
			"pragma":           "no-cache",
			"priority":         "u=1, i",
			"referer":          "https://monica.im/",
			"sec-fetch-dest":   "empty",
			"sec-fetch-mode":   "cors",
			"sec-fetch-site":   "same-site",
			"user-agent":       "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/139.0.0.0 Safari/537.36",
			"x-client-id":      "3ff5e948-10e9-4a32-8626-f8560b238a42",
			"x-client-locale":  "zh_CN",
			"x-client-type":    "web",
			"x-client-version": "5.4.3",
			"x-from-channel":   "NA",
			"x-product-name":   "Monica",
			"x-time-zone":      "Asia/Shanghai;-480",
			"dnt":              "1",
		})

	// 设置代理
	if cfg.Proxy.HTTPProxy != "" || cfg.Proxy.HTTPSProxy != "" {
		proxyURL := cfg.Proxy.HTTPProxy
		if proxyURL == "" {
			proxyURL = cfg.Proxy.HTTPSProxy
		}
		client.SetProxy(proxyURL)
	}

	// 设置Cookie
	if cfg.Monica.Cookie != "" {
		client.SetHeader("Cookie", cfg.Monica.Cookie)
	}

	// 准备请求数据
	requestData := map[string]interface{}{
		"modules": []string{"genius_bot", "credits"},
	}

	// 发送请求
	resp, err := client.R().SetBody(requestData).Post("https://api.monica.im/api/usagev2/get_quotas")
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}

	if resp.StatusCode() != 200 {
		return nil, fmt.Errorf("HTTP错误: %d", resp.StatusCode())
	}

	// 解析响应
	var quotaResp MonicaQuotaResponse
	if err := json.Unmarshal(resp.Body(), &quotaResp); err != nil {
		return nil, fmt.Errorf("响应解析失败: %w", err)
	}

	if quotaResp.Code != 0 {
		return nil, fmt.Errorf("API错误: %s", quotaResp.Msg)
	}

	return &quotaResp, nil
}
