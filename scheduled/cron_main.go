package scheduled

import (
	"claude-code-relay/common"
	"claude-code-relay/constant"
	"claude-code-relay/model"
	"claude-code-relay/relay"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/robfig/cron/v3"
)

// CronService 定时任务服务
type CronService struct {
	cron *cron.Cron
	jobs map[string]cron.EntryID
}

// NewCronService 创建定时任务服务
func NewCronService() *CronService {
	return &CronService{
		cron: cron.New(cron.WithSeconds()),
		jobs: make(map[string]cron.EntryID),
	}
}

// Start 启动所有定时任务
func (s *CronService) Start() error {
	// 任务配置表 - 更清晰的配置方式
	tasks := map[string]struct {
		spec    string
		handler func() error
	}{
		"reset_daily":    {"0 0 0 * * *", s.resetDailyStats},
		"clean_logs":     {"0 0 1 * * *", s.cleanExpiredLogs},
		"recover_abnormal": {"0 */30 * * * *", s.recoverAbnormalAccounts},
		"check_rate_limit": {"0 */10 * * * *", s.checkRateLimitExpiredAccounts},
		"refresh_tokens":   {"0 */15 * * * *", s.refreshExpiredTokens},
	}

	for name, task := range tasks {
		entryID, err := s.cron.AddFunc(task.spec, s.wrapTask(name, task.handler))
		if err != nil {
			return fmt.Errorf("添加任务 %s 失败: %w", name, err)
		}
		s.jobs[name] = entryID
	}

	s.cron.Start()
	common.SysLog("定时任务服务启动成功")
	return nil
}

// Stop 停止服务
func (s *CronService) Stop() {
	if s.cron != nil {
		ctx := s.cron.Stop()
		<-ctx.Done()
		common.SysLog("定时任务服务已停止")
	}
}

// wrapTask 包装任务，统一处理日志和错误
func (s *CronService) wrapTask(name string, handler func() error) func() {
	return func() {
		start := time.Now()
		common.SysLog(fmt.Sprintf("[%s] 开始执行", name))
		
		if err := handler(); err != nil {
			common.SysError(fmt.Sprintf("[%s] 执行失败: %v", name, err))
			return
		}
		
		common.SysLog(fmt.Sprintf("[%s] 执行完成，耗时 %v", name, time.Since(start)))
	}
}

// resetDailyStats 重置每日统计
func (s *CronService) resetDailyStats() error {
	// 统一的重置字段
	resetFields := map[string]any{
		"today_usage_count":                 0,
		"today_input_tokens":                0,
		"today_output_tokens":               0,
		"today_cache_read_input_tokens":     0,
		"today_cache_creation_input_tokens": 0,
		"today_total_cost":                  0,
	}

	// 批量重置
	for modelName, modelType := range map[string]interface{}{
		"accounts": &model.Account{},
		"api_keys": &model.ApiKey{},
	} {
		result := model.DB.Model(modelType).Where("1 = 1").Updates(resetFields)
		if result.Error != nil {
			return fmt.Errorf("重置 %s 统计失败: %w", modelName, result.Error)
		}
		common.SysLog(fmt.Sprintf("已重置 %d 个 %s 的统计数据", result.RowsAffected, modelName))
	}
	
	return nil
}

// cleanExpiredLogs 清理过期日志
func (s *CronService) cleanExpiredLogs() error {
	retentionMonths := getEnvInt("LOG_RETENTION_MONTHS", 3)
	expiredDate := time.Now().AddDate(0, -retentionMonths, 0)
	
	result := model.DB.Where("created_at < ?", expiredDate).Delete(&model.Log{})
	if result.Error != nil {
		return fmt.Errorf("清理日志失败: %w", result.Error)
	}
	
	common.SysLog(fmt.Sprintf("已清理 %d 条过期日志（%d个月前）", result.RowsAffected, retentionMonths))
	return nil
}

