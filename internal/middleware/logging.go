package middleware

import (
	"monica-proxy/internal/config"
	"monica-proxy/internal/logger"
	"time"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

// RequestLogger 创建一个请求日志记录中间件
func RequestLogger(cfg *config.Config) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// 如果禁用了请求日志，直接处理请求
			if !cfg.Logging.EnableRequestLog {
				return next(c)
			}

			start := time.Now()
			req := c.Request()
			res := c.Response()

			// 从 Echo 的 RequestID 中间件读取请求ID（不再自行生成）
			requestID := req.Header.Get(echo.HeaderXRequestID)
			if requestID == "" {
				requestID = res.Header().Get(echo.HeaderXRequestID)
			}

			// 处理请求
			err := next(c)

			// 计算耗时
			duration := time.Since(start)

			// 构建日志字段
			fields := []zap.Field{
				zap.String("method", req.Method),
				zap.String("uri", req.RequestURI),
				zap.Int("status", res.Status),
				zap.Duration("latency", duration),
				zap.String("remote_addr", c.RealIP()),
				zap.String("request_id", requestID),
				zap.String("user_agent", req.UserAgent()),
			}

			// 添加响应大小信息
			if res.Size > 0 {
				fields = append(fields, zap.Int64("response_size", res.Size))
			}

			// 根据错误情况记录不同级别的日志
			if err != nil {
				fields = append(fields, zap.Error(err))
				logger.Error("请求失败", fields...)
			} else {
				// 根据状态码决定日志级别
				switch {
				case res.Status >= 500:
					logger.Error("请求完成但服务器错误", fields...)
				case res.Status >= 400:
					logger.Warn("请求完成但客户端错误", fields...)
				default:
					logger.Info("请求完成", fields...)
				}
			}

			return err
		}
	}
}
