package types

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"monica-proxy/internal/config"
	"monica-proxy/internal/logger"
	"monica-proxy/internal/utils"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/cespare/xxhash/v2"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

var (
	fileCache sync.Map
)

// FileUploadSource 文件上传来源类型
type FileUploadSource int

const (
	SourceBase64 FileUploadSource = iota // Base64编码
	SourceURL                            // 网络URL
	SourceBytes                          // 字节数据
)

// UniversalFileUploadRequest 通用文件上传请求
type UniversalFileUploadRequest struct {
	Data      interface{}      `json:"data"`       // 文件数据 (base64 string, URL string, or []byte)
	Source    FileUploadSource `json:"source"`     // 数据来源类型
	FileName  string           `json:"file_name"`  // 可选的文件名
	MimeType  string           `json:"mime_type"`  // 可选的MIME类型
	ParseFile bool             `json:"parse_file"` // 是否需要LLM解析文件内容
}

// UploadUniversalFile 通用文件上传函数 - 支持所有OpenAI兼容的文件类型
func UploadUniversalFile(ctx context.Context, cfg *config.Config, req *UniversalFileUploadRequest) (*FileInfo, error) {
	// 1. 预处理文件数据
	fileData, fileName, mimeType, err := preprocessFileData(req)
	if err != nil {
		return nil, fmt.Errorf("preprocess file data failed: %v", err)
	}

	// 2. 生成缓存key
	cacheKey := generateFileCacheKey(fileData, fileName, mimeType)

	// 3. 检查缓存
	if value, exists := fileCache.Load(cacheKey); exists {
		logger.Debug("File found in cache", zap.String("cache_key", cacheKey))
		return value.(*FileInfo), nil
	}

	// 4. 验证文件格式和大小
	fileInfo, err := validateFileData(fileData, fileName, mimeType)
	if err != nil {
		return nil, fmt.Errorf("validate file failed: %v", err)
	}

	logger.Info("Uploading file to Monica",
		zap.String("file_name", fileInfo.FileName),
		zap.String("file_type", fileInfo.FileType),
		zap.Int64("file_size", fileInfo.FileSize),
	)

	// 5. 获取预签名URL (对应第一个接口)
	preSignReq := &PreSignRequest{
		FilenameList: []string{fileInfo.FileName},
		Module:       FileModule,
		Location:     FileLocation,
		ObjID:        uuid.New().String(),
	}

	var preSignResp PreSignResponse
	_, err = utils.RestyDefaultClient.R().
		SetContext(ctx).
		SetHeader("cookie", cfg.Monica.Cookie).
		SetBody(preSignReq).
		SetResult(&preSignResp).
		Post(PreSignURL)

	if err != nil {
		return nil, fmt.Errorf("get pre-sign url failed: %v", err)
	}

	if len(preSignResp.Data.PreSignURLList) == 0 || len(preSignResp.Data.ObjectURLList) == 0 {
		return nil, fmt.Errorf("no pre-sign url or object url returned")
	}

	// 6. 上传文件数据到S3
	_, err = utils.RestyDefaultClient.R().
		SetContext(ctx).
		SetHeader("Content-Type", fileInfo.FileType).
		SetBody(fileData).
		Put(preSignResp.Data.PreSignURLList[0])

	if err != nil {
		return nil, fmt.Errorf("upload file to S3 failed: %v", err)
	}

	// 7. 创建LLM文件对象 (对应第二个接口)
	fileInfo.ObjectURL = preSignResp.Data.ObjectURLList[0]
	fileInfo.Parse = req.ParseFile

	uploadReq := &MonicaFileUploadRequest{
		Data: []FileInfo{*fileInfo},
	}

	var uploadResp FileUploadResponse
	_, err = utils.RestyDefaultClient.R().
		SetContext(ctx).
		SetHeader("cookie", cfg.Monica.Cookie).
		SetBody(uploadReq).
		SetResult(&uploadResp).
		Post(FileUploadURL)

	if err != nil {
		return nil, fmt.Errorf("create LLM file object failed: %v", err)
	}

	if len(uploadResp.Data.Items) > 0 {
		item := uploadResp.Data.Items[0]
		fileInfo.FileName = item.FileName
		fileInfo.FileType = item.FileType
		fileInfo.FileSize = item.FileSize
		fileInfo.FileUID = item.FileUID
		fileInfo.FileExt = item.FileType
		fileInfo.FileTokens = item.FileTokens
		fileInfo.FileChunks = item.FileChunks
	}

	fileInfo.UseFullText = true
	fileInfo.FileURL = preSignResp.Data.CDNURLList[0]

	// 8. 等待LLM处理完成 (对应第三个接口)
	if req.ParseFile {
		err = waitForFileProcessing(ctx, cfg, fileInfo.FileUID)
		if err != nil {
			return nil, fmt.Errorf("wait for file processing failed: %v", err)
		}

		// 重新获取处理后的文件信息
		processedInfo, err := getProcessedFileInfo(ctx, cfg, fileInfo.FileUID)
		if err != nil {
			return nil, fmt.Errorf("get processed file info failed: %v", err)
		}

		fileInfo.FileTokens = processedInfo.FileTokens
		fileInfo.FileChunks = processedInfo.FileChunks
	}

	// 9. 清理敏感信息并缓存
	fileInfo.URL = ""
	fileInfo.ObjectURL = ""

	fileCache.Store(cacheKey, fileInfo)

	logger.Info("File uploaded successfully",
		zap.String("file_uid", fileInfo.FileUID),
		zap.Int64("file_tokens", fileInfo.FileTokens),
		zap.Int64("file_chunks", fileInfo.FileChunks),
	)

	return fileInfo, nil
}

