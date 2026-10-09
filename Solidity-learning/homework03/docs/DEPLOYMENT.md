# 部署指南

本文档说明如何在本地演练部署，以及如何部署到 **Sepolia 测试网**、升级到 V2、在 Etherscan 验证源码，并在测试网上演示一场完整的拍卖。

所有命令都在 `homework03/` 目录下执行。

## 目录

1. [准备工作](#1-准备工作)
2. [本地演练](#2-本地演练)
3. [配置敏感信息（keystore）](#3-配置敏感信息keystore)
4. [部署到 Sepolia](#4-部署到-sepolia)
5. [在 Etherscan 验证源码](#5-在-etherscan-验证源码)
6. [升级到 V2](#6-升级到-v2)
7. [测试网演示一场拍卖](#7-测试网演示一场拍卖)
8. [常见问题](#8-常见问题)

## 1. 准备工作

### 1.1 安装依赖并跑通测试

```bash
npm install
npx hardhat compile
npx hardhat test
```

测试全部通过后再进行部署。

> 使用 npm 12 安装时，可能出现 `esbuild ... install scripts blocked` 的警告。这是 npm 12 默认阻止依赖的安装脚本所致，不影响编译、测试和部署，可以忽略。

### 1.2 准备账户与测试 ETH

| 准备项 | 说明 |
| --- | --- |
| **部署账户** | 建议新建一个**专用于测试网**的钱包账户，不要使用存有真实资产的账户。部署账户会成为拍卖合约的 owner、AuctionNFT 的 owner 和手续费接收地址 |
| **Sepolia 测试 ETH** | 部署 + 升级约消耗 **760 万 gas**（按 10 gwei 计约 0.076 ETH），建议准备 **0.1 ETH 以上**；可从 [Google Cloud Faucet](https://cloud.google.com/application/web3/faucet/ethereum/sepolia)、[Alchemy Faucet](https://www.alchemy.com/faucets/ethereum-sepolia) 等水龙头领取 |
| **第二个账户（可选）** | 用于[第 7 节](#7-测试网演示一场拍卖)演示出价：合约禁止卖家给自己出价，需要另一个账户作为买家，准备 **0.005 ETH** 左右即可（出价约 $6 的 ETH，另加 gas） |

各步骤的 gas 消耗（本地以 production 配置实测）：

| 步骤 | gas |
| --- | --- |
| 部署 AuctionNFT | 约 128 万 |
| 部署 MockUSDC | 约 52 万 |
| 部署 NFTAuction V1 实现合约 | 约 252 万 |
| 部署 ERC1967Proxy（含 initialize） | 约 36 万 |
| 配置两个喂价 | 约 11 万 |
| **部署合计** | **约 480 万** |
| 部署 NFTAuctionV2 实现合约 | 约 269 万 |
| `upgradeToAndCall` | 约 9 万 |
| **升级合计** | **约 278 万** |

### 1.3 准备三项敏感配置

| 配置项 | 获取方式 |
| --- | --- |
| `SEPOLIA_RPC_URL` | 在 [Alchemy](https://www.alchemy.com/)、[Infura](https://www.infura.io/) 等服务商创建 Sepolia 应用，复制 HTTPS 地址，形如 `https://eth-sepolia.g.alchemy.com/v2/<key>` |
| `SEPOLIA_PRIVATE_KEY` | 部署账户的私钥（MetaMask：账户详情 → 显示私钥），`0x` 开头的 64 位十六进制 |
| `ETHERSCAN_API_KEY` | 登录 [Etherscan](https://etherscan.io/myapikey) 创建 API Key（同一个 Key 可用于 Sepolia） |
| `SEPOLIA_BIDDER_PRIVATE_KEY`（可选） | 买家账户的私钥，仅[第 7.1 节](#71-脚本自动演示推荐)的演示脚本使用；部署、升级、验证不需要它 |

## 2. 本地演练

部署到测试网之前，建议先在本地节点上完整走一遍部署和升级流程。本地网络会自动部署 Mock 喂价（ETH = $3000，USDC = $1），无需任何敏感配置。

```bash
# 终端 1：启动本地节点（保持运行）
npm run node

# 终端 2
npm run deploy:local     # 部署 V1，结果写入 deployments/localhost.json
npm run upgrade:local    # 升级到 V2，并校验升级前后状态一致
```

如果还想演练[第 7.1 节](#71-脚本自动演示推荐)的拍卖演示，在部署之后、升级之前运行一次 `npm run demo:local`，升级之后再运行一次。本地网络会直接推进链上时间，不需要真的等待。

`deploy:local` 最后会输出自检信息，应看到：

```text
5. 部署自检
  version：1
  ...
  1 ETH = $3000.0
  1 mUSDC = $1.0
```

`upgrade:local` 最后应看到：

```text
✓ 状态校验通过：代理地址不变，原有数据完整保留
```

> 本地节点重启后链上数据会清空，但 `deployments/localhost.json` 仍在，此时 `deploy:local` 会拒绝执行。请先删除该文件，或使用 `REDEPLOY=true` 重新部署（见 [8. 常见问题](#8-常见问题)）。

## 3. 配置敏感信息（keystore）

项目使用 **Hardhat 3 keystore** 加密保存敏感信息，`hardhat.config.ts` 通过 `configVariable` 读取，**不写入任何文件，也不使用 `.env`**。

```bash
npx hardhat keystore set SEPOLIA_RPC_URL
npx hardhat keystore set SEPOLIA_PRIVATE_KEY
npx hardhat keystore set ETHERSCAN_API_KEY

# 可选：仅测试网演示脚本使用（买家账户）
npx hardhat keystore set SEPOLIA_BIDDER_PRIVATE_KEY
```

- 第一次执行时会要求设置 keystore 密码，之后每次读取（部署、升级、验证）都需要输入该密码；
- 命令执行后按提示粘贴对应的值，输入内容不会回显；
- 修改已有的值需要加 `--force`，例如 `npx hardhat keystore set SEPOLIA_RPC_URL --force`。

其他常用命令：

```bash
npx hardhat keystore list    # 查看已保存的 key（不显示值）
npx hardhat keystore path    # 查看 keystore 文件位置
npx hardhat keystore delete SEPOLIA_PRIVATE_KEY
```

## 4. 部署到 Sepolia

```bash
npm run deploy:sepolia
```

> **始终通过 npm 脚本执行部署、升级和验证。** 这三个脚本都使用 `--build-profile production`（开启优化器）。如果混用不同的编译配置，Etherscan 验证会因字节码不一致而失败。

脚本按以下顺序执行：

1. 部署 `AuctionNFT`（owner 为部署账户）和 `MockUSDC`（名称 Mock USD Coin，符号 mUSDC，6 位精度）；
2. 读取 Chainlink 喂价的 `description()` 并与配置比对，防止地址填错，同时打印当前价格；
3. 部署 `NFTAuction` 实现合约，再部署 `ERC1967Proxy`，**在同一笔交易里调用 `initialize`**，防止被他人抢先初始化；
4. 通过代理调用 `setPriceFeed`：ETH 对应 ETH/USD，MockUSDC 对应 USDC/USD；
5. 自检：输出 version、owner、feeRecipient，以及 1 ETH 和 1 mUSDC 换算成的美元价格。

使用的 Chainlink 喂价（配置在 [`scripts/lib/config.ts`](../scripts/lib/config.ts)，地址来源：[Chainlink 官方文档](https://docs.chain.link/data-feeds/price-feeds/addresses?network=ethereum&page=1#sepolia-testnet)）：

| 喂价 | 地址 |
| --- | --- |
| ETH / USD | `0x694AA1769357215DE4FAC081bf1f309aDC325306` |
| USDC / USD | `0xA2F78ab2355fe2f984D808B5CeE7FD0A93D5270E` |

### 部署记录

部署结果写入 `deployments/sepolia.json`（该文件需要提交到仓库），升级和验证脚本都从这里读取地址：

```jsonc
{
  "network": "sepolia",
  "chainId": 11155111,
  "deployer": "0x...",
  "feeRecipient": "0x...",
  "deployedAt": "...",
  "priceFeeds": { "ETH": "0x694A...", "USDC": "0xA2F7..." },
  "contracts": {
    "AuctionNFT":      { "address": "0x...", "constructorArgs": [...], "txHash": "0x..." },
    "MockUSDC":        { ... },
    "NFTAuctionProxy": { ... },   // 拍卖合约入口，升级后不变
    "NFTAuctionV1":    { ... },   // V1 实现合约
    "NFTAuctionV2":    { ... }    // 升级后追加
  },
  "upgrades": [ { "version": 2, "implementation": "0x...", "txHash": "0x...", "timestamp": "..." } ]
}
```

已有部署记录时脚本会拒绝重新部署，防止误覆盖地址。确实需要重新部署时见 [8. 常见问题](#8-常见问题)。

## 5. 在 Etherscan 验证源码

```bash
npm run verify:sepolia
```

依次验证 AuctionNFT、MockUSDC、NFTAuction V1 实现合约、ERC1967Proxy，以及 V2 实现合约（已升级时）。某个合约验证失败不会中断其余合约，最后统一查看输出。已经验证过的合约再次执行不会出错，因此**升级后重新执行一次即可补充验证 V2**。

验证完成后，在 Etherscan 打开**代理地址**：

1. 进入 **Contract** 标签页，若未自动识别为代理，点击 **More Options → Is this a proxy?**，再点 **Verify** 和 **Save**；
2. 识别后会出现 **Read as Proxy / Write as Proxy** 标签页，可以直接按 NFTAuction 的 ABI 调用代理。

## 6. 升级到 V2

```bash
npm run upgrade:sepolia
npm run verify:sepolia    # 补充验证 V2 实现合约
```

升级脚本的流程：

1. 校验当前账户是合约 owner，且当前版本为 1；
2. 记录升级前的快照：实现合约地址、`nextAuctionId`、`feeRecipient`、`priceStaleThreshold`；
3. 部署 `NFTAuctionV2` 实现合约；
4. 调用 `upgradeToAndCall`，**在同一笔交易里调用 `initializeV2(300, 300)`**，开启防狙击（最后 5 分钟内出价，延长 5 分钟）；
5. 校验实现合约已切换到 V2，且升级前后状态一致，最后把升级信息追加到 `deployments/sepolia.json`。

成功时输出：

```text
升级后：
  version：2
  ...
  防狙击：最后 300 秒内出价，延长 300 秒

✓ 状态校验通过：代理地址不变，原有数据完整保留
```

> **想演示"升级后数据保留"**：升级前先运行一次演示脚本，在 V1 上创建拍卖并出价，升级后再运行一次，见[第 7.1 节](#71-脚本自动演示推荐)。

## 7. 测试网演示一场拍卖

演示需要两个账户：部署账户 **A** 作为卖家，另一个账户 **B** 作为买家（合约禁止卖家给自己出价）。可以用演示脚本自动执行，也可以在 Etherscan 上手动操作。

### 7.1 脚本自动演示（推荐）

演示脚本 [`scripts/demo.ts`](../scripts/demo.ts) 运行两次，每次运行时根据演示记录和链上状态自动判断该执行哪个阶段。配合升级可以完整演示"升级后数据保留"和 V2 的防狙击延时：

```bash
npx hardhat keystore set SEPOLIA_BIDDER_PRIVATE_KEY   # 首次：保存买家 B 的私钥

npm run demo:sepolia      # 第一阶段（V1）：创建拍卖并出价
npm run upgrade:sepolia   # 升级到 V2（需在拍卖结束前完成，拍卖时长 15 分钟）
npm run verify:sepolia    # 补充验证 V2 实现合约
npm run demo:sepolia      # 第二阶段（V2）：核对数据、加价触发延时、结算、提现
```

| 阶段 | 合约版本 | 执行内容 |
| --- | --- | --- |
| 第一阶段 | V1 | A 铸造 NFT → A 授权 → A 创建拍卖（起拍价 $5，时长 15 分钟，接受 ETH 与 mUSDC）→ B 用约 $6 的 ETH 出价 |
| 第二阶段 | V2 | 打印升级后的拍卖数据（应与升级前一致）→ 等到最后 5 分钟 → B 领取并授权 10 mUSDC → B 用 mUSDC 加价，结束时间延长 300 秒并触发 `AuctionExtended` 事件 → 等待到期 → 结算 → A 提现货款与手续费，B 取回被超越的 ETH 出价 |

- 第二阶段会在测试网上**真实等待**链上时间，最长约 15 分钟，期间每 15 秒输出一次剩余时间；
- 每笔交易执行后立即写入 `deployments/sepolia-demo.json`（含交易哈希和 Etherscan 链接），该文件需要提交到仓库，作为演示凭证；
- 演示完成后再次运行，脚本会拒绝执行，防止覆盖已有记录；如需重新演示，先把该文件改名备份；
- 买家私钥由演示脚本单独从 keystore 读取，没有加入 `hardhat.config.ts` 的 `accounts`，因此部署、升级、验证不需要它；
- 演示金额在 [`scripts/lib/config.ts`](../scripts/lib/config.ts) 的 `DEMO` 中配置。

如果升级前拍卖已经到期，第二阶段会跳过防狙击演示，直接结算和提现。

### 7.2 在 Etherscan 上手动操作

以下操作都在 Etherscan 上完成：连接 MetaMask 后，在合约的 **Write Contract**（代理为 **Write as Proxy**）标签页调用。

> **金额单位**：Etherscan 的 `payable` 输入框以 **ETH** 为单位；其余参数都是整数，需要自己换算精度：美元金额为 18 位精度（$10 = `10000000000000000000`），mUSDC 为 6 位精度（50 mUSDC = `50000000`）。

| 步骤 | 账户 | 合约 | 调用 | 参数示例 |
| --- | --- | --- | --- | --- |
| 1. 铸造 NFT | A | AuctionNFT | `mint` | `to` = A 的地址，`uri` = `ipfs://demo/1`；新 tokenId 可在 `nextTokenId` 减 1 或交易日志中查看（从 1 开始） |
| 2. 授权 | A | AuctionNFT | `approve` | `to` = **代理地址**，`tokenId` = 1 |
| 3. 创建拍卖 | A | 代理 | `createAuction` | `nft` = AuctionNFT 地址，`tokenId` = 1，`paymentToken` = MockUSDC 地址（只收 ETH 时填 `0x0000000000000000000000000000000000000000`），`startPriceUsd` = `10000000000000000000`（$10），`duration` = `600`（10 分钟，范围 60 ~ 2592000 秒） |
| 4. ETH 出价 | B | 代理 | `bidWithETH` | `payableAmount` 填 `0.005`（ETH），`auctionId` = 0 |
| 5. 领取测试代币 | B | MockUSDC | `mint` | `to` = B 的地址，`amount` = `100000000`（100 mUSDC，任何人都可以铸造） |
| 6. 授权代币 | B | MockUSDC | `approve` | `spender` = **代理地址**，`value` = `100000000` |
| 7. mUSDC 加价 | B | 代理 | `bidWithToken` | `auctionId` = 0，`amount` = 出价金额，换算成美元后需高于当前最高价 |
| 8. 结算 | 任何人 | 代理 | `endAuction` | `auctionId` = 0，需到期后调用 |
| 9. 提现 | A / B | 代理 | `withdraw` | `token` = 币种地址（ETH 填零地址，mUSDC 填 MockUSDC 地址） |

查看状态（**Read as Proxy**）：

| 调用 | 用途 |
| --- | --- |
| `getAuction(auctionId)` | 拍卖详情：卖家、结束时间、最高出价者、出价币种与金额、美元价值、状态（0 进行中 / 1 已结束 / 2 已取消） |
| `toUsd(token, amount)` | 预估某笔出价的美元价值，决定出价金额前可先查询 |
| `pendingReturns(user, token)` | 某账户在某币种上的待提现金额 |
| `getLatestPrice(token)` | 当前喂价及精度 |
| `getFeeTiers()` / `getFeeBps(amountUsd)` | 手续费阶梯 / 某成交额对应的费率 |
| `version()` | 1 为 V1，2 为 V2 |

演示中可以观察到：

- 第 7 步 B 用 mUSDC 超过了自己之前的 ETH 出价，旧出价不会丢失，而是记入 `pendingReturns(B, 0x000…000)`，可以随时提现；
- V2 下，在拍卖最后 5 分钟内出价，`getAuction` 中的 `endTime` 会延长 300 秒，并触发 `AuctionExtended` 事件；
- 结算后 NFT 归 B；成交价 < $1,000 时收取 5% 手续费，记入 feeRecipient（即 A）。由于 A 同时是卖家和手续费接收地址，A 的待提现金额等于全部成交额。

## 8. 常见问题

**输入 keystore 密码后报 `Field.fromBytes: expected 32 bytes, got 20`**
`SEPOLIA_PRIVATE_KEY` 里存的是**钱包地址**（20 字节，`0x` + 40 位十六进制），而不是**私钥**（32 字节，`0x` + 64 位十六进制）。此时还没有发出任何交易。用 `npx hardhat keystore set SEPOLIA_PRIVATE_KEY --force` 重新填入私钥即可。

**演示脚本报 `未找到买家私钥` 或 `买家与卖家是同一账户`**
执行 `npx hardhat keystore set SEPOLIA_BIDDER_PRIVATE_KEY` 保存买家私钥；买家必须是部署账户以外的另一个账户。

**`xxx 已有部署记录，如需重新部署请设置 REDEPLOY=true`**
该网络已有部署记录。确实要重新部署时，在命令前设置环境变量（新地址会覆盖旧记录）：

```bash
# Git Bash / macOS / Linux
REDEPLOY=true npm run deploy:sepolia
# PowerShell
$env:REDEPLOY="true"; npm run deploy:sepolia; Remove-Item Env:REDEPLOY
```

**`未找到 deployments/xxx.json，请先运行 scripts/deploy.ts`**
升级和验证依赖部署记录，请先部署；本地节点重启后需要重新部署。

**Etherscan 验证失败，提示字节码不一致**
部署和验证使用了不同的编译配置。请统一通过 `npm run deploy:sepolia` / `verify:sepolia` 执行，不要直接运行 `npx hardhat run` 且漏掉 `--build-profile production`。

**出价或创建拍卖时报 `StalePrice`**
喂价的最后更新时间超过了过期阈值（默认 1 天）。Sepolia 喂价的更新频率较低，可由 owner 在代理上调用 `setPriceStaleThreshold` 放宽阈值，例如 `172800`（2 天）。

**出价时报 `BidTooLow(bidUsd, minRequiredUsd)`**
出价换算成美元后不足：首次出价需 ≥ 起拍价，之后需 > 当前最高价。两个参数都是 18 位精度的美元金额，可先用 `toUsd` 预估。

**报 `insufficient funds`**
部署账户的 Sepolia ETH 不足，请去水龙头领取。

**升级时报 `当前账户 ... 不是合约 owner`**
keystore 中的 `SEPOLIA_PRIVATE_KEY` 与部署时使用的账户不一致。
