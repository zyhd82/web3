// 任务二 - 使用 abigen 生成的 Go 绑定代码与 Sepolia 上的 Counter 合约交互
//
// 用法：
//
//	go run ./task2/interact -deploy          # 部署新合约后进行交互
//	go run ./task2/interact                  # 使用 .env 中的 COUNTER_ADDRESS 进行交互
//	go run ./task2/interact -add 5           # 在 increment 之后再调用 add(5)
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math/big"
	"os"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/joho/godotenv"

	"homework05/task2/counter"
)

func main() {
	deploy := flag.Bool("deploy", false, "部署一个新的 Counter 合约")
	initial := flag.Int64("initial", 0, "部署时的初始计数值")
	addressFlag := flag.String("address", "", "合约地址（不传则读取 .env 中的 COUNTER_ADDRESS）")
	addDelta := flag.Int64("add", 0, "大于 0 时，额外调用 add(delta)")
	flag.Parse()

	_ = godotenv.Load()
	rpcURL := os.Getenv("SEPOLIA_RPC_URL")
	privateKeyHex := os.Getenv("PRIVATE_KEY")
	if rpcURL == "" || privateKeyHex == "" {
		log.Fatal("请在 .env 中配置 SEPOLIA_RPC_URL 和 PRIVATE_KEY")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	client, err := ethclient.DialContext(ctx, rpcURL)
	if err != nil {
		log.Fatalf("连接节点失败: %v", err)
	}
	defer client.Close()

	privateKey, err := crypto.HexToECDSA(strings.TrimPrefix(privateKeyHex, "0x"))
	if err != nil {
		log.Fatalf("解析私钥失败: %v", err)
	}
	chainID, err := client.ChainID(ctx)
	if err != nil {
		log.Fatalf("获取链 ID 失败: %v", err)
	}

	// TransactOpts 封装了发送方地址和签名器，绑定代码发送交易时会自动填充 nonce、gas 等参数
	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, chainID)
	if err != nil {
		log.Fatalf("创建交易签名器失败: %v", err)
	}
	auth.Context = ctx
	fmt.Printf("当前账户 : %s\n", auth.From.Hex())

	// 1. 获取合约实例：部署新合约，或连接已有合约
	var address common.Address
	var instance *counter.Counter
	if *deploy {
		address, instance = deployCounter(ctx, client, auth, big.NewInt(*initial))
	} else {
		addrHex := *addressFlag
		if addrHex == "" {
			addrHex = os.Getenv("COUNTER_ADDRESS")
		}
		if !common.IsHexAddress(addrHex) {
			log.Fatal("未指定合约地址：请使用 -deploy 部署，或通过 -address / COUNTER_ADDRESS 指定")
		}
		address = common.HexToAddress(addrHex)
		instance, err = counter.NewCounter(address, client)
		if err != nil {
			log.Fatalf("加载合约失败: %v", err)
		}
	}
	fmt.Printf("合约地址 : %s\n", address.Hex())

	// 2. 只读调用（eth_call，不消耗 gas）
	callOpts := &bind.CallOpts{Context: ctx}
	owner, err := instance.Owner(callOpts)
	if err != nil {
		log.Fatalf("读取 owner 失败（请确认合约地址在 Sepolia 上且已部署）: %v", err)
	}
	before, err := instance.GetCount(callOpts)
	if err != nil {
		log.Fatalf("读取计数失败: %v", err)
	}
	fmt.Printf("合约 owner: %s\n", owner.Hex())
	fmt.Printf("调用前计数: %s\n", before)

	// 3. 写入调用：increment()
	tx, err := instance.Increment(auth)
	if err != nil {
		log.Fatalf("调用 increment 失败: %v", err)
	}
	waitAndPrint(ctx, client, instance, "increment()", tx)

	// 4. 可选写入调用：add(delta)
	if *addDelta > 0 {
		tx, err = instance.Add(auth, big.NewInt(*addDelta))
		if err != nil {
			log.Fatalf("调用 add 失败: %v", err)
		}
		waitAndPrint(ctx, client, instance, fmt.Sprintf("add(%d)", *addDelta), tx)
	}

	// 5. 再次读取，确认结果
	after, err := instance.GetCount(callOpts)
	if err != nil {
		log.Fatalf("读取计数失败: %v", err)
	}
	fmt.Printf("调用后计数: %s\n", after)
}

// deployCounter 使用绑定代码中的 DeployCounter 部署合约，并等待部署完成
func deployCounter(ctx context.Context, client *ethclient.Client, auth *bind.TransactOpts, initial *big.Int) (common.Address, *counter.Counter) {
	address, tx, instance, err := counter.DeployCounter(auth, client, initial)
	if err != nil {
		log.Fatalf("部署合约失败: %v", err)
	}
	fmt.Printf("部署交易 : %s\n", tx.Hash().Hex())
	fmt.Println("等待合约部署上链...")
	if _, err := bind.WaitDeployed(ctx, client, tx); err != nil {
		log.Fatalf("等待部署失败: %v", err)
	}
	fmt.Println("部署成功！可将下面的地址写入 .env 的 COUNTER_ADDRESS，之后无需重复部署")
	return address, instance
}

// waitAndPrint 等待交易被打包，输出收据信息，并从日志中解析 CountChanged 事件
func waitAndPrint(ctx context.Context, client *ethclient.Client, instance *counter.Counter, method string, tx *types.Transaction) {
	fmt.Printf("---- %s ----\n", method)
	fmt.Printf("交易哈希 : %s\n", tx.Hash().Hex())
	receipt, err := bind.WaitMined(ctx, client, tx)
	if err != nil {
		log.Fatalf("等待交易上链失败: %v", err)
	}
	if receipt.Status != types.ReceiptStatusSuccessful {
		log.Fatalf("交易执行失败，区块号 %d", receipt.BlockNumber)
	}
	fmt.Printf("交易状态 : 成功，区块号 %d，消耗 gas %d\n", receipt.BlockNumber, receipt.GasUsed)

	// 使用绑定代码生成的 ParseCountChanged 解析事件日志
	for _, l := range receipt.Logs {
		if ev, err := instance.ParseCountChanged(*l); err == nil {
			fmt.Printf("事件     : CountChanged(caller=%s, newCount=%s)\n", ev.Caller.Hex(), ev.NewCount)
		}
	}
}
