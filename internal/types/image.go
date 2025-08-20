package types

import (
	"context"
	"fmt"
	"monica-proxy/internal/config"
	"monica-proxy/internal/utils"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/cespare/xxhash/v2"
	"github.com/google/uuid"
)

const MaxFileSize = 10 * 1024 * 1024 // 10MB

var imageCache sync.Map

// sampleAndHash 对base64字符串进行采样并计算xxHash
func sampleAndHash(data string) string {
	// 如果数据长度小于1024，直接计算整个字符串的哈希
	if len(data) <= 1024 {
		return fmt.Sprintf("%x", xxhash.Sum64String(data))
	}

	// 采样策略：
	// 1. 取前256字节
	// 2. 取中间256字节
	// 3. 取最后256字节
	var samples []string
	samples = append(samples, data[:256])
	mid := len(data) / 2
	samples = append(samples, data[mid-128:mid+128])
	samples = append(samples, data[len(data)-256:])

	// 将采样数据拼接后计算哈希
	return fmt.Sprintf("%x", xxhash.Sum64String(strings.Join(samples, "")))
}

// UploadBase64Image 上传base64编码的图片到Monica
func UploadBase64Image(ctx context.Context, cfg *config.Config, base64Data string) (*FileInfo, error) {
	// 1. 生成缓存key
	cacheKey := sampleAndHash(base64Data)

	// 2. 检查缓存
	if value, exists := imageCache.Load(cacheKey); exists {
		return value.(*FileInfo), nil
	}

	// 3. 解析base64数据
	// 移除 "data:image/png;base64," 这样的前缀
	parts := strings.Split(base64Data, ",")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid base64 image format")
	}

	// 获取图片类型
	mimeType := strings.TrimSuffix(strings.TrimPrefix(parts[0], "data:"), ";base64")
	if !strings.HasPrefix(mimeType, "image/") {
		return nil, fmt.Errorf("invalid image mime type: %s", mimeType)
	}

	// 解码base64数据
	imageData, err := utils.Base64Decode(parts[1])
	if err != nil {
		return nil, fmt.Errorf("decode base64 failed: %v", err)
	}

	// 4. 验证图片格式和大小
	fileInfo, err := validateImageBytes(imageData, mimeType)
	if err != nil {
		return nil, fmt.Errorf("validate image failed: %v", err)
	}
	// log.Printf("file info: %+v", fileInfo)

	// 5. 获取预签名URL
	preSignReq := &PreSignRequest{
		FilenameList: []string{fileInfo.FileName},
		Module:       ImageModule,
		Location:     ImageLocation,
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
	// log.Printf("preSign info: %+v", preSignResp)

	// 6. 上传图片数据
	_, err = utils.RestyDefaultClient.R().
		SetContext(ctx).
		SetHeader("Content-Type", fileInfo.FileType).
		SetBody(imageData).
		Put(preSignResp.Data.PreSignURLList[0])

	if err != nil {
		return nil, fmt.Errorf("upload file failed: %v", err)
	}

	// 7. 创建文件对象
	fileInfo.ObjectURL = preSignResp.Data.ObjectURLList[0]
	uploadReq := &FileUploadRequest{
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
		return nil, fmt.Errorf("create file object failed: %v", err)
	}
	// log.Printf("uploadResp: %+v", uploadResp)
	if len(uploadResp.Data.Items) > 0 {
		fileInfo.FileName = uploadResp.Data.Items[0].FileName
		fileInfo.FileType = uploadResp.Data.Items[0].FileType
		fileInfo.FileSize = uploadResp.Data.Items[0].FileSize
		fileInfo.FileUID = uploadResp.Data.Items[0].FileUID
		fileInfo.FileExt = uploadResp.Data.Items[0].FileType
		fileInfo.FileTokens = uploadResp.Data.Items[0].FileTokens
		fileInfo.FileChunks = uploadResp.Data.Items[0].FileChunks
	}

	fileInfo.UseFullText = true
	fileInfo.FileURL = preSignResp.Data.CDNURLList[0]

	// 8. 获取文件llm读取结果知道有返回
	var batchResp FileBatchGetResponse
	reqMap := make(map[string][]string)
	reqMap["file_uids"] = []string{fileInfo.FileUID}
	var retryCount = 1
	for {
		if retryCount > 5 {
			return nil, fmt.Errorf("retry limit exceeded")
		}
		_, err = utils.RestyDefaultClient.R().
			SetContext(ctx).
			SetHeader("cookie", cfg.Monica.Cookie).
			SetBody(reqMap).
			SetResult(&batchResp).
			Post(FileGetURL)
		if err != nil {
			return nil, fmt.Errorf("batch get file failed: %v", err)
		}
		if len(batchResp.Data.Items) > 0 && batchResp.Data.Items[0].FileChunks > 0 {
			break
		} else {
			retryCount++
		}
		time.Sleep(1 * time.Second)
	}
	fileInfo.FileChunks = batchResp.Data.Items[0].FileChunks
	fileInfo.FileTokens = batchResp.Data.Items[0].FileTokens
	fileInfo.URL = ""
	fileInfo.ObjectURL = ""

	// 9. 保存到缓存
	imageCache.Store(cacheKey, fileInfo)

	return fileInfo, nil
}

// validateImageBytes 验证图片字节数据的格式和大小
func validateImageBytes(imageData []byte, mimeType string) (*FileInfo, error) {
	if len(imageData) > MaxFileSize {
		return nil, fmt.Errorf("file size exceeds limit: %d > %d", len(imageData), MaxFileSize)
	}

	contentType := http.DetectContentType(imageData)
	if !SupportedImageTypes[contentType] {
		return nil, fmt.Errorf("unsupported image type: %s", contentType)
	}

	// 根据MIME类型生成文件扩展名
	ext := ".png"
	switch mimeType {
	case "image/jpeg":
		ext = ".jpg"
	case "image/gif":
		ext = ".gif"
	case "image/webp":
		ext = ".webp"
	}

	fileName := fmt.Sprintf("%s%s", uuid.New().String(), ext)

	return &FileInfo{
		FileName: fileName,
		FileSize: int64(len(imageData)),
		FileType: contentType,
	}, nil
}
