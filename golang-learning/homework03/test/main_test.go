package test

import (
	"homework03/configs"
	"os"
	"testing"

	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
)

// 在某个测试文件（如 main_test.go）中定义
func TestMain(m *testing.M) {
	// 1. 加载环境变量
	configs.InitEnv()
	if err := godotenv.Load(); err != nil {
		logrus.Warn("No .env file found, using system environment variables")
	}

	// 2. 初始化数据库
	configs.InitDatabase()

	// 3. 运行所有测试
	code := m.Run()

	// 4. 测试结束后清理资源（可选）
	// db.Close()

	// 5. 退出，返回测试状态码
	os.Exit(code)
}
