package service

import (
	"context"
	"monica-proxy/internal/config"
	"monica-proxy/internal/errors"
	"monica-proxy/internal/logger"
	"monica-proxy/internal/monica"
	"monica-proxy/internal/types"

	"go.uber.org/zap"
)

// ImageService 图像服务接口
type ImageService interface {
	// GenerateImage 生成图像
	GenerateImage(ctx context.Context, req *types.ImageGenerationRequest) (*types.ImageGenerationResponse, error)
}

// imageService 图像服务实现
type imageService struct {
	config *config.Config
}

// NewImageService 创建图像服务实例
func NewImageService(cfg *config.Config) ImageService {
	return &imageService{
		config: cfg,
	}
}

// GenerateImage 生成图像
func (s *imageService) GenerateImage(ctx context.Context, req *types.ImageGenerationRequest) (*types.ImageGenerationResponse, error) {
	// 验证请求
	if req.Prompt == "" {
		return nil, errors.NewInvalidInputError("提示词不能为空", nil)
	}

	// 设置默认值
	if req.Model == "" {
		req.Model = "dall-e-3"
	}
	if req.N <= 0 {
		req.N = 1
	}
	if req.Size == "" {
		req.Size = "1024x1024"
	}

	// 日志记录请求
	logger.Info("处理图像生成请求",
		zap.String("model", req.Model),
		zap.String("size", req.Size),
		zap.Int("count", req.N),
	)

	// 调用Monica API生成图像
	response, err := monica.GenerateImage(ctx, s.config, req)
	if err != nil {
		logger.Error("生成图像失败", zap.Error(err))
		return nil, errors.NewImageGenerationError(err)
	}

	return response, nil
}
