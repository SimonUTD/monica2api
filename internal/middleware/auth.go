package middleware

import (
	"monica-proxy/internal/config"
	"monica-proxy/internal/logger"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

// BearerAuth 创建一个Bearer Token认证中间件
func BearerAuth(cfg *config.Config) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// 获取Authorization header
			auth := c.Request().Header.Get("Authorization")

			// 检查header格式
			if auth == "" || !strings.HasPrefix(auth, "Bearer ") {
				if cfg.Logging.MaskSensitive {
					logger.Warn("无效的授权头",
						zap.String("method", c.Request().Method),
						zap.String("uri", c.Request().RequestURI),
						zap.String("remote_addr", c.RealIP()),
					)
				} else {
					logger.Warn("无效的授权头",
						zap.String("method", c.Request().Method),
						zap.String("uri", c.Request().RequestURI),
						zap.String("remote_addr", c.RealIP()),
						zap.String("auth_header", auth),
					)
				}
				return echo.NewHTTPError(http.StatusUnauthorized, "invalid authorization header")
			}

			// 提取token
			token := strings.TrimPrefix(auth, "Bearer ")

			// 验证token
			if token != cfg.Security.BearerToken || token == "" {
				if cfg.Logging.MaskSensitive {
					logger.Warn("无效的Token",
						zap.String("method", c.Request().Method),
						zap.String("uri", c.Request().RequestURI),
						zap.String("remote_addr", c.RealIP()),
					)
				} else {
					logger.Warn("无效的Token",
						zap.String("method", c.Request().Method),
						zap.String("uri", c.Request().RequestURI),
						zap.String("remote_addr", c.RealIP()),
						zap.String("token", token),
					)
				}
				return echo.NewHTTPError(http.StatusUnauthorized, "invalid token")
			}

			return next(c)
		}
	}
}
