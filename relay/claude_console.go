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
	"errors"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tidwall/sjson"
)

const (
	// Console默认超时配置
	consoleDefaultTimeout = 120 * time.Second

	// 状态码
	consoleStatusOK         = 200
	consoleStatusBadRequest = 400
)

// Console错误类型定义
var (
	consoleErrRequestBodyRead = gin.H{"error": map[string]any{"type": "request_body_error", "message": "Failed to read request body"}}
	consoleErrCreateRequest   = gin.H{"error": map[string]any{"type": "internal_server_error", "message": "Failed to create request"}}
	consoleErrProxyConfig     = gin.H{"error": map[string]any{"type": "proxy_configuration_error", "message": "Invalid proxy URI"}}
	consoleErrTimeout         = gin.H{"error": map[string]any{"type": "timeout_error", "message": "Request was canceled or timed out"}}
	consoleErrNetworkError    = gin.H{"error": map[string]any{"type": "network_error", "message": "Failed to execute request"}}
	consoleErrDecompression   = gin.H{"error": map[string]any{"type": "decompression_error", "message": "Failed to create decompressor"}}
	consoleErrResponseRead    = gin.H{"error": map[string]any{"type": "response_read_error", "message": "Failed to read error response"}}
	consoleErrResponseError   = gin.H{"error": map[string]any{"type": "response_error", "message": "Request failed"}}
)

// HandleClaudeConsoleRequest 处理Claude Console平台的请求
func HandleClaudeConsoleRequest(c *gin.Context, account *model.Account) {
	startTime := time.Now()

	apiKey := extractConsoleAPIKey(c)

	body, err := parseConsoleRequest(c)
	if err != nil {
		return
	}

	client := createConsoleHTTPClient(account)
	if client == nil {
		c.JSON(http.StatusInternalServerError, consoleErrProxyConfig)
		return
	}

	req, err := createConsoleRequest(c, body, account)
	if err != nil {
		c.JSON(http.StatusInternalServerError, appendConsoleErrorMessage(consoleErrCreateRequest, err.Error()))
		return
	}

	resp, err := client.Do(req)
	if err != nil {
		handleConsoleRequestError(c, err)
		return
	}
	defer common.CloseIO(resp.Body)

	accountService := service.NewAccountService()

	if resp.StatusCode >= consoleStatusBadRequest {
		accountService.UpdateAccountStatus(account, resp.StatusCode, nil)
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": map[string]any{
				"type":    "response_error",
				"message": "Request failed with status " + strconv.Itoa(resp.StatusCode),
			},
		})
		return
	}

	responseReader, err := createConsoleResponseReader(resp)
	if err != nil {
		c.JSON(http.StatusInternalServerError, appendConsoleErrorMessage(consoleErrDecompression, err.Error()))
		return
	}

	usageTokens := handleConsoleSuccessResponse(c, resp, responseReader)

	go accountService.UpdateAccountStatus(account, resp.StatusCode, usageTokens)

	if apiKey != nil {
		go service.UpdateApiKeyStatus(apiKey, resp.StatusCode, usageTokens)
	}

	saveConsoleRequestLog(startTime, apiKey, account, resp.StatusCode, usageTokens)
}

// extractConsoleAPIKey 从上下文中提取API Key
func extractConsoleAPIKey(c *gin.Context) *model.ApiKey {
	if keyInfo, exists := c.Get("api_key"); exists {
		return keyInfo.(*model.ApiKey)
	}
	return nil
}

// parseConsoleRequest 解析Console请求
func parseConsoleRequest(c *gin.Context) ([]byte, error) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, appendConsoleErrorMessage(consoleErrRequestBodyRead, err.Error()))
		return nil, err
	}

	body, _ = sjson.SetBytes(body, "stream", true)
	body, _ = sjson.SetBytes(body, "metadata.user_id", common.GetInstanceID())
	return body, nil
}

