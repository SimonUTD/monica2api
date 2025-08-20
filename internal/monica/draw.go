package monica

import (
	"context"
	"fmt"
	"monica-proxy/internal/config"
	"monica-proxy/internal/types"
	"monica-proxy/internal/utils"
	"time"

	"github.com/bytedance/sonic"
	"github.com/google/uuid"
)

// GenerateImage 使用 Monica 的文生图 API 生成图片
func GenerateImage(ctx context.Context, cfg *config.Config, req *types.ImageGenerationRequest) (*types.ImageGenerationResponse, error) {
	// 1. 参数验证和默认值设置
	if req.Model == "" {
		req.Model = "dall-e-3" // 默认使用 dall-e-3
	}
	if req.N <= 0 {
		req.N = 1 // 默认生成1张图片
	}
	if req.Size == "" {
		req.Size = "1024x1024" // 默认尺寸
	}

	// 2. 转换尺寸为 Monica 的格式
	aspectRatio := sizeToAspectRatio(req.Size)

	// 3. 构建请求体
	monicaReq := &types.MonicaImageRequest{
		TaskUID:     uuid.New().String(),
		ImageCount:  req.N,
		Prompt:      req.Prompt,
		ModelType:   "sdxl", // Monica 目前只支持 sdxl
		AspectRatio: aspectRatio,
		TaskType:    "text_to_image",
	}

	// 4. 发送请求生成图片
	resp, err := utils.RestyDefaultClient.R().
		SetContext(ctx).
		SetBody(monicaReq).
		SetHeader("cookie", cfg.Monica.Cookie).
		Post(types.ImageGenerateURL)

	if err != nil {
		return nil, fmt.Errorf("failed to send image generation request: %v", err)
	}

	// 5. 解析响应
	var monicaResp struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			ImageToolsID int `json:"image_tools_id"`
			ExpectedTime int `json:"expected_time"`
		} `json:"data"`
	}

	if err := sonic.Unmarshal(resp.Body(), &monicaResp); err != nil {
		return nil, fmt.Errorf("failed to parse image generation response: %v", err)
	}

	if monicaResp.Code != 0 {
		return nil, fmt.Errorf("image generation failed: %s", monicaResp.Msg)
	}

	// 6. 轮询获取生成结果
	imageToolsID := monicaResp.Data.ImageToolsID
	expectedTime := monicaResp.Data.ExpectedTime

	// 设置轮询超时时间为预期时间的2倍
	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(expectedTime*2)*time.Second)
	defer cancel()

	var generatedImages []types.ImageGenerationData
	for {
		select {
		case <-timeoutCtx.Done():
			return nil, fmt.Errorf("timeout waiting for image generation")
		default:
			var resultData struct {
				Code int    `json:"code"`
				Msg  string `json:"msg"`
				Data struct {
					Record struct {
						Result struct {
							CDNURLList []string `json:"cdn_url_list"`
						} `json:"result"`
					} `json:"record"`
				} `json:"data"`
			}

			// 查询生成结果
			_, err := utils.RestyDefaultClient.R().
				SetContext(ctx).
				SetBody(map[string]any{
					"image_tools_id": imageToolsID,
				}).
				SetHeader("cookie", cfg.Monica.Cookie).
				SetResult(&resultData).
				Post(types.ImageResultURL)

			if err != nil {
				return nil, fmt.Errorf("failed to get image generation result: %v", err)
			}

			if resultData.Code != 0 {
				return nil, fmt.Errorf("failed to get image result: %s", resultData.Msg)
			}

			// 检查是否有图片生成完成
			if len(resultData.Data.Record.Result.CDNURLList) > 0 {
				// 构建返回数据
				for _, url := range resultData.Data.Record.Result.CDNURLList {
					generatedImages = append(generatedImages, types.ImageGenerationData{
						URL:           url,
						RevisedPrompt: req.Prompt, // Monica 不提供修改后的提示词
					})
				}

				// 返回结果
				return &types.ImageGenerationResponse{
					Created: time.Now().Unix(),
					Data:    generatedImages,
				}, nil
			}

			// 等待一段时间后继续轮询
			time.Sleep(time.Second)
		}
	}
}

// sizeToAspectRatio 将 OpenAI 的尺寸格式转换为 Monica 的宽高比格式
func sizeToAspectRatio(size string) string {
	switch size {
	case "1024x1024":
		return "1:1"
	case "1792x1024":
		return "16:9"
	case "1024x1792":
		return "9:16"
	default:
		return "1:1" // 默认使用1:1
	}
}
