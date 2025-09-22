package types

import (
	"context"
	"fmt"
	"monica-proxy/internal/config"
	"monica-proxy/internal/logger"
	"regexp"
	"strings"
	"sync/atomic"
	"time"

	lop "github.com/samber/lo/parallel"

	"github.com/google/uuid"
	"github.com/sashabaranov/go-openai"
	"go.uber.org/zap"
)

const (
	BotChatURL    = "https://api.monica.im/api/custom_bot/chat"
	PreSignURL    = "https://api.monica.im/api/file_object/pre_sign_list_by_module"
	FileUploadURL = "https://api.monica.im/api/files/batch_create_llm_file"
	FileGetURL    = "https://api.monica.im/api/files/batch_get_file"

	// 图片生成相关 API
	ImageGenerateURL = "https://api.monica.im/api/image_tools/text_to_image"
	ImageResultURL   = "https://api.monica.im/api/image_tools/loop_result"
)

// 文件相关常量
const (
	MaxFileSize          = 100 * 1024 * 1024 // 100MB (遵循OpenAI限制)
	MaxImageSize         = 10 * 1024 * 1024  // 10MB
	FileModule           = "chat_bot"
	FileLocation         = "files"
	FileUploadTimeout    = 60 * time.Second // 文件上传超时时间
	MaxConcurrentUploads = 5                // 最大并发上传数
)

// OpenAI兼容的文件类型映射
var SupportedFileTypes = map[string]FileTypeInfo{
	// 图片类型
	"image/jpeg": {Extension: ".jpg", Category: "image", MaxSize: MaxImageSize},
	"image/png":  {Extension: ".png", Category: "image", MaxSize: MaxImageSize},
	"image/gif":  {Extension: ".gif", Category: "image", MaxSize: MaxImageSize},
	"image/webp": {Extension: ".webp", Category: "image", MaxSize: MaxImageSize},

	// 文档类型
	"application/pdf":    {Extension: ".pdf", Category: "document", MaxSize: MaxFileSize},
	"application/msword": {Extension: ".doc", Category: "document", MaxSize: MaxFileSize},
	"application/vnd.openxmlformats-officedocument.wordprocessingml.document": {Extension: ".docx", Category: "document", MaxSize: MaxFileSize},
	"application/vnd.ms-excel": {Extension: ".xls", Category: "document", MaxSize: MaxFileSize},
	"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":         {Extension: ".xlsx", Category: "document", MaxSize: MaxFileSize},
	"application/vnd.ms-powerpoint":                                             {Extension: ".ppt", Category: "document", MaxSize: MaxFileSize},
	"application/vnd.openxmlformats-officedocument.presentationml.presentation": {Extension: ".pptx", Category: "document", MaxSize: MaxFileSize},

	// 文本类型
	"text/plain":       {Extension: ".txt", Category: "text", MaxSize: MaxFileSize},
	"text/markdown":    {Extension: ".md", Category: "text", MaxSize: MaxFileSize},
	"text/csv":         {Extension: ".csv", Category: "text", MaxSize: MaxFileSize},
	"application/json": {Extension: ".json", Category: "text", MaxSize: MaxFileSize},
	"application/xml":  {Extension: ".xml", Category: "text", MaxSize: MaxFileSize},
	"text/xml":         {Extension: ".xml", Category: "text", MaxSize: MaxFileSize},

	// 代码类型
	"text/javascript":        {Extension: ".js", Category: "code", MaxSize: MaxFileSize},
	"application/javascript": {Extension: ".js", Category: "code", MaxSize: MaxFileSize},
	"text/html":              {Extension: ".html", Category: "code", MaxSize: MaxFileSize},
	"text/css":               {Extension: ".css", Category: "code", MaxSize: MaxFileSize},
	"application/x-python":   {Extension: ".py", Category: "code", MaxSize: MaxFileSize},
	"text/x-python":          {Extension: ".py", Category: "code", MaxSize: MaxFileSize},

	// 音频类型 (Monica可能支持)
	"audio/mpeg": {Extension: ".mp3", Category: "audio", MaxSize: MaxFileSize},
	"audio/wav":  {Extension: ".wav", Category: "audio", MaxSize: MaxFileSize},
	"audio/ogg":  {Extension: ".ogg", Category: "audio", MaxSize: MaxFileSize},
	"audio/mp4":  {Extension: ".m4a", Category: "audio", MaxSize: MaxFileSize},

	// 视频类型 (Monica可能支持)
	"video/mp4": {Extension: ".mp4", Category: "video", MaxSize: MaxFileSize},
	"video/avi": {Extension: ".avi", Category: "video", MaxSize: MaxFileSize},
	"video/mov": {Extension: ".mov", Category: "video", MaxSize: MaxFileSize},
}

