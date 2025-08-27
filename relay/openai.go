package relay

import (
	"bufio"
	"bytes"
	"claude-code-relay/common"
	"claude-code-relay/model"
	"claude-code-relay/service"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// Claude API 类型定义
type ClaudeTool struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	InputSchema any    `json:"input_schema"`
}

type ClaudeContentBlock struct {
	Type      string               `json:"type"`
	Text      string               `json:"text,omitempty"`
	Source    *ClaudeContentSource `json:"source,omitempty"`
	ID        string               `json:"id,omitempty"`
	Name      string               `json:"name,omitempty"`
	Input     map[string]any       `json:"input,omitempty"`
	ToolUseID string               `json:"tool_use_id,omitempty"`
	Content   any                  `json:"content,omitempty"`
}

type ClaudeContentSource struct {
	Type      string `json:"type"`
	MediaType string `json:"media_type"`
	Data      string `json:"data"`
}

type ClaudeMessage struct {
	Role    string `json:"role"`
	Content any    `json:"content"`
}

type ClaudeToolChoice struct {
	Type string `json:"type"`
	Name string `json:"name,omitempty"`
}

type ClaudeRequest struct {
	Model         string            `json:"model"`
	Messages      []ClaudeMessage   `json:"messages"`
	System        any               `json:"system,omitempty"`
	MaxTokens     int               `json:"max_tokens"`
	StopSequences []string          `json:"stop_sequences,omitempty"`
	Stream        bool              `json:"stream,omitempty"`
	Temperature   *float64          `json:"temperature,omitempty"`
	TopP          *float64          `json:"top_p,omitempty"`
	TopK          *int              `json:"top_k,omitempty"`
	Tools         []ClaudeTool      `json:"tools,omitempty"`
	ToolChoice    *ClaudeToolChoice `json:"tool_choice,omitempty"`
	Metadata      map[string]any    `json:"metadata,omitempty"`
}

// OpenAI API 类型定义
type OpenAIMessage struct {
	Role       string           `json:"role"`
	Content    any              `json:"content"`
	ToolCalls  []OpenAIToolCall `json:"tool_calls,omitempty"`
	ToolCallID string           `json:"tool_call_id,omitempty"`
}

type OpenAIToolCall struct {
	ID       string             `json:"id"`
	Type     string             `json:"type"`
	Function OpenAIFunctionCall `json:"function"`
}

type OpenAIFunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type OpenAITool struct {
	Type     string            `json:"type"`
	Function OpenAIFunctionDef `json:"function"`
}

type OpenAIFunctionDef struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Parameters  any    `json:"parameters"`
}

type OpenAIToolChoice struct {
	Type     string            `json:"type"`
	Function OpenAIFunctionDef `json:"function"`
}

type OpenAIRequest struct {
	Model       string          `json:"model"`
	Messages    []OpenAIMessage `json:"messages"`
	MaxTokens   *int            `json:"max_tokens,omitempty"`
	Temperature *float64        `json:"temperature,omitempty"`
	TopP        *float64        `json:"top_p,omitempty"`
	Stop        []string        `json:"stop,omitempty"`
	Stream      bool            `json:"stream,omitempty"`
	Tools       []OpenAITool    `json:"tools,omitempty"`
	ToolChoice  any             `json:"tool_choice,omitempty"`
}

// OpenAI 响应类型定义
type OpenAIResponse struct {
	ID      string         `json:"id"`
	Object  string         `json:"object"`
	Created int64          `json:"created"`
	Model   string         `json:"model"`
	Choices []OpenAIChoice `json:"choices"`
	Usage   OpenAIUsage    `json:"usage"`
}

type OpenAIChoice struct {
	Index        int           `json:"index"`
	Message      OpenAIMessage `json:"message"`
	FinishReason string        `json:"finish_reason"`
}

type OpenAIUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// Claude 响应类型定义
type ClaudeResponse struct {
	ID         string               `json:"id"`
	Type       string               `json:"type"`
	Role       string               `json:"role"`
	Model      string               `json:"model"`
	Content    []ClaudeContentBlock `json:"content"`
	StopReason string               `json:"stop_reason"`
	Usage      ClaudeUsage          `json:"usage"`
}

type ClaudeUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

type OpenAITargetConfig struct {
	BaseURL   string
	ModelName string
}

// HandleOpenAIRequest 处理 OpenAI 请求的中转
func HandleOpenAIRequest(c *gin.Context, account *model.Account) {
	// 记录请求开始时间用于计算耗时
	startTime := time.Now()

	// 从上下文中获取API Key信息
	var apiKey *model.ApiKey
	if keyInfo, exists := c.Get("api_key"); exists {
		apiKey = keyInfo.(*model.ApiKey)
	}
	ctx := c.Request.Context()

	// 读取请求体
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": map[string]any{
				"type":    "request_body_error",
				"message": "Failed to read request body: " + err.Error(),
			},
		})
		return
	}

	// 解析Claude请求
	var claudeReq ClaudeRequest
	if err := json.Unmarshal(body, &claudeReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": map[string]any{
				"type":    "json_parse_error",
				"message": "Failed to parse request JSON: " + err.Error(),
			},
		})
		return
	}

	// 直接使用账号配置的请求地址和默认模型
	if account.RequestURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": map[string]any{
				"type":    "configuration_error",
				"message": "账号未配置请求地址",
			},
		})
		return
	}

	targetConfig := &OpenAITargetConfig{
		BaseURL:   account.RequestURL, // 使用账号配置的请求地址
		ModelName: "gpt-4o",           // 默认模型，会被模型映射覆盖
	}

	// 应用模型映射
	mappedModelName := applyModelMapping(claudeReq.Model, account.ModelMapping, targetConfig.ModelName)

	// 转换Claude请求为OpenAI格式
	openaiReq := convertClaudeToOpenAI(claudeReq, mappedModelName)

	// 序列化OpenAI请求
	openaiBody, err := json.Marshal(openaiReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": map[string]any{
				"type":    "json_marshal_error",
				"message": "Failed to marshal OpenAI request: " + err.Error(),
			},
		})
		return
	}

	// 创建OpenAI API请求
	openaiURL := targetConfig.BaseURL + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, "POST", openaiURL, bytes.NewBuffer(openaiBody))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": map[string]any{
				"type":    "internal_server_error",
				"message": "Failed to create request: " + err.Error(),
			},
		})
		return
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+account.SecretKey)

	// 创建HTTP客户端
	httpClientTimeout, _ := time.ParseDuration(os.Getenv("HTTP_CLIENT_TIMEOUT") + "s")
	if httpClientTimeout == 0 {
		httpClientTimeout = 120 * time.Second
	}

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	// 配置代理
	if account.ProxyURI != "" {
		proxyURL, err := url.Parse(account.ProxyURI)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": map[string]any{
					"type":    "proxy_configuration_error",
					"message": "Invalid proxy URI: " + err.Error(),
				},
			})
			return
		}
		transport.Proxy = http.ProxyURL(proxyURL)
	}

	client := &http.Client{
		Timeout:   httpClientTimeout,
		Transport: transport,
	}

	// 发送请求
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("OpenAI API request failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": map[string]any{
				"type":    "network_error",
				"message": "Failed to execute request: " + err.Error(),
			},
		})
		return
	}
	defer common.CloseIO(resp.Body)

	// 检查响应状态
	accountService := service.NewAccountService()
	if resp.StatusCode >= 400 {
		accountService.UpdateAccountStatus(account, resp.StatusCode, nil)
		bodyBytes, _ := io.ReadAll(resp.Body)
		c.Data(resp.StatusCode, "application/json", bodyBytes)
		return
	}

	// 统一使用流式响应处理（传递原始Claude模型名称，用于日志记录）
	handleStreamingResponse(c, resp, claudeReq.Model, claudeReq.Stream, account, apiKey, startTime)
}

