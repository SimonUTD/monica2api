package monica

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"monica-proxy/internal/types"
	"monica-proxy/internal/utils"
	"net/http"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/sashabaranov/go-openai"
)

const (
	sseObject     = "chat.completion.chunk"
	sseFinish     = "[DONE]"
	flushInterval = 100 * time.Millisecond // 刷新间隔
	bufferSize    = 4096                   // 缓冲区大小

	dataPrefix    = "data: "
	dataPrefixLen = len(dataPrefix)
	lineEnd       = "\n\n"
)

// SSEData 用于解析 Monica SSE json
type SSEData struct {
	Text        string      `json:"text"`
	Finished    bool        `json:"finished"`
	AgentStatus AgentStatus `json:"agent_status,omitempty"`
}

type AgentStatus struct {
	UID      string `json:"uid"`
	Type     string `json:"type"`
	Text     string `json:"text"`
	Metadata struct {
		Title           string `json:"title"`
		ReasoningDetail string `json:"reasoning_detail"`
	} `json:"metadata"`
}

var (
	sseDataPool = sync.Pool{
		New: func() any {
			return &SSEData{}
		},
	}
	
	// 字符串构建器池，复用strings.Builder
	stringBuilderPool = sync.Pool{
		New: func() any {
			return &strings.Builder{}
		},
	}
	
	// 缓冲区池，复用字节缓冲区
	bufferPool = sync.Pool{
		New: func() any {
			buf := make([]byte, bufferSize)
			return &buf
		},
	}
)

// processMonicaSSE 处理Monica的SSE数据
type processMonicaSSE struct {
	reader *bufio.Reader
	model  string
	ctx    context.Context
}

// handleSSEData 处理单条SSE数据
type handleSSEData func(*SSEData) error

// processSSEStream 处理SSE流
func (p *processMonicaSSE) processSSEStream(handler handleSSEData) error {
	var line []byte
	var err error
	for {
		// 检查上下文是否已取消
		select {
		case <-p.ctx.Done():
			return p.ctx.Err()
		default:
		}
		
		line, err = p.reader.ReadBytes('\n')
		if err != nil {
			// EOF 和 上下文取消 都是正常结束，不应视为错误
			if err == io.EOF || errors.Is(err, context.Canceled) {
				return nil
			}
			return fmt.Errorf("read error: %w", err)
		}

		// Monica SSE 的行前缀一般是 "data: "
		if len(line) < dataPrefixLen || !bytes.HasPrefix(line, []byte(dataPrefix)) {
			continue
		}

		jsonStr := line[dataPrefixLen : len(line)-1] // 去掉\n
		if len(jsonStr) == 0 {
			continue
		}

		// 如果是 [DONE] 则结束
		if bytes.Equal(jsonStr, []byte(sseFinish)) {
			return nil
		}

		// 从对象池获取一个对象
		sseData := sseDataPool.Get().(*SSEData)
		
		// 解析 JSON
		if err := sonic.Unmarshal(jsonStr, sseData); err != nil {
			// 立即归还对象到池中
			*sseData = SSEData{}
			sseDataPool.Put(sseData)
			return fmt.Errorf("unmarshal error: %w", err)
		}

		// 调用处理函数
		if err := handler(sseData); err != nil {
			// 立即归还对象到池中
			*sseData = SSEData{}
			sseDataPool.Put(sseData)
			return err
		}
		
		// 使用完后立即归还对象到池中
		*sseData = SSEData{}
		sseDataPool.Put(sseData)
	}
}

// CollectMonicaSSEToCompletion 将 Monica SSE 转换为完整的 ChatCompletion 响应
func CollectMonicaSSEToCompletion(model string, r io.Reader) (*openai.ChatCompletionResponse, error) {
	ctx := context.Background()
	
	// 从池中获取字符串构建器
	fullContentBuilder := stringBuilderPool.Get().(*strings.Builder)
	defer func() {
		fullContentBuilder.Reset()
		stringBuilderPool.Put(fullContentBuilder)
	}()
	
	processor := &processMonicaSSE{
		reader: bufio.NewReaderSize(r, bufferSize),
		model:  model,
		ctx:    ctx,
	}

	// 处理SSE数据
	err := processor.processSSEStream(func(sseData *SSEData) error {
		// 如果是 agent_status，跳过
		if sseData.AgentStatus.Type != "" {
			return nil
		}
		// 累积内容
		fullContentBuilder.WriteString(sseData.Text)
		return nil
	})

	if err != nil {
		return nil, err
	}

	// 构造完整的响应
	response := &openai.ChatCompletionResponse{
		ID:      fmt.Sprintf("chatcmpl-%s", utils.RandStringUsingMathRand(29)),
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   model,
		Choices: []openai.ChatCompletionChoice{
			{
				Index: 0,
				Message: openai.ChatCompletionMessage{
					Role:    "assistant",
					Content: fullContentBuilder.String(),
				},
				FinishReason: "stop",
			},
		},
		Usage: openai.Usage{
			// Monica API 不提供 token 使用信息，这里暂时填 0
			PromptTokens:     0,
			CompletionTokens: 0,
			TotalTokens:      0,
		},
	}

	return response, nil
}

