package utils

import (
	"math/rand"
	"strings"
	"sync"
	"time"
)

var (
	randSource = rand.New(rand.NewSource(time.Now().UnixNano()))
	letters    = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

	// 字符串构建器池，用于生成随机字符串
	randStringBuilderPool = sync.Pool{
		New: func() any {
			return &strings.Builder{}
		},
	}
)

// RandStringUsingMathRand 生成指定长度的随机字符串
func RandStringUsingMathRand(n int) string {
	if n <= 0 {
		return ""
	}

	// 从池中获取字符串构建器
	sb := randStringBuilderPool.Get().(*strings.Builder)
	defer func() {
		sb.Reset()
		randStringBuilderPool.Put(sb)
	}()

	// 预分配容量
	sb.Grow(n)

	// 生成随机字符
	for range n {
		sb.WriteRune(letters[randSource.Intn(len(letters))])
	}

	return sb.String()
}