// FileTypeInfo 文件类型信息
type FileTypeInfo struct {
	Extension string // 文件扩展名
	Category  string // 文件类别
	MaxSize   int64  // 最大文件大小
}

// AttachmentRequest 附件请求结构
type AttachmentRequest struct {
	Type     string `json:"type"`                // 附件类型: image_url, document, audio, etc.
	Data     string `json:"data"`                // 文件数据 (base64, URL等)
	FileName string `json:"file_name,omitempty"` // 文件名
	MimeType string `json:"mime_type,omitempty"` // MIME类型
}

// 扩展openai包的消息内容类型定义
type DocumentContent struct {
	URL      string `json:"url"`                 // 文档URL或base64
	Name     string `json:"name,omitempty"`      // 文档名称
	MimeType string `json:"mime_type,omitempty"` // MIME类型
}

type AudioContent struct {
	URL      string `json:"url"`                 // 音频URL或base64
	Name     string `json:"name,omitempty"`      // 音频名称
	MimeType string `json:"mime_type,omitempty"` // MIME类型
}

type ChatGPTRequest struct {
	Model    string        `json:"model"`    // gpt-3.5-turbo, gpt-4, ...
	Messages []ChatMessage `json:"messages"` // 对话数组
	Stream   bool          `json:"stream"`   // 是否流式返回
}

type ChatMessage struct {
	Role    string `json:"role"`    // "system", "user", "assistant"
	Content any    `json:"content"` // 可以是字符串或MessageContent数组
}

// MessageContent 消息内容
type MessageContent struct {
	Type     string `json:"type"`                // "text" 或 "image_url"
	Text     string `json:"text,omitempty"`      // 文本内容
	ImageURL string `json:"image_url,omitempty"` // 图片URL
}

// MonicaRequest 为 Monica 自定义 AI 的请求格式
type MonicaRequest struct {
	TaskUID  string    `json:"task_uid"`
	BotUID   string    `json:"bot_uid"`
	Data     DataField `json:"data"`
	Language string    `json:"language"`
	TaskType string    `json:"task_type"`
	ToolData ToolData  `json:"tool_data"`
}

// DataField 在 Monica 的 body 中
type DataField struct {
	ConversationID  string `json:"conversation_id"`
	PreParentItemID string `json:"pre_parent_item_id"`
	Items           []Item `json:"items"`
	TriggerBy       string `json:"trigger_by"`
	UseModel        string `json:"use_model,omitempty"`
	IsIncognito     bool   `json:"is_incognito"`
	UseNewMemory    bool   `json:"use_new_memory"`
}

type Item struct {
	ConversationID string      `json:"conversation_id"`
	ParentItemID   string      `json:"parent_item_id,omitempty"`
	ItemID         string      `json:"item_id"`
	ItemType       string      `json:"item_type"`
	Data           ItemContent `json:"data"`
}

type ItemContent struct {
	Type                   string     `json:"type"`
	Content                string     `json:"content"`
	MaxToken               int        `json:"max_token,omitempty"`
	IsIncognito            bool       `json:"is_incognito,omitempty"` // 是否无痕模式
	FromTaskType           string     `json:"from_task_type,omitempty"`
	ManualWebSearchEnabled bool       `json:"manual_web_search_enabled,omitempty"` // 网页搜索
	UseModel               string     `json:"use_model,omitempty"`
	FileInfos              []FileInfo `json:"file_infos,omitempty"`
}

// ToolData 这里演示放空
type ToolData struct {
	SysSkillList []string `json:"sys_skill_list"`
}

