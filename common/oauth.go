package common

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// OAuthConfig OAuth配置常量
type OAuthConfig struct {
	AuthorizeURL string
	TokenURL     string
	ClientID     string
	RedirectURI  string
	Scopes       string
}

// DefaultOAuthConfig 默认OAuth配置
var DefaultOAuthConfig = &OAuthConfig{
	AuthorizeURL: "https://claude.ai/oauth/authorize",
	TokenURL:     "https://console.anthropic.com/v1/oauth/token",
	ClientID:     "9d1c250a-e61b-44d9-88ed-5944d1962f5e",
	RedirectURI:  "https://console.anthropic.com/oauth/code/callback",
	Scopes:       "org:create_api_key user:profile user:inference",
}

// OAuthParams OAuth参数结构
type OAuthParams struct {
	AuthURL       string `json:"auth_url"`
	CodeVerifier  string `json:"code_verifier"`
	State         string `json:"state"`
	CodeChallenge string `json:"code_challenge"`
}

// TokenResponse token响应结构
type TokenResponse struct {
	AccessToken  string   `json:"access_token"`
	RefreshToken string   `json:"refresh_token"`
	ExpiresAt    int64    `json:"expires_at"`
	Scopes       []string `json:"scopes"`
	IsMax        bool     `json:"is_max"`
}

// ClaudeCredentials Claude凭证格式
type ClaudeCredentials struct {
	ClaudeAiOauth *TokenResponse `json:"claudeAiOauth"`
}

// OAuthHelper OAuth助手类
type OAuthHelper struct {
	config *OAuthConfig
}

// NewOAuthHelper 创建OAuth助手实例
func NewOAuthHelper(config *OAuthConfig) *OAuthHelper {
	if config == nil {
		config = DefaultOAuthConfig
	}
	return &OAuthHelper{config: config}
}

// GenerateState 生成随机的state参数
func (o *OAuthHelper) GenerateState() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate state: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}

// GenerateCodeVerifier 生成随机的code verifier（PKCE）
func (o *OAuthHelper) GenerateCodeVerifier() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate code verifier: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

// GenerateCodeChallenge 生成code challenge（PKCE）
func (o *OAuthHelper) GenerateCodeChallenge(codeVerifier string) string {
	hash := sha256.Sum256([]byte(codeVerifier))
	return base64.RawURLEncoding.EncodeToString(hash[:])
}

// GenerateAuthURL 生成授权URL
func (o *OAuthHelper) GenerateAuthURL(codeChallenge, state string) string {
	params := url.Values{
		"code":                  {"true"},
		"client_id":             {o.config.ClientID},
		"response_type":         {"code"},
		"redirect_uri":          {o.config.RedirectURI},
		"scope":                 {o.config.Scopes},
		"code_challenge":        {codeChallenge},
		"code_challenge_method": {"S256"},
		"state":                 {state},
	}

	return fmt.Sprintf("%s?%s", o.config.AuthorizeURL, params.Encode())
}

// GenerateOAuthParams 生成OAuth授权URL和相关参数
func (o *OAuthHelper) GenerateOAuthParams() (*OAuthParams, error) {
	state, err := o.GenerateState()
	if err != nil {
		return nil, fmt.Errorf("failed to generate state: %w", err)
	}

	codeVerifier, err := o.GenerateCodeVerifier()
	if err != nil {
		return nil, fmt.Errorf("failed to generate code verifier: %w", err)
	}

	codeChallenge := o.GenerateCodeChallenge(codeVerifier)
	authURL := o.GenerateAuthURL(codeChallenge, state)

	return &OAuthParams{
		AuthURL:       authURL,
		CodeVerifier:  codeVerifier,
		State:         state,
		CodeChallenge: codeChallenge,
	}, nil
}

// ParseCallbackURL 解析回调URL或授权码
func (o *OAuthHelper) ParseCallbackURL(input string) (string, error) {
	if input == "" {
		return "", fmt.Errorf("请提供有效的授权码或回调URL")
	}

	input = strings.TrimSpace(input)

	// 情况1: 尝试作为完整URL解析
	if strings.HasPrefix(input, "http://") || strings.HasPrefix(input, "https://") {
		parsedURL, err := url.Parse(input)
		if err != nil {
			return "", fmt.Errorf("无效的URL格式，请检查回调URL是否正确: %w", err)
		}

		code := parsedURL.Query().Get("code")
		if code == "" {
			return "", fmt.Errorf("回调URL中未找到授权码(code参数)")
		}

		return code, nil
	}

	// 情况2: 直接的授权码（可能包含URL fragments）
	// 移除URL fragments和参数
	parts := strings.Split(input, "#")
	if len(parts) > 0 {
		parts = strings.Split(parts[0], "&")
	}

	cleanedCode := ""
	if len(parts) > 0 {
		cleanedCode = parts[0]
	}

	if cleanedCode == "" || len(cleanedCode) < 10 {
		return "", fmt.Errorf("授权码格式无效，请确保复制了完整的Authorization Code")
	}

	// 基本格式验证：授权码应该只包含字母、数字、下划线、连字符
	for _, r := range cleanedCode {
		if !((r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') ||
			(r >= '0' && r <= '9') || r == '_' || r == '-') {
			return "", fmt.Errorf("授权码包含无效字符，请检查是否复制了正确的Authorization Code")
		}
	}

	return cleanedCode, nil
}

// CreateTokenExchangeParams 创建token交换参数
func (o *OAuthHelper) CreateTokenExchangeParams(authorizationCode, codeVerifier, state string) map[string]any {
	// 清理授权码，移除URL片段
	cleanedCode := strings.Split(strings.Split(authorizationCode, "#")[0], "&")[0]

	return map[string]any{
		"grant_type":    "authorization_code",
		"client_id":     o.config.ClientID,
		"code":          cleanedCode,
		"redirect_uri":  o.config.RedirectURI,
		"code_verifier": codeVerifier,
		"state":         state,
	}
}

// GetTokenExchangeHeaders 获取token交换请求头
func (o *OAuthHelper) GetTokenExchangeHeaders() map[string]string {
	return map[string]string{
		"Content-Type":    "application/json",
		"User-Agent":      "claude-cli/1.0.56 (external, cli)",
		"Accept":          "application/json, text/plain, */*",
		"Accept-Language": "en-US,en;q=0.9",
		"Referer":         "https://claude.ai/",
		"Origin":          "https://claude.ai",
	}
}

// FormatTokenResponse 格式化token响应为标准格式
func (o *OAuthHelper) FormatTokenResponse(accessToken, refreshToken string, expiresIn int, scopes string) *TokenResponse {
	scopeList := strings.Fields(scopes)
	if len(scopeList) == 0 {
		scopeList = []string{"user:inference", "user:profile"}
	}

	expiresAt := time.Now().Unix() + int64(expiresIn)

	return &TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    expiresAt,
		Scopes:       scopeList,
		IsMax:        true,
	}
}

