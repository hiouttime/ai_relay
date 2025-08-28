package common

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
)

var (
	InfoLogger  *log.Logger
	ErrorLogger *log.Logger
)

func SetupLogger() {
	logDir := "./logs"
	if err := os.MkdirAll(logDir, 0755); err != nil {
		log.Fatal("创建日志目录失败:", err)
	}

	logFile := filepath.Join(logDir, "app.log")
	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatal("打开日志文件失败:", err)
	}

	InfoLogger = log.New(file, "[INFO] ", log.Ldate|log.Ltime|log.Lshortfile)
	ErrorLogger = log.New(file, "[ERROR] ", log.Ldate|log.Ltime|log.Lshortfile)
}

func SysLog(format string, args ...interface{}) {
	var message string
	if len(args) > 0 {
		message = fmt.Sprintf(format, args...)
	} else {
		message = format
	}
	logMessage := fmt.Sprintf("[SYSTEM] %s", message)
	log.Println(logMessage)
	if InfoLogger != nil {
		InfoLogger.Println(message)
	}
}

func SysError(format string, args ...interface{}) {
	var message string
	if len(args) > 0 {
		message = fmt.Sprintf(format, args...)
	} else {
		message = format
	}
	logMessage := fmt.Sprintf("[ERROR] %s", message)
	log.Println(logMessage)
	if ErrorLogger != nil {
		ErrorLogger.Println(message)
	}
}

func FatalLog(message string) {
	SysError(message)
	os.Exit(1)
}