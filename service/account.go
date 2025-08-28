package service

import (
	"claude-code-relay/common"
	"claude-code-relay/model"
	"errors"
	"log"
	"time"
)

type AccountService struct{}

func NewAccountService() *AccountService {
	return &AccountService{}
}

// GetAccountList 获取账号列表
func (s *AccountService) GetAccountList(page, limit int, userID *uint) (*model.AccountListResponse, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}

	accounts, total, err := model.GetAccountList(page, limit, userID)
	if err != nil {
		return nil, errors.New("获取账号列表失败")
	}

	result := &model.AccountListResponse{
		Accounts: accounts,
		Total:    total,
		Page:     page,
		Limit:    limit,
	}

	return result, nil
}

// CreateAccount 创建账号
func (s *AccountService) CreateAccount(req *model.CreateAccountRequest, userID uint) (*model.Account, error) {
	// 设置今日请求次数：获取同用户、同分组、同优先级可用账号的最大今日请求次数，然后减1
	todayUsageCount := req.TodayUsageCount
	if todayUsageCount == 0 && req.Priority > 0 {
		maxUsageCount, err := model.GetMaxTodayUsageCountFromAvailableAccounts(userID, req.GroupID, req.Priority)
		if err != nil {
			return nil, errors.New("获取最大今日请求次数失败")
		}
		todayUsageCount = maxUsageCount - 1
		if todayUsageCount < 0 {
			todayUsageCount = 0
		}
	}

	account := &model.Account{
		Name:            req.Name,
		PlatformType:    req.PlatformType,
		RequestURL:      req.RequestURL,
		SecretKey:       req.SecretKey,
		GroupID:         req.GroupID,
		Priority:        req.Priority,
		Weight:          req.Weight,
		EnableProxy:     req.EnableProxy,
		ProxyURI:        req.ProxyURI,
		ModelMapping:    req.ModelMapping,
		ActiveStatus:    req.ActiveStatus,
		IsMax:           req.IsMax,
		AccessToken:     req.AccessToken,
		RefreshToken:    req.RefreshToken,
		ExpiresAt:       req.ExpiresAt,
		TodayUsageCount: todayUsageCount,
		UserID:          userID,
	}

	if err := model.CreateAccount(account); err != nil {
		return nil, errors.New("创建账号失败")
	}

	return account, nil
}

// GetAccountByID 根据ID获取账号详情
func (s *AccountService) GetAccountByID(id uint, userID *uint) (*model.Account, error) {
	account, err := model.GetAccountByID(id)
	if err != nil {
		return nil, errors.New("账号不存在")
	}

	// 如果指定了用户ID，验证账号是否属于该用户
	if userID != nil && account.UserID != *userID {
		return nil, errors.New("无权访问此账号")
	}

	return account, nil
}

// UpdateAccount 更新账号
func (s *AccountService) UpdateAccount(id uint, req *model.UpdateAccountRequest, userID *uint) (*model.Account, error) {
	account, err := s.GetAccountByID(id, userID)
	if err != nil {
		return nil, err
	}

	// 更新字段
	account.Name = req.Name
	account.PlatformType = req.PlatformType
	account.RequestURL = req.RequestURL
	if req.GroupID != nil {
		account.GroupID = *req.GroupID
	}
	account.Priority = req.Priority
	account.Weight = req.Weight
	account.EnableProxy = req.EnableProxy
	account.ProxyURI = req.ProxyURI
	account.ModelMapping = req.ModelMapping
	account.ActiveStatus = req.ActiveStatus
	account.IsMax = req.IsMax

	if req.SecretKey != "" {
		account.SecretKey = req.SecretKey
	}

	if req.AccessToken != "" {
		account.AccessToken = req.AccessToken
	}

	if req.RefreshToken != "" {
		account.RefreshToken = req.RefreshToken
	}

	if req.ExpiresAt > 0 {
		account.ExpiresAt = req.ExpiresAt
	}

	// 更新TodayUsageCount字段，如果请求中设置了该字段，则更新
	if req.TodayUsageCount > 0 {
		account.TodayUsageCount = req.TodayUsageCount
	}

	if err := model.UpdateAccount(account); err != nil {
		return nil, errors.New("更新账号失败")
	}

	return account, nil
}

// DeleteAccount 删除账号
func (s *AccountService) DeleteAccount(id uint, userID *uint) error {
	account, err := s.GetAccountByID(id, userID)
	if err != nil {
		return err
	}

	if err := model.DeleteAccount(account.ID); err != nil {
		return errors.New("删除账号失败")
	}

	return nil
}

