package errors

import (
	"fmt"
	"net/http"
)

// ErrorCode 定义错误码
type ErrorCode int

const (
	// 系统错误码 (1000-1999)
	ErrInternal ErrorCode = 1000 + iota
	ErrBadRequest
	ErrUnauthorized
	ErrForbidden
	ErrNotFound
	ErrTimeout
	ErrRequestFailed

	// 业务错误码 (2000-2999)
	ErrInvalidInput ErrorCode = 2000 + iota
	ErrInvalidModel
	ErrEmptyMessage
	ErrImageGeneration
	ErrModelMapping
	ErrFileUpload
)

// AppError 应用错误
type AppError struct {
	Code    ErrorCode // 错误码
	Message string    // 错误消息
	Err     error     // 原始错误
	Status  int       // HTTP状态码
}

// Error 实现error接口
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%d] %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("[%d] %s", e.Code, e.Message)
}

// Unwrap 实现errors.Unwrap接口
func (e *AppError) Unwrap() error {
	return e.Err
}

// HTTPResponse 生成HTTP响应
func (e *AppError) HTTPResponse() (int, map[string]interface{}) {
	return e.Status, map[string]interface{}{
		"error": map[string]interface{}{
			"code":    e.Code,
			"message": e.Message,
		},
	}
}

// NewInternalError 创建内部错误
func NewInternalError(err error) *AppError {
	return &AppError{
		Code:    ErrInternal,
		Message: "服务器内部错误",
		Err:     err,
		Status:  http.StatusInternalServerError,
	}
}

// NewBadRequestError 创建请求错误
func NewBadRequestError(message string, err error) *AppError {
	return &AppError{
		Code:    ErrBadRequest,
		Message: message,
		Err:     err,
		Status:  http.StatusBadRequest,
	}
}

// NewUnauthorizedError 创建未授权错误
func NewUnauthorizedError(message string) *AppError {
	return &AppError{
		Code:    ErrUnauthorized,
		Message: message,
		Status:  http.StatusUnauthorized,
	}
}

// NewInvalidInputError 创建无效输入错误
func NewInvalidInputError(message string, err error) *AppError {
	return &AppError{
		Code:    ErrInvalidInput,
		Message: message,
		Err:     err,
		Status:  http.StatusBadRequest,
	}
}

// NewEmptyMessageError 创建空消息错误
func NewEmptyMessageError() *AppError {
	return &AppError{
		Code:    ErrEmptyMessage,
		Message: "消息内容不能为空",
		Status:  http.StatusBadRequest,
	}
}

// NewImageGenerationError 创建图片生成错误
func NewImageGenerationError(err error) *AppError {
	return &AppError{
		Code:    ErrImageGeneration,
		Message: "图片生成失败",
		Err:     err,
		Status:  http.StatusInternalServerError,
	}
}

// NewRequestFailedError 创建请求失败错误
func NewRequestFailedError(message string, err error) *AppError {
	return &AppError{
		Code:    ErrRequestFailed,
		Message: fmt.Sprintf("请求失败: %s", message),
		Err:     err,
		Status:  http.StatusBadGateway,
	}
}

// NewModelMappingError 创建模型映射错误
func NewModelMappingError(model string) *AppError {
	return &AppError{
		Code:    ErrModelMapping,
		Message: fmt.Sprintf("不支持的模型: %s", model),
		Status:  http.StatusBadRequest,
	}
}

// NewFileUploadError 创建文件上传错误
func NewFileUploadError(err error) *AppError {
	return &AppError{
		Code:    ErrFileUpload,
		Message: "文件上传失败",
		Err:     err,
		Status:  http.StatusInternalServerError,
	}
}
