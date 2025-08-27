package relay

import (
	"bytes"
	"claude-code-relay/common"
	"claude-code-relay/model"
	"claude-code-relay/service"
	"compress/flate"
	"compress/gzip"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/tidwall/sjson"

	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
)

const (
	ClaudeAPIURL        = "https://api.anthropic.com/v1/messages"
	ClaudeOAuthTokenURL = "https://console.anthropic.com/v1/oauth/token"
	ClaudeOAuthClientID = "9d1c250a-e61b-44d9-88ed-5944d1962f5e"

	// 默认超时配置
	defaultHTTPTimeout = 120 * time.Second
	tokenRefreshBuffer = 300 // 5分钟
	rateLimitDuration  = 5 * time.Hour

	// 状态码
	statusRateLimit  = 429
	statusOK         = 200
	statusBadRequest = 400

	// 账号状态
	accountStatusActive    = 1
	accountStatusDisabled  = 2
	accountStatusRateLimit = 3
)

// 错误类型定义
var (
	errRequestBody     = gin.H{"error": map[string]any{"type": "request_error", "message": "Incorrect request body"}}
	errMissingModel    = gin.H{"error": map[string]any{"type": "request_error", "message": "The model field is missing in the request body"}}
	errModelNotAllowed = gin.H{"error": map[string]any{"type": "request_error", "message": "This model is not allowed."}}
	errAuthFailed      = gin.H{"error": map[string]any{"type": "authentication_error", "message": "Failed to get valid access token"}}
	errCreateRequest   = gin.H{"error": map[string]any{"type": "internal_server_error", "message": "Failed to create request"}}
	errProxyConfig     = gin.H{"error": map[string]any{"type": "proxy_configuration_error", "message": "Invalid proxy URI"}}
	errTimeout         = gin.H{"error": map[string]any{"type": "timeout_error", "message": "Request was canceled or timed out"}}
	errNetworkError    = gin.H{"error": map[string]any{"type": "network_error", "message": "Failed to execute request"}}
	errDecompression   = gin.H{"error": map[string]any{"type": "decompression_error", "message": "Failed to create decompressor"}}
	errResponseRead    = gin.H{"error": map[string]any{"type": "response_read_error", "message": "Failed to read error response"}}
	errResponseError   = gin.H{"error": map[string]any{"type": "response_error", "message": "Request failed"}}
)

// OAuthTokenResponse 表示OAuth token刷新响应
type OAuthTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
}

// HandleClaudeRequest 处理Claude官方API平台的请求
func HandleClaudeRequest(c *gin.Context, account *model.Account) {
	startTime := time.Now()

	apiKey := extractAPIKey(c)

	requestData, err := parseAndValidateRequest(c)
	if err != nil {
		return
	}

	if apiKey != nil {
		if err := validateModelRestriction(c, apiKey, requestData.ModelName); err != nil {
			return
		}
	}

	accessToken, err := GetValidAccessToken(account)
	if err != nil {
		log.Printf("获取有效访问token失败: %v", err)
		c.JSON(http.StatusInternalServerError, appendErrorMessage(errAuthFailed, err.Error()))
		return
	}

	client := createHTTPClient(account)
	if client == nil {
		c.JSON(http.StatusInternalServerError, errProxyConfig)
		return
	}

	req, err := createClaudeRequest(c, requestData.Body, accessToken)
	if err != nil {
		c.JSON(http.StatusInternalServerError, appendErrorMessage(errCreateRequest, err.Error()))
		return
	}

	resp, err := client.Do(req)
	if err != nil {
		handleRequestError(c, err)
		return
	}
	defer common.CloseIO(resp.Body)

	responseReader, err := createResponseReader(resp)
	if err != nil {
		c.JSON(http.StatusInternalServerError, appendErrorMessage(errDecompression, err.Error()))
		return
	}

	var usageTokens *common.TokenUsage
	if resp.StatusCode < statusBadRequest {
		usageTokens = handleSuccessResponse(c, resp, responseReader)
	} else {
		handleErrorResponse(c, resp, responseReader, account)
	}

	updateAccountAndStats(account, resp.StatusCode, usageTokens)

	if apiKey != nil {
		go service.UpdateApiKeyStatus(apiKey, resp.StatusCode, usageTokens)
	}

	saveRequestLog(startTime, apiKey, account, resp.StatusCode, usageTokens, true)
}

// requestData 封装请求数据
type requestData struct {
	Body      []byte
	ModelName string
}

