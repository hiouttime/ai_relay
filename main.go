package main

import (
	"claude-code-relay/common"
	"claude-code-relay/middleware"
	"claude-code-relay/model"
	"claude-code-relay/router"
	"claude-code-relay/scheduled"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// 加载环境变量
	err := godotenv.Load(".env")
	if err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	// 设置全局时间格式
	time.Local, _ = time.LoadLocation("Asia/Shanghai")

	// 设置日志
	common.SetupLogger()
	common.SysLog("Claude Code Relay started")

	// 设置Gin模式
	if os.Getenv("GIN_MODE") != "debug" {
		gin.SetMode(gin.ReleaseMode)
	}

	// 初始化数据库
	err = model.InitDB()
	if err != nil {
		common.FatalLog("failed to initialize database: " + err.Error())
	}
	defer func() {
		if err := model.CloseDB(); err != nil {
			common.FatalLog("failed to close database: " + err.Error())
		}
	}()

	// 初始化Redis
	err = common.InitRedisClient()
	if err != nil {
		common.FatalLog("failed to initialize Redis: " + err.Error())
	}

	// 初始化定时任务服务
	if err := scheduled.Init(); err != nil {
		common.FatalLog("failed to initialize cron service: " + err.Error())
	}
	defer scheduled.Shutdown()

	// 初始化HTTP服务器
	server := gin.New()
	server.Use(gin.CustomRecovery(func(c *gin.Context, err any) {
		common.SysError(fmt.Sprintf("panic detected: %v", err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"message": fmt.Sprintf("系统异常: %v", err),
				"type":    "system_error",
			},
		})
	}))

	// 请求ID中间件
	server.Use(middleware.RequestId())

	// 设置日志中间件
	middleware.SetUpLogger(server)

	// 设置跨域中间件
	server.Use(middleware.CORS())

	// 设置API前后端路由
	router.SetAPIRouter(server)
	// 设置Claude Code专用路由
	router.SetClaudeCodeRouter(server)

	// 启动服务器
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// 设置信号处理，优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		common.SysLog("Server starting on port " + port)
		err = server.Run(":" + port)
		if err != nil {
			common.FatalLog("failed to start HTTP server: " + err.Error())
		}
	}()

	// 等待退出信号
	<-quit
	common.SysLog("Shutting down server...")

	// 停止定时任务服务
	scheduled.Shutdown()

	common.SysLog("Server stopped gracefully")
}