// createConsoleHTTPClient 创建Console HTTP客户端
func createConsoleHTTPClient(account *model.Account) *http.Client {
	timeout := parseConsoleHTTPTimeout()

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	if account.ProxyURI != "" {
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

// parseConsoleHTTPTimeout 解析Console HTTP超时时间
func parseConsoleHTTPTimeout() time.Duration {
	if timeoutStr := os.Getenv("HTTP_CLIENT_TIMEOUT"); timeoutStr != "" {
		if timeout, err := time.ParseDuration(timeoutStr + "s"); err == nil {
			return timeout
		}
	}
	return consoleDefaultTimeout
}

// createConsoleRequest 创建Console请求
func createConsoleRequest(c *gin.Context, body []byte, account *model.Account) (*http.Request, error) {
	requestURL := account.RequestURL + "/v1/messages"

	req, err := http.NewRequestWithContext(
		c.Request.Context(),
		c.Request.Method,
		requestURL,
		bytes.NewBuffer(body),
	)
	if err != nil {
		return nil, err
	}

	copyConsoleRequestHeaders(c, req)
	setConsoleAPIHeaders(req, account.SecretKey)
	setConsoleStreamHeaders(c, req)

	return req, nil
}

// copyConsoleRequestHeaders 复制Console原始请求头
func copyConsoleRequestHeaders(c *gin.Context, req *http.Request) {
	for name, values := range c.Request.Header {
		for _, value := range values {
			req.Header.Add(name, value)
		}
	}
}

// setConsoleAPIHeaders 设置Console API请求头
func setConsoleAPIHeaders(req *http.Request, secretKey string) {
	fixedHeaders := buildConsoleAPIHeaders(secretKey)
	for name, value := range fixedHeaders {
		req.Header.Set(name, value)
	}
}

// buildConsoleAPIHeaders 构建Console API请求头
func buildConsoleAPIHeaders(secretKey string) map[string]string {
	return map[string]string{
		"x-api-key":                                 secretKey,
		"Authorization":                             "Bearer " + secretKey,
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

// setConsoleStreamHeaders 设置Console流式请求头
func setConsoleStreamHeaders(c *gin.Context, req *http.Request) {
	if c.Request.Header.Get("Accept") == "" {
		req.Header.Set("Accept", "text/event-stream")
	}
}

// handleConsoleRequestError 处理Console请求错误
func handleConsoleRequestError(c *gin.Context, err error) {
	if errors.Is(err, context.Canceled) {
		c.JSON(http.StatusRequestTimeout, consoleErrTimeout)
		return
	}

	log.Println("request conversation failed:", err.Error())
	c.JSON(http.StatusInternalServerError, appendConsoleErrorMessage(consoleErrNetworkError, err.Error()))
}

// createConsoleResponseReader 创建Console响应读取器（处理压缩）
func createConsoleResponseReader(resp *http.Response) (io.Reader, error) {
	contentEncoding := resp.Header.Get("Content-Encoding")

	switch strings.ToLower(contentEncoding) {
	case "gzip":
		gzipReader, err := gzip.NewReader(resp.Body)
		if err != nil {
			log.Printf("[Claude Console] 创建gzip解压缩器失败: %v", err)
			return nil, err
		}
		return gzipReader, nil
	case "deflate":
		return flate.NewReader(resp.Body), nil
	default:
		return resp.Body, nil
	}
}

// handleConsoleSuccessResponse 处理Console成功响应
func handleConsoleSuccessResponse(c *gin.Context, resp *http.Response, responseReader io.Reader) *common.TokenUsage {
	c.Status(resp.StatusCode)
	copyConsoleResponseHeaders(c, resp)
	setConsoleStreamResponseHeaders(c)

	c.Writer.Flush()

	usageTokens, err := common.ParseStreamResponse(c.Writer, responseReader)
	if err != nil {
		log.Println("stream copy and parse failed:", err.Error())
	}

	return usageTokens
}

// copyConsoleResponseHeaders 复制Console响应头
func copyConsoleResponseHeaders(c *gin.Context, resp *http.Response) {
	for name, values := range resp.Header {
		if strings.ToLower(name) != "content-length" {
			for _, value := range values {
				c.Header(name, value)
			}
		}
	}
}

// setConsoleStreamResponseHeaders 设置Console流式响应头
func setConsoleStreamResponseHeaders(c *gin.Context) {
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	if c.Writer.Header().Get("Content-Type") == "" {
		c.Header("Content-Type", "text/event-stream")
	}
}

// saveConsoleRequestLog 保存Console请求日志
func saveConsoleRequestLog(startTime time.Time, apiKey *model.ApiKey, account *model.Account, statusCode int, usageTokens *common.TokenUsage) {
	if statusCode >= consoleStatusOK && statusCode < 300 && usageTokens != nil && apiKey != nil {
		duration := time.Since(startTime).Milliseconds()
		logService := service.NewLogService()
		go func() {
			_, err := logService.CreateLogFromTokenUsage(usageTokens, apiKey.UserID, apiKey.ID, account.ID, duration, true)
			if err != nil {
				log.Printf("保存日志失败: %v", err)
			}
		}()
	}
}

// appendConsoleErrorMessage 为Console错误消息追加详细信息
func appendConsoleErrorMessage(baseError gin.H, message string) gin.H {
	errorMap := baseError["error"].(map[string]any)
	errorMap["message"] = errorMap["message"].(string) + ": " + message
	return gin.H{"error": errorMap}
}

// TestHandleClaudeConsoleRequest 测试处理Claude Console请求的函数
func TestHandleClaudeConsoleRequest(account *model.Account) (int, string) {
	body, _ := sjson.SetBytes([]byte(TestRequestBody), "stream", true)

	req, err := http.NewRequest("POST", account.RequestURL+"/v1/messages?beta=true", bytes.NewBuffer(body))
	if err != nil {
		return http.StatusInternalServerError, "Failed to create request: " + err.Error()
	}

	fixedHeaders := buildConsoleAPIHeaders(account.SecretKey)
	fixedHeaders["Content-Type"] = "application/json"
	fixedHeaders["Accept"] = "text/event-stream"

	for name, value := range fixedHeaders {
		req.Header.Set(name, value)
	}

	client := createConsoleHTTPClient(account)
	if client == nil {
		return http.StatusInternalServerError, "Failed to create HTTP client"
	}

	resp, err := client.Do(req)
	if err != nil {
		return http.StatusInternalServerError, "Request failed: " + err.Error()
	}
	defer common.CloseIO(resp.Body)

	return resp.StatusCode, ""
}