// extractAPIKey 从上下文中提取API Key
func extractAPIKey(c *gin.Context) *model.ApiKey {
	if keyInfo, exists := c.Get("api_key"); exists {
		return keyInfo.(*model.ApiKey)
	}
	return nil
}

// parseAndValidateRequest 解析并验证请求
func parseAndValidateRequest(c *gin.Context) (*requestData, error) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, errRequestBody)
		return nil, err
	}

	body, _ = sjson.SetBytes(body, "stream", true)
	body, _ = sjson.SetBytes(body, "metadata.user_id", common.GetInstanceID())

	modelName := gjson.GetBytes(body, "model").String()
	if modelName == "" {
		c.JSON(http.StatusServiceUnavailable, errMissingModel)
		return nil, errors.New("missing model")
	}

	return &requestData{Body: body, ModelName: modelName}, nil
}

// validateModelRestriction 验证模型限制
func validateModelRestriction(c *gin.Context, apiKey *model.ApiKey, modelName string) error {
	if apiKey.ModelRestriction == "" {
		return nil
	}

	allowedModels := strings.Split(apiKey.ModelRestriction, ",")
	for _, allowedModel := range allowedModels {
		if strings.EqualFold(strings.TrimSpace(allowedModel), modelName) {
			return nil
		}
	}

	c.JSON(http.StatusForbidden, errModelNotAllowed)
	return errors.New("model not allowed")
}

// createHTTPClient 创建HTTP客户端
func createHTTPClient(account *model.Account) *http.Client {
	timeout := parseHTTPTimeout()

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	if account.EnableProxy && account.ProxyURI != "" {
		proxyURL, err := url.Parse(account.ProxyURI)
		if err != nil {
			log.Printf("invalid proxy URI: %s", err.Error())
			return nil
		}
		transport.Proxy = http.ProxyURL(proxyURL)
	}

	return &http.Client{
		Timeout:   timeout,
		Transport: transport,
	}
}

// parseHTTPTimeout 解析HTTP超时时间
func parseHTTPTimeout() time.Duration {
	if timeoutStr := os.Getenv("HTTP_CLIENT_TIMEOUT"); timeoutStr != "" {
		if timeout, err := time.ParseDuration(timeoutStr + "s"); err == nil {
			return timeout
		}
	}
	return defaultHTTPTimeout
}

// createClaudeRequest 创建Claude请求
func createClaudeRequest(c *gin.Context, body []byte, accessToken string) (*http.Request, error) {
	req, err := http.NewRequestWithContext(
		c.Request.Context(),
		c.Request.Method,
		ClaudeAPIURL,
		bytes.NewBuffer(body),
	)
	if err != nil {
		return nil, err
	}

	copyRequestHeaders(c, req)
	setClaudeAPIHeaders(req, accessToken)
	setStreamHeaders(c, req)

	return req, nil
}

// copyRequestHeaders 复制原始请求头
func copyRequestHeaders(c *gin.Context, req *http.Request) {
	for name, values := range c.Request.Header {
		for _, value := range values {
			req.Header.Add(name, value)
		}
	}
}

// setClaudeAPIHeaders 设置Claude API请求头
func setClaudeAPIHeaders(req *http.Request, accessToken string) {
	fixedHeaders := buildClaudeAPIHeaders(accessToken)
	for name, value := range fixedHeaders {
		req.Header.Set(name, value)
	}

	req.Header.Del("X-Api-Key")
	req.Header.Del("Cookie")
}

// setStreamHeaders 设置流式请求头
func setStreamHeaders(c *gin.Context, req *http.Request) {
	if c.Request.Header.Get("Accept") == "" {
		req.Header.Set("Accept", "text/event-stream")
	}
}

// handleRequestError 处理请求错误
func handleRequestError(c *gin.Context, err error) {
	if errors.Is(err, context.Canceled) {
		c.JSON(http.StatusRequestTimeout, errTimeout)
		return
	}

	log.Printf("❌ 请求失败: %v", err)
	c.JSON(http.StatusInternalServerError, appendErrorMessage(errNetworkError, err.Error()))
}

// createResponseReader 创建响应读取器（处理压缩）
func createResponseReader(resp *http.Response) (io.Reader, error) {
	contentEncoding := resp.Header.Get("Content-Encoding")

	switch strings.ToLower(contentEncoding) {
	case "gzip":
		gzipReader, err := gzip.NewReader(resp.Body)
		if err != nil {
			log.Printf("[Claude API] 创建gzip解压缩器失败: %v", err)
			return nil, err
		}
		return gzipReader, nil
	case "deflate":
		return flate.NewReader(resp.Body), nil
	default:
		return resp.Body, nil
	}
}

