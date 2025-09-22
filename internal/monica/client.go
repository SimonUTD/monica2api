package monica

import (
	"context"
	"encoding/json"
	"fmt"
	"monica-proxy/internal/config"
	"monica-proxy/internal/errors"
	"monica-proxy/internal/logger"
	"monica-proxy/internal/types"
	"monica-proxy/internal/utils"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
	"go.uber.org/zap"
)

// SendMonicaRequest 发起对 Monica AI 的请求(使用 resty)
func SendMonicaRequest(ctx context.Context, cfg *config.Config, mReq *types.MonicaRequest) (*resty.Response, error) {
	startTime := time.Now()
	requestID := fmt.Sprintf("monica-%d", startTime.UnixNano())

	// 记录请求详情
	if cfg.Logging.EnableRequestLog {
		requestBody, _ := json.Marshal(mReq)
		fields := []zap.Field{
			zap.String("request_id", requestID),
			zap.String("api_type", "monica_chat"),
			zap.String("url", types.BotChatURL),
			zap.String("method", "POST"),
		}

		if cfg.Logging.MaskSensitive {
			fields = append(fields, zap.String("request_body", maskMonicaRequestData(string(requestBody))))
		} else {
			fields = append(fields, zap.String("request_body", string(requestBody)))
		}

		logger.Info("[环节2] 本软件请求Monica - 发送Monica API请求", fields...)
	}

	// 发起请求
	resp, err := utils.RestySSEClient.R().
		SetContext(ctx).
		SetHeader("cookie", cfg.Monica.Cookie).
		SetBody(mReq).
		Post(types.BotChatURL)

	duration := time.Since(startTime)

	// 记录响应详情
	if cfg.Logging.EnableRequestLog {
		fields := []zap.Field{
			zap.String("request_id", requestID),
			zap.String("api_type", "monica_chat"),
			zap.Duration("duration", duration),
		}

		if err != nil {
			fields = append(fields, zap.Error(err))
			logger.Error("Monica API请求失败", fields...)
		} else {
			fields = append(fields, 
				zap.Int("status_code", resp.StatusCode()),
				zap.Int64("response_size", resp.Size()),
			)

			// 对于非流式响应，记录响应体
			if resp.StatusCode() < 400 && resp.RawResponse != nil && resp.Body() != nil {
				if cfg.Logging.MaskSensitive {
					fields = append(fields, zap.String("response_body", maskMonicaResponseData(string(resp.Body()))))
				} else {
					fields = append(fields, zap.String("response_body", string(resp.Body())))
				}
			}

			// 根据状态码决定日志级别
			switch {
			case resp.StatusCode() >= 500:
				logger.Error("Monica API响应服务器错误", fields...)
			case resp.StatusCode() >= 400:
				logger.Warn("Monica API响应客户端错误", fields...)
			default:
				logger.Info("[环节3] Monica返回本软件 - Monica API请求完成", fields...)
			}
		}
	}

	if err != nil {
		return nil, errors.NewRequestFailedError("Monica API调用失败", err)
	}

	return resp, nil
}

