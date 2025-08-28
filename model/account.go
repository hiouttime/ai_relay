package model

import (
	"gorm.io/gorm"
	"time"
)

type Account struct {
	ID                            uint           `json:"id" gorm:"primaryKey"`
	Name                          string         `json:"name" gorm:"type:varchar(100);not null;comment:账号名称"`
	PlatformType                  string         `json:"platform_type" gorm:"type:varchar(50);not null;comment:平台类型(claude/claude_console)"`
	RequestURL                    string         `json:"request_url" gorm:"type:varchar(500);comment:请求地址"`
	SecretKey                     string         `json:"secret_key" gorm:"type:text;comment:请求秘钥"`
	AccessToken                   string         `json:"access_token" gorm:"type:text;comment:claude的官方token"`
	RefreshToken                  string         `json:"refresh_token" gorm:"type:text;comment:claude的官方刷新token"`
	ExpiresAt                     int            `json:"expires_at" gorm:"default:0;comment:token过期时间戳"`
	IsMax                         bool           `json:"is_max" gorm:"default:false;comment:是否是max账号"`
	GroupID                       int            `json:"group_id" gorm:"default:0;comment:分组ID"`
	Priority                      int            `json:"priority" gorm:"default:100;comment:优先级(数字越小越高)"`
	Weight                        int            `json:"weight" gorm:"default:100;comment:权重(数字越大越高)"`
	TodayUsageCount               int            `json:"today_usage_count" gorm:"default:0;comment:今日使用次数"`
	TodayInputTokens              int            `json:"today_input_tokens" gorm:"default:0;comment:今日输入tokens"`
	TodayOutputTokens             int            `json:"today_output_tokens" gorm:"default:0;comment:今日输出tokens"`
	TodayCacheReadInputTokens     int            `json:"today_cache_read_input_tokens" gorm:"default:0;comment:今日缓存读取输入tokens"`
	TodayCacheCreationInputTokens int            `json:"today_cache_creation_input_tokens" gorm:"default:0;comment:今日缓存创建输入tokens"`
	TodayTotalCost                float64        `json:"today_total_cost" gorm:"default:0;comment:今日使用总费用(USD)"`
	EnableProxy                   bool           `json:"enable_proxy" gorm:"default:false;comment:是否启用代理"`
	ProxyURI                      string         `json:"proxy_uri" gorm:"type:varchar(500);comment:代理URI字符串"`
	ModelMapping                  string         `json:"model_mapping" gorm:"type:text;comment:模型映射配置(格式:claude-model:openai-model,多个用逗号分隔)"`
	LastUsedTime                  *Time          `json:"last_used_time" gorm:"comment:最后使用时间;type:datetime"`
	RateLimitEndTime              *Time          `json:"rate_limit_end_time" gorm:"comment:限流结束时间;type:datetime"`
	CurrentStatus                 int            `json:"current_status" gorm:"default:1;comment:当前状态(1:正常,2:接口异常,3:账号异常/限流)"`
	ActiveStatus                  int            `json:"active_status" gorm:"default:1;comment:激活状态(1:激活,2:禁用)"`
	UserID                        uint           `json:"user_id" gorm:"not null;comment:所属用户ID"`
	CreatedAt                     Time           `json:"created_at" gorm:"type:datetime;default:CURRENT_TIMESTAMP"`
	UpdatedAt                     Time           `json:"updated_at" gorm:"type:datetime;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP"`
	DeletedAt                     gorm.DeletedAt `json:"-" gorm:"index"`

	// 关联查询
	User  User   `json:"user,omitempty" gorm:"foreignKey:UserID"`
	Group *Group `json:"group" gorm:"-"`

	// 最近一周统计数据（不存储到数据库，运行时计算）
	WeeklyCost  float64 `json:"weekly_cost" gorm:"-"`  // 最近一周使用费用
	WeeklyCount int64   `json:"weekly_count" gorm:"-"` // 最近一周使用次数
}

// 账号列表请求参数
type AccountListRequest struct {
	Page   int   `json:"page" form:"page" binding:"min=1"`
	Limit  int   `json:"limit" form:"limit" binding:"min=1,max=100"`
	UserID *uint `json:"user_id" form:"user_id"`
}

