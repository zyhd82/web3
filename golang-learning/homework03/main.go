package main

import (
	"homework03/configs"

	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
)

func main() {
	// 加载环境变量
	configs.InitEnv()
	if err := godotenv.Load(); err != nil {
		logrus.Warn("No .env file found, using system environment variables")
	}
	//初始化数据库
	configs.InitDatabase()

}