// recoverAbnormalAccounts 恢复异常账号
func (s *CronService) recoverAbnormalAccounts() error {
	var accounts []model.Account
	if err := model.DB.Where("current_status = 2 AND active_status = 1").Find(&accounts).Error; err != nil {
		return fmt.Errorf("查询异常账号失败: %w", err)
	}

	if len(accounts) == 0 {
		return nil
	}

	recovered := 0
	for _, acc := range accounts {
		if s.testAccount(&acc) {
			if err := model.DB.Model(&acc).Update("current_status", 1).Error; err != nil {
				common.SysError(fmt.Sprintf("更新账号 %s 状态失败: %v", acc.Name, err))
				continue
			}
			recovered++
		}
	}

	common.SysLog(fmt.Sprintf("恢复了 %d/%d 个异常账号", recovered, len(accounts)))
	return nil
}

// checkRateLimitExpiredAccounts 检查限流过期账号
func (s *CronService) checkRateLimitExpiredAccounts() error {
	now := time.Now()
	result := model.DB.Model(&model.Account{}).
		Where("current_status = 3 AND active_status = 1 AND rate_limit_end_time < ?", now).
		Updates(map[string]any{
			"current_status": 1,
			"rate_limit_end_time": nil,
		})
		
	if result.Error != nil {
		return fmt.Errorf("恢复限流账号失败: %w", result.Error)
	}
	
	if result.RowsAffected > 0 {
		common.SysLog(fmt.Sprintf("已恢复 %d 个限流账号", result.RowsAffected))
	}
	
	return nil
}

// refreshExpiredTokens 刷新即将过期的Token
func (s *CronService) refreshExpiredTokens() error {
	var accounts []model.Account
	threshold := time.Now().Unix() + 300 // 提前5分钟
	
	err := model.DB.Where(
		"active_status = 1 AND platform_type = ? AND expires_at > 0 AND expires_at <= ?",
		constant.PlatformClaude, threshold,
	).Find(&accounts).Error
	
	if err != nil {
		return fmt.Errorf("查询待刷新账号失败: %w", err)
	}

	if len(accounts) == 0 {
		return nil
	}

	refreshed := 0
	for _, acc := range accounts {
		if _, err := relay.GetValidAccessToken(&acc); err != nil {
			common.SysError(fmt.Sprintf("刷新账号 %s Token失败: %v", acc.Name, err))
			continue
		}
		refreshed++
	}

	common.SysLog(fmt.Sprintf("Token刷新: %d个需要刷新，%d个成功", len(accounts), refreshed))
	return nil
}

// testAccount 测试账号是否正常
func (s *CronService) testAccount(account *model.Account) bool {
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
		return false
	}

	return errMsg == "" && statusCode >= 200 && statusCode < 300
}

// getEnvInt 获取环境变量整数值
func getEnvInt(key string, defaultValue int) int {
	if val := os.Getenv(key); val != "" {
		if num, err := strconv.Atoi(val); err == nil && num > 0 {
			return num
		}
	}
	return defaultValue
}

// 单例实例和初始化函数
var instance *CronService

// Init 初始化定时任务服务
func Init() error {
	if instance != nil {
		return nil
	}
	
	instance = NewCronService()
	return instance.Start()
}

// Shutdown 关闭定时任务服务
func Shutdown() {
	if instance != nil {
		instance.Stop()
		instance = nil
	}
}

// GetInstance 获取服务实例（用于手动触发任务）
func GetInstance() *CronService {
	return instance
}

// ManualTrigger 手动触发指定任务
func (s *CronService) ManualTrigger(taskName string) error {
	tasks := map[string]func() error{
		"reset_daily":     s.resetDailyStats,
		"clean_logs":      s.cleanExpiredLogs,
		"recover_abnormal": s.recoverAbnormalAccounts,
		"check_rate_limit": s.checkRateLimitExpiredAccounts,
		"refresh_tokens":   s.refreshExpiredTokens,
	}
	
	handler, ok := tasks[taskName]
	if !ok {
		return fmt.Errorf("未知任务: %s", taskName)
	}
	
	common.SysLog(fmt.Sprintf("手动触发任务: %s", taskName))
	return handler()
}