// GetAccountsByUserID 获取指定用户的所有账号
func (s *AccountService) GetAccountsByUserID(userID uint) ([]model.Account, error) {
	accounts, err := model.GetAccountsByUserID(userID)
	if err != nil {
		return nil, errors.New("获取用户账号列表失败")
	}

	return accounts, nil
}

// UpdateAccountActiveStatus 更新账号激活状态
func (s *AccountService) UpdateAccountActiveStatus(id uint, activeStatus int, userID *uint) error {
	account, err := s.GetAccountByID(id, userID)
	if err != nil {
		return err
	}

	// 如果是启用账号（从禁用变为激活），设置今日请求次数
	if account.ActiveStatus == 2 && activeStatus == 1 {
		maxUsageCount, err := model.GetMaxTodayUsageCountFromAvailableAccounts(account.UserID, account.GroupID, account.Priority)
		if err != nil {
			return errors.New("获取最大今日请求次数失败")
		}
		todayUsageCount := maxUsageCount - 1
		if todayUsageCount < 0 {
			todayUsageCount = 0
		}
		account.TodayUsageCount = todayUsageCount
	}

	account.ActiveStatus = activeStatus

	if err := model.UpdateAccount(account); err != nil {
		return errors.New("更新账号激活状态失败")
	}

	return nil
}

// UpdateAccountCurrentStatus 更新账号当前状态(值不能为3)
func (s *AccountService) UpdateAccountCurrentStatus(id uint, currentStatus int, userID *uint) error {
	if currentStatus == 3 {
		return errors.New("当前状态不能设置为3")
	}

	account, err := s.GetAccountByID(id, userID)
	if err != nil {
		return err
	}

	account.CurrentStatus = currentStatus

	if err := model.UpdateAccount(account); err != nil {
		return errors.New("更新账号当前状态失败")
	}

	return nil
}

// UpdateAccountStatus 根据响应状态码更新账号状态
func (s *AccountService) UpdateAccountStatus(account *model.Account, statusCode int, usage *common.TokenUsage) {
	// 根据状态码设置CurrentStatus
	switch {
	case statusCode == 429:
		// 限流状态
		account.CurrentStatus = 3
	case statusCode >= 400:
		// 接口异常
		account.CurrentStatus = 2
	case statusCode == 200 || statusCode == 201:
		// 正常状态
		account.CurrentStatus = 1

		// 请求成功时更新最后使用时间和今日使用次数
		now := time.Now()

		// 判断最后使用时间是否为当天
		if account.LastUsedTime != nil {
			lastUsedDate := time.Time(*account.LastUsedTime).Format("2006-01-02")
			todayDate := now.Format("2006-01-02")

			if lastUsedDate == todayDate {
				// 同一天，使用次数+1
				account.TodayUsageCount++
			} else {
				// 不同天，重置为1
				account.TodayUsageCount = 1
			}
		} else {
			// 首次使用，设置为1
			account.TodayUsageCount = 1
		}

		// 更新token使用量和费用（如果有的话）
		if usage != nil {
			// 计算本次请求的费用
			costResult := common.CalculateCost(usage)
			currentCost := costResult.Costs.Total

			if account.LastUsedTime != nil {
				lastUsedDate := time.Time(*account.LastUsedTime).Format("2006-01-02")
				todayDate := now.Format("2006-01-02")

				if lastUsedDate == todayDate {
					// 同一天，累加各类tokens和费用
					account.TodayInputTokens += usage.InputTokens
					account.TodayOutputTokens += usage.OutputTokens
					account.TodayCacheReadInputTokens += usage.CacheReadInputTokens
					account.TodayCacheCreationInputTokens += usage.CacheCreationInputTokens
					account.TodayTotalCost += currentCost
				} else {
					// 不同天，重置各类tokens和费用
					account.TodayInputTokens = usage.InputTokens
					account.TodayOutputTokens = usage.OutputTokens
					account.TodayCacheReadInputTokens = usage.CacheReadInputTokens
					account.TodayCacheCreationInputTokens = usage.CacheCreationInputTokens
					account.TodayTotalCost = currentCost
				}
			} else {
				// 首次使用，设置各类tokens和费用
				account.TodayInputTokens = usage.InputTokens
				account.TodayOutputTokens = usage.OutputTokens
				account.TodayCacheReadInputTokens = usage.CacheReadInputTokens
				account.TodayCacheCreationInputTokens = usage.CacheCreationInputTokens
				account.TodayTotalCost = currentCost
			}
		}

		// 更新最后使用时间
		nowTime := model.Time(now)
		account.LastUsedTime = &nowTime
	default:
		// 其他状态码保持原状态
		return
	}

	// 更新数据库
	if err := model.UpdateAccount(account); err != nil {
		log.Printf("failed to update account status: %v", err)
	}
}
