package service

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"mime/multipart"
	"monica-proxy/internal/config"
	"monica-proxy/internal/errors"
	"monica-proxy/internal/logger"
	"monica-proxy/internal/types"
	"net/http"
	"path/filepath"
	"time"

	"go.uber.org/zap"
)

// FileService 文件服务接口
type FileService interface {
	// UploadFile 上传文件
	UploadFile(ctx context.Context, fileHeader *multipart.FileHeader, purpose string) (*types.FileObject, error)
	// GetFile 获取文件信息
	GetFile(ctx context.Context, fileID string) (*types.FileObject, error)
	// ListFiles 列出文件
	ListFiles(ctx context.Context) ([]*types.FileObject, error)
	// DeleteFile 删除文件
	DeleteFile(ctx context.Context, fileID string) error
}

// fileService 文件服务实现
type fileService struct {
	config *config.Config
}

// NewFileService 创建文件服务实例
func NewFileService(cfg *config.Config) FileService {
	return &fileService{
		config: cfg,
	}
}

// UploadFile 上传文件
func (s *fileService) UploadFile(ctx context.Context, fileHeader *multipart.FileHeader, purpose string) (*types.FileObject, error) {
	startTime := time.Now()
	requestID := fmt.Sprintf("upload-%d", startTime.UnixNano())

	// 验证文件大小
	if fileHeader.Size > types.MaxFileSize {
		if s.config.Logging.EnableRequestLog {
			logger.Error("文件大小验证失败",
				zap.String("request_id", requestID),
				zap.String("operation", "file_upload"),
				zap.String("filename", fileHeader.Filename),
				zap.Int64("file_size", fileHeader.Size),
				zap.Int64("max_size", types.MaxFileSize),
				zap.Duration("duration", time.Since(startTime)),
			)
		}
		return nil, errors.NewBadRequestError(
			fmt.Sprintf("文件大小超出限制: %d bytes > %d bytes", fileHeader.Size, types.MaxFileSize),
			nil,
		)
	}

	// 打开文件
	file, err := fileHeader.Open()
	if err != nil {
		if s.config.Logging.EnableRequestLog {
			logger.Error("无法打开上传的文件",
				zap.String("request_id", requestID),
				zap.String("operation", "file_upload"),
				zap.String("filename", fileHeader.Filename),
				zap.Error(err),
				zap.Duration("duration", time.Since(startTime)),
			)
		}
		return nil, errors.NewInternalError(err)
	}
	defer file.Close()

	// 读取文件内容
	fileData, err := io.ReadAll(file)
	if err != nil {
		if s.config.Logging.EnableRequestLog {
			logger.Error("无法读取文件内容",
				zap.String("request_id", requestID),
				zap.String("operation", "file_upload"),
				zap.String("filename", fileHeader.Filename),
				zap.Error(err),
				zap.Duration("duration", time.Since(startTime)),
			)
		}
		return nil, errors.NewInternalError(err)
	}

	// 检测MIME类型
	mimeType := http.DetectContentType(fileData)
	if mimeType == "" {
		// 尝试从文件扩展名推断
		ext := filepath.Ext(fileHeader.Filename)
		mimeType = getMimeTypeFromExtension(ext)
	}

	if s.config.Logging.EnableRequestLog {
		logger.Info("开始上传文件",
			zap.String("request_id", requestID),
			zap.String("operation", "file_upload"),
			zap.String("filename", fileHeader.Filename),
			zap.String("mime_type", mimeType),
			zap.Int64("size", fileHeader.Size),
			zap.String("purpose", purpose),
			zap.String("file_hash", hashFileData(fileData)),
		)
	}

	// 创建上传请求
	uploadReq := &types.UniversalFileUploadRequest{
		Data:      fileData,
		Source:    types.SourceBytes,
		FileName:  fileHeader.Filename,
		MimeType:  mimeType,
		ParseFile: shouldParseFile(purpose, mimeType),
	}

	// 上传文件到Monica
	fileInfo, err := types.UploadUniversalFile(ctx, s.config, uploadReq)
	
	duration := time.Since(startTime)
	
	if err != nil {
		if s.config.Logging.EnableRequestLog {
			logger.Error("上传文件到Monica失败",
				zap.String("request_id", requestID),
				zap.String("operation", "file_upload"),
				zap.String("filename", fileHeader.Filename),
				zap.String("file_uid", uploadReq.FileName),
				zap.Error(err),
				zap.Duration("duration", duration),
			)
		}
		return nil, errors.NewInternalError(err)
	}

	// 转换为OpenAI兼容的文件对象
	fileObject := &types.FileObject{
		ID:        fileInfo.FileUID,
		Object:    "file",
		Bytes:     int(fileInfo.FileSize),
		CreatedAt: time.Now().Unix(),
		Filename:  fileInfo.FileName,
		Purpose:   purpose,
		Status:    "processed",
		StatusDetails: map[string]interface{}{
			"tokens":        fileInfo.FileTokens,
			"chunks":        fileInfo.FileChunks,
			"upload_duration": duration.String(),
		},
	}

	if s.config.Logging.EnableRequestLog {
		logger.Info("文件上传成功",
			zap.String("request_id", requestID),
			zap.String("operation", "file_upload"),
			zap.String("file_id", fileObject.ID),
			zap.String("filename", fileObject.Filename),
			zap.String("purpose", purpose),
			zap.Int64("tokens", fileInfo.FileTokens),
			zap.Int64("chunks", fileInfo.FileChunks),
			zap.Duration("duration", duration),
			zap.Any("status_details", fileObject.StatusDetails),
		)
	}

	return fileObject, nil
}