// 账号列表响应结构
type AccountListResponse struct {
	Accounts []Account `json:"accounts"`
	Total    int64     `json:"total"`
	Page     int       `json:"page"`
	Limit    int       `json:"limit"`
}

// 账号创建请求参数
type CreateAccountRequest struct {
	Name            string `json:"name" binding:"required,min=1,max=100"`
	PlatformType    string `json:"platform_type" binding:"required,oneof=claude claude_console gemini openai"`
	RequestURL      string `json:"request_url"`
	SecretKey       string `json:"secret_key"`
	GroupID         int    `json:"group_id"`
	Priority        int    `json:"priority"`
	Weight          int    `json:"weight" binding:"min=1"`
	EnableProxy     bool   `json:"enable_proxy"`
	ProxyURI        string `json:"proxy_uri"`
	ModelMapping    string `json:"model_mapping"`
	ActiveStatus    int    `json:"active_status" binding:"oneof=1 2"`
	IsMax           bool   `json:"is_max"` // 是否是max账号
	AccessToken     string `json:"access_token"`
	RefreshToken    string `json:"refresh_token"`
	ExpiresAt       int    `json:"expires_at" binding:"min=0"`
	TodayUsageCount int    `json:"today_usage_count"` // 今日使用次数
}

// 账号更新请求参数
type UpdateAccountRequest struct {
	Name            string `json:"name" binding:"required,min=1,max=100"`
	PlatformType    string `json:"platform_type" binding:"required,oneof=claude claude_console openai gemini"`
	RequestURL      string `json:"request_url"`
	SecretKey       string `json:"secret_key"`
	GroupID         *int   `json:"group_id" binding:"omitempty,min=0"`
	Priority        int    `json:"priority" binding:"min=1"`
	Weight          int    `json:"weight" binding:"min=1"`
	EnableProxy     bool   `json:"enable_proxy"`
	ProxyURI        string `json:"proxy_uri"`
	ModelMapping    string `json:"model_mapping"`
	ActiveStatus    int    `json:"active_status" binding:"oneof=1 2"`
	IsMax           bool   `json:"is_max"` // 是否是max账号
	AccessToken     string `json:"access_token"`
	RefreshToken    string `json:"refresh_token"`
	ExpiresAt       int    `json:"expires_at" binding:"min=0"`
	TodayUsageCount int    `json:"today_usage_count"` // 今日使用次数
}

// 账号激活状态更新请求参数
type UpdateAccountActiveStatusRequest struct {
	ActiveStatus *int `json:"active_status" binding:"required"`
}

// 账号当前状态更新请求参数
type UpdateAccountCurrentStatusRequest struct {
	CurrentStatus *int `json:"current_status" binding:"required"`
}

func (a *Account) TableName() string {
	return "accounts"
}

// 创建账号
func CreateAccount(account *Account) error {
	account.ID = 0
	return DB.Create(account).Error
}

// 根据ID获取账号
func GetAccountByID(id uint) (*Account, error) {
	var account Account
	err := DB.Preload("User").First(&account, id).Error
	if err != nil {
		return nil, err
	}

	// 如果有分组ID，查询分组信息
	if account.GroupID > 0 {
		var group Group
		if err := DB.Where("id = ?", account.GroupID).First(&group).Error; err == nil {
			account.Group = &group
		}
	}

	return &account, nil
}

// 更新账号
func UpdateAccount(account *Account) error {
	return DB.Save(account).Error
}

// 删除账号（软删除）
func DeleteAccount(id uint) error {
	return DB.Delete(&Account{}, id).Error
}

