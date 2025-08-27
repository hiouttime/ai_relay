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

// NewCronService 创建定时任务服务实例
func NewCronService() *CronService {
	// 使用带秒的cron解析器
	c := cron.New(cron.WithSeconds())
	return &CronService{cron: c}
}

// Start 启动定时任务
func (s *CronService) Start() {
	// 每天凌晨0点清理统计数据
	_, err := s.cron.AddFunc("0 0 0 * * *", s.resetDailyStats)
	if err != nil {
		log.Printf("Failed to add daily reset cron job: %v", err)
		return
	}

	// 每天凌晨1点清理过期日志
	_, err = s.cron.AddFunc("0 0 1 * * *", s.cleanExpiredLogs)
	if err != nil {
		log.Printf("Failed to add log cleanup cron job: %v", err)
		return
	}

	// 每30分钟执行一次账号异常恢复测试
	_, err = s.cron.AddFunc("0 */30 * * * *", s.recoverAbnormalAccounts)
	if err != nil {
		log.Printf("Failed to add account recovery cron job: %v", err)
		return
	}

	// 每10分钟检查限流过期账号
	_, err = s.cron.AddFunc("0 */10 * * * *", s.checkRateLimitExpiredAccounts)
	if err != nil {
		log.Printf("Failed to add rate limit check cron job: %v", err)
		return
	}

	// 启动定时任务
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

// resetDailyStats 重置每日统计数据
func (s *CronService) resetDailyStats() {
	startTime := time.Now()
	common.SysLog("Starting daily stats reset task")

	// 重置Account表的今日统计数据
	err := s.resetAccountStats()
	if err != nil {
		common.SysError("Failed to reset account daily stats: " + err.Error())
	} else {
		common.SysLog("Account daily stats reset successfully")
	}

	// 重置ApiKey表的今日统计数据
	err = s.resetApiKeyStats()
	if err != nil {
		common.SysError("Failed to reset api key daily stats: " + err.Error())
	} else {
		common.SysLog("API Key daily stats reset successfully")
	}

	duration := time.Since(startTime)
	common.SysLog("Daily stats reset task completed in " + duration.String())
}

// resetAccountStats 重置账户今日统计数据
func (s *CronService) resetAccountStats() error {
	result := model.DB.Model(&model.Account{}).Where("1 = 1").Updates(map[string]any{
		"today_usage_count":                 0,
		"today_input_tokens":                0,
		"today_output_tokens":               0,
		"today_cache_read_input_tokens":     0,
		"today_cache_creation_input_tokens": 0,
		"today_total_cost":                  0,
	})

	if result.Error != nil {
		return result.Error
	}

	log.Printf("Reset daily stats for %d accounts", result.RowsAffected)
	return nil
}

// resetApiKeyStats 重置API Key今日统计数据
func (s *CronService) resetApiKeyStats() error {
	result := model.DB.Model(&model.ApiKey{}).Where("1 = 1").Updates(map[string]any{
		"today_usage_count":                 0,
		"today_input_tokens":                0,
		"today_output_tokens":               0,
		"today_cache_read_input_tokens":     0,
		"today_cache_creation_input_tokens": 0,
		"today_total_cost":                  0,
	})

	if result.Error != nil {
		return result.Error
	}

	log.Printf("Reset daily stats for %d api keys", result.RowsAffected)
	return nil
}

// ManualResetStats 手动重置统计数据（用于测试或管理员操作）
func (s *CronService) ManualResetStats() error {
	common.SysLog("Manual daily stats reset triggered")

	err := s.resetAccountStats()
	if err != nil {
		return err
	}

	err = s.resetApiKeyStats()
	if err != nil {
		return err
	}

	common.SysLog("Manual daily stats reset completed")
	return nil
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

// cleanExpiredLogs 清理过期日志
func (s *CronService) cleanExpiredLogs() {
	startTime := time.Now()
	common.SysLog("Starting expired logs cleanup task")

	// 从环境变量获取日志保留月数，默认为3个月
	retentionMonths := getLogRetentionMonths()

	logService := service.NewLogService()
	deletedCount, err := logService.DeleteExpiredLogs(retentionMonths)
	if err != nil {
		common.SysError("Failed to clean expired logs: " + err.Error())
	} else {
		common.SysLog("Cleaned expired logs successfully, deleted " + strconv.FormatInt(deletedCount, 10) + " records (older than " + strconv.Itoa(retentionMonths) + " months)")
	}

	duration := time.Since(startTime)
	common.SysLog("Expired logs cleanup task completed in " + duration.String())
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
	startTime := time.Now()
	common.SysLog("Starting abnormal accounts recovery task")

	// 筛选current_status==2且active_status==1的账号
	var abnormalAccounts []model.Account
	err := model.DB.Where("current_status = ? AND active_status = ?", 2, 1).Find(&abnormalAccounts).Error
	if err != nil {
		common.SysError("Failed to query abnormal accounts: " + err.Error())
		return
	}

	if len(abnormalAccounts) == 0 {
		common.SysLog("No abnormal accounts found for recovery testing")
		return
	}

	common.SysLog(fmt.Sprintf("Found %d abnormal accounts to test", len(abnormalAccounts)))

	recoveredCount := 0
	failedCount := 0

	// 逐个测试异常账号
	for _, account := range abnormalAccounts {
		if s.testAndRecoverAccount(&account) {
			recoveredCount++
			common.SysLog(fmt.Sprintf("Account %s (ID: %d) recovered successfully", account.Name, account.ID))
		} else {
			failedCount++
		}
	}

	duration := time.Since(startTime)
	common.SysLog(fmt.Sprintf("Abnormal accounts recovery task completed in %s. Recovered: %d, Failed: %d", duration.String(), recoveredCount, failedCount))
}

// testAndRecoverAccount 测试并恢复单个账号
func (s *CronService) testAndRecoverAccount(account *model.Account) bool {
	var statusCode int
	var err string

	// 根据平台类型调用不同的测试函数
	switch account.PlatformType {
	case constant.PlatformClaude:
		statusCode, err = relay.TestsHandleClaudeRequest(account)
	case constant.PlatformClaudeConsole:
		statusCode, err = relay.TestHandleClaudeConsoleRequest(account)
	case constant.PlatformOpenAI:
		statusCode, err = relay.TestHandleOpenAIRequest(account)
	default:
		common.SysError(fmt.Sprintf("Unsupported platform type for account %s (ID: %d): %s", account.Name, account.ID, account.PlatformType))
		return false
	}

	common.SysLog(fmt.Sprintf("Testing account %s (ID: %d) with status code: %d, error: %s", account.Name, account.ID, statusCode, err))

	// 检查测试结果：状态码在200-300之间且无错误视为成功
	if err == "" && statusCode >= 200 && statusCode < 300 {
		// 测试成功，恢复账号状态为正常
		updateErr := model.DB.Model(account).Update("current_status", 1).Error
		if updateErr != nil {
			common.SysError(fmt.Sprintf("Failed to recover account %s (ID: %d): %v", account.Name, account.ID, updateErr))
			return false
		}
		return true
	} else {
		// 测试失败，记录日志但不改变状态
		if err != "" {
			common.SysLog(fmt.Sprintf("Account %s (ID: %d) test failed: %s", account.Name, account.ID, err))
		} else {
			common.SysLog(fmt.Sprintf("Account %s (ID: %d) test failed with status code: %d", account.Name, account.ID, statusCode))
		}
		return false
	}
}

// ManualCleanExpiredLogs 手动清理过期日志（用于测试或管理员操作）
func (s *CronService) ManualCleanExpiredLogs() (int64, error) {
	common.SysLog("Manual expired logs cleanup triggered")

	retentionMonths := getLogRetentionMonths()
	logService := service.NewLogService()
	deletedCount, err := logService.DeleteExpiredLogs(retentionMonths)
	if err != nil {
		return 0, err
	}

	common.SysLog("Manual expired logs cleanup completed, deleted " + strconv.FormatInt(deletedCount, 10) + " records")
	return deletedCount, nil
}

// checkRateLimitExpiredAccounts 检查限流过期账号和Token过期情况
func (s *CronService) checkRateLimitExpiredAccounts() {
	startTime := time.Now()
	common.SysLog("Starting rate limit expired accounts and token expiry check task")

	// 筛选current_status==3且active_status==1的账号
	var rateLimitedAccounts []model.Account
	err := model.DB.Where("current_status = ? AND active_status = ?", 3, 1).Find(&rateLimitedAccounts).Error
	if err != nil {
		common.SysError("Failed to query rate limited accounts: " + err.Error())
		return
	}

	if len(rateLimitedAccounts) == 0 {
		common.SysLog("No rate limited accounts found for checking")
		return
	}

	common.SysLog(fmt.Sprintf("Found %d rate limited accounts to check", len(rateLimitedAccounts)))

	recoveredCount := 0
	now := time.Now()

	// 检查每个限流账号的限流结束时间
	for _, account := range rateLimitedAccounts {
		// 检查限流结束时间是否已过期
		if account.RateLimitEndTime != nil && now.After(time.Time(*account.RateLimitEndTime)) {
			// 限流时间已过，将账号状态恢复为正常并清空限流结束时间
			err := model.DB.Model(&account).Updates(map[string]any{
				"current_status":      1,
				"rate_limit_end_time": nil,
			}).Error
			if err != nil {
				common.SysError(fmt.Sprintf("Failed to recover rate limited account %s (ID: %d): %v", account.Name, account.ID, err))
				continue
			}
			recoveredCount++
			common.SysLog(fmt.Sprintf("Rate limited account %s (ID: %d) recovered successfully, limit expired at %v", account.Name, account.ID, account.RateLimitEndTime))
		} else if account.RateLimitEndTime != nil {
			common.SysLog(fmt.Sprintf("Account %s (ID: %d) still rate limited until %v", account.Name, account.ID, account.RateLimitEndTime))
		} else {
			common.SysLog(fmt.Sprintf("Account %s (ID: %d) has no rate limit end time, skipping", account.Name, account.ID))
		}
	}

	duration := time.Since(startTime)
	common.SysLog(fmt.Sprintf("Rate limit expired accounts check task completed in %s. Recovered: %d", duration.String(), recoveredCount))
	
	// 检查并刷新即将过期的Token
	s.checkAndRefreshExpiredTokens()
}

// checkAndRefreshExpiredTokens 检查并刷新即将过期的Token
func (s *CronService) checkAndRefreshExpiredTokens() {
	startTime := time.Now()
	common.SysLog("Starting token expiry check and refresh task")
	
	// 查询所有启用的Claude账号
	var activeAccounts []model.Account
	err := model.DB.Where("active_status = ? AND platform_type = ?", 1, constant.PlatformClaude).Find(&activeAccounts).Error
	if err != nil {
		common.SysError("Failed to query active accounts for token refresh: " + err.Error())
		return
	}
	
	if len(activeAccounts) == 0 {
		common.SysLog("No active Claude accounts found for token refresh check")
		return
	}
	
	common.SysLog(fmt.Sprintf("Found %d active Claude accounts to check", len(activeAccounts)))
	
	refreshedCount := 0
	failedCount := 0
	
	for _, account := range activeAccounts {
		// 直接调用getValidAccessToken，它会自动处理过期检查和刷新
		_, err := relay.GetValidAccessToken(&account)
		if err != nil {
			common.SysError(fmt.Sprintf("Failed to refresh token for account %s (ID: %d): %v", 
				account.Name, account.ID, err))
			failedCount++
		} else {
			refreshedCount++
		}
	}
	
	duration := time.Since(startTime)
	common.SysLog(fmt.Sprintf("Token refresh task completed in %s. Checked: %d, Failed: %d", 
		duration.String(), refreshedCount, failedCount))
}
