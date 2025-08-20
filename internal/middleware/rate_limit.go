package middleware

import (
	"context"
	"monica-proxy/internal/config"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
	"golang.org/x/time/rate"
)

// clientEntry 客户端限流器条目
type clientEntry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// RateLimiter 限流器结构
type RateLimiter struct {
	mu      sync.RWMutex
	clients map[string]*clientEntry
	rate    rate.Limit
	burst   int
	ctx     context.Context
	cancel  context.CancelFunc
}

// NewRateLimiter 创建新的限流器
func NewRateLimiter(rps int) *RateLimiter {
	ctx, cancel := context.WithCancel(context.Background())
	rl := &RateLimiter{
		clients: make(map[string]*clientEntry),
		rate:    rate.Limit(rps),
		burst:   rps, // 突发请求等于RPS，更保守的策略
		ctx:     ctx,
		cancel:  cancel,
	}

	// 启动清理协程
	go rl.cleanupClients()

	return rl
}

// GetLimiter 获取特定客户端的限流器
func (rl *RateLimiter) GetLimiter(clientIP string) *rate.Limiter {
	// 先尝试读锁
	rl.mu.RLock()
	entry, exists := rl.clients[clientIP]
	if exists {
		entry.lastSeen = time.Now() // 更新最后访问时间
		rl.mu.RUnlock()
		return entry.limiter
	}
	rl.mu.RUnlock()

	// 需要创建新的限流器，使用写锁
	rl.mu.Lock()
	defer rl.mu.Unlock()

	// 双重检查，避免并发创建
	if entry, exists := rl.clients[clientIP]; exists {
		entry.lastSeen = time.Now()
		return entry.limiter
	}

	// 创建新的限流器
	limiter := rate.NewLimiter(rl.rate, rl.burst)
	rl.clients[clientIP] = &clientEntry{
		limiter:  limiter,
		lastSeen: time.Now(),
	}

	return limiter
}

// cleanupClients 定期清理不活跃的客户端限流器
func (rl *RateLimiter) cleanupClients() {
	ticker := time.NewTicker(2 * time.Minute) // 更频繁的清理
	defer ticker.Stop()

	for {
		select {
		case <-rl.ctx.Done():
			return // 优雅停止
		case <-ticker.C:
			rl.cleanupInactiveClients()
		}
	}
}

// cleanupInactiveClients 清理不活跃的客户端
func (rl *RateLimiter) cleanupInactiveClients() {
	now := time.Now()
	threshold := 10 * time.Minute // 10分钟不活跃就清理

	rl.mu.Lock()
	defer rl.mu.Unlock()

	for ip, entry := range rl.clients {
		if now.Sub(entry.lastSeen) > threshold {
			delete(rl.clients, ip)
		}
	}
}

// Close 关闭限流器，停止清理协程
func (rl *RateLimiter) Close() {
	rl.cancel()
}

// getClientIP 安全地获取客户端IP
func getClientIP(c echo.Context) string {
	// 优先级：X-Real-IP > X-Forwarded-For > RemoteAddr
	if ip := c.Request().Header.Get("X-Real-IP"); ip != "" {
		if parsedIP := net.ParseIP(ip); parsedIP != nil {
			return ip
		}
	}

	if ip := c.Request().Header.Get("X-Forwarded-For"); ip != "" {
		// X-Forwarded-For 可能包含多个IP，取第一个
		if firstIP := getFirstIP(ip); firstIP != "" {
			if parsedIP := net.ParseIP(firstIP); parsedIP != nil {
				return firstIP
			}
		}
	}

	// 回退到 RemoteAddr
	if ip, _, err := net.SplitHostPort(c.Request().RemoteAddr); err == nil {
		return ip
	}

	return c.Request().RemoteAddr
}

// getFirstIP 从逗号分隔的IP列表中获取第一个IP
func getFirstIP(ips string) string {
	for i, char := range ips {
		if char == ',' {
			return ips[:i]
		}
	}
	return ips
}

// 全局限流器实例，避免重复创建
var globalRateLimiter *RateLimiter
var rateLimiterOnce sync.Once

// RateLimit 创建限流中间件
func RateLimit(cfg *config.Config) echo.MiddlewareFunc {
	// 如果禁用限流，返回空中间件
	if !cfg.Security.RateLimitEnabled {
		return func(next echo.HandlerFunc) echo.HandlerFunc {
			return next
		}
	}

	// 确保只创建一次限流器
	rateLimiterOnce.Do(func() {
		globalRateLimiter = NewRateLimiter(cfg.Security.RateLimitRPS)
	})

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// 安全地获取客户端IP
			clientIP := getClientIP(c)

			// 获取该客户端的限流器
			limiter := globalRateLimiter.GetLimiter(clientIP)

			// 检查是否允许请求
			if !limiter.Allow() {
				return echo.NewHTTPError(http.StatusTooManyRequests, map[string]any{
					"error": map[string]any{
						"code":        "rate_limit_exceeded",
						"message":     "请求过于频繁，请稍后再试",
						"limit":       cfg.Security.RateLimitRPS,
						"retry_after": "1s",
					},
				})
			}

			return next(c)
		}
	}
}