// PreSignRequest 预签名请求
type PreSignRequest struct {
	FilenameList []string `json:"filename_list"`
	Module       string   `json:"module"`
	Location     string   `json:"location"`
	ObjID        string   `json:"obj_id"`
}

// PreSignResponse 预签名响应
type PreSignResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		PreSignURLList []string `json:"pre_sign_url_list"`
		ObjectURLList  []string `json:"object_url_list"`
		CDNURLList     []string `json:"cdn_url_list"`
	} `json:"data"`
}

// MonicaImageRequest 文生图请求结构
type MonicaImageRequest struct {
	TaskUID     string `json:"task_uid"`     // 任务ID
	ImageCount  int    `json:"image_count"`  // 生成图片数量
	Prompt      string `json:"prompt"`       // 提示词
	ModelType   string `json:"model_type"`   // 模型类型，目前只支持 sdxl
	AspectRatio string `json:"aspect_ratio"` // 宽高比，如 1:1, 16:9, 9:16
	TaskType    string `json:"task_type"`    // 任务类型，固定为 text_to_image
}

// FileInfo 文件信息
type FileInfo struct {
	URL        string `json:"url,omitempty"`
	FileURL    string `json:"file_url"`
	FileUID    string `json:"file_uid"`
	Parse      bool   `json:"parse"`
	FileName   string `json:"file_name"`
	FileSize   int64  `json:"file_size"`
	FileType   string `json:"file_type"`
	FileExt    string `json:"file_ext"`
	FileTokens int64  `json:"file_tokens"`
	FileChunks int64  `json:"file_chunks"`
	ObjectURL  string `json:"object_url,omitempty"`
	//Embedding    bool                   `json:"embedding"`
	FileMetaInfo map[string]any `json:"file_meta_info,omitempty"`
	UseFullText  bool           `json:"use_full_text"`
}

// MonicaFileUploadRequest Monica API文件上传请求
type MonicaFileUploadRequest struct {
	Data []FileInfo `json:"data"`
}

// FileUploadResponse 文件上传响应
type FileUploadResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		Items []struct {
			FileName   string `json:"file_name"`
			FileType   string `json:"file_type"`
			FileSize   int64  `json:"file_size"`
			FileUID    string `json:"file_uid"`
			FileTokens int64  `json:"file_tokens"`
			FileChunks int64  `json:"file_chunks"`
			// 其他字段暂时不需要
		} `json:"items"`
	} `json:"data"`
}

// FileBatchGetResponse 获取文件llm处理是否完成
type FileBatchGetResponse struct {
	Data struct {
		Items []struct {
			FileName     string `json:"file_name"`
			FileType     string `json:"file_type"`
			FileSize     int    `json:"file_size"`
			ObjectUrl    string `json:"object_url"`
			Url          string `json:"url"`
			FileMetaInfo struct {
			} `json:"file_meta_info"`
			DriveFileUid  string `json:"drive_file_uid"`
			FileUid       string `json:"file_uid"`
			IndexState    int    `json:"index_state"`
			IndexDesc     string `json:"index_desc"`
			ErrorMessage  string `json:"error_message"`
			FileTokens    int64  `json:"file_tokens"`
			FileChunks    int64  `json:"file_chunks"`
			IndexProgress int    `json:"index_progress"`
		} `json:"items"`
	} `json:"data"`
}

// OpenAIModel represents a model in the OpenAI API format
type OpenAIModel struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	OwnedBy string `json:"owned_by"`
}

// OpenAIModelList represents the response format for the /v1/models endpoint
type OpenAIModelList struct {
	Object string        `json:"object"`
	Data   []OpenAIModel `json:"data"`
}

