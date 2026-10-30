// 任务一 - 发送交易：在 Sepolia 测试网构造一笔 ETH 转账交易，签名后广播并输出交易哈希
//
// 用法：
//
//	go run ./task1/send_tx -to 0x接收方地址 -amount 0.001
//	go run ./task1/send_tx -to 0x接收方地址 -amount 0.001 -wait   # 等待交易上链
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

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/joho/godotenv"
)

func main() {
	toFlag := flag.String("to", "", "接收方地址（不传则读取 .env 中的 TO_ADDRESS）")
	amountFlag := flag.String("amount", "0.001", "转账金额，单位 ETH")
	wait := flag.Bool("wait", false, "是否等待交易被打包并输出收据")
	flag.Parse()

	_ = godotenv.Load()
	rpcURL := os.Getenv("SEPOLIA_RPC_URL")
	privateKeyHex := os.Getenv("PRIVATE_KEY")
	if rpcURL == "" || privateKeyHex == "" {
		log.Fatal("请在 .env 中配置 SEPOLIA_RPC_URL 和 PRIVATE_KEY")
	}

	toHex := *toFlag
	if toHex == "" {
		toHex = os.Getenv("TO_ADDRESS")
	}
	if !common.IsHexAddress(toHex) {
		log.Fatalf("接收方地址无效: %q", toHex)
	}
	toAddress := common.HexToAddress(toHex)

	value, err := ethToWei(*amountFlag)
	if err != nil {
		log.Fatalf("转账金额无效: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	client, err := ethclient.DialContext(ctx, rpcURL)
	if err != nil {
		log.Fatalf("连接节点失败: %v", err)
	}
	defer client.Close()

	// 1. 解析私钥并推导出发送方地址
	privateKey, err := crypto.HexToECDSA(strings.TrimPrefix(privateKeyHex, "0x"))
	if err != nil {
		log.Fatalf("解析私钥失败: %v", err)
	}
	fromAddress := crypto.PubkeyToAddress(privateKey.PublicKey)

	// 2. 余额检查
	balance, err := client.BalanceAt(ctx, fromAddress, nil)
	if err != nil {
		log.Fatalf("查询余额失败: %v", err)
	}

	// 3. 获取 nonce、链 ID、gas 参数
	nonce, err := client.PendingNonceAt(ctx, fromAddress)
	if err != nil {
		log.Fatalf("获取 nonce 失败: %v", err)
	}
	chainID, err := client.ChainID(ctx)
	if err != nil {
		log.Fatalf("获取链 ID 失败: %v", err)
	}

	// EIP-1559 费用：小费取节点建议值，费用上限 = 2 * baseFee + 小费，留出 baseFee 上涨空间
	tipCap, err := client.SuggestGasTipCap(ctx)
	if err != nil {
		log.Fatalf("获取建议小费失败: %v", err)
	}
	header, err := client.HeaderByNumber(ctx, nil)
	if err != nil {
		log.Fatalf("获取最新区块头失败: %v", err)
	}
	feeCap := new(big.Int).Add(new(big.Int).Mul(header.BaseFee, big.NewInt(2)), tipCap)

	// 普通 ETH 转账固定消耗 21000 gas
	const gasLimit uint64 = 21000

	maxCost := new(big.Int).Add(value, new(big.Int).Mul(feeCap, big.NewInt(int64(gasLimit))))
	if balance.Cmp(maxCost) < 0 {
		log.Fatalf("余额不足：当前 %s wei，最多需要 %s wei", balance, maxCost)
	}

	// 4. 构造交易（EIP-1559 动态费用交易）
	tx := types.NewTx(&types.DynamicFeeTx{
		ChainID:   chainID,
		Nonce:     nonce,
		GasTipCap: tipCap,
		GasFeeCap: feeCap,
		Gas:       gasLimit,
		To:        &toAddress,
		Value:     value,
	})

	// 5. 使用私钥签名
	signedTx, err := types.SignTx(tx, types.LatestSignerForChainID(chainID), privateKey)
	if err != nil {
		log.Fatalf("签名交易失败: %v", err)
	}

	// 6. 广播交易
	if err := client.SendTransaction(ctx, signedTx); err != nil {
		log.Fatalf("发送交易失败: %v", err)
	}

	fmt.Println("========== 交易已发送 ==========")
	fmt.Printf("发送方   : %s\n", fromAddress.Hex())
	fmt.Printf("接收方   : %s\n", toAddress.Hex())
	fmt.Printf("金额     : %s ETH (%s wei)\n", *amountFlag, value)
	fmt.Printf("Nonce    : %d\n", nonce)
	fmt.Printf("交易哈希 : %s\n", signedTx.Hash().Hex())
	fmt.Printf("浏览器   : https://sepolia.etherscan.io/tx/%s\n", signedTx.Hash().Hex())

	if !*wait {
		return
	}

	// 可选：轮询等待交易被打包
	fmt.Println("等待交易上链...")
	for {
		receipt, err := client.TransactionReceipt(ctx, signedTx.Hash())
		if err == nil {
			status := "成功"
			if receipt.Status != types.ReceiptStatusSuccessful {
				status = "失败"
			}
			fmt.Printf("交易状态 : %s，区块号 %d，消耗 gas %d\n", status, receipt.BlockNumber, receipt.GasUsed)
			return
		}
		select {
		case <-ctx.Done():
			log.Fatalf("等待超时，可稍后在浏览器查看交易状态")
		case <-time.After(3 * time.Second):
		}
	}
}

// ethToWei 将十进制 ETH 字符串精确转换为 wei（1 ETH = 1e18 wei），避免浮点误差
func ethToWei(eth string) (*big.Int, error) {
	r, ok := new(big.Rat).SetString(eth)
	if !ok || r.Sign() <= 0 {
		return nil, fmt.Errorf("%q 不是正数", eth)
	}
	r.Mul(r, new(big.Rat).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)))
	if !r.IsInt() {
		return nil, fmt.Errorf("%q 精度超过 18 位小数", eth)
	}
	return r.Num(), nil
}