// handleSuccessResponse 处理成功响应
func handleSuccessResponse(c *gin.Context, resp *http.Response, responseReader io.Reader) *common.TokenUsage {
	c.Status(resp.StatusCode)
	copyResponseHeaders(c, resp)
	setStreamResponseHeaders(c)

	c.Writer.Flush()

	usageTokens, err := common.ParseStreamResponse(c.Writer, responseReader)
	if err != nil {
		log.Println("stream copy and parse failed:", err.Error())
	}

	return usageTokens
}

// handleErrorResponse 处理错误响应
func handleErrorResponse(c *gin.Context, resp *http.Response, responseReader io.Reader, account *model.Account) {
	responseBody, err := io.ReadAll(responseReader)
	if err != nil {
		log.Printf("❌ 读取错误响应失败: %v", err)
		c.JSON(http.StatusInternalServerError, appendErrorMessage(errResponseRead, err.Error()))
		return
	}

	log.Printf("❌ 错误响应内容: %s", string(responseBody))

	c.Status(resp.StatusCode)
	copyResponseHeaders(c, resp)

	handleRateLimit(resp, responseBody, account)

	c.JSON(http.StatusServiceUnavailable, gin.H{
		"error": map[string]any{
			"type":    "response_error",
			"message": "Request failed with status " + strconv.Itoa(resp.StatusCode),
		},
	})
}

// copyResponseHeaders 复制响应头
func copyResponseHeaders(c *gin.Context, resp *http.Response) {
	for name, values := range resp.Header {
		if strings.ToLower(name) != "content-length" {
			for _, value := range values {
				c.Header(name, value)
			}
		}
	}
}

// setStreamResponseHeaders 设置流式响应头
func setStreamResponseHeaders(c *gin.Context) {
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	if c.Writer.Header().Get("Content-Type") == "" {
		c.Header("Content-Type", "text/event-stream")
	}
}

// handleRateLimit 处理限流逻辑
func handleRateLimit(resp *http.Response, responseBody []byte, account *model.Account) {
	isRateLimited, resetTimestamp := detectRateLimit(resp, responseBody)
	if !isRateLimited {
		return
	}

	log.Printf("🚫 检测到账号 %s 被限流，状态码: %d", account.Name, resp.StatusCode)

	account.CurrentStatus = accountStatusRateLimit

	if resetTimestamp > 0 {
		resetTime := time.Unix(resetTimestamp, 0)
		rateLimitEndTime := model.Time(resetTime)
		account.RateLimitEndTime = &rateLimitEndTime
		log.Printf("账号 %s 限流至 %s", account.Name, resetTime.Format(time.RFC3339))
	} else {
		resetTime := time.Now().Add(rateLimitDuration)
		rateLimitEndTime := model.Time(resetTime)
		account.RateLimitEndTime = &rateLimitEndTime
		log.Printf("账号 %s 限流至 %s (默认5小时)", account.Name, resetTime.Format(time.RFC3339))
	}

	if err := model.UpdateAccount(account); err != nil {
		log.Printf("更新账号限流状态失败: %v", err)
	}
}

// detectRateLimit 检测限流状态
func detectRateLimit(resp *http.Response, responseBody []byte) (bool, int64) {
	if resp.StatusCode == statusRateLimit {
		if resetHeader := resp.Header.Get("anthropic-ratelimit-unified-reset"); resetHeader != "" {
			if timestamp, err := strconv.ParseInt(resetHeader, 10, 64); err == nil {
				resetTime := time.Unix(timestamp, 0)
				log.Printf("🕐 提取到限流重置时间戳: %d (%s)", timestamp, resetTime.Format(time.RFC3339))
				return true, timestamp
			}
		}
		return true, 0
	}

	if len(responseBody) > 0 {
		errorBodyStr := strings.ToLower(string(responseBody))
		rateLimitKeyword := "exceed your account's rate limit"

		if errorData := gjson.Get(string(responseBody), "error.message"); errorData.Exists() {
			if strings.Contains(strings.ToLower(errorData.String()), rateLimitKeyword) {
				return true, 0
			}
		} else if strings.Contains(errorBodyStr, rateLimitKeyword) {
			return true, 0
		}
	}

	return false, 0
}

