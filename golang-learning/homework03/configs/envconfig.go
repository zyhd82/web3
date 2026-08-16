package configs

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func InitEnv() {
	// 获取当前工作目录
	rootDir, err := os.Getwd()
	if err != nil {
		log.Fatalf("Failed to get current working directory: %s", err)
	}

	// 这是单元测试
	if strings.HasSuffix(rootDir, "test") {
		rootDir = filepath.Dir(rootDir)
	}

	// 设置工作目录
	if err := os.Chdir(rootDir); err != nil {
		log.Fatalf("Failed to change working directory: %s", err)
	}
	fmt.Println("工作目录：", rootDir)

}