// FormatClaudeCredentials 格式化为Claude标准格式
func (o *OAuthHelper) FormatClaudeCredentials(tokenData *TokenResponse) *ClaudeCredentials {
	return &ClaudeCredentials{
		ClaudeAiOauth: tokenData,
	}
}

// ValidateState 验证state参数
func (o *OAuthHelper) ValidateState(receivedState, expectedState string) bool {
	return receivedState == expectedState
}

// IsTokenExpired 检查token是否过期
func (o *OAuthHelper) IsTokenExpired(expiresAt int64) bool {
	return time.Now().UnixMilli() >= expiresAt
}

// ExchangeCodeForTokens 使用授权码交换访问令牌
func (o *OAuthHelper) ExchangeCodeForTokens(authorizationCode, codeVerifier, state, proxyURI string) (*TokenResponse, error) {
	// 创建HTTP客户端
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// 如果提供了代理URI，配置代理
	if proxyURI != "" {
		proxyURL, err := url.Parse(proxyURI)
		if err != nil {
			return nil, fmt.Errorf("invalid proxy URI: %w", err)
		}
		transport := &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
		}
		client.Transport = transport
		SysLog(fmt.Sprintf("Using proxy: %s", proxyURI))
	}

	// 创建请求参数
	params := o.CreateTokenExchangeParams(authorizationCode, codeVerifier, state)

	// 将参数转换为JSON
	jsonData, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request params: %w", err)
	}

	// 创建HTTP请求
	req, err := http.NewRequest("POST", o.config.TokenURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// 设置请求头
	headers := o.GetTokenExchangeHeaders()
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	// 记录请求信息
	cleanedCode := o.CleanAuthorizationCode(authorizationCode)
	SysLog(fmt.Sprintf("Attempting OAuth token exchange - URL: %s, Code length: %d, Code prefix: %s..., Has proxy: %t",
		o.config.TokenURL, len(cleanedCode), cleanedCode[:min(10, len(cleanedCode))], proxyURI != ""))

	// 发送请求
	resp, err := client.Do(req)
	if err != nil {
		SysError(fmt.Sprintf("OAuth token exchange network error: %v", err))
		return nil, fmt.Errorf("token exchange failed: network error - %w", err)
	}
	defer resp.Body.Close()

	// 读取响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// 检查响应状态码
	if resp.StatusCode != http.StatusOK {
		SysError(fmt.Sprintf("OAuth token exchange failed - Status: %d, Body: %s", resp.StatusCode, string(body)))

		// 尝试解析错误响应
		var errorResp map[string]any
		if json.Unmarshal(body, &errorResp) == nil {
			if errorMsg, ok := errorResp["error"].(string); ok {
				errorDesc := ""
				if desc, exists := errorResp["error_description"].(string); exists {
					errorDesc = " - " + desc
				}
				return nil, fmt.Errorf("token exchange failed: HTTP %d: %s%s", resp.StatusCode, errorMsg, errorDesc)
			}
		}

		return nil, fmt.Errorf("token exchange failed: HTTP %d - %s", resp.StatusCode, string(body))
	}

	// 解析成功响应
	var tokenResp map[string]any
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("failed to parse token response: %w", err)
	}

	// 打印 json 字符串格式的响应
	SysLog(fmt.Sprintf("OAuth token exchange response: %s", string(body)))

	// 提取token信息
	accessToken, ok := tokenResp["access_token"].(string)
	if !ok || accessToken == "" {
		return nil, fmt.Errorf("access_token not found in response")
	}

	refreshToken, _ := tokenResp["refresh_token"].(string)

	expiresIn := 3600 // 默认1小时
	if exp, ok := tokenResp["expires_in"].(float64); ok {
		expiresIn = int(exp)
	}

	scopes := ""
	if scope, ok := tokenResp["scope"].(string); ok {
		scopes = scope
	}

	// 格式化token响应
	result := o.FormatTokenResponse(accessToken, refreshToken, expiresIn, scopes)

	SysLog(fmt.Sprintf("OAuth token exchange successful - Has access token: %t, Has refresh token: %t, Scopes: %v",
		result.AccessToken != "", result.RefreshToken != "", result.Scopes))

	return result, nil
}

// min 辅助函数，返回两个整数中的较小值
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// GetTokenURL 获取token交换URL
func (o *OAuthHelper) GetTokenURL() string {
	return o.config.TokenURL
}

// CleanAuthorizationCode 清理授权码
func (o *OAuthHelper) CleanAuthorizationCode(code string) string {
	return strings.Split(strings.Split(code, "#")[0], "&")[0]
}
