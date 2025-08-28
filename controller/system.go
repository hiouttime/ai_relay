package controller

import (
	"claude-code-relay/constant"
	"claude-code-relay/model"
	"claude-code-relay/scheduled"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

func GetStatus(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "服务运行正常",
		"code":    constant.Success,
		"data": gin.H{
			"status":  "running",
			"version": "1.0.0",
		},
	})
}

func GetApiLogs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}

	logs, total, err := model.GetApiLogs(page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "获取日志失败",
			"code":  constant.InternalServerError,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "获取成功",
		"code":    constant.Success,
		"data": gin.H{
			"logs":  logs,
			"total": total,
			"page":  page,
			"limit": limit,
		},
	})
}

func GetDashboard(c *gin.Context) {
	// 获取统计数据
	var userCount, taskCount, completedTaskCount int64

	model.DB.Model(&model.User{}).Count(&userCount)
	model.DB.Model(&model.Task{}).Count(&taskCount)
	model.DB.Model(&model.Task{}).Where("status = ?", constant.TaskStatusCompleted).Count(&completedTaskCount)

	c.JSON(http.StatusOK, gin.H{
		"message": "获取成功",
		"code":    constant.Success,
		"data": gin.H{
			"user_count":           userCount,
			"task_count":           taskCount,
			"completed_task_count": completedTaskCount,
			"pending_task_count":   taskCount - completedTaskCount,
		},
	})
}

// ManualResetStats 手动重置统计数据（测试用）
func ManualResetStats(c *gin.Context) {
	cronService := scheduled.GetInstance()
	if cronService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "定时任务服务未初始化",
			"code":  constant.InternalServerError,
		})
		return
	}

	err := cronService.ManualTrigger("reset_daily")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "重置统计数据失败: " + err.Error(),
			"code":  constant.InternalServerError,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "统计数据重置成功",
		"code":    constant.Success,
	})
}

// ManualCleanLogs 手动清理过期日志（测试用）
func ManualCleanLogs(c *gin.Context) {
	cronService := scheduled.GetInstance()
	if cronService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "定时任务服务未初始化",
			"code":  constant.InternalServerError,
		})
		return
	}

	err := cronService.ManualTrigger("clean_logs")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "清理日志失败: " + err.Error(),
			"code":  constant.InternalServerError,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "日志清理成功",
		"code":    constant.Success,
	})
}