// 分页获取账号列表
func GetAccountList(page, limit int, userID *uint) ([]Account, int64, error) {
	var accounts []Account
	var total int64

	query := DB.Model(&Account{})

	// 如果指定了用户ID，则只查询该用户的账号
	if userID != nil {
		query = query.Where("user_id = ?", *userID)
	}

	// 统计总数
	err := query.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * limit
	err = query.Preload("User").
		Offset(offset).
		Limit(limit).
		Order("priority ASC, created_at DESC").
		Find(&accounts).Error

	if err != nil {
		return nil, 0, err
	}

	// 批量查询分组信息
	groupIDs := make(map[int]bool)
	for _, account := range accounts {
		if account.GroupID > 0 {
			groupIDs[account.GroupID] = true
		}
	}

	if len(groupIDs) > 0 {
		var ids []int
		for id := range groupIDs {
			ids = append(ids, id)
		}

		var groups []Group
		DB.Where("id IN ?", ids).Find(&groups)

		// 创建分组映射
		groupMap := make(map[int]*Group)
		for i := range groups {
			groupMap[int(groups[i].ID)] = &groups[i]
		}

		// 为每个账号设置对应的分组信息
		for i := range accounts {
			if accounts[i].GroupID > 0 {
				if group, exists := groupMap[accounts[i].GroupID]; exists {
					accounts[i].Group = group
				}
			}
		}
	}

	// 批量查询最近一周的统计数据
	if err := setWeeklyStatsForAccounts(accounts); err != nil {
		// 如果查询失败，只记录错误但不影响主要功能
		// 统计字段将保持默认值0
	}

	return accounts, total, nil
}

// 根据用户ID获取账号列表
func GetAccountsByUserID(userID uint) ([]Account, error) {
	var accounts []Account
	err := DB.Where("user_id = ?", userID).Find(&accounts).Error
	if err != nil {
		return nil, err
	}
	return accounts, nil
}

// 根据分组ID获取可用账号列表（按优先级和使用次数排序）
func GetAvailableAccountsByGroupID(groupID int) ([]Account, error) {
	var accounts []Account
	err := DB.Where("group_id = ? AND active_status = 1 AND (current_status = 1 OR (current_status = 3 AND (rate_limit_end_time IS NULL OR rate_limit_end_time < ?)))", groupID, time.Now()).
		Order("priority ASC, today_usage_count ASC").
		Find(&accounts).Error
	if err != nil {
		return nil, err
	}
	return accounts, nil
}

// 获取指定用户、分组和优先级下可用账号的最大今日请求次数
func GetMaxTodayUsageCountFromAvailableAccounts(userID uint, groupID int, priority int) (int, error) {
	var maxUsageCount int
	err := DB.Model(&Account{}).
		Where("user_id = ? AND group_id = ? AND priority = ? AND active_status = 1 AND current_status IN (1, 2)", userID, groupID, priority).
		Select("COALESCE(MAX(today_usage_count), 0)").
		Scan(&maxUsageCount).Error
	return maxUsageCount, err
}

// setWeeklyStatsForAccounts 为账号列表设置最近一周的统计数据
func setWeeklyStatsForAccounts(accounts []Account) error {
	if len(accounts) == 0 {
		return nil
	}

	// 计算本周开始时间（周一00:00:00）到现在
	now := time.Now()
	weekday := int(now.Weekday())
	if weekday == 0 { // 如果是周日，调整为7
		weekday = 7
	}

	// 计算本周一的日期
	weekStart := now.AddDate(0, 0, -(weekday - 1))
	weekStart = time.Date(weekStart.Year(), weekStart.Month(), weekStart.Day(), 0, 0, 0, 0, weekStart.Location())

	// 提取所有账号ID
	accountIDs := make([]uint, len(accounts))
	for i, account := range accounts {
		accountIDs[i] = account.ID
	}

	// 查询最近一周的统计数据
	type WeeklyStats struct {
		AccountID  uint    `json:"account_id"`
		TotalCost  float64 `json:"total_cost"`
		TotalCount int64   `json:"total_count"`
	}

	var weeklyStats []WeeklyStats
	err := DB.Table("logs").
		Select("account_id, SUM(total_cost) as total_cost, COUNT(*) as total_count").
		Where("account_id IN ? AND created_at >= ? AND created_at <= ?", accountIDs, weekStart, now).
		Group("account_id").
		Scan(&weeklyStats).Error

	if err != nil {
		return err
	}

	// 创建统计数据映射
	statsMap := make(map[uint]WeeklyStats)
	for _, stat := range weeklyStats {
		statsMap[stat.AccountID] = stat
	}

	// 为每个账号设置统计数据
	for i := range accounts {
		if stat, exists := statsMap[accounts[i].ID]; exists {
			accounts[i].WeeklyCost = stat.TotalCost
			accounts[i].WeeklyCount = stat.TotalCount
		} else {
			accounts[i].WeeklyCost = 0
			accounts[i].WeeklyCount = 0
		}
	}

	return nil
}