// extractSystemMessage 从system字段中提取系统消息文本
// 支持字符串和数组格式的system字段
func extractSystemMessage(systemField any) string {
	if systemField == nil {
		return ""
	}

	switch s := systemField.(type) {
	case string:
		// 直接字符串格式
		return s
	case []any:
		// 数组格式，提取所有text内容
		var textParts []string
		for _, item := range s {
			if itemMap, ok := item.(map[string]any); ok {
				if itemType, exists := itemMap["type"]; exists && itemType == "text" {
					if text, ok := itemMap["text"].(string); ok {
						textParts = append(textParts, text)
					}
				}
			}
		}
		return strings.Join(textParts, "\n")
	default:
		// 其他类型转为JSON字符串
		if jsonBytes, err := json.Marshal(s); err == nil {
			return string(jsonBytes)
		}
		return ""
	}
}

// applyModelMapping 应用模型映射配置
// 格式: claude-haiku-20250303:gpt-4o-mini,claude-sonnet:gpt-4o
// 如果没有找到映射，返回默认的目标模型名称
func applyModelMapping(claudeModel, modelMapping, defaultTargetModel string) string {
	if modelMapping == "" {
		return defaultTargetModel
	}

	// 解析映射配置
	mappings := strings.Split(modelMapping, ",")
	for _, mapping := range mappings {
		mapping = strings.TrimSpace(mapping)
		if mapping == "" {
			continue
		}

		parts := strings.Split(mapping, ":")
		if len(parts) != 2 {
			continue
		}

		sourceModel := strings.TrimSpace(parts[0])
		targetModel := strings.TrimSpace(parts[1])

		// 支持模糊匹配，只要包含关键字就匹配
		if strings.Contains(claudeModel, sourceModel) || claudeModel == sourceModel {
			return targetModel
		}
	}

	// 如果没有找到映射，返回默认目标模型
	return defaultTargetModel
}

// recursivelyCleanSchema 递归清理JSON Schema，使其兼容严格API如Google Gemini
func recursivelyCleanSchema(schema any) any {
	if schema == nil {
		return schema
	}

	switch s := schema.(type) {
	case map[string]any:
		newSchema := make(map[string]any)
		for key, value := range s {
			if key == "$schema" || key == "additionalProperties" {
				continue
			}
			newSchema[key] = recursivelyCleanSchema(value)
		}

		// 处理string类型的format字段
		if newSchema["type"] == "string" && newSchema["format"] != nil {
			format := newSchema["format"].(string)
			supportedFormats := []string{"date-time", "enum"}
			supported := false
			for _, sf := range supportedFormats {
				if format == sf {
					supported = true
					break
				}
			}
			if !supported {
				delete(newSchema, "format")
			}
		}
		return newSchema
	case []any:
		newSlice := make([]any, len(s))
		for i, item := range s {
			newSlice[i] = recursivelyCleanSchema(item)
		}
		return newSlice
	default:
		return schema
	}
}