var modelToBotMap = map[string]string{
	"gpt-5":        "gpt_5",
	"gpt-4o":       "gpt_4_o_chat",
	"gpt-4o-mini":  "gpt_4_o_mini_chat",
	"gpt-4.1":      "gpt_4_1",
	"gpt-4.1-mini": "gpt_4_1_mini",
	"gpt-4.1-nano": "gpt_4_1_nano",
	"gpt-4-5":      "gpt_4_5_chat",
	"o1-preview":   "openai_o_1",
	"o3":           "o3",
	"o3-mini":      "openai_o_3_mini",
	"o4-mini":      "o4_mini",

	"claude-4-sonnet":            "claude_4_sonnet",
	"claude-4-sonnet-thinking":   "claude_4_sonnet_think",
	"claude-4-opus":              "claude_4_opus",
	"claude-4-opus-thinking":     "claude_4_opus_think",
	"claude-3-7-sonnet-thinking": "claude_3_7_sonnet_think",
	"claude-3-7-sonnet":          "claude_3_7_sonnet",
	"claude-3-5-sonnet":          "claude_3.5_sonnet",
	"claude-3-5-haiku":           "claude_3.5_haiku",

	"gemini-2.5-pro":   "gemini_2_5_pro",
	"gemini-2.5-flash": "gemini_2_5_flash",
	"gemini-2.0-flash": "gemini_2_0",
	"gemini-1":         "gemini_1_5",

	"deepseek-reasoner": "deepseek_reasoner",
	"deepseek-chat":     "deepseek_chat",
	"deepclaude":        "deepclaude",

	"sonar":               "sonar",
	"sonar-reasoning-pro": "sonar_reasoning_pro",

	"grok-3-beta": "grok_3_beta",
	"grok-4":      "grok_4",
}

func modelToBot(model string) string {
	if botUID, ok := modelToBotMap[model]; ok {
		return botUID
	}
	// 如果未找到映射，则返回原始模型名称
	logger.Warn("未找到模型映射，使用原始名称", zap.String("model", model))
	return model
}

// CustomBotRequest 定义custom bot的请求结构
type CustomBotRequest struct {
	TaskUID        string        `json:"task_uid"`
	BotUID         string        `json:"bot_uid"`
	Data           CustomBotData `json:"data"`
	Language       string        `json:"language"`
	Locale         string        `json:"locale"`
	TaskType       string        `json:"task_type"`
	BotData        BotData       `json:"bot_data"`
	AIRespLanguage string        `json:"ai_resp_language,omitempty"`
}

// CustomBotData custom bot的数据字段
type CustomBotData struct {
	ConversationID      string `json:"conversation_id"`
	Items               []Item `json:"items"`
	PreGeneratedReplyID string `json:"pre_generated_reply_id"`
	PreParentItemID     string `json:"pre_parent_item_id"`
	Origin              string `json:"origin"`
	OriginPageTitle     string `json:"origin_page_title"`
	TriggerBy           string `json:"trigger_by"`
	UseModel            string `json:"use_model"`
	IsIncognito         bool   `json:"is_incognito"`
	UseNewMemory        bool   `json:"use_new_memory"`
	UseMemorySuggestion bool   `json:"use_memory_suggestion"`
}

// BotData bot配置数据
type BotData struct {
	Description    string        `json:"description"`
	LogoURL        string        `json:"logo_url"`
	Name           string        `json:"name"`
	Classification string        `json:"classification"`
	Prompt         string        `json:"prompt"`
	Type           string        `json:"type"`
	UID            string        `json:"uid"`
	ExampleList    []interface{} `json:"example_list"`
	ToolData       BotToolData   `json:"tool_data"`
}

// BotToolData bot工具数据
type BotToolData struct {
	KnowledgeList    []interface{} `json:"knowledge_list"`
	UserSkillList    []interface{} `json:"user_skill_list"`
	SysSkillList     []interface{} `json:"sys_skill_list"`
	UseModel         string        `json:"use_model"`
	ScheduleTaskList []interface{} `json:"schedule_task_list"`
}

// Custom Bot相关的URL
const (
	CustomBotSaveURL    = "https://api.monica.im/api/custom_bot/save_bot"
	CustomBotPublishURL = "https://api.monica.im/api/custom_bot/publish_bot"
	CustomBotPinURL     = "https://api.monica.im/api/custom_bot/pin_bot"
	CustomBotChatURL    = "https://api.monica.im/api/custom_bot/preview_chat"
)

