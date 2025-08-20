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

// ChatService 聊天服务接口
type ChatService interface {
	// HandleChatCompletion 处理聊天完成请求
	HandleChatCompletion(ctx context.Context, req *openai.ChatCompletionRequest) (interface{}, error)
}

// chatService 聊天服务实现
type chatService struct {
	config *config.Config
}

// NewChatService 创建聊天服务实例
func NewChatService(cfg *config.Config) ChatService {
	return &chatService{
		config: cfg,
	}
}

// HandleChatCompletion 处理聊天完成请求
func (s *chatService) HandleChatCompletion(ctx context.Context, req *openai.ChatCompletionRequest) (interface{}, error) {
	// 验证请求
	if len(req.Messages) == 0 {
		return nil, errors.NewEmptyMessageError()
	}

	// 日志记录请求
	// logger.Info("处理聊天请求",
	// 	zap.String("model", req.Model),
	// 	zap.Int("message_count", len(req.Messages)),
	// 	zap.Bool("stream", req.Stream),
	// )

	// 转换请求格式
	monicaReq, err := types.ChatGPTToMonica(s.config, *req)
	if err != nil {
		logger.Error("转换请求失败", zap.Error(err))
		return nil, errors.NewInternalError(err)
	}

	// 调用Monica API
	stream, err := monica.SendMonicaRequest(ctx, s.config, monicaReq)
	if err != nil {
		logger.Error("调用Monica API失败", zap.Error(err))
		// 如果已经是AppError，直接返回，否则包装为内部错误
		if appErr, ok := err.(*errors.AppError); ok {
			return nil, appErr
		}
		return nil, errors.NewInternalError(err)
	}
	// 根据是否使用流式响应处理结果
	if req.Stream {
		// 这里只返回stream，实际的流处理在handler层
		// 流式响应时不关闭响应体，让handler层负责关闭
		return stream.RawBody(), nil
	}

	// 非流式响应，确保在此函数结束时关闭响应体
	defer stream.RawBody().Close()

	// 处理非流式响应
	response, err := monica.CollectMonicaSSEToCompletion(req.Model, stream.RawBody())
	if err != nil {
		logger.Error("处理Monica响应失败", zap.Error(err))
		return nil, errors.NewInternalError(err)
	}

	return response, nil
}