// convertClaudeToOpenAI 将Claude请求转换为OpenAI格式
func convertClaudeToOpenAI(claudeReq ClaudeRequest, modelName string) OpenAIRequest {
	var openaiMessages []OpenAIMessage

	// 添加system消息（支持字符串和数组格式）
	systemMessage := extractSystemMessage(claudeReq.System)
	if systemMessage != "" {
		openaiMessages = append(openaiMessages, OpenAIMessage{
			Role:    "system",
			Content: systemMessage,
		})
	}

	// 转换消息
	for _, message := range claudeReq.Messages {
		if message.Role == "user" {
			// 处理用户消息
			if contentBlocks, ok := message.Content.([]any); ok {
				var toolResults []any
				var otherContent []any

				for _, block := range contentBlocks {
					if blockMap, ok := block.(map[string]any); ok {
						if blockMap["type"] == "tool_result" {
							toolResults = append(toolResults, blockMap)
						} else {
							otherContent = append(otherContent, blockMap)
						}
					}
				}

				// 添加工具结果消息
				for _, result := range toolResults {
					if resultMap, ok := result.(map[string]any); ok {
						var content string
						if resultMap["content"] != nil {
							if str, ok := resultMap["content"].(string); ok {
								content = str
							} else {
								contentBytes, _ := json.Marshal(resultMap["content"])
								content = string(contentBytes)
							}
						}

						openaiMessages = append(openaiMessages, OpenAIMessage{
							Role:       "tool",
							ToolCallID: resultMap["tool_use_id"].(string),
							Content:    content,
						})
					}
				}

				// 添加其他用户内容
				if len(otherContent) > 0 {
					var convertedContent []map[string]any
					for _, block := range otherContent {
						if blockMap, ok := block.(map[string]any); ok {
							if blockMap["type"] == "text" {
								convertedContent = append(convertedContent, map[string]any{
									"type": "text",
									"text": blockMap["text"],
								})
							} else if blockMap["type"] == "image" {
								if source, ok := blockMap["source"].(map[string]any); ok {
									imageURL := fmt.Sprintf("data:%s;base64,%s", source["media_type"], source["data"])
									convertedContent = append(convertedContent, map[string]any{
										"type": "image_url",
										"image_url": map[string]string{
											"url": imageURL,
										},
									})
								}
							}
						}
					}
					openaiMessages = append(openaiMessages, OpenAIMessage{
						Role:    "user",
						Content: convertedContent,
					})
				}
			} else {
				// 简单文本消息
				openaiMessages = append(openaiMessages, OpenAIMessage{
					Role:    "user",
					Content: message.Content,
				})
			}
		} else if message.Role == "assistant" {
			// 处理助手消息
			var textParts []string
			var toolCalls []OpenAIToolCall

			if contentBlocks, ok := message.Content.([]any); ok {
				for _, block := range contentBlocks {
					if blockMap, ok := block.(map[string]any); ok {
						if blockMap["type"] == "text" {
							textParts = append(textParts, blockMap["text"].(string))
						} else if blockMap["type"] == "tool_use" {
							arguments := "{}"
							if blockMap["input"] != nil {
								argBytes, _ := json.Marshal(blockMap["input"])
								arguments = string(argBytes)
							}

							toolCalls = append(toolCalls, OpenAIToolCall{
								ID:   blockMap["id"].(string),
								Type: "function",
								Function: OpenAIFunctionCall{
									Name:      blockMap["name"].(string),
									Arguments: arguments,
								},
							})
						}
					}
				}
			}

			assistantMessage := OpenAIMessage{
				Role:    "assistant",
				Content: strings.Join(textParts, "\n"),
			}
			if len(toolCalls) > 0 {
				assistantMessage.ToolCalls = toolCalls
			}
			if assistantMessage.Content == "" {
				assistantMessage.Content = nil
			}

			openaiMessages = append(openaiMessages, assistantMessage)
		}
	}

	// 构建OpenAI请求（强制流式处理）
	openaiReq := OpenAIRequest{
		Model:       modelName,
		Messages:    openaiMessages,
		Temperature: claudeReq.Temperature,
		TopP:        claudeReq.TopP,
		Stream:      true, // 强制流式处理
		Stop:        claudeReq.StopSequences,
	}

	// 转换工具
	if len(claudeReq.Tools) > 0 {
		for _, tool := range claudeReq.Tools {
			cleanedParameters := recursivelyCleanSchema(tool.InputSchema)
			openaiReq.Tools = append(openaiReq.Tools, OpenAITool{
				Type: "function",
				Function: OpenAIFunctionDef{
					Name:        tool.Name,
					Description: tool.Description,
					Parameters:  cleanedParameters,
				},
			})
		}
	}

	// 转换工具选择
	if claudeReq.ToolChoice != nil {
		if claudeReq.ToolChoice.Type == "auto" || claudeReq.ToolChoice.Type == "any" {
			openaiReq.ToolChoice = "auto"
		} else if claudeReq.ToolChoice.Type == "tool" {
			openaiReq.ToolChoice = map[string]any{
				"type": "function",
				"function": map[string]string{
					"name": claudeReq.ToolChoice.Name,
				},
			}
		}
	}

	return openaiReq
}

