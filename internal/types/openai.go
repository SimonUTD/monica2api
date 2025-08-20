package types

import "github.com/sashabaranov/go-openai"

// ImageGenerationRequest represents a request to create an image using DALL-E
type ImageGenerationRequest struct {
	Model          string `json:"model"`           // Required. Currently supports: dall-e-3
	Prompt         string `json:"prompt"`          // Required. A text description of the desired image(s)
	N              int    `json:"n,omitempty"`     // Optional. The number of images to generate. Default is 1
	Quality        string `json:"quality,omitempty"` // Optional. The quality of the image that will be generated
	ResponseFormat string `json:"response_format,omitempty"` // Optional. The format in which the generated images are returned
	Size           string `json:"size,omitempty"`   // Optional. The size of the generated images
	Style          string `json:"style,omitempty"`  // Optional. The style of the generated images
	User           string `json:"user,omitempty"`   // Optional. A unique identifier representing your end-user
}

// ImageGenerationResponse represents the response from the DALL-E image generation API
type ImageGenerationResponse struct {
	Created int64                     `json:"created"`
	Data    []ImageGenerationData    `json:"data"`
}

// ImageGenerationData represents a single image in the response
type ImageGenerationData struct {
	URL        string `json:"url,omitempty"`        // The URL of the generated image
	B64JSON    string `json:"b64_json,omitempty"`   // Base64 encoded JSON of the generated image
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