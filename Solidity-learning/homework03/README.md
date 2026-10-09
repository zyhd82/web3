# NFT 拍卖市场（NFT Auction Market）

基于 **Hardhat 3** 的 NFT 拍卖市场，主要特性：

- **ETH / ERC20 混合出价**：通过 **Chainlink 价格预言机**把不同币种的出价统一换算为美元，再比较高低；
- **UUPS 可升级**：拍卖合约部署在 `ERC1967Proxy` 之后，演示从 V1 升级到 V2（新增防狙击延时），升级后原有数据完整保留；
- **动态手续费**：按成交额的美元价值分档收费，阶梯由管理员配置；
- **安全设计**：退款与货款统一 pull 提现、重入保护、预言机数据校验、检查-生效-交互顺序。

| 项目 | 版本 |
| --- | --- |
| Solidity | 0.8.28 |
| Hardhat | 3.x（mocha + ethers v6，TypeScript 测试） |
| OpenZeppelin | 5.6（`contracts` + `contracts-upgradeable`） |
| Node.js | 22 及以上（本项目在 Node 24 / npm 12 下开发） |

## 目录

- [功能说明](#功能说明)
- [合约总览](#合约总览)
- [项目结构](#项目结构)
- [快速开始](#快速开始)
- [测试与覆盖率](#测试与覆盖率)
- [部署](#部署)
- [Sepolia 部署地址](#sepolia-部署地址)
- [Sepolia 演示记录](#sepolia-演示记录)
- [相关文档](#相关文档)

## 功能说明

### 拍卖流程

1. **创建拍卖**：NFT 持有者先 `approve` 拍卖合约，再调用 `createAuction` 指定 NFT、可接受的 ERC20（可选）、起拍价（美元）和时长（60 秒 ~ 30 天），NFT 托管进拍卖合约。
2. **出价**：任何人（卖家除外）在到期前用 `bidWithETH` 或 `bidWithToken` 出价。出价按当时的 Chainlink 价格换算成美元：首次出价需 **≥ 起拍价**，之后需 **> 当前最高价**。被超越的出价记入出价者的待提现余额。
3. **结算**：到期后**任何人**都可以调用 `endAuction`：
   - 有人出价：NFT 转给最高出价者；成交金额扣除手续费后记入卖家待提现余额，手续费记入 `feeRecipient`；
   - 无人出价（流拍）：NFT 退回卖家。
4. **取消**：拍卖无人出价时，卖家可以调用 `cancelAuction` 取消，NFT 退回。
5. **提现**：所有应收款（被超越的出价、货款、手续费）都通过 `withdraw(币种)` 自行取回，合约不会主动推送资金。

### 价格换算

- `priceFeeds[token]` 保存代币对应的 Chainlink 喂价合约，`address(0)` 代表 ETH；
- 读取价格时校验 `answer > 0` 且数据未过期（默认阈值 1 天，管理员可调），精度读取 `decimals()`，不硬编码；
- 合约内美元金额统一为 **18 位精度**（`1e18` = $1），例如起拍价 $100 传入 `100000000000000000000`。

### 动态手续费

| 成交额（美元） | 费率 |
| --- | --- |
| < $1,000 | 5% |
| $1,000 ~ $10,000 | 3% |
| ≥ $10,000 | 1% |

手续费按成交币种的原始金额计算，与货款同币种。管理员可通过 `setFeeTiers` 修改阶梯（最多 10 档，单档费率上限 10%）。

### 合约升级（V2）

`NFTAuctionV2` 继承 V1，只追加状态变量，新增 **防狙击延时**：拍卖最后 5 分钟内有人出价，结束时间自动延长 5 分钟（参数可由管理员调整，窗口设为 0 即关闭）。升级通过 `upgradeToAndCall` 完成，并在同一笔交易里调用 `initializeV2` 完成初始化；代理地址不变，`version()` 从 1 变为 2。

## 合约总览

| 合约 | 说明 | 部署方式 |
| --- | --- | --- |
| [`AuctionNFT`](contracts/AuctionNFT.sol) | 拍品 NFT（ERC721URIStorage + Ownable），仅 owner 可铸造，tokenId 从 1 开始 | 普通部署 |
| [`NFTAuction`](contracts/NFTAuction.sol) | 拍卖核心逻辑 V1（UUPS） | 实现合约 + `ERC1967Proxy` |
| [`NFTAuctionV2`](contracts/NFTAuctionV2.sol) | 升级版本，新增防狙击延时 | 部署实现合约后 `upgradeToAndCall` |
| [`MockERC20`](contracts/mocks/MockERC20.sol) | 可自由铸造的测试代币，测试网上作为 mUSDC 出价币种 | 本地 + 测试网 |
| [`MockV3Aggregator`](contracts/mocks/MockV3Aggregator.sol) | 模拟 Chainlink 喂价 | 仅本地 |
| [`MaliciousBidder`](contracts/mocks/MaliciousBidder.sol) | 模拟恶意出价者（重入、拒收 ETH） | 仅测试 |
| [`AggregatorV3Interface`](contracts/interfaces/AggregatorV3Interface.sol) | Chainlink 官方接口 | — |

用户只与**代理地址**交互；实现合约可以替换，状态全部保存在代理中。

## 项目结构

```text
homework03/
├── contracts/
│   ├── AuctionNFT.sol            # 拍品 NFT
│   ├── NFTAuction.sol            # 拍卖合约 V1（UUPS）
│   ├── NFTAuctionV2.sol          # 拍卖合约 V2（防狙击延时）
│   ├── interfaces/               # Chainlink AggregatorV3Interface
│   └── mocks/                    # MockERC20、MockV3Aggregator、MaliciousBidder
├── test/
│   ├── helpers/fixtures.ts       # 公共部署场景与工具函数
│   ├── AuctionNFT.test.ts
│   ├── Mocks.test.ts
│   ├── NFTAuction.test.ts        # V1 全部功能
│   └── NFTAuctionUpgrade.test.ts # 升级、数据保留、防狙击
├── scripts/
│   ├── deploy.ts                 # 部署 V1（本地自动部署 Mock 喂价）
│   ├── upgrade.ts                # 升级到 V2 并校验状态
│   ├── verify.ts                 # Etherscan 源码验证
│   ├── demo.ts                   # 演示一场完整拍卖（跨越升级，演示数据保留与防狙击）
│   └── lib/                      # 网络配置、部署记录读写
├── deployments/                  # 部署记录与测试网演示记录
├── docs/
│   ├── DESIGN.md                 # 合约详细设计
│   └── DEPLOYMENT.md             # 部署指南
├── hardhat.config.ts
└── TASK_REQUIREMENTS.md          # 需求与已确认的决策
```

## 快速开始

所有命令都在 `homework03/` 目录下执行。

```bash
npm install          # 安装依赖
npx hardhat compile  # 编译合约
npx hardhat test     # 运行全部测试
```

> 使用 npm 12 安装时可能出现 `esbuild ... install scripts blocked` 的警告，不影响使用，可以忽略。

在本地节点上演练部署和升级（需要两个终端）：

```bash
# 终端 1：启动本地节点（保持运行）
npm run node

# 终端 2：部署 V1，再升级到 V2
npm run deploy:local
npm run upgrade:local
```

本地网络会自动部署 Mock 喂价（ETH = $3000，USDC = $1），部署记录写入 `deployments/localhost.json`。重启本地节点后链上数据会清空，请先删除该文件再重新部署。

## 测试与覆盖率

```bash
npx hardhat test                           # 全部测试
npx hardhat test test/NFTAuction.test.ts   # 单个文件
npx hardhat test --grep "出价"             # 按用例名过滤
npx hardhat test --coverage                # 覆盖率（HTML 报告在 coverage/html）
```

共 **132 个用例**，覆盖正常路径、权限校验、边界值和失败回滚：

| 测试文件 | 覆盖内容 |
| --- | --- |
| `AuctionNFT.test.ts` | 铸造权限、tokenId 自增、tokenURI、转移 |
| `Mocks.test.ts` | Mock 喂价与 Mock 代币 |
| `NFTAuction.test.ts` | 初始化、管理员配置、价格换算与喂价校验、动态手续费、创建 / 出价 / 加价退款 / 结算 / 取消 / 提现、重入与 DoS 防护 |
| `NFTAuctionUpgrade.test.ts` | 升级权限、升级前后数据保留、V2 防狙击延时、`initializeV2` 只能调用一次、再次升级 |

覆盖率结果：

| 合约 | 行覆盖率 | 语句覆盖率 |
| --- | --- | --- |
| NFTAuction.sol | 100% | 100% |
| NFTAuctionV2.sol | 100% | 100% |
| AuctionNFT.sol | 100% | 100% |
| mocks/*.sol | 100% | 100% |

## 部署

部署到 Sepolia 测试网并在 Etherscan 验证源码，完整步骤见 **[部署指南](docs/DEPLOYMENT.md)**。简要流程：

```bash
npx hardhat keystore set SEPOLIA_RPC_URL
npx hardhat keystore set SEPOLIA_PRIVATE_KEY
npx hardhat keystore set ETHERSCAN_API_KEY

npm run deploy:sepolia    # 部署 V1
npm run verify:sepolia    # 验证源码
npm run upgrade:sepolia   # 升级到 V2
npm run verify:sepolia    # 验证 V2 实现合约
```

在测试网上演示一场完整拍卖（需要另一个账户作为买家）：升级前运行一次 `npm run demo:sepolia`，升级后再运行一次，详见部署指南第 7 节。

## Sepolia 部署地址

> 部署于 2026-10-09，部署账户 `0x507d6556376A819efda673B5c271d7898239f543`（同时是 owner 与手续费接收地址），完整记录见 [`deployments/sepolia.json`](deployments/sepolia.json)。所有合约均已在 Etherscan 验证源码。

| 合约 | 地址 |
| --- | --- |
| NFTAuction 代理（交互入口） | [`0x63189dB992c04A3Acc6c12d69Accaafe912b5a69`](https://sepolia.etherscan.io/address/0x63189dB992c04A3Acc6c12d69Accaafe912b5a69) |
| NFTAuction V1 实现 | [`0xfB9CDbE830228632EC2209481A7a8AC29D457325`](https://sepolia.etherscan.io/address/0xfB9CDbE830228632EC2209481A7a8AC29D457325) |
| NFTAuction V2 实现（当前） | [`0x6c228EC3A73Db48E9dc9c2e92e32E4E9FA0567c4`](https://sepolia.etherscan.io/address/0x6c228EC3A73Db48E9dc9c2e92e32E4E9FA0567c4) |
| AuctionNFT | [`0xB42d582Cd6ECA20357F4DB83f65484c9E945f5ac`](https://sepolia.etherscan.io/address/0xB42d582Cd6ECA20357F4DB83f65484c9E945f5ac) |
| MockUSDC（mUSDC） | [`0x160A1d6767c8d0302753c1D44DaBD08647539AED`](https://sepolia.etherscan.io/address/0x160A1d6767c8d0302753c1D44DaBD08647539AED) |

使用的 Chainlink 喂价：

| 喂价 | 地址 |
| --- | --- |
| ETH / USD | [`0x694AA1769357215DE4FAC081bf1f309aDC325306`](https://sepolia.etherscan.io/address/0x694AA1769357215DE4FAC081bf1f309aDC325306) |
| USDC / USD | [`0xA2F78ab2355fe2f984D808B5CeE7FD0A93D5270E`](https://sepolia.etherscan.io/address/0xA2F78ab2355fe2f984D808B5CeE7FD0A93D5270E) |

## Sepolia 演示记录

2026-10-09 在 Sepolia 上用 [`scripts/demo.ts`](scripts/demo.ts) 演示了一场**跨越升级**的完整拍卖：拍卖在 V1 上创建并出价，[升级到 V2](https://sepolia.etherscan.io/tx/0x848337c847cf82a0a94ca1e61510153a099742fc0662293c1ef27e22da1e1b1f) 后继续出价并结算。卖家 A 为部署账户，买家 B 为 `0x6EeD95ac2e883067413781b71FE18d54798e3E40`。完整记录见 [`deployments/sepolia-demo.json`](deployments/sepolia-demo.json)。

| # | 版本 | 操作 | 交易 |
| --- | --- | --- | --- |
| 1 | V1 | A 铸造 NFT #1 | [0x6f68b9…1c16](https://sepolia.etherscan.io/tx/0x6f68b9ba91c3166e7f7226ffc5fca358ee7e79cb3ea34a243ca51c5125341c16) |
| 2 | V1 | A 授权拍卖合约转移 NFT #1 | [0x6a5d0f…84c7](https://sepolia.etherscan.io/tx/0x6a5d0f0f1d189c7aa346f24ff4ffdf0cd3f8ceb4498740130e51905ad2c284c7) |
| 3 | V1 | A 创建拍卖 #0（起拍价 $5，时长 15 分钟，接受 ETH 与 mUSDC） | [0x51b280…ce9b](https://sepolia.etherscan.io/tx/0x51b280ce641485201129a27a6265718758cda6ada6be26f597832e9f8143ce9b) |
| 4 | V1 | B 用 0.0025 ETH 出价（约 $6.25，按 Chainlink ETH/USD 换算） | [0x2ba5d7…9131](https://sepolia.etherscan.io/tx/0x2ba5d71036f2ae47ba18f646a2490cf62ff01d2318818f96de6d70b0d59e9131) |
| — | — | **升级到 V2**：拍卖 #0 的数据完整保留 | [0x848337…1b1f](https://sepolia.etherscan.io/tx/0x848337c847cf82a0a94ca1e61510153a099742fc0662293c1ef27e22da1e1b1f) |
| 5 | V2 | B 领取 10 mUSDC 测试代币 | [0x225e27…7bd4](https://sepolia.etherscan.io/tx/0x225e2716a027c68ed27ce7debb6d8ff66769956320c40276e6573d9f838f7bd4) |
| 6 | V2 | B 授权拍卖合约转移 mUSDC | [0x9f63bb…a694](https://sepolia.etherscan.io/tx/0x9f63bb9e6e770bea21abd755550716555bb0314768a632903e218cfa9ee2a694) |
| 7 | V2 | B 在最后 5 分钟内用 10 mUSDC 加价（约 $10）：结束时间延长 300 秒，触发 `AuctionExtended`；被超越的 0.0025 ETH 记入 B 的待提现 | [0xf8fa91…67c1](https://sepolia.etherscan.io/tx/0xf8fa9189c967f004dd011c218d3d3d3d9c226fd9075f6d26d315f023f13d67c1) |
| 8 | V2 | 结算：NFT #1 转给 B，成交 10 mUSDC，手续费 0.5 mUSDC（< $1,000 档，5%） | [0x4311d0…85b2](https://sepolia.etherscan.io/tx/0x4311d04a90d7753cd7d9d8980f2cadb6c4e2d3a606cbde1434fbd096066d85b2) |
| 9 | V2 | A 提现 10 mUSDC（货款 9.5 + 手续费 0.5，A 同时是手续费接收地址） | [0xe56655…c660](https://sepolia.etherscan.io/tx/0xe56655971d65b864aa67dbcf19d8a80b987cb369a4094dac8799df5a053bc660) |
| 10 | V2 | B 取回被超越的 0.0025 ETH 出价 | [0x0968ed…fc14](https://sepolia.etherscan.io/tx/0x0968ed8ecba185aaa59ae8046335637569c60481b9040186adea4db1598fbc14) |

演示结束后的链上状态：拍卖 #0 已结束，NFT #1 归 B；拍卖时长由 900 秒延长到 1200 秒；所有账户的待提现余额均为 0。

## 相关文档

- [TASK_REQUIREMENTS.md](TASK_REQUIREMENTS.md)：需求与已确认的决策
- [docs/DESIGN.md](docs/DESIGN.md)：合约详细设计（数据结构、存储布局、函数、事件、错误）
- [docs/DEPLOYMENT.md](docs/DEPLOYMENT.md)：部署、升级、验证与测试网演示步骤
