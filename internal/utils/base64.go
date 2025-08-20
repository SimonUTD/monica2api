package utils

import (
	"encoding/base64"
)

// Base64Decode 解码base64字符串为字节数组
func Base64Decode(data string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(data)
}