// convertOpenAIToClaudeResponse 将OpenAI响应转换为Claude格式
func convertOpenAIToClaudeResponse(openaiResp OpenAIResponse, model string) ClaudeResponse {
	var contentBlocks []ClaudeContentBlock

	if len(openaiResp.Choices) > 0 {
		choice := openaiResp.Choices[0]

		// 添加文本内容
		if choice.Message.Content != nil {
			if content, ok := choice.Message.Content.(string); ok && content != "" {
				contentBlocks = append(contentBlocks, ClaudeContentBlock{
					Type: "text",
					Text: content,
				})
			}
		}

		// 添加工具调用
		if len(choice.Message.ToolCalls) > 0 {
			for _, toolCall := range choice.Message.ToolCalls {
				var input map[string]any
				if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &input); err != nil {
					input = make(map[string]any)
				}

				contentBlocks = append(contentBlocks, ClaudeContentBlock{
					Type:  "tool_use",
					ID:    toolCall.ID,
					Name:  toolCall.Function.Name,
					Input: input,
				})
			}
		}
	}

	// 映射停止原因
	stopReasonMap := map[string]string{
		"stop":       "end_turn",
		"length":     "max_tokens",
		"tool_calls": "tool_use",
	}

	var stopReason string
	if len(openaiResp.Choices) > 0 {
		stopReason = stopReasonMap[openaiResp.Choices[0].FinishReason]
	}
	if stopReason == "" {
		stopReason = "end_turn"
	}

	return ClaudeResponse{
		ID:         openaiResp.ID,
		Type:       "message",
		Role:       "assistant",
		Model:      model,
		Content:    contentBlocks,
		StopReason: stopReason,
		Usage: ClaudeUsage{
			InputTokens:  openaiResp.Usage.PromptTokens,
			OutputTokens: openaiResp.Usage.CompletionTokens,
		},
	}
}

// handleStreamingResponse 处理流式响应
func handleStreamingResponse(c *gin.Context, resp *http.Response, model string, isClientStream bool, account *model.Account, apiKey *model.ApiKey, startTime time.Time) {
	// 设置流式响应头
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Writer.Flush()

	// 创建流式转换器并处理OpenAI流式响应
	transformer := createStreamTransformer(model)
	usageTokens := processOpenAIStreamResponse(c.Writer, resp.Body, transformer, isClientStream)

	// 如果没有usage信息，创建0值的TokenUsage用于日志记录
	if usageTokens == nil {
		usageTokens = &common.TokenUsage{
			InputTokens:  0,
			OutputTokens: 0,
			Model:        model,
		}
	}

	// 更新账号状态和统计信息
	accountService := service.NewAccountService()
	go accountService.UpdateAccountStatus(account, resp.StatusCode, usageTokens)

	// 更新API Key统计信息
	if apiKey != nil {
		go service.UpdateApiKeyStatus(apiKey, resp.StatusCode, usageTokens)
	}

	// 保存日志记录
	if resp.StatusCode >= 200 && resp.StatusCode < 300 && apiKey != nil {
		duration := time.Since(startTime).Milliseconds()
		logService := service.NewLogService()
		go func() {
			_, err := logService.CreateLogFromTokenUsage(usageTokens, apiKey.UserID, apiKey.ID, account.ID, duration, isClientStream)
			if err != nil {
				log.Printf("保存日志失败: %v", err)
			}
		}()
	}
}

