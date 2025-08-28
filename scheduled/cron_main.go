package scheduled

import (
	"claude-code-relay/common"
	"claude-code-relay/constant"
	"claude-code-relay/model"
	"claude-code-relay/relay"
	"claude-code-relay/service"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/robfig/cron/v3"
)

// 全局定时任务服务实例
var GlobalCronService *CronService

// CronService 定时任务服务
type CronService struct {
	cron *cron.Cron
}

// cronJob 定时任务配置
type cronJob struct {
	spec    string
	handler func()
	name    string
}

// NewCronService 创建定时任务服务实例
func NewCronService() *CronService {
	c := cron.New(cron.WithSeconds())
	return &CronService{cron: c}
}

// Start 启动定时任务
func (s *CronService) Start() {
	jobs := []cronJob{
		{"0 0 0 * * *", s.resetDailyStats, "daily stats reset"},
		{"0 0 1 * * *", s.cleanExpiredLogs, "log cleanup"},
		{"0 */30 * * * *", s.recoverAbnormalAccounts, "account recovery"},
		{"0 */10 * * * *", s.checkRateLimitExpiredAccounts, "rate limit check"},
		{"0 */15 * * * *", s.checkAndRefreshExpiredTokens, "token refresh"}, // 独立的Token刷新任务
	}

	for _, job := range jobs {
		if _, err := s.cron.AddFunc(job.spec, s.withLog(job.name, job.handler)); err != nil {
			log.Printf("Failed to add %s cron job: %v", job.name, err)
			return
		}
	}

	s.cron.Start()
	common.SysLog("Cron service started successfully")
}

// Stop 停止定时任务
func (s *CronService) Stop() {
	if s.cron != nil {
		ctx := s.cron.Stop()
		<-ctx.Done()
		common.SysLog("Cron service stopped")
	}
}

// withLog 包装任务函数，统一处理日志记录
func (s *CronService) withLog(taskName string, handler func()) func() {
	return func() {
		startTime := time.Now()
		common.SysLog(fmt.Sprintf("Starting %s task", taskName))
		handler()
		common.SysLog(fmt.Sprintf("%s task completed in %s", taskName, time.Since(startTime)))
	}
}

// resetDailyStats 重置每日统计数据
func (s *CronService) resetDailyStats() {
	if err := s.resetStats(&model.Account{}, "accounts"); err != nil {
		common.SysError("Failed to reset account daily stats: " + err.Error())
	}

	if err := s.resetStats(&model.ApiKey{}, "api keys"); err != nil {
		common.SysError("Failed to reset api key daily stats: " + err.Error())
	}
}

// resetStats 通用的统计数据重置函数
func (s *CronService) resetStats(modelType interface{}, entityName string) error {
	updates := map[string]any{
		"today_usage_count":                 0,
		"today_input_tokens":                0,
		"today_output_tokens":               0,
		"today_cache_read_input_tokens":     0,
		"today_cache_creation_input_tokens": 0,
		"today_total_cost":                  0,
	}

	result := model.DB.Model(modelType).Where("1 = 1").Updates(updates)
	if result.Error != nil {
		return result.Error
	}

	common.SysLog(fmt.Sprintf("Reset daily stats for %d %s", result.RowsAffected, entityName))
	return nil
}

// cleanExpiredLogs 清理过期日志
func (s *CronService) cleanExpiredLogs() {
	retentionMonths := getLogRetentionMonths()
	logService := service.NewLogService()
	
	deletedCount, err := logService.DeleteExpiredLogs(retentionMonths)
	if err != nil {
		common.SysError("Failed to clean expired logs: " + err.Error())
		return
	}
	
	common.SysLog(fmt.Sprintf("Cleaned %d expired logs (older than %d months)", deletedCount, retentionMonths))
}

// getLogRetentionMonths 从环境变量获取日志保留月数
func getLogRetentionMonths() int {
	monthsStr := os.Getenv("LOG_RETENTION_MONTHS")
	if monthsStr == "" {
		return 3 // 默认保留3个月
	}

	months, err := strconv.Atoi(monthsStr)
	if err != nil || months <= 0 {
		log.Printf("Invalid LOG_RETENTION_MONTHS value: %s, using default value 3", monthsStr)
		return 3
	}

	return months
}

// recoverAbnormalAccounts 恢复异常账号测试
func (s *CronService) recoverAbnormalAccounts() {
	accounts, err := s.queryAccounts(2, 1)
	if err != nil {
		common.SysError("Failed to query abnormal accounts: " + err.Error())
		return
	}

	if len(accounts) == 0 {
		common.SysLog("No abnormal accounts found for recovery testing")
		return
	}

	common.SysLog(fmt.Sprintf("Found %d abnormal accounts to test", len(accounts)))
	
	recoveredCount := 0
	for _, account := range accounts {
		if s.testAndRecover(&account) {
			recoveredCount++
			common.SysLog(fmt.Sprintf("Account %s (ID: %d) recovered successfully", account.Name, account.ID))
		}
	}

	common.SysLog(fmt.Sprintf("Recovered %d of %d abnormal accounts", recoveredCount, len(accounts)))
}

