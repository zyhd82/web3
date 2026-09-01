package config

import (
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
)

// Config 汇总应用启动所需的全部配置项
type Config struct {
	ServerPort string

	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string

	JWTSecret     string
	JWTExpireHour int

	LogLevel string
	LogFile  string
}

var Cfg *Config

// Load 加载 .env（若存在）并解析出全局配置
func Load() *Config {
	if err := godotenv.Load(); err != nil {
		logrus.Warn("未找到 .env 文件，使用系统环境变量")
	}

	Cfg = &Config{
		ServerPort: getEnv("SERVER_PORT", "8080"),

		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "3306"),
		DBUser:     getEnv("DB_USER", "root"),
		DBPassword: getEnv("DB_PASSWORD", ""),
		DBName:     getEnv("DB_NAME", "golang_blog"),

		JWTSecret:     getEnv("JWT_SECRET", "please-change-this-secret"),
		JWTExpireHour: getEnvInt("JWT_EXPIRE_HOUR", 24),

		LogLevel: getEnv("LOG_LEVEL", "info"),
		LogFile:  getEnv("LOG_FILE", "logs/app.log"),
	}
	return Cfg
}

// TokenExpireDuration JWT 过期时长
func (c *Config) TokenExpireDuration() time.Duration {
	return time.Duration(c.JWTExpireHour) * time.Hour
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		logrus.Warnf("环境变量 %s=%s 不是合法整数，使用默认值 %d", key, value, defaultValue)
		return defaultValue
	}
	return parsed
}