// preprocessFileData 预处理不同来源的文件数据
func preprocessFileData(req *UniversalFileUploadRequest) ([]byte, string, string, error) {
	var fileData []byte
	var fileName, mimeType string
	var err error

	switch req.Source {
	case SourceBase64:
		// 处理Base64数据
		base64Data, ok := req.Data.(string)
		if !ok {
			return nil, "", "", fmt.Errorf("invalid base64 data type")
		}
		fileData, fileName, mimeType, err = parseBase64File(base64Data, req.FileName, req.MimeType)

	case SourceURL:
		// 处理URL数据
		urlData, ok := req.Data.(string)
		if !ok {
			return nil, "", "", fmt.Errorf("invalid URL data type")
		}
		fileData, fileName, mimeType, err = downloadFileFromURL(urlData, req.FileName, req.MimeType)

	case SourceBytes:
		// 处理字节数据
		bytesData, ok := req.Data.([]byte)
		if !ok {
			return nil, "", "", fmt.Errorf("invalid bytes data type")
		}
		fileData = bytesData
		fileName = req.FileName
		mimeType = req.MimeType

		// 如果没有提供MIME类型，尝试检测
		if mimeType == "" {
			mimeType = http.DetectContentType(fileData)
		}

	default:
		return nil, "", "", fmt.Errorf("unsupported source type: %v", req.Source)
	}

	if err != nil {
		return nil, "", "", err
	}

	// 生成默认文件名（如果没有提供）
	if fileName == "" {
		typeInfo, exists := SupportedFileTypes[mimeType]
		if exists {
			fileName = fmt.Sprintf("%s%s", uuid.New().String(), typeInfo.Extension)
		} else {
			fileName = fmt.Sprintf("%s.bin", uuid.New().String())
		}
	}

	return fileData, fileName, mimeType, nil
}