// processOpenAIStreamResponse 处理OpenAI流式响应并转换为Claude格式
func processOpenAIStreamResponse(writer gin.ResponseWriter, reader io.Reader, transformer *StreamTransformer, isClientStream bool) *common.TokenUsage {
	scanner := bufio.NewScanner(reader)

	var totalPromptTokens, totalCompletionTokens int
	var responseContent strings.Builder
	var toolCalls []OpenAIToolCall
	var finishReason string

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// 跳过空行和非data行
		if line == "" || !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := line[6:] // 移除 "data: " 前缀

		// 处理结束标记
		if strings.TrimSpace(data) == "[DONE]" {
			if isClientStream {
				transformer.sendFinalEvents(writer)
			}
			break
		}

		// 解析OpenAI流式数据
		var openaiChunk map[string]any
		if err := json.Unmarshal([]byte(data), &openaiChunk); err != nil {
			continue // 忽略解析错误的chunk
		}

		// 提取usage信息
		if usage, ok := openaiChunk["usage"].(map[string]any); ok {
			if promptTokens, ok := usage["prompt_tokens"].(float64); ok {
				totalPromptTokens = int(promptTokens)
			}
			if completionTokens, ok := usage["completion_tokens"].(float64); ok {
				totalCompletionTokens = int(completionTokens)
			}
		}

		// 处理流式数据
		if choices, ok := openaiChunk["choices"].([]any); ok && len(choices) > 0 {
			if choice, ok := choices[0].(map[string]any); ok {
				// 获取finish_reason
				if reason, ok := choice["finish_reason"].(string); ok && reason != "" {
					finishReason = reason
				}

				if delta, ok := choice["delta"].(map[string]any); ok {
					// 收集文本内容
					if content, ok := delta["content"].(string); ok {
						responseContent.WriteString(content)
					}

					// 收集工具调用增量数据
					if toolCallsData, ok := delta["tool_calls"].([]any); ok {
						for _, tc := range toolCallsData {
							if tcMap, ok := tc.(map[string]any); ok {
								index := int(tcMap["index"].(float64))

								// 确保toolCalls数组足够长
								for len(toolCalls) <= index {
									toolCalls = append(toolCalls, OpenAIToolCall{})
								}

								// 更新工具调用信息
								if id, ok := tcMap["id"].(string); ok {
									toolCalls[index].ID = id
									toolCalls[index].Type = "function"
								}

								if function, ok := tcMap["function"].(map[string]any); ok {
									if name, ok := function["name"].(string); ok {
										toolCalls[index].Function.Name = name
									}
									if args, ok := function["arguments"].(string); ok {
										toolCalls[index].Function.Arguments += args
									}
								}
							}
						}
					}
				}
			}
		}

		// 如果客户端需要流式响应，实时转发
		if isClientStream {
			transformer.processChunk(writer, openaiChunk)
		}
	}

	// 如果客户端不需要流式响应，发送完整的非流式响应
	if !isClientStream {
		// 构建Claude格式的内容块
		var contentBlocks []ClaudeContentBlock

		// 添加文本内容
		if responseContent.Len() > 0 {
			contentBlocks = append(contentBlocks, ClaudeContentBlock{
				Type: "text",
				Text: responseContent.String(),
			})
		}

		// 添加工具调用内容
		for _, toolCall := range toolCalls {
			if toolCall.ID != "" && toolCall.Function.Name != "" {
				var input map[string]any
				if toolCall.Function.Arguments != "" {
					json.Unmarshal([]byte(toolCall.Function.Arguments), &input)
				}
				if input == nil {
					input = make(map[string]any)
				}

				contentBlocks = append(contentBlocks, ClaudeContentBlock{
					Type:  "tool_use",
					ID:    toolCall.ID,
					Name:  toolCall.Function.Name,
					Input: input,
				})
			}
		}

		// 映射停止原因
		stopReasonMap := map[string]string{
			"stop":       "end_turn",
			"length":     "max_tokens",
			"tool_calls": "tool_use",
		}
		stopReason := stopReasonMap[finishReason]
		if stopReason == "" {
			stopReason = "end_turn"
		}

		claudeResponse := ClaudeResponse{
			ID:         fmt.Sprintf("msg_%s", generateRandomID()),
			Type:       "message",
			Role:       "assistant",
			Model:      transformer.model,
			Content:    contentBlocks,
			StopReason: stopReason,
			Usage: ClaudeUsage{
				InputTokens:  totalPromptTokens,
				OutputTokens: totalCompletionTokens,
			},
		}

		// 设置非流式响应头
		writer.Header().Set("Content-Type", "application/json")
		jsonBytes, _ := json.Marshal(claudeResponse)
		writer.Write(jsonBytes)
	}

	// 返回token使用统计
	if totalPromptTokens > 0 || totalCompletionTokens > 0 {
		return &common.TokenUsage{
			InputTokens:  totalPromptTokens,
			OutputTokens: totalCompletionTokens,
			Model:        transformer.model,
		}
	}

	return nil
}

// StreamTransformer 流式转换器结构
type StreamTransformer struct {
	initialized       bool
	messageID         string
	model             string
	toolCalls         map[int]*ToolCallState
	contentBlockIndex int
}

