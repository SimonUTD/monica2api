package middleware

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"monica-proxy/internal/config"
	"monica-proxy/internal/logger"
	"net"
	"net/http"
	"strings"
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

			// 记录请求详情
			var requestBody []byte
			if req.Body != nil {
				// 读取请求体
				requestBody, _ = io.ReadAll(req.Body)
				// 恢复请求体以便后续处理
				req.Body = io.NopCloser(bytes.NewBuffer(requestBody))
			}

			// 记录请求头（脱敏处理）
			headers := logHeaders(req.Header, cfg.Logging.MaskSensitive)

			// 创建响应体捕获器
			responseBody := &bytes.Buffer{}
			if res.Writer != nil {
				res.Writer = &responseWriter{ResponseWriter: res.Writer, body: responseBody}
			}

			// 处理请求
			err := next(c)

			// 计算耗时
			duration := time.Since(start)

			// 构建日志字段
			fields := []zap.Field{
				zap.String("method", req.Method),
				zap.String("uri", req.RequestURI),
				zap.String("protocol", req.Proto),
				zap.Int("status", res.Status),
				zap.Duration("latency", duration),
				zap.String("remote_addr", c.RealIP()),
				zap.String("request_id", requestID),
				zap.String("user_agent", req.UserAgent()),
				zap.Any("headers", headers),
			}

			// 添加请求体（如果有且不是文件上传）
			if len(requestBody) > 0 && !isFileUpload(req) {
				if cfg.Logging.MaskSensitive {
					fields = append(fields, zap.String("request_body", maskSensitiveData(string(requestBody))))
				} else {
					fields = append(fields, zap.String("request_body", string(requestBody)))
				}
			}

			// 添加响应体（如果有）
			if responseBody.Len() > 0 {
				if cfg.Logging.MaskSensitive {
					fields = append(fields, zap.String("response_body", maskSensitiveData(responseBody.String())))
				} else {
					fields = append(fields, zap.String("response_body", responseBody.String()))
				}
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

// responseWriter 包装 http.ResponseWriter 以捕获响应体
type responseWriter struct {
	http.ResponseWriter
	body *bytes.Buffer
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	rw.body.Write(b)
	return rw.ResponseWriter.Write(b)
}

func (rw *responseWriter) WriteHeader(statusCode int) {
	rw.ResponseWriter.WriteHeader(statusCode)
}

func (rw *responseWriter) Header() http.Header {
	return rw.ResponseWriter.Header()
}

func (rw *responseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hj, ok := rw.ResponseWriter.(http.Hijacker); ok {
		return hj.Hijack()
	}
	return nil, nil, fmt.Errorf("response writer cannot hijack")
}

// isFileUpload 检查是否为文件上传请求
func isFileUpload(req *http.Request) bool {
	return req.Method == "POST" && 
		req.Header.Get("Content-Type") != "" && 
		(req.Header.Get("Content-Type")[:19] == "multipart/form-data" || 
			req.Header.Get("Content-Type")[:33] == "application/x-www-form-urlencoded")
}

// logHeaders 记录请求头并脱敏敏感信息
func logHeaders(headers map[string][]string, maskSensitive bool) map[string][]string {
	result := make(map[string][]string)
	
	for key, values := range headers {
		if maskSensitive {
			switch key {
			case "Authorization", "Cookie", "Token", "apikey", "Api-Key":
				result[key] = []string{"***"}
			default:
				result[key] = values
			}
		} else {
			result[key] = values
		}
	}
	
	return result
}

// maskSensitiveData 脱敏敏感数据
func maskSensitiveData(data string) string {
	// 尝试解析为JSON
	var jsonMap map[string]interface{}
	if err := json.Unmarshal([]byte(data), &jsonMap); err == nil {
		return maskJSONSensitiveFields(jsonMap)
	}
	
	// 如果不是JSON，简单处理包含敏感信息的字符串
	lowerData := strings.ToLower(data)
	sensitivePatterns := []string{
		"authorization", "bearer", "api_key", "apikey", 
		"token", "password", "secret", "credential",
	}
	
	for _, pattern := range sensitivePatterns {
		if strings.Contains(lowerData, pattern) {
			return "*** 敏感数据已脱敏 ***"
		}
	}
	
	return data
}

// maskJSONSensitiveFields 脱敏JSON中的敏感字段
func maskJSONSensitiveFields(data map[string]interface{}) string {
	masked := make(map[string]interface{})
	
	for key, value := range data {
		lowerKey := strings.ToLower(key)
		switch {
		case strings.Contains(lowerKey, "authorization") ||
			strings.Contains(lowerKey, "bearer") ||
			strings.Contains(lowerKey, "api_key") ||
			strings.Contains(lowerKey, "apikey") ||
			strings.Contains(lowerKey, "token") ||
			strings.Contains(lowerKey, "password") ||
			strings.Contains(lowerKey, "secret") ||
			strings.Contains(lowerKey, "credential") ||
			strings.Contains(lowerKey, "cookie"):
			masked[key] = "***"
		default:
			masked[key] = value
		}
	}
	
	if result, err := json.Marshal(masked); err == nil {
		return string(result)
	}
	return "*** 敏感数据已脱敏 ***"
}