// parseBase64File 解析Base64编码的文件
func parseBase64File(base64Data, fileName, mimeType string) ([]byte, string, string, error) {
	// 处理 "data:mime/type;base64,data" 格式
	if strings.Contains(base64Data, ",") {
		parts := strings.Split(base64Data, ",")
		if len(parts) == 2 {
			// 提取MIME类型
			if mimeType == "" {
				header := parts[0]
				if strings.HasPrefix(header, "data:") && strings.Contains(header, ";base64") {
					mimeType = strings.TrimSuffix(strings.TrimPrefix(header, "data:"), ";base64")
				}
			}
			base64Data = parts[1]
		}
	}

	// 解码Base64数据
	fileData, err := base64.StdEncoding.DecodeString(base64Data)
	if err != nil {
		return nil, "", "", fmt.Errorf("decode base64 failed: %v", err)
	}

	// 检测MIME类型（如果没有提供）
	if mimeType == "" {
		mimeType = http.DetectContentType(fileData)
	}

	// 生成文件名（如果没有提供）
	if fileName == "" {
		typeInfo, exists := SupportedFileTypes[mimeType]
		if exists {
			fileName = fmt.Sprintf("%s%s", uuid.New().String(), typeInfo.Extension)
		}
	}

	return fileData, fileName, mimeType, nil
}

// downloadFileFromURL 从URL下载文件
func downloadFileFromURL(fileURL, fileName, mimeType string) ([]byte, string, string, error) {
	resp, err := http.Get(fileURL)
	if err != nil {
		return nil, "", "", fmt.Errorf("download file from URL failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", "", fmt.Errorf("download failed with status: %d", resp.StatusCode)
	}

	// 读取文件数据
	fileData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", "", fmt.Errorf("read file data failed: %v", err)
	}

	// 从Content-Type头获取MIME类型
	if mimeType == "" {
		mimeType = resp.Header.Get("Content-Type")
		if mimeType == "" {
			mimeType = http.DetectContentType(fileData)
		}
	}

	// 从URL推断文件名
	if fileName == "" {
		fileName = filepath.Base(fileURL)
		if fileName == "." || fileName == "/" {
			typeInfo, exists := SupportedFileTypes[mimeType]
			if exists {
				fileName = fmt.Sprintf("%s%s", uuid.New().String(), typeInfo.Extension)
			}
		}
	}

	return fileData, fileName, mimeType, nil
}

// validateFileData 验证文件数据的格式和大小
func validateFileData(fileData []byte, fileName, mimeType string) (*FileInfo, error) {
	typeInfo, supported := SupportedFileTypes[mimeType]
	if !supported {
		return nil, fmt.Errorf("unsupported file type: %s", mimeType)
	}

	if int64(len(fileData)) > typeInfo.MaxSize {
		return nil, fmt.Errorf("file size exceeds limit: %d > %d bytes", len(fileData), typeInfo.MaxSize)
	}

	// 验证文件内容类型
	detectedType := http.DetectContentType(fileData)
	if !isCompatibleMimeType(detectedType, mimeType) {
		logger.Warn("MIME type mismatch",
			zap.String("provided", mimeType),
			zap.String("detected", detectedType),
			zap.String("file_name", fileName),
		)
	}

	return &FileInfo{
		FileName: fileName,
		FileSize: int64(len(fileData)),
		FileType: mimeType,
		FileExt:  typeInfo.Extension,
	}, nil
}

// isCompatibleMimeType 检查检测到的MIME类型是否与提供的兼容
func isCompatibleMimeType(detected, provided string) bool {
	if detected == provided {
		return true
	}

	// 一些常见的兼容性检查
	compatibleTypes := map[string][]string{
		"application/pdf":  {"application/pdf"},
		"text/plain":       {"text/plain", "application/octet-stream"},
		"text/markdown":    {"text/plain", "text/markdown"},
		"application/json": {"text/plain", "application/json"},
	}

	if compatible, exists := compatibleTypes[provided]; exists {
		for _, c := range compatible {
			if detected == c {
				return true
			}
		}
	}

	return false
}

