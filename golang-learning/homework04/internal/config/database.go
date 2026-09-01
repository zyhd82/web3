package config

import (
	"fmt"

	"github.com/sirupsen/logrus"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// DB 全局数据库连接实例
var DB *gorm.DB

// InitDatabase 建立 MySQL 连接；models 用于后续步骤传入需要 AutoMigrate 的表结构
func InitDatabase(cfg *Config, models ...interface{}) error {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.DBUser, cfg.DBPassword, cfg.DBHost, cfg.DBPort, cfg.DBName)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Warn),
		// 项目约定不用数据库物理外键（见 docs/DATABASE_CONVENTIONS.md），
		// 关闭 AutoMigrate 时的外键约束自动创建，引用完整性交给 service 层保证。
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		return fmt.Errorf("连接 MySQL 数据库失败: %w", err)
	}

	if len(models) > 0 {
		if err := db.AutoMigrate(models...); err != nil {
			return fmt.Errorf("自动迁移表结构失败: %w", err)
		}
	}

	DB = db
	logrus.Info("MySQL 数据库连接成功")
	return nil
}
