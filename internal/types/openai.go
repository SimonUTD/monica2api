package types

import (
	"mime/multipart"

	"github.com/sashabaranov/go-openai"
)

// 扩展openai包以支持更多文件类型
// ExtendedChatCompletionMessage 扩展的聊天消息，支持文档和音频
type ExtendedChatCompletionMessage struct {
	openai.ChatCompletionMessage
	MultiContent []ExtendedChatMessagePart `json:"multi_content,omitempty"`
}

// ExtendedChatMessagePart 扩展的消息部分，支持更多内容类型
type ExtendedChatMessagePart struct {
	Type     string                      `json:"type"`                // "text", "image_url", "document", "audio", etc.
	Text     string                      `json:"text,omitempty"`      // 文本内容
	ImageURL *openai.ChatMessageImageURL `json:"image_url,omitempty"` // 图片URL
	Document *DocumentContent            `json:"document,omitempty"`  // 文档内容
	Audio    *AudioContent               `json:"audio,omitempty"`     // 音频内容
}

// ExtendedChatCompletionRequest 扩展的聊天完成请求
type ExtendedChatCompletionRequest struct {
	Model       string                          `json:"model"`
	Messages    []ExtendedChatCompletionMessage `json:"messages"`
	Stream      *bool                           `json:"stream,omitempty"`
	Temperature *float32                        `json:"temperature,omitempty"`
	MaxTokens   int                             `json:"max_tokens,omitempty"`
	// 其他OpenAI标准参数...
}

// FileObject OpenAI兼容的文件对象
type FileObject struct {
	ID            string                 `json:"id"`                       // 文件ID
	Object        string                 `json:"object"`                   // 对象类型，固定为 "file"
	Bytes         int                    `json:"bytes"`                    // 文件大小（字节）
	CreatedAt     int64                  `json:"created_at"`               // 创建时间戳
	Filename      string                 `json:"filename"`                 // 文件名
	Purpose       string                 `json:"purpose"`                  // 文件用途
	Status        string                 `json:"status"`                   // 处理状态
	StatusDetails map[string]interface{} `json:"status_details,omitempty"` // 状态详情
}

// FileListResponse OpenAI兼容的文件列表响应
type FileListResponse struct {
	Object string       `json:"object"` // 固定为 "list"
	Data   []FileObject `json:"data"`   // 文件列表
}

// FileUploadRequest OpenAI兼容的文件上传请求
type FileUploadRequest struct {
	File    *multipart.FileHeader `json:"file"`    // 上传的文件
	Purpose string                `json:"purpose"` // 文件用途
}

// DeleteFileResponse OpenAI兼容的删除文件响应
type DeleteFileResponse struct {
	ID      string `json:"id"`      // 文件ID
	Object  string `json:"object"`  // 固定为 "file"
	Deleted bool   `json:"deleted"` // 是否删除成功
}

// ImageGenerationRequest represents a request to create an image using DALL-E
type ImageGenerationRequest struct {
	Model          string `json:"model"`                     // Required. Currently supports: dall-e-3
	Prompt         string `json:"prompt"`                    // Required. A text description of the desired image(s)
	N              int    `json:"n,omitempty"`               // Optional. The number of images to generate. Default is 1
	Quality        string `json:"quality,omitempty"`         // Optional. The quality of the image that will be generated
	ResponseFormat string `json:"response_format,omitempty"` // Optional. The format in which the generated images are returned
	Size           string `json:"size,omitempty"`            // Optional. The size of the generated images
	Style          string `json:"style,omitempty"`           // Optional. The style of the generated images
	User           string `json:"user,omitempty"`            // Optional. A unique identifier representing your end-user
}

// ImageGenerationResponse represents the response from the DALL-E image generation API
type ImageGenerationResponse struct {
	Created int64                 `json:"created"`
	Data    []ImageGenerationData `json:"data"`
}

// ImageGenerationData represents a single image in the response
type ImageGenerationData struct {
	URL           string `json:"url,omitempty"`            // The URL of the generated image
	B64JSON       string `json:"b64_json,omitempty"`       // Base64 encoded JSON of the generated image
	RevisedPrompt string `json:"revised_prompt,omitempty"` // The prompt that was used to generate the image
}

type ChatCompletionStreamResponse struct {
	ID                  string                       `json:"id"`
	Object              string                       `json:"object"`
	Created             int64                        `json:"created"`
	Model               string                       `json:"model"`
	Choices             []ChatCompletionStreamChoice `json:"choices"`
	SystemFingerprint   string                       `json:"system_fingerprint"`
	PromptAnnotations   []openai.PromptAnnotation    `json:"prompt_annotations,omitempty"`
	PromptFilterResults []openai.PromptFilterResult  `json:"prompt_filter_results,omitempty"`
	Usage               *openai.Usage                `json:"usage,omitempty"`
}

type ChatCompletionStreamChoice struct {
	Index        int                                        `json:"index"`
	Delta        openai.ChatCompletionStreamChoiceDelta     `json:"delta"`
	Logprobs     *openai.ChatCompletionStreamChoiceLogprobs `json:"logprobs,omitempty"`
	FinishReason openai.FinishReason                        `json:"finish_reason"`
}
