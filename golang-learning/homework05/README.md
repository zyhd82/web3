# homework05 - Go 与以太坊交互

## 任务一：区块链读写

### 1. 环境搭建

1. 安装 Go（本项目使用 Go 1.26），确认 `go version` 可用。
2. 安装依赖（已写入 go.mod，执行一次即可）：
   ```bash
   go mod download
   ```
   主要依赖：`github.com/ethereum/go-ethereum`（ethclient、签名等）、`github.com/joho/godotenv`（读取 .env）。
3. 注册 [Infura](https://app.infura.io/)，创建 API Key，并在该 Key 下启用 Ethereum Sepolia 网络，
   得到节点地址 `https://sepolia.infura.io/v3/<API_KEY>`。
4. 准备 Sepolia 测试账户：
   - 在 MetaMask 新建一个**仅用于测试**的账户，导出私钥；
   - 通过水龙头领取测试币（如 Google Cloud Web3 Faucet、Alchemy Sepolia Faucet）。
5. 复制配置文件并填写：
   ```bash
   cp .env.example .env
   ```

### 2. 查询区块

```bash
go run ./task1/query_block                  # 查询最新区块
go run ./task1/query_block -number 5000000  # 查询指定区块
```

输出示例：

```
========== 区块信息 ==========
链 ID       : 11155111
区块号      : 5000000
区块哈希    : 0x72247ea9191db039158939f7ba958e638a32df4f61a43edcae60cb7a686a2d55
父区块哈希  : 0x69c3f269c0e96530feb9c3ec01ab8c87c77a8c489148d34bdd3ce00d78f00b3c
时间戳      : 1704111828 (2024-01-01 20:23:48)
交易数量    : 81
Gas 上限    : 30000000
Gas 使用量  : 15348380
...
```

### 3. 发送交易

```bash
go run ./task1/send_tx -to 0x接收方地址 -amount 0.001
go run ./task1/send_tx -to 0x接收方地址 -amount 0.001 -wait   # 同时等待交易上链
```

流程：解析私钥得到发送方地址 → 检查余额 → 获取 nonce / 链 ID / EIP-1559 费用参数
→ 构造 `DynamicFeeTx` → `types.SignTx` 签名 → `SendTransaction` 广播 → 输出交易哈希。

交易哈希可在 `https://sepolia.etherscan.io/tx/<哈希>` 查看。

## 任务二：使用 abigen 生成 Go 绑定并与合约交互

目录结构：

```
task2/
├── contracts/Counter.sol   # 计数器合约源码
├── build/                  # 编译产物：Counter.abi、Counter.bin
├── counter/counter.go      # abigen 生成的 Go 绑定（勿手动修改）
└── interact/main.go        # 使用绑定代码部署/调用合约
```

### 1. 编译合约（生成 ABI 和字节码）

使用 npm 版 solc（solcjs），无需单独安装：

```powershell
cd task2
npx solc@0.8.28 --optimize --abi --bin --base-path contracts -o build contracts/Counter.sol
# solcjs 输出的文件名带前缀，重命名为简洁的名字
Move-Item -Force build/Counter_sol_Counter.abi build/Counter.abi
Move-Item -Force build/Counter_sol_Counter.bin build/Counter.bin
```

### 2. 安装 abigen 并生成 Go 绑定

abigen 版本应与 go.mod 中的 go-ethereum 版本保持一致（本项目为 v1.17.10）：

```powershell
go install github.com/ethereum/go-ethereum/cmd/abigen@v1.17.10
abigen --abi build/Counter.abi --bin build/Counter.bin --pkg counter --type Counter --out counter/counter.go
```

传入 `--bin` 后，绑定代码中会额外生成 `DeployCounter` 部署函数。

### 3. 与合约交互

```powershell
# 首次：部署合约（初始值 10），然后调用 increment()，再调用 add(5)
go run ./task2/interact -deploy -initial 10 -add 5

# 把输出的合约地址写入 .env 的 COUNTER_ADDRESS 后，可直接交互
go run ./task2/interact
```

程序流程：`NewKeyedTransactorWithChainID` 创建签名器 → `DeployCounter` / `NewCounter` 获取合约实例
→ `GetCount` 只读调用 → `Increment` 发送交易 → `bind.WaitMined` 等待上链
→ `ParseCountChanged` 解析事件 → 再次 `GetCount` 输出结果。