// ToolCallState 工具调用状态
type ToolCallState struct {
	ID          string
	Name        string
	Args        string
	ClaudeIndex int
	Started     bool
}

// createStreamTransformer 创建流式转换器
func createStreamTransformer(model string) *StreamTransformer {
	return &StreamTransformer{
		initialized:       false,
		messageID:         fmt.Sprintf("msg_%s", generateRandomID()),
		model:             model,
		toolCalls:         make(map[int]*ToolCallState),
		contentBlockIndex: 0,
	}
}

// generateRandomID 生成随机ID
func generateRandomID() string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	result := make([]byte, 9)
	for i := range result {
		result[i] = chars[rand.Intn(len(chars))]
	}
	return string(result)
}

// sendEvent 发送SSE事件
func (st *StreamTransformer) sendEvent(writer gin.ResponseWriter, eventType string, data any) {
	jsonData, _ := json.Marshal(data)
	fmt.Fprintf(writer, "event: %s\ndata: %s\n\n", eventType, jsonData)
	writer.Flush()
}

// processChunk 处理单个流式chunk
func (st *StreamTransformer) processChunk(writer gin.ResponseWriter, openaiChunk map[string]any) {
	// 初始化消息开始事件
	if !st.initialized {
		st.sendEvent(writer, "message_start", map[string]any{
			"type": "message_start",
			"message": map[string]any{
				"id":          st.messageID,
				"type":        "message",
				"role":        "assistant",
				"model":       st.model,
				"content":     []any{},
				"stop_reason": nil,
				"usage": map[string]int{
					"input_tokens":  0,
					"output_tokens": 0,
				},
			},
		})

		st.sendEvent(writer, "content_block_start", map[string]any{
			"type":  "content_block_start",
			"index": 0,
			"content_block": map[string]any{
				"type": "text",
				"text": "",
			},
		})

		st.initialized = true
	}

	// 处理choices数组
	if choices, ok := openaiChunk["choices"].([]any); ok && len(choices) > 0 {
		if choice, ok := choices[0].(map[string]any); ok {
			if delta, ok := choice["delta"].(map[string]any); ok {
				// 处理文本内容
				if content, ok := delta["content"].(string); ok {
					st.sendEvent(writer, "content_block_delta", map[string]any{
						"type":  "content_block_delta",
						"index": 0,
						"delta": map[string]any{
							"type": "text_delta",
							"text": content,
						},
					})
				}

				// 处理工具调用
				if toolCalls, ok := delta["tool_calls"].([]any); ok {
					for _, tc := range toolCalls {
						if tcMap, ok := tc.(map[string]any); ok {
							st.processToolCallDelta(writer, tcMap)
						}
					}
				}
			}
		}
	}
}

// processToolCallDelta 处理工具调用增量
func (st *StreamTransformer) processToolCallDelta(writer gin.ResponseWriter, tcDelta map[string]any) {
	index := int(tcDelta["index"].(float64))

	// 初始化工具调用状态
	if _, exists := st.toolCalls[index]; !exists {
		st.toolCalls[index] = &ToolCallState{
			ID:          "",
			Name:        "",
			Args:        "",
			ClaudeIndex: 0,
			Started:     false,
		}
	}

	toolCall := st.toolCalls[index]

	// 更新工具调用信息
	if id, ok := tcDelta["id"].(string); ok {
		toolCall.ID = id
	}

	if function, ok := tcDelta["function"].(map[string]any); ok {
		if name, ok := function["name"].(string); ok {
			toolCall.Name = name
		}
		if args, ok := function["arguments"].(string); ok {
			toolCall.Args += args
		}
	}

	// 如果工具调用准备就绪且未开始，发送开始事件
	if toolCall.ID != "" && toolCall.Name != "" && !toolCall.Started {
		st.contentBlockIndex++
		toolCall.ClaudeIndex = st.contentBlockIndex
		toolCall.Started = true

		st.sendEvent(writer, "content_block_start", map[string]any{
			"type":  "content_block_start",
			"index": toolCall.ClaudeIndex,
			"content_block": map[string]any{
				"type":  "tool_use",
				"id":    toolCall.ID,
				"name":  toolCall.Name,
				"input": map[string]any{},
			},
		})
	}

	// 如果有新的参数内容，发送增量事件
	if toolCall.Started {
		if function, ok := tcDelta["function"].(map[string]any); ok {
			if args, ok := function["arguments"].(string); ok {
				st.sendEvent(writer, "content_block_delta", map[string]any{
					"type":  "content_block_delta",
					"index": toolCall.ClaudeIndex,
					"delta": map[string]any{
						"type":         "input_json_delta",
						"partial_json": args,
					},
				})
			}
		}
	}
}