// updateAccountAndStats 更新账号状态和统计
func updateAccountAndStats(account *model.Account, statusCode int, usageTokens *common.TokenUsage) {
	if statusCode >= statusOK && statusCode < 300 {
		clearRateLimitIfExpired(account)
	}

	accountService := service.NewAccountService()
	accountService.UpdateAccountStatus(account, statusCode, usageTokens)
}

// clearRateLimitIfExpired 清除已过期的限流状态
func clearRateLimitIfExpired(account *model.Account) {
	if account.CurrentStatus == accountStatusRateLimit && account.RateLimitEndTime != nil {
		now := time.Now()
		if now.After(time.Time(*account.RateLimitEndTime)) {
			account.CurrentStatus = accountStatusActive
			account.RateLimitEndTime = nil
			if err := model.UpdateAccount(account); err != nil {
				log.Printf("重置账号限流状态失败: %v", err)
			} else {
				log.Printf("账号 %s 限流状态已自动重置", account.Name)
			}
		}
	}
}

// saveRequestLog 保存请求日志
func saveRequestLog(startTime time.Time, apiKey *model.ApiKey, account *model.Account, statusCode int, usageTokens *common.TokenUsage, isStream bool) {
	if statusCode >= statusOK && statusCode < 300 && usageTokens != nil && apiKey != nil {
		duration := time.Since(startTime).Milliseconds()
		logService := service.NewLogService()
		go func() {
			_, err := logService.CreateLogFromTokenUsage(usageTokens, apiKey.UserID, apiKey.ID, account.ID, duration, isStream)
			if err != nil {
				log.Printf("保存日志失败: %v", err)
			}
		}()
	}
}

// appendErrorMessage 为错误消息追加详细信息
func appendErrorMessage(baseError gin.H, message string) gin.H {
	errorMap := baseError["error"].(map[string]any)
	errorMap["message"] = errorMap["message"].(string) + ": " + message
	return gin.H{"error": errorMap}
}