// StreamMonicaSSEToClient 将 Monica SSE 转成前端可用的流
func StreamMonicaSSEToClient(model string, w io.Writer, r io.Reader) error {
	ctx := context.Background()
	writer := bufio.NewWriterSize(w, bufferSize)
	defer writer.Flush()

	chatId := utils.RandStringUsingMathRand(29)
	now := time.Now().Unix()
	fingerprint := utils.RandStringUsingMathRand(10)

	// 创建一个定时刷新的 ticker
	ticker := time.NewTicker(flushInterval)
	defer ticker.Stop()

	// 创建一个 done channel 用于清理
	done := make(chan struct{})
	defer close(done)

	// 启动一个 goroutine 定期刷新缓冲区
	go func() {
		for {
			select {
			case <-ticker.C:
				if f, ok := w.(http.Flusher); ok {
					writer.Flush()
					f.Flush()
				}
			case <-done:
				return
			}
		}
	}()

	processor := &processMonicaSSE{
		reader: bufio.NewReaderSize(r, bufferSize),
		model:  model,
		ctx:    ctx,
	}

	var thinkFlag bool
	return processor.processSSEStream(func(sseData *SSEData) error {
		var sseMsg types.ChatCompletionStreamResponse
		switch {
		case sseData.Finished:
			sseMsg = types.ChatCompletionStreamResponse{
				ID:      "chatcmpl-" + chatId,
				Object:  sseObject,
				Created: now,
				Model:   model,
				Choices: []types.ChatCompletionStreamChoice{
					{
						Index: 0,
						Delta: openai.ChatCompletionStreamChoiceDelta{
							Role: openai.ChatMessageRoleAssistant,
						},
						FinishReason: openai.FinishReasonStop,
					},
				},
			}
		case sseData.AgentStatus.Type == "thinking":
			thinkFlag = true
			sseMsg = types.ChatCompletionStreamResponse{
				ID:                "chatcmpl-" + chatId,
				Object:            sseObject,
				SystemFingerprint: fingerprint,
				Created:           now,
				Model:             model,
				Choices: []types.ChatCompletionStreamChoice{
					{
						Index: 0,
						Delta: openai.ChatCompletionStreamChoiceDelta{
							Role:    openai.ChatMessageRoleAssistant,
							Content: `<think>`,
						},
						FinishReason: openai.FinishReasonNull,
					},
				},
			}
		case sseData.AgentStatus.Type == "thinking_detail_stream":
			sseMsg = types.ChatCompletionStreamResponse{
				ID:                "chatcmpl-" + chatId,
				Object:            sseObject,
				SystemFingerprint: fingerprint,
				Created:           now,
				Model:             model,
				Choices: []types.ChatCompletionStreamChoice{
					{
						Index: 0,
						Delta: openai.ChatCompletionStreamChoiceDelta{
							Role:    openai.ChatMessageRoleAssistant,
							Content: sseData.AgentStatus.Metadata.ReasoningDetail,
						},
						FinishReason: openai.FinishReasonNull,
					},
				},
			}
		default:
			if thinkFlag {
				sseData.Text = "</think>" + sseData.Text
				thinkFlag = false
			}
			sseMsg = types.ChatCompletionStreamResponse{
				ID:                "chatcmpl-" + chatId,
				Object:            sseObject,
				SystemFingerprint: fingerprint,
				Created:           now,
				Model:             model,
				Choices: []types.ChatCompletionStreamChoice{
					{
						Index: 0,
						Delta: openai.ChatCompletionStreamChoiceDelta{
							Role:    openai.ChatMessageRoleAssistant,
							Content: sseData.Text,
						},
						FinishReason: openai.FinishReasonNull,
					},
				},
			}
		}

		// 从池中获取字符串构建器
		sb := stringBuilderPool.Get().(*strings.Builder)
		sb.WriteString("data: ")
		sendLine, _ := sonic.MarshalString(sseMsg)
		sb.WriteString(sendLine)
		sb.WriteString("\n\n")

		// 写入缓冲区
		if _, err := writer.WriteString(sb.String()); err != nil {
			// 归还字符串构建器到池中
			sb.Reset()
			stringBuilderPool.Put(sb)
			return fmt.Errorf("write error: %w", err)
		}
		
		// 使用完毕，归还字符串构建器到池中
		sb.Reset()
		stringBuilderPool.Put(sb)

		// 如果发现 finished=true，就可以结束
		if sseData.Finished {
			writer.WriteString(dataPrefix)
			writer.WriteString(sseFinish)
			writer.WriteString(lineEnd)
			writer.Flush()
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
			return nil
		}

		sseData.AgentStatus.Type = ""
		sseData.Finished = false
		return nil
	})
}