// SendCustomBotRequest 发送custom bot请求
func SendCustomBotRequest(ctx context.Context, cfg *config.Config, customBotReq *types.CustomBotRequest) (*resty.Response, error) {
	startTime := time.Now()
	requestID := fmt.Sprintf("custombot-%d", startTime.UnixNano())

	// 记录请求详情
	if cfg.Logging.EnableRequestLog {
		requestBody, _ := json.Marshal(customBotReq)
		fields := []zap.Field{
			zap.String("request_id", requestID),
			zap.String("api_type", "custom_bot"),
			zap.String("url", types.CustomBotChatURL),
			zap.String("method", "POST"),
			zap.String("bot_uid", customBotReq.BotUID),
		}

		if cfg.Logging.MaskSensitive {
			fields = append(fields, zap.String("request_body", maskMonicaRequestData(string(requestBody))))
		} else {
			fields = append(fields, zap.String("request_body", string(requestBody)))
		}

		logger.Info("发送Custom Bot API请求", fields...)
	}

	// 发起请求
	resp, err := utils.RestySSEClient.R().
		SetContext(ctx).
		SetHeader("cookie", cfg.Monica.Cookie).
		SetBody(customBotReq).
		Post(types.CustomBotChatURL)

	duration := time.Since(startTime)

	// 记录响应详情
	if cfg.Logging.EnableRequestLog {
		fields := []zap.Field{
			zap.String("request_id", requestID),
			zap.String("api_type", "custom_bot"),
			zap.Duration("duration", duration),
			zap.String("bot_uid", customBotReq.BotUID),
		}

		if err != nil {
			fields = append(fields, zap.Error(err))
			logger.Error("Custom Bot API请求失败", fields...)
		} else {
			fields = append(fields, 
				zap.Int("status_code", resp.StatusCode()),
				zap.Int64("response_size", resp.Size()),
			)

			// 对于非流式响应，记录响应体
			if resp.StatusCode() < 400 && resp.RawResponse != nil && resp.Body() != nil {
				if cfg.Logging.MaskSensitive {
					fields = append(fields, zap.String("response_body", maskMonicaResponseData(string(resp.Body()))))
				} else {
					fields = append(fields, zap.String("response_body", string(resp.Body())))
				}
			}

			// 根据状态码决定日志级别
			switch {
			case resp.StatusCode() >= 500:
				logger.Error("Custom Bot API响应服务器错误", fields...)
			case resp.StatusCode() >= 400:
				logger.Warn("Custom Bot API响应客户端错误", fields...)
			default:
				logger.Info("Custom Bot API请求完成", fields...)
			}
		}
	}

	if err != nil {
		return nil, errors.NewRequestFailedError("Custom Bot API调用失败", err)
	}

	return resp, nil
}

// maskMonicaRequestData 脱敏Monica请求中的敏感数据
func maskMonicaRequestData(data string) string {
	var request map[string]interface{}
	if err := json.Unmarshal([]byte(data), &request); err == nil {
		return maskMonicaJSONFields(request)
	}
	return data
}

// maskMonicaResponseData 脱敏Monica响应中的敏感数据
func maskMonicaResponseData(data string) string {
	// 对于SSE流数据，按行处理
	lines := strings.Split(data, "\n")
	for i, line := range lines {
		if strings.HasPrefix(line, "data: ") {
			jsonData := strings.TrimPrefix(line, "data: ")
			var response map[string]interface{}
			if err := json.Unmarshal([]byte(jsonData), &response); err == nil {
				masked := maskMonicaJSONFields(response)
				lines[i] = "data: " + masked
			}
		}
	}
	return strings.Join(lines, "\n")
}

// maskMonicaJSONFields 脱敏Monica JSON中的敏感字段
func maskMonicaJSONFields(data map[string]interface{}) string {
	masked := make(map[string]interface{})
	
	for key, value := range data {
		lowerKey := strings.ToLower(key)
		switch {
		case strings.Contains(lowerKey, "cookie") ||
			strings.Contains(lowerKey, "token") ||
			strings.Contains(lowerKey, "secret") ||
			strings.Contains(lowerKey, "key") ||
			strings.Contains(lowerKey, "password"):
			masked[key] = "***"
		case strings.Contains(lowerKey, "data"):
			// 对data字段进行递归处理
			if dataMap, ok := value.(map[string]interface{}); ok {
				masked[key] = maskMonicaDataFields(dataMap)
			} else {
				masked[key] = value
			}
		default:
			masked[key] = value
		}
	}
	
	if result, err := json.Marshal(masked); err == nil {
		return string(result)
	}
	return "*** 数据已脱敏 ***"
}

// maskMonicaDataFields 脱敏Monica数据字段中的敏感信息
func maskMonicaDataFields(data map[string]interface{}) map[string]interface{} {
	masked := make(map[string]interface{})
	
	for key, value := range data {
		lowerKey := strings.ToLower(key)
		switch {
		case strings.Contains(lowerKey, "cookie") ||
			strings.Contains(lowerKey, "token") ||
			strings.Contains(lowerKey, "secret"):
			masked[key] = "***"
		default:
			masked[key] = value
		}
	}
	
	return masked
}