// checkRateLimitExpiredAccounts 检查限流过期账号
func (s *CronService) checkRateLimitExpiredAccounts() {
	accounts, err := s.queryAccounts(3, 1)
	if err != nil {
		common.SysError("Failed to query rate limited accounts: " + err.Error())
		return
	}

	if len(accounts) == 0 {
		common.SysLog("No rate limited accounts found for checking")
		return
	}

	common.SysLog(fmt.Sprintf("Found %d rate limited accounts to check", len(accounts)))
	
	recoveredCount := 0
	now := time.Now()

	for _, account := range accounts {
		if account.RateLimitEndTime != nil && now.After(time.Time(*account.RateLimitEndTime)) {
			if err := s.recoverRateLimit(&account); err != nil {
				common.SysError(fmt.Sprintf("Failed to recover rate limited account %s (ID: %d): %v", 
					account.Name, account.ID, err))
				continue
			}
			recoveredCount++
			common.SysLog(fmt.Sprintf("Rate limited account %s (ID: %d) recovered", account.Name, account.ID))
		}
	}

	if recoveredCount > 0 {
		common.SysLog(fmt.Sprintf("Recovered %d rate limited accounts", recoveredCount))
	}
}

// checkAndRefreshExpiredTokens 检查并刷新即将过期的Token
func (s *CronService) checkAndRefreshExpiredTokens() {
	var accounts []model.Account
	err := model.DB.Where("active_status = ? AND platform_type = ?", 1, constant.PlatformClaude).Find(&accounts).Error
	if err != nil {
		common.SysError("Failed to query active Claude accounts: " + err.Error())
		return
	}

	if len(accounts) == 0 {
		common.SysLog("No active Claude accounts found for token refresh")
		return
	}

	common.SysLog(fmt.Sprintf("Checking tokens for %d Claude accounts", len(accounts)))
	
	needRefreshCount := 0
	refreshedCount := 0
	failedCount := 0
	now := time.Now().Unix()
	
	for _, account := range accounts {
		// 检查是否需要刷新（提前5分钟）
		expiresAt := int64(account.ExpiresAt)
		if expiresAt > 0 && now >= (expiresAt-300) {
			needRefreshCount++
			common.SysLog(fmt.Sprintf("Account %s (ID: %d) token expires at %d, needs refresh", 
				account.Name, account.ID, expiresAt))
			
			if _, err := relay.GetValidAccessToken(&account); err != nil {
				common.SysError(fmt.Sprintf("Failed to refresh token for account %s (ID: %d): %v", 
					account.Name, account.ID, err))
				failedCount++
			} else {
				refreshedCount++
				common.SysLog(fmt.Sprintf("Successfully refreshed token for account %s (ID: %d)", 
					account.Name, account.ID))
			}
		}
	}

	if needRefreshCount == 0 {
		common.SysLog(fmt.Sprintf("Checked %d accounts, no tokens need refreshing", len(accounts)))
	} else {
		common.SysLog(fmt.Sprintf("Token refresh summary: %d accounts checked, %d needed refresh, %d refreshed, %d failed", 
			len(accounts), needRefreshCount, refreshedCount, failedCount))
	}
}

// queryAccounts 查询指定状态的账号
func (s *CronService) queryAccounts(currentStatus, activeStatus int) ([]model.Account, error) {
	var accounts []model.Account
	err := model.DB.Where("current_status = ? AND active_status = ?", currentStatus, activeStatus).Find(&accounts).Error
	return accounts, err
}

// testAndRecover 测试并恢复单个账号
func (s *CronService) testAndRecover(account *model.Account) bool {
	var statusCode int
	var errMsg string

	switch account.PlatformType {
	case constant.PlatformClaude:
		statusCode, errMsg = relay.TestsHandleClaudeRequest(account)
	case constant.PlatformClaudeConsole:
		statusCode, errMsg = relay.TestHandleClaudeConsoleRequest(account)
	case constant.PlatformOpenAI:
		statusCode, errMsg = relay.TestHandleOpenAIRequest(account)
	default:
		common.SysError(fmt.Sprintf("Unsupported platform type: %s", account.PlatformType))
		return false
	}

	if errMsg == "" && statusCode >= 200 && statusCode < 300 {
		if updateErr := model.DB.Model(account).Update("current_status", 1).Error; updateErr != nil {
			common.SysError(fmt.Sprintf("Failed to update account status: %v", updateErr))
			return false
		}
		return true
	}

	return false
}

// recoverRateLimit 恢复限流账号
func (s *CronService) recoverRateLimit(account *model.Account) error {
	return model.DB.Model(account).Updates(map[string]any{
		"current_status":      1,
		"rate_limit_end_time": nil,
	}).Error
}

// ManualResetStats 手动重置统计数据
func (s *CronService) ManualResetStats() error {
	common.SysLog("Manual daily stats reset triggered")
	s.resetDailyStats()
	return nil
}

// ManualCleanExpiredLogs 手动清理过期日志
func (s *CronService) ManualCleanExpiredLogs() (int64, error) {
	common.SysLog("Manual expired logs cleanup triggered")
	
	retentionMonths := getLogRetentionMonths()
	logService := service.NewLogService()
	deletedCount, err := logService.DeleteExpiredLogs(retentionMonths)
	
	if err == nil {
		common.SysLog(fmt.Sprintf("Manual cleanup deleted %d records", deletedCount))
	}
	
	return deletedCount, err
}

// InitCronService 初始化全局定时任务服务
func InitCronService() {
	GlobalCronService = NewCronService()
	GlobalCronService.Start()
}

// StopCronService 停止全局定时任务服务
func StopCronService() {
	if GlobalCronService != nil {
		GlobalCronService.Stop()
	}
}