// waitForFileProcessing 等待文件LLM处理完成
func waitForFileProcessing(ctx context.Context, cfg *config.Config, fileUID string) error {
	const maxRetries = 10
	const retryInterval = 2 * time.Second

	reqMap := map[string][]string{
		"file_uids": {fileUID},
	}

	for i := 0; i < maxRetries; i++ {
		var batchResp FileBatchGetResponse
		_, err := utils.RestyDefaultClient.R().
			SetContext(ctx).
			SetHeader("cookie", cfg.Monica.Cookie).
			SetBody(reqMap).
			SetResult(&batchResp).
			Post(FileGetURL)

		if err != nil {
			return fmt.Errorf("batch get file failed: %v", err)
		}

		if len(batchResp.Data.Items) > 0 {
			item := batchResp.Data.Items[0]

			// index_state: 3 表示处理完成
			if item.IndexState == 3 && item.FileChunks > 0 {
				logger.Info("File processing completed",
					zap.String("file_uid", fileUID),
					zap.Int("index_state", item.IndexState),
					zap.Int64("file_tokens", item.FileTokens),
					zap.Int64("file_chunks", item.FileChunks),
				)
				return nil
			}

			// 检查是否有错误
			if item.ErrorMessage != "" {
				return fmt.Errorf("file processing failed: %s", item.ErrorMessage)
			}

			logger.Debug("File still processing",
				zap.String("file_uid", fileUID),
				zap.Int("index_state", item.IndexState),
				zap.Int("retry", i+1),
			)
		}

		// 等待后重试
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(retryInterval):
			continue
		}
	}

	return fmt.Errorf("file processing timeout after %d retries", maxRetries)
}

// getProcessedFileInfo 获取处理完成的文件信息
func getProcessedFileInfo(ctx context.Context, cfg *config.Config, fileUID string) (*FileBatchGetResponse_Item, error) {
	reqMap := map[string][]string{
		"file_uids": {fileUID},
	}

	var batchResp FileBatchGetResponse
	_, err := utils.RestyDefaultClient.R().
		SetContext(ctx).
		SetHeader("cookie", cfg.Monica.Cookie).
		SetBody(reqMap).
		SetResult(&batchResp).
		Post(FileGetURL)

	if err != nil {
		return nil, fmt.Errorf("get processed file info failed: %v", err)
	}

	if len(batchResp.Data.Items) == 0 {
		return nil, fmt.Errorf("no file info returned")
	}

	item := batchResp.Data.Items[0]
	return &FileBatchGetResponse_Item{
		FileName:      item.FileName,
		FileType:      item.FileType,
		FileSize:      int64(item.FileSize),
		ObjectUrl:     item.ObjectUrl,
		Url:           item.Url,
		FileMetaInfo:  make(map[string]any),
		DriveFileUid:  item.DriveFileUid,
		FileUid:       item.FileUid,
		IndexState:    item.IndexState,
		IndexDesc:     item.IndexDesc,
		ErrorMessage:  item.ErrorMessage,
		FileTokens:    item.FileTokens,
		FileChunks:    item.FileChunks,
		IndexProgress: item.IndexProgress,
	}, nil
}

// generateFileCacheKey 生成文件缓存键
func generateFileCacheKey(fileData []byte, fileName, mimeType string) string {
	// 对文件数据进行采样和哈希
	var samples []string

	dataLen := len(fileData)
	if dataLen <= 1024 {
		// 小文件直接哈希
		samples = append(samples, string(fileData))
	} else {
		// 大文件采样
		samples = append(samples, string(fileData[:256]))
		mid := dataLen / 2
		samples = append(samples, string(fileData[mid-128:mid+128]))
		samples = append(samples, string(fileData[dataLen-256:]))
	}

	// 添加文件元信息
	samples = append(samples, fileName, mimeType)

	return fmt.Sprintf("%x", xxhash.Sum64String(strings.Join(samples, "")))
}

// FileBatchGetResponse_Item 单个文件的批量获取响应项
type FileBatchGetResponse_Item struct {
	FileName      string         `json:"file_name"`
	FileType      string         `json:"file_type"`
	FileSize      int64          `json:"file_size"`
	ObjectUrl     string         `json:"object_url"`
	Url           string         `json:"url"`
	FileMetaInfo  map[string]any `json:"file_meta_info"`
	DriveFileUid  string         `json:"drive_file_uid"`
	FileUid       string         `json:"file_uid"`
	IndexState    int            `json:"index_state"`
	IndexDesc     string         `json:"index_desc"`
	ErrorMessage  string         `json:"error_message"`
	FileTokens    int64          `json:"file_tokens"`
	FileChunks    int64          `json:"file_chunks"`
	IndexProgress int            `json:"index_progress"`
}
