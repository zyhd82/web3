package logger

import (
	"io"
	"os"

	"github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Init 配置全局 logrus 输出格式、级别，并将日志同时写入控制台和滚动文件
func Init(level, filePath string) {
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
	})

	parsedLevel, err := logrus.ParseLevel(level)
	if err != nil {
		parsedLevel = logrus.InfoLevel
	}
	logrus.SetLevel(parsedLevel)

	fileWriter := &lumberjack.Logger{
		Filename:   filePath,
		MaxSize:    50, // MB
		MaxBackups: 5,
		MaxAge:     30, // days
		Compress:   true,
	}

	logrus.SetOutput(io.MultiWriter(os.Stdout, fileWriter))
}
