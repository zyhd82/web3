// 任务一 - 查询区块：连接 Sepolia 测试网，查询指定区块号的区块信息并输出到控制台
//
// 用法：
//
//	go run ./task1/query_block              # 查询最新区块
//	go run ./task1/query_block -number 5000000
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math/big"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/joho/godotenv"
)

func main() {
	// 命令行参数：区块号，小于 0 表示查询最新区块
	number := flag.Int64("number", -1, "要查询的区块号，不传则查询最新区块")
	flag.Parse()

	// 加载 .env（文件不存在时忽略，允许直接使用系统环境变量）
	_ = godotenv.Load()
	rpcURL := os.Getenv("SEPOLIA_RPC_URL")
	if rpcURL == "" {
		log.Fatal("请在 .env 中配置 SEPOLIA_RPC_URL")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 使用 ethclient 连接 Sepolia 测试网
	client, err := ethclient.DialContext(ctx, rpcURL)
	if err != nil {
		log.Fatalf("连接节点失败: %v", err)
	}
	defer client.Close()

	chainID, err := client.ChainID(ctx)
	if err != nil {
		log.Fatalf("获取链 ID 失败: %v", err)
	}

	// BlockByNumber 传 nil 时返回最新区块
	var blockNumber *big.Int
	if *number >= 0 {
		blockNumber = big.NewInt(*number)
	}

	block, err := client.BlockByNumber(ctx, blockNumber)
	if err != nil {
		log.Fatalf("查询区块失败: %v", err)
	}

	// 输出区块信息
	fmt.Println("========== 区块信息 ==========")
	fmt.Printf("链 ID       : %s\n", chainID)
	fmt.Printf("区块号      : %d\n", block.Number().Uint64())
	fmt.Printf("区块哈希    : %s\n", block.Hash().Hex())
	fmt.Printf("父区块哈希  : %s\n", block.ParentHash().Hex())
	fmt.Printf("时间戳      : %d (%s)\n", block.Time(),
		time.Unix(int64(block.Time()), 0).Format("2006-01-02 15:04:05"))
	fmt.Printf("交易数量    : %d\n", len(block.Transactions()))
	fmt.Printf("Gas 上限    : %d\n", block.GasLimit())
	fmt.Printf("Gas 使用量  : %d\n", block.GasUsed())
	fmt.Printf("矿工/提议者 : %s\n", block.Coinbase().Hex())
	if baseFee := block.BaseFee(); baseFee != nil {
		fmt.Printf("基础费用    : %s wei\n", baseFee)
	}

	// 额外演示：通过区块哈希单独查询交易数量，与上面的结果应一致
	count, err := client.TransactionCount(ctx, block.Hash())
	if err != nil {
		log.Fatalf("查询交易数量失败: %v", err)
	}
	fmt.Printf("交易数量校验: %d\n", count)
}
