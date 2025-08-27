package middleware

import (
	"claude-code-relay/constant"
	"claude-code-relay/model"
	"encoding/json"
	"io"
	"net/http"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
)

// SystemMessage 系统消息结构体
type SystemMessage struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// RequestBody Claude API请求体结构
type RequestBody struct {
	System any `json:"system"`
}

// ClaudeCodeAuth API Key鉴权中间件
func ClaudeCodeAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 判断是否来自真实的 Claude Code 请求
		if !isRealClaudeCodeRequest(c) {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "仅支持来自 Claude Code 的请求",
				"code":  40003,
			})
			c.Abort()
			return
		}

		// 从多个可能的请求头中获取API Key
		apiKey := getApiKeyFromHeaders(c)
		if apiKey == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "缺少API Key",
				"code":  40001,
			})
			c.Abort()
			return
		}

		// 从数据库查询API Key
		keyInfo, err := model.GetApiKeyByKey(apiKey)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "无效的API Key",
				"code":  40001,
			})
			c.Abort()
			return
		}

		// 判断是否达到每日限额
		if keyInfo.DailyLimit > 0 && keyInfo.TodayTotalCost >= keyInfo.DailyLimit {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "API Key已达到每日使用限额",
				"code":  40004,
			})
			c.Abort()
			return
		}

		// API Key已经在model层验证了状态和过期时间
		// 将API Key信息存储到上下文中供后续使用
		c.Set("api_key_id", keyInfo.ID)
		c.Set("api_key", keyInfo)
		c.Set("user_id", keyInfo.UserID)
		c.Set("group_id", keyInfo.GroupID)

		c.Next()
	}
}

// getApiKeyFromHeaders 从多个可能的请求头中提取API Key
func getApiKeyFromHeaders(c *gin.Context) string {
	// 1. 检查 X-API-Key
	if apiKey := c.GetHeader("x-api-key"); apiKey != "" {
		return apiKey
	}

	// 2. 检查 X-Goog-API-Key (Google Cloud API格式)
	if apiKey := c.GetHeader("X-Goog-API-Key"); apiKey != "" {
		return apiKey
	}

	// 3. 检查 Authorization Bearer Token
	if authHeader := c.GetHeader("Authorization"); authHeader != "" {
		if strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
			return strings.TrimSpace(authHeader[7:])
		}
	}

	// 4. 检查 API-Key
	if apiKey := c.GetHeader("API-Key"); apiKey != "" {
		return apiKey
	}

	return ""
}

// isRealClaudeCodeRequest 判断是否是真实的 Claude Code 请求
func isRealClaudeCodeRequest(c *gin.Context) bool {
	// 检查 User-Agent 是否匹配 Claude Code 格式
	userAgent := c.GetHeader("User-Agent")
	isClaudeCodeUserAgent := isClaudeCodeUserAgent(userAgent)

	// 检查系统提示词
	hasClaudeCodeSystemPrompt := hasClaudeCodeSystemPrompt(c)

	// 只有当 User-Agent 匹配且系统提示词正确时，才认为是真实的 Claude Code 请求
	return isClaudeCodeUserAgent && hasClaudeCodeSystemPrompt
}

// isClaudeCodeUserAgent 检查 User-Agent 是否匹配 Claude Code 格式
func isClaudeCodeUserAgent(userAgent string) bool {
	if userAgent == "" {
		return false
	}
	// 匹配 claude-cli/x.x.x 格式
	matched, _ := regexp.MatchString(`claude-cli/\d+\.\d+\.\d+`, userAgent)
	return matched
}

// hasClaudeCodeSystemPrompt 检查请求中是否包含 Claude Code 系统提示词
func hasClaudeCodeSystemPrompt(c *gin.Context) bool {
	// 读取请求体
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return false
	}

	// 重新设置请求体，以便后续处理可以再次读取
	c.Request.Body = io.NopCloser(strings.NewReader(string(bodyBytes)))

	// 解析请求体
	var requestBody RequestBody
	if err := json.Unmarshal(bodyBytes, &requestBody); err != nil {
		return false
	}

	if requestBody.System == nil {
		return false
	}

	// 如果是字符串格式，一定不是真实的 Claude Code 请求
	if systemStr, ok := requestBody.System.(string); ok {
		_ = systemStr // 避免未使用变量警告
		return false
	}

	// 处理数组格式
	if systemArray, ok := requestBody.System.([]any); ok && len(systemArray) > 0 {
		if firstItem, ok := systemArray[0].(map[string]any); ok {
			// 检查第一个元素是否包含 Claude Code 提示词
			if itemType, exists := firstItem["type"]; exists && itemType == "text" {
				if text, exists := firstItem["text"]; exists && text == constant.ClaudeCodeSystemPrompt {
					return true
				}
			}
		}
	}

	return false
}
