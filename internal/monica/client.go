package monica

import (
	"context"
	"monica-proxy/internal/config"
	"monica-proxy/internal/errors"
	"monica-proxy/internal/logger"
	"monica-proxy/internal/types"
	"monica-proxy/internal/utils"

	"github.com/go-resty/resty/v2"
	"go.uber.org/zap"
)

// SendMonicaRequest 发起对 Monica AI 的请求(使用 resty)
func SendMonicaRequest(ctx context.Context, cfg *config.Config, mReq *types.MonicaRequest) (*resty.Response, error) {
	// 发起请求
	resp, err := utils.RestySSEClient.R().
		SetContext(ctx).
		SetHeader("cookie", cfg.Monica.Cookie).
		SetBody(mReq).
		Post(types.BotChatURL)

	if err != nil {
		logger.Error("Monica API请求失败", zap.Error(err))
		return nil, errors.NewRequestFailedError("Monica API调用失败", err)
	}

	// 如果需要在这里做更多判断，可自行补充
	return resp, nil
}

// SendCustomBotRequest 发送custom bot请求
func SendCustomBotRequest(ctx context.Context, cfg *config.Config, customBotReq *types.CustomBotRequest) (*resty.Response, error) {
	// 发起请求
	resp, err := utils.RestySSEClient.R().
		SetContext(ctx).
		SetHeader("cookie", cfg.Monica.Cookie).
		SetBody(customBotReq).
		Post(types.CustomBotChatURL)

	if err != nil {
		logger.Error("Custom Bot API请求失败", zap.Error(err))
		return nil, errors.NewRequestFailedError("Custom Bot API调用失败", err)
	}

	return resp, nil
}
