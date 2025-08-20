package service

import (
	"context"
	"monica-proxy/internal/config"
	"monica-proxy/internal/errors"
	"monica-proxy/internal/logger"
	"monica-proxy/internal/monica"
	"monica-proxy/internal/types"

	"github.com/sashabaranov/go-openai"
	"go.uber.org/zap"
)

// CustomBotService 定义自定义Bot服务接口
type CustomBotService interface {
	HandleCustomBotChat(ctx context.Context, req *openai.ChatCompletionRequest, botUID string) (interface{}, error)
}

type customBotService struct {
	config *config.Config
}

// NewCustomBotService 创建自定义Bot服务实例
func NewCustomBotService(cfg *config.Config) CustomBotService {
	return &customBotService{
		config: cfg,
	}
}

// HandleCustomBotChat 处理自定义Bot对话请求
func (s *customBotService) HandleCustomBotChat(ctx context.Context, req *openai.ChatCompletionRequest, botUID string) (interface{}, error) {
	// 验证请求
	if len(req.Messages) == 0 {
		return nil, errors.NewEmptyMessageError()
	}

	// 日志记录请求
	logger.Info("处理Custom Bot聊天请求",
		zap.String("model", req.Model),
		zap.String("bot_uid", botUID),
		zap.Int("message_count", len(req.Messages)),
		zap.Bool("stream", req.Stream),
	)

	// 转换请求格式
	customBotReq, err := types.ChatGPTToCustomBot(s.config, *req, botUID)
	if err != nil {
		logger.Error("转换Custom Bot请求失败", zap.Error(err))
		return nil, errors.NewInternalError(err)
	}

	// 调用Monica Custom Bot API
	stream, err := monica.SendCustomBotRequest(ctx, s.config, customBotReq)
	if err != nil {
		logger.Error("调用Custom Bot API失败", zap.Error(err))
		// 如果已经是AppError，直接返回，否则包装为内部错误
		if appErr, ok := err.(*errors.AppError); ok {
			return nil, appErr
		}
		return nil, errors.NewInternalError(err)
	}

	// 根据是否使用流式响应处理结果
	if req.Stream {
		// 流式响应时不关闭响应体，让handler层负责关闭
		return stream.RawBody(), nil
	}

	// 非流式响应，确保在此函数结束时关闭响应体
	defer stream.RawBody().Close()

	// 处理非流式响应
	response, err := monica.CollectMonicaSSEToCompletion(req.Model, stream.RawBody())
	if err != nil {
		logger.Error("处理Custom Bot响应失败", zap.Error(err))
		return nil, errors.NewInternalError(err)
	}

	return response, nil
}