// GetFile 获取文件信息
func (s *fileService) GetFile(ctx context.Context, fileID string) (*types.FileObject, error) {
	startTime := time.Now()
	requestID := fmt.Sprintf("getfile-%d", startTime.UnixNano())

	if s.config.Logging.EnableRequestLog {
		logger.Info("获取文件信息请求",
			zap.String("request_id", requestID),
			zap.String("operation", "get_file"),
			zap.String("file_id", fileID),
		)
	}

	// TODO: 实现从Monica获取文件信息的逻辑
	// 目前返回一个基本的文件对象
	fileObject := &types.FileObject{
		ID:     fileID,
		Object: "file",
		Status: "processed",
	}

	duration := time.Since(startTime)

	if s.config.Logging.EnableRequestLog {
		logger.Info("获取文件信息完成",
			zap.String("request_id", requestID),
			zap.String("operation", "get_file"),
			zap.String("file_id", fileID),
			zap.Duration("duration", duration),
			zap.Any("file_object", fileObject),
		)
	}

	return fileObject, nil
}

// ListFiles 列出文件
func (s *fileService) ListFiles(ctx context.Context) ([]*types.FileObject, error) {
	startTime := time.Now()
	requestID := fmt.Sprintf("listfiles-%d", startTime.UnixNano())

	if s.config.Logging.EnableRequestLog {
		logger.Info("列出文件请求",
			zap.String("request_id", requestID),
			zap.String("operation", "list_files"),
		)
	}

	// TODO: 实现从Monica获取文件列表的逻辑
	// 目前返回空列表
	files := []*types.FileObject{}

	duration := time.Since(startTime)

	if s.config.Logging.EnableRequestLog {
		logger.Info("列出文件完成",
			zap.String("request_id", requestID),
			zap.String("operation", "list_files"),
			zap.Duration("duration", duration),
			zap.Int("file_count", len(files)),
		)
	}

	return files, nil
}

// DeleteFile 删除文件
func (s *fileService) DeleteFile(ctx context.Context, fileID string) error {
	startTime := time.Now()
	requestID := fmt.Sprintf("deletefile-%d", startTime.UnixNano())

	if s.config.Logging.EnableRequestLog {
		logger.Info("文件删除请求",
			zap.String("request_id", requestID),
			zap.String("operation", "delete_file"),
			zap.String("file_id", fileID),
		)
	}

	// TODO: 实现从Monica删除文件的逻辑
	// 目前返回成功
	
	duration := time.Since(startTime)

	if s.config.Logging.EnableRequestLog {
		logger.Info("文件删除完成",
			zap.String("request_id", requestID),
			zap.String("operation", "delete_file"),
			zap.String("file_id", fileID),
			zap.Duration("duration", duration),
		)
	}

	return nil
}

// shouldParseFile 判断是否需要LLM解析文件内容
func shouldParseFile(purpose, mimeType string) bool {
	// 根据用途和文件类型决定是否需要解析
	switch purpose {
	case "assistants", "vision", "batch":
		return true
	case "fine-tune":
		return false
	default:
		// 对于文档类型，默认解析
		typeInfo, exists := types.SupportedFileTypes[mimeType]
		if exists && (typeInfo.Category == "document" || typeInfo.Category == "text") {
			return true
		}
		return false
	}
}

// getMimeTypeFromExtension 从文件扩展名获取MIME类型
func getMimeTypeFromExtension(ext string) string {
	switch ext {
	case ".pdf":
		return "application/pdf"
	case ".doc":
		return "application/msword"
	case ".docx":
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	case ".xls":
		return "application/vnd.ms-excel"
	case ".xlsx":
		return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	case ".ppt":
		return "application/vnd.ms-powerpoint"
	case ".pptx":
		return "application/vnd.openxmlformats-officedocument.presentationml.presentation"
	case ".txt":
		return "text/plain"
	case ".md":
		return "text/markdown"
	case ".csv":
		return "text/csv"
	case ".json":
		return "application/json"
	case ".xml":
		return "application/xml"
	case ".js":
		return "text/javascript"
	case ".html":
		return "text/html"
	case ".css":
		return "text/css"
	case ".py":
		return "text/x-python"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".mp3":
		return "audio/mpeg"
	case ".wav":
		return "audio/wav"
	case ".ogg":
		return "audio/ogg"
	case ".m4a":
		return "audio/mp4"
	case ".mp4":
		return "video/mp4"
	case ".avi":
		return "video/avi"
	case ".mov":
		return "video/mov"
	default:
		return "application/octet-stream"
	}
}

// hashFileData 计算文件数据的MD5哈希值
func hashFileData(data []byte) string {
	hash := md5.Sum(data)
	return hex.EncodeToString(hash[:])
}
