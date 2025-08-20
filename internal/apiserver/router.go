package apiserver

import (
	"fmt"
	"io"
	"monica-proxy/internal/config"
	"monica-proxy/internal/errors"
	"monica-proxy/internal/logger"
	"monica-proxy/internal/middleware"
	"monica-proxy/internal/monica"
	"monica-proxy/internal/service"
	"monica-proxy/internal/types"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/sashabaranov/go-openai"
	"go.uber.org/zap"
)

// RegisterRoutes 注册 Echo 路由
func RegisterRoutes(e *echo.Echo, cfg *config.Config) {
	// 设置自定义错误处理器
	e.HTTPErrorHandler = middleware.ErrorHandler()

	// 添加中间件
	e.Use(middleware.BearerAuth(cfg))
	e.Use(middleware.RequestLogger(cfg))

	// 初始化服务实例
	chatService := service.NewChatService(cfg)
	modelService := service.NewModelService(cfg)
	imageService := service.NewImageService(cfg)
	customBotService := service.NewCustomBotService(cfg)

	// ChatGPT 风格的请求转发到 /v1/chat/completions
	e.POST("/v1/chat/completions", createChatCompletionHandler(chatService, customBotService, cfg))
	// 获取支持的模型列表
	e.GET("/v1/models", createListModelsHandler(modelService))
	// DALL-E 风格的图片生成请求
	e.POST("/v1/images/generations", createImageGenerationHandler(imageService))
	// Custom Bot 测试接口
	e.POST("/v1/chat/custom-bot/:bot_uid", createCustomBotHandler(customBotService, cfg))
	// 新增不带bot_uid的路由，使用环境变量中的BOT_UID
	e.POST("/v1/chat/custom-bot", createCustomBotHandler(customBotService, cfg))
}

// createChatCompletionHandler 创建聊天完成处理器
func createChatCompletionHandler(chatService service.ChatService, customBotService service.CustomBotService, cfg *config.Config) echo.HandlerFunc {
	return func(c echo.Context) error {
		var req openai.ChatCompletionRequest
		if err := c.Bind(&req); err != nil {
			return errors.NewBadRequestError("无效的请求数据", err)
		}

		ctx := c.Request().Context()
		var result interface{}
		var err error

		// 检查是否启用了 Custom Bot 模式
		if cfg.Monica.EnableCustomBotMode {
			// 使用 Custom Bot Service 处理请求
			result, err = customBotService.HandleCustomBotChat(ctx, &req, cfg.Monica.BotUID)
		} else {
			// 使用普通的 Chat Service 处理请求
			result, err = chatService.HandleChatCompletion(ctx, &req)
		}

		if err != nil {
			return err
		}

		// 根据请求参数决定响应方式
		if req.Stream {
			// 对于流式请求，result是一个io.ReadCloser
			rawBody, ok := result.(io.Reader)
			if !ok {
				return errors.NewInternalError(nil)
			}

			// 确保关闭响应体
			closer, isCloser := rawBody.(io.Closer)
			if isCloser {
				defer closer.Close()
			}

			// 设置响应头
			c.Response().Header().Set(echo.HeaderContentType, "text/event-stream")
			c.Response().Header().Set("Cache-Control", "no-cache")
			c.Response().Header().Set("Transfer-Encoding", "chunked")
			c.Response().WriteHeader(http.StatusOK)

			// 流式处理响应
			if err := monica.StreamMonicaSSEToClient(req.Model, c.Response().Writer, rawBody); err != nil {
				return errors.NewInternalError(err)
			}
			return nil
		} else {
			// 对于非流式请求，直接返回JSON响应
			return c.JSON(http.StatusOK, result)
		}
	}
}

// createListModelsHandler 创建模型列表处理器
func createListModelsHandler(modelService service.ModelService) echo.HandlerFunc {
	return func(c echo.Context) error {
		// 调用服务获取模型列表
		models := modelService.GetSupportedModels()

		// 构造响应格式
		result := make(map[string][]struct {
			Id string `json:"id"`
		})

		result["data"] = make([]struct {
			Id string `json:"id"`
		}, 0)

		for _, model := range models {
			result["data"] = append(result["data"], struct {
				Id string `json:"id"`
			}{
				Id: model,
			})
		}
		return c.JSON(http.StatusOK, result)
	}
}

// createImageGenerationHandler 创建图片生成处理器
func createImageGenerationHandler(imageService service.ImageService) echo.HandlerFunc {
	return func(c echo.Context) error {
		// 解析请求
		var req types.ImageGenerationRequest
		if err := c.Bind(&req); err != nil {
			return errors.NewBadRequestError("无效的请求数据", err)
		}

		// 调用服务生成图片
		resp, err := imageService.GenerateImage(c.Request().Context(), &req)
		if err != nil {
			return err
		}

		// 返回结果
		return c.JSON(http.StatusOK, resp)
	}
}

// createCustomBotHandler 创建Custom Bot处理器
func createCustomBotHandler(service service.CustomBotService, cfg *config.Config) echo.HandlerFunc {
	return func(c echo.Context) error {
		// 获取bot UID，优先从路由参数获取，如果没有则从环境变量获取
		botUID := c.Param("bot_uid")
		if botUID == "" {
			// 从配置（环境变量）中获取
			botUID = cfg.Monica.BotUID
			if botUID == "" {
				return errors.NewBadRequestError("bot_uid参数不能为空，请在URL中指定或设置BOT_UID环境变量", nil)
			}
		}

		var req openai.ChatCompletionRequest
		if err := c.Bind(&req); err != nil {
			return errors.NewBadRequestError("请求体解析失败", err)
		}

		ctx := c.Request().Context()
		result, err := service.HandleCustomBotChat(ctx, &req, botUID)
		if err != nil {
			return err
		}

		// 如果是流式响应
		if req.Stream {
			// 设置响应头
			c.Response().Header().Set("Content-Type", "text/event-stream")
			c.Response().Header().Set("Cache-Control", "no-cache")
			c.Response().Header().Set("Connection", "keep-alive")
			c.Response().Header().Set("Transfer-Encoding", "chunked")

			// 获取响应体（io.ReadCloser）
			stream, ok := result.(io.ReadCloser)
			if !ok {
				return errors.NewInternalError(fmt.Errorf("流式响应类型错误"))
			}
			defer stream.Close()

			// 转换并写入响应
			err := monica.StreamMonicaSSEToClient(req.Model, c.Response().Writer, stream)
			if err != nil {
				logger.Error("流式响应写入失败", zap.Error(err))
				return err
			}

			c.Response().Flush()
			return nil
		}

		// 非流式响应
		return c.JSON(http.StatusOK, result)
	}
}