// GetSupportedModels 获取支持的模型列表
func GetSupportedModels() []string {
	models := []string{
		"gpt-5",
		"gpt-4o",
		"gpt-4o-mini",
		"gpt-4-5",
		"gpt-4.1",
		"gpt-4.1-mini",
		"gpt-4.1-nano",

		"claude-4-sonnet",
		"claude-4-sonnet-thinking",
		"claude-4-opus",
		"claude-4-opus-thinking",
		"claude-3-7-sonnet-thinking",
		"claude-3-7-sonnet",
		"claude-3-5-sonnet",
		"claude-3-5-haiku",

		"gemini-2.5-pro",
		"gemini-2.5-flash",
		"gemini-2.0-flash",
		"gemini-1",

		"o1-preview",
		"o3",
		"o3-mini",
		"o4-mini",

		"deepseek-reasoner",
		"deepseek-chat",
		"deepclaude",
		"sonar",
		"sonar-reasoning-pro",
		"grok-3-beta",
		"grok-4",
	}
	return models
}

// ChatGPTToMonica 将 ChatGPTRequest 转换为 MonicaRequest
func ChatGPTToMonica(cfg *config.Config, chatReq openai.ChatCompletionRequest) (*MonicaRequest, error) {
	if len(chatReq.Messages) == 0 {
		return nil, fmt.Errorf("empty messages")
	}

	// 生成会话ID
	conversationID := fmt.Sprintf("conv:%s", uuid.New().String())

	// 转换消息

	// 设置默认欢迎消息头，不加上就有几率去掉问题最后的十几个token，不清楚是不是bug
	defaultItem := Item{
		ItemID:         fmt.Sprintf("msg:%s", uuid.New().String()),
		ConversationID: conversationID,
		ItemType:       "reply",
		Data:           ItemContent{Type: "text", Content: "__RENDER_BOT_WELCOME_MSG__"},
	}
	var items = make([]Item, 1, len(chatReq.Messages))
	items[0] = defaultItem
	preItemID := defaultItem.ItemID

	for _, msg := range chatReq.Messages {
		if msg.Role == "system" {
			// monica不支持设置prompt，所以直接跳过
			continue
		}
		var msgContext string
		var attachments []AttachmentRequest

		// 处理多内容消息 (当前主要支持图片，为将来扩展做准备)
		if len(msg.MultiContent) > 0 {
			for _, content := range msg.MultiContent {
				switch content.Type {
				case "text":
					msgContext = content.Text
					
					// 检测文本内容中的文件信息
					if strings.Contains(msgContext, "[file name]:") && strings.Contains(msgContext, "[file content begin]") {
						// 提取文件名和文件内容
						fileName, fileContent, found := extractFileFromText(msgContext)
						if found {
							attachments = append(attachments, AttachmentRequest{
								Type:     "document",
								Data:     fileContent,
								FileName: fileName,
								MimeType: "text/markdown",
							})
							// 清空文本内容，避免重复
							msgContext = ""
						}
					}
					
				case "image_url":
					// 图片处理 (当前支持)
					attachments = append(attachments, AttachmentRequest{
						Type: "image_url",
						Data: content.ImageURL.URL,
					})
				// TODO: 未来支持更多文件类型
				// case "document", "file":
				//     // 文档/文件处理
				// case "audio":
				//     // 音频处理
				default:
					logger.Warn("不支持的内容类型", zap.String("type", string(content.Type)))
				}
			}
		}

		itemID := fmt.Sprintf("msg:%s", uuid.New().String())
		itemType := "question"
		if msg.Role == "assistant" {
			itemType = "reply"
		}

		var content ItemContent

		// 处理附件上传
		if len(attachments) > 0 {
			// 创建带超时的上下文
			uploadCtx, cancel := context.WithTimeout(context.Background(), FileUploadTimeout)
			defer cancel()

			// 统计上传成功和失败数量
			var successCount, failureCount int64

			// 并发上传所有类型的文件
			uploadResults := lop.Map(attachments, func(attachment AttachmentRequest, _ int) *FileInfo {
				// 确定文件来源类型
				var source FileUploadSource
				var fileData interface{}
				
				if strings.HasPrefix(attachment.Data, "data:") {
					source = SourceBase64
					fileData = attachment.Data
				} else if strings.HasPrefix(attachment.Data, "http") {
					source = SourceURL
					fileData = attachment.Data
				} else {
					// 对于纯文本内容，使用字节数据
					source = SourceBytes
					fileData = []byte(attachment.Data)
				}

				// 创建上传请求
				fileReq := &UniversalFileUploadRequest{
					Data:      fileData,
					Source:    source,
					FileName:  attachment.FileName,
					MimeType:  attachment.MimeType,
					ParseFile: true, // 默认启用LLM解析
				}

				f, err := UploadUniversalFile(uploadCtx, cfg, fileReq)
				if err != nil {
					atomic.AddInt64(&failureCount, 1)
					logger.Error("文件上传失败",
						zap.Error(err),
						zap.String("file_type", attachment.Type),
						zap.String("file_name", attachment.FileName),
					)
					return nil
				}

				if f == nil {
					atomic.AddInt64(&failureCount, 1)
					logger.Warn("文件上传返回空结果",
						zap.String("file_name", attachment.FileName))
					return nil
				}

				atomic.AddInt64(&successCount, 1)
				return f
			})

			// 过滤掉失败的上传
			fileInfoList := make([]FileInfo, 0, len(uploadResults))
			for _, result := range uploadResults {
				if result != nil {
					fileInfoList = append(fileInfoList, *result)
				}
			}

			// 记录上传统计信息
			if failureCount > 0 {
				logger.Warn("文件上传完成",
					zap.Int64("success_count", successCount),
					zap.Int64("failure_count", failureCount),
					zap.Int("total_attachments", len(attachments)),
				)
			} else {
				logger.Info("所有文件上传成功",
					zap.Int64("success_count", successCount),
					zap.Int("total_attachments", len(attachments)),
				)
			}

			content = ItemContent{
				Type:        "file_with_text",
				Content:     msgContext,
				FileInfos:   fileInfoList,
				IsIncognito: true,
			}
		} else {
			content = ItemContent{
				Type:        "text",
				Content:     msg.Content,
				IsIncognito: true,
			}
		}

		item := Item{
			ConversationID: conversationID,
			ItemID:         itemID,
			ParentItemID:   preItemID,
			ItemType:       itemType,
			Data:           content,
		}
		items = append(items, item)
		preItemID = itemID
	}

	// 构建请求
	mReq := &MonicaRequest{
		TaskUID: fmt.Sprintf("task:%s", uuid.New().String()),
		BotUID:  modelToBot(chatReq.Model),
		Data: DataField{
			ConversationID:  conversationID,
			Items:           items,
			PreParentItemID: preItemID,
			TriggerBy:       "auto",
			IsIncognito:     true,
			UseModel:        "", //TODO 好像写啥都没影响
			UseNewMemory:    false,
		},
		Language: "auto",
		TaskType: "chat",
	}

	// indent, err := json.MarshalIndent(mReq, "", "  ")
	// if err != nil {
	// 	return nil, err
	// }
	// log.Printf("send: \n%s\n", indent)

	return mReq, nil
}