// sendFinalEvents 发送最终事件
func (st *StreamTransformer) sendFinalEvents(writer gin.ResponseWriter) {
	// 发送内容块结束事件
	st.sendEvent(writer, "content_block_stop", map[string]any{
		"type":  "content_block_stop",
		"index": 0,
	})

	// 发送所有工具调用的结束事件
	for _, toolCall := range st.toolCalls {
		if toolCall.Started {
			st.sendEvent(writer, "content_block_stop", map[string]any{
				"type":  "content_block_stop",
				"index": toolCall.ClaudeIndex,
			})
		}
	}

	// 发送消息增量事件（包含停止原因）
	st.sendEvent(writer, "message_delta", map[string]any{
		"type": "message_delta",
		"delta": map[string]any{
			"stop_reason":   "end_turn",
			"stop_sequence": nil,
		},
		"usage": map[string]int{
			"output_tokens": 0,
		},
	})

	// 发送消息停止事件
	st.sendEvent(writer, "message_stop", map[string]any{
		"type": "message_stop",
	})
}

// TestHandleOpenAIRequest 仅用于单元测试，返回状态码和响应内容
func TestHandleOpenAIRequest(account *model.Account) (int, string) {
	requestBody := GetTestRequestBody(100)

	// 解析Claude请求
	var claudeReq ClaudeRequest
	if err := json.Unmarshal([]byte(requestBody), &claudeReq); err != nil {
		return http.StatusBadRequest, "Failed to parse request JSON: " + err.Error()
	}

	// 检查账号配置
	if account.RequestURL == "" {
		return http.StatusBadRequest, "账号未配置请求地址"
	}

	targetConfig := &OpenAITargetConfig{
		BaseURL:   account.RequestURL,
		ModelName: "gpt-4o-mini",
	}

	// 应用模型映射
	mappedModelName := applyModelMapping(claudeReq.Model, account.ModelMapping, targetConfig.ModelName)

	// 转换Claude请求为OpenAI格式
	openaiReq := convertClaudeToOpenAI(claudeReq, mappedModelName)

	// 序列化OpenAI请求
	openaiBody, err := json.Marshal(openaiReq)
	if err != nil {
		return http.StatusInternalServerError, "Failed to marshal OpenAI request: " + err.Error()
	}

	// 创建OpenAI API请求
	openaiURL := targetConfig.BaseURL + "/chat/completions"
	req, err := http.NewRequest("POST", openaiURL, bytes.NewBuffer(openaiBody))
	if err != nil {
		return http.StatusInternalServerError, "Failed to create request: " + err.Error()
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+account.SecretKey)

	// 创建HTTP客户端
	httpClientTimeout := 30 * time.Second
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	// 配置代理
	if account.ProxyURI != "" {
		proxyURL, err := url.Parse(account.ProxyURI)
		if err != nil {
			return http.StatusInternalServerError, "Invalid proxy URI: " + err.Error()
		}
		transport.Proxy = http.ProxyURL(proxyURL)
	}

	client := &http.Client{
		Timeout:   httpClientTimeout,
		Transport: transport,
	}

	// 发送请求
	resp, err := client.Do(req)
	if err != nil {
		return http.StatusInternalServerError, "Request failed: " + err.Error()
	}
	defer common.CloseIO(resp.Body)

	// 读取错误响应内容
	if resp.StatusCode >= 400 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return resp.StatusCode, string(bodyBytes)
	}

	return resp.StatusCode, ""
}