// TestsHandleClaudeRequest 用于测试的Claude请求处理函数，功能同HandleClaudeRequest但不更新日志和账号状态
// 主要用于单元测试和集成测试，避免对数据库和日志系统的
func TestsHandleClaudeRequest(account *model.Account) (int, string) {
	body, _ := sjson.SetBytes([]byte(TestRequestBody), "stream", true)

	// 获取有效的访问token
	accessToken, err := GetValidAccessToken(account)
	if err != nil {
		return http.StatusInternalServerError, "Failed to get valid access token: " + err.Error()
	}

	req, err := http.NewRequest("POST", ClaudeAPIURL, bytes.NewBuffer(body))
	if err != nil {
		return http.StatusInternalServerError, "Failed to create request: " + err.Error()
	}

	// 使用公共的请求头构建方法
	fixedHeaders := buildClaudeAPIHeaders(accessToken)

	for name, value := range fixedHeaders {
		req.Header.Set(name, value)
	}

	httpClientTimeout := 30 * time.Second
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	if account.EnableProxy && account.ProxyURI != "" {
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

	resp, err := client.Do(req)
	if err != nil {
		return http.StatusInternalServerError, "Request failed: " + err.Error()
	}
	defer common.CloseIO(resp.Body)

	// 打印响应内容
	if resp.StatusCode >= 400 {
		responseBody, _ := io.ReadAll(resp.Body)
		log.Println("Response Status:", resp.Status)
		log.Println("Response body:", string(responseBody))
	}
	return resp.StatusCode, ""
}

// buildClaudeAPIHeaders 构建Claude API请求头
func buildClaudeAPIHeaders(accessToken string) map[string]string {
	return map[string]string{
		"Authorization":                             "Bearer " + accessToken,
		"anthropic-version":                         "2023-06-01",
		"X-Stainless-Retry-Count":                   "0",
		"X-Stainless-Timeout":                       "600",
		"X-Stainless-Lang":                          "js",
		"X-Stainless-Package-Version":               "0.55.1",
		"X-Stainless-OS":                            "MacOS",
		"X-Stainless-Arch":                          "arm64",
		"X-Stainless-Runtime":                       "node",
		"x-stainless-helper-method":                 "stream",
		"x-app":                                     "cli",
		"User-Agent":                                "claude-cli/1.0.44 (external, cli)",
		"anthropic-beta":                            "claude-code-20250219,oauth-2025-04-20,interleaved-thinking-2025-05-14,fine-grained-tool-streaming-2025-05-14",
		"X-Stainless-Runtime-Version":               "v20.18.1",
		"anthropic-dangerous-direct-browser-access": "true",
	}
}

// GetValidAccessToken 获取有效的访问token，如果过期则自动刷新
func GetValidAccessToken(account *model.Account) (string, error) {
	// 检查当前token是否存在
	if account.AccessToken == "" {
		return "", errors.New("账号缺少访问token")
	}

	// 检查token是否过期（提前5分钟刷新）
	now := time.Now().Unix()
	expiresAt := int64(account.ExpiresAt)

	// 如果过期时间存在且距离过期不到5分钟，或者已经过期，则需要刷新
	if expiresAt > 0 && now >= (expiresAt-tokenRefreshBuffer) {
		log.Printf("账号 %s 的token即将过期或已过期，尝试刷新", account.Name)

		if account.RefreshToken == "" {
			return "", errors.New("账号缺少刷新token，无法自动刷新")
		}

		// 刷新token
		newAccessToken, newRefreshToken, newExpiresAt, err := refreshToken(account)
		if err != nil {
			log.Printf("刷新token失败: %v", err)
			// 刷新失败时，如果当前token未完全过期，仍尝试使用
			if now < expiresAt {
				log.Printf("刷新失败但token未完全过期，尝试使用当前token")
				return account.AccessToken, nil
			}

			// token已过期且刷新失败，禁用此账号
			log.Printf("token已过期且刷新失败，禁用账号: %s", account.Name)
			account.CurrentStatus = accountStatusDisabled // 设置为禁用状态
			if updateErr := model.UpdateAccount(account); updateErr != nil {
				log.Printf("禁用账号失败: %v", updateErr)
			} else {
				log.Printf("账号 %s 已被自动禁用", account.Name)
			}
			return "", fmt.Errorf("token已过期且刷新失败: %v", err)
		}

		// 更新账号信息
		account.AccessToken = newAccessToken
		account.RefreshToken = newRefreshToken
		account.ExpiresAt = int(newExpiresAt)

		// 保存到数据库
		if err := model.UpdateAccount(account); err != nil {
			log.Printf("更新账号token信息到数据库失败: %v", err)
			// 不返回错误，因为内存中的token已经更新
		}

		log.Printf("账号 %s token刷新成功", account.Name)
		return newAccessToken, nil
	}

	// token还有效，直接返回
	return account.AccessToken, nil
}

// refreshToken 使用refresh token获取新的access token
func refreshToken(account *model.Account) (accessToken, refreshToken string, expiresAt int64, err error) {
	payload := map[string]any{
		"grant_type":    "refresh_token",
		"refresh_token": account.RefreshToken,
		"client_id":     ClaudeOAuthClientID,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", "", 0, fmt.Errorf("序列化请求数据失败: %v", err)
	}

	req, err := http.NewRequest("POST", ClaudeOAuthTokenURL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return "", "", 0, fmt.Errorf("创建刷新请求失败: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("User-Agent", "claude-cli/1.0.56 (external, cli)")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Referer", "https://claude.ai/")
	req.Header.Set("Origin", "https://claude.ai")

	// 创建HTTP客户端，配置代理（如果启用）
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	if account.EnableProxy && account.ProxyURI != "" {
		proxyURL, err := url.Parse(account.ProxyURI)
		if err == nil {
			transport.Proxy = http.ProxyURL(proxyURL)
		}
	}

	client := &http.Client{
		Timeout:   30 * time.Second,
		Transport: transport,
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", "", 0, fmt.Errorf("刷新token请求失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", 0, fmt.Errorf("读取刷新响应失败: %v", err)
	}

	if resp.StatusCode != 200 {
		return "", "", 0, fmt.Errorf("刷新token失败，状态码: %d, 响应: %s", resp.StatusCode, string(body))
	}

	var tokenResp OAuthTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", "", 0, fmt.Errorf("解析token响应失败: %v", err)
	}

	if tokenResp.AccessToken == "" {
		return "", "", 0, errors.New("刷新响应中缺少access_token")
	}

	// 计算过期时间戳
	expiresAt = time.Now().Unix() + int64(tokenResp.ExpiresIn)

	log.Printf("Token刷新成功，新token: %s，将在 %d 秒后过期", maskToken(tokenResp.AccessToken), tokenResp.ExpiresIn)

	return tokenResp.AccessToken, tokenResp.RefreshToken, expiresAt, nil
}

// maskToken 遮蔽token用于安全日志输出
func maskToken(token string) string {
	if len(token) <= 8 {
		return strings.Repeat("*", len(token))
	}
	return token[:4] + strings.Repeat("*", len(token)-8) + token[len(token)-4:]
}