// ChatGPTToCustomBot 转换ChatGPT请求到Custom Bot请求
func ChatGPTToCustomBot(cfg *config.Config, chatReq openai.ChatCompletionRequest, botUID string) (*CustomBotRequest, error) {
	if len(chatReq.Messages) == 0 {
		return nil, fmt.Errorf("empty messages")
	}

	// 生成会话ID
	conversationID := fmt.Sprintf("conv:%s", uuid.New().String())

	// 设置默认欢迎消息
	defaultItem := Item{
		ItemID:         fmt.Sprintf("msg:%s", uuid.New().String()),
		ConversationID: conversationID,
		ItemType:       "reply",
		Data:           ItemContent{Type: "text", Content: "__RENDER_BOT_WELCOME_MSG__"},
	}
	var items = make([]Item, 1, len(chatReq.Messages))
	items[0] = defaultItem
	preItemID := defaultItem.ItemID

	// 提取system消息作为prompt
	var systemPrompt string
	// 转换消息
	for _, msg := range chatReq.Messages {
		if msg.Role == "system" {
			// 将system消息作为prompt
			systemPrompt = msg.Content
			continue
		}

		var msgContext string
		var imgUrl []*openai.ChatMessageImageURL
		if len(msg.MultiContent) > 0 {
			for _, content := range msg.MultiContent {
				switch content.Type {
				case "text":
					msgContext = content.Text
				case "image_url":
					imgUrl = append(imgUrl, content.ImageURL)
				}
			}
		}

		itemID := fmt.Sprintf("msg:%s", uuid.New().String())
		itemType := "question"
		if msg.Role == "assistant" {
			itemType = "reply"
		}

		var content ItemContent
		if len(imgUrl) > 0 {
			// 处理图片上传
			uploadCtx, cancel := context.WithTimeout(context.Background(), FileUploadTimeout)
			defer cancel()

			var successCount, failureCount int64
			uploadResults := lop.Map(imgUrl, func(item *openai.ChatMessageImageURL, _ int) *FileInfo {
				f, err := UploadBase64Image(uploadCtx, cfg, item.URL)
				if err != nil {
					atomic.AddInt64(&failureCount, 1)
					logger.Error("上传图片失败",
						zap.Error(err),
						zap.String("image_url", item.URL),
					)
					return nil
				}
				atomic.AddInt64(&successCount, 1)
				return f
			})

			fileIfoList := make([]FileInfo, 0, len(uploadResults))
			for _, result := range uploadResults {
				if result != nil {
					fileIfoList = append(fileIfoList, *result)
				}
			}

			content = ItemContent{
				Type:        "file_with_text",
				Content:     msgContext,
				FileInfos:   fileIfoList,
				IsIncognito: false,
			}
		} else {
			content = ItemContent{
				Type:        "text",
				Content:     msg.Content,
				IsIncognito: false,
			}
		}

		item := Item{
			ConversationID: conversationID,
			ItemID:         itemID,
			ParentItemID:   preItemID,
			ItemType:       itemType,
			Data:           content,
		}
		items = append(items, item)
		preItemID = itemID
	}

	// 生成reply ID
	preGeneratedReplyID := fmt.Sprintf("msg:%s", uuid.New().String())

	// 构建请求
	customBotReq := &CustomBotRequest{
		TaskUID: fmt.Sprintf("task:%s", uuid.New().String()),
		BotUID:  botUID,
		Data: CustomBotData{
			ConversationID:      conversationID,
			Items:               items,
			PreGeneratedReplyID: preGeneratedReplyID,
			PreParentItemID:     preItemID,
			Origin:              fmt.Sprintf("https://monica.im/bots/%s", botUID),
			OriginPageTitle:     "Monica Bot Test",
			TriggerBy:           "auto",
			UseModel:            chatReq.Model, // 使用请求中的模型
			IsIncognito:         false,
			UseNewMemory:        true,
			UseMemorySuggestion: true,
		},
		Language: "auto",
		Locale:   "zh_CN",
		TaskType: "chat",
		BotData: BotData{
			Description:    "Test Bot",
			LogoURL:        "https://assets.monica.im/assets/img/default_bot_icon.jpg",
			Name:           "Test Bot",
			Classification: "custom",
			Prompt:         systemPrompt,
			Type:           "custom_bot",
			UID:            botUID,
			ExampleList:    []interface{}{},
			ToolData: BotToolData{
				KnowledgeList:    []interface{}{},
				UserSkillList:    []interface{}{},
				SysSkillList:     []interface{}{},
				UseModel:         chatReq.Model,
				ScheduleTaskList: []interface{}{},
			},
		},
		AIRespLanguage: "Chinese (Simplified)",
	}

	return customBotReq, nil
}

// extractFileFromText 从文本内容中提取文件信息
func extractFileFromText(text string) (fileName, fileContent string, found bool) {
	// 查找文件名
	fileNamePattern := regexp.MustCompile(`\[file name\]:\s*([^\n]+)`)
	fileNameMatch := fileNamePattern.FindStringSubmatch(text)
	if len(fileNameMatch) < 2 {
		return "", "", false
	}
	fileName = strings.TrimSpace(fileNameMatch[1])
	
	// 查找文件内容开始和结束标记
	contentStart := strings.Index(text, "[file content begin]")
	if contentStart == -1 {
		return "", "", false
	}
	contentStart += len("[file content begin]")
	
	contentEnd := strings.Index(text, "[file content end]")
	if contentEnd == -1 {
		return "", "", false
	}
	
	// 提取文件内容
	fileContent = strings.TrimSpace(text[contentStart:contentEnd])
	
	return fileName, fileContent, true
}
