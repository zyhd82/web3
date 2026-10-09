# homework03 任务需求文档

> 本文件用于记录 NFT 拍卖市场项目的任务目标与需求，避免重复沟通确认。后续如有需求变更，请更新此文件。

## 项目名称

NFT 拍卖市场（NFT Auction Market）

## 项目目标

1. 使用 **Hardhat** 框架开发一个 NFT 拍卖市场。
2. 使用 **Chainlink 价格预言机（Data Feeds）** 计算 ERC20 和以太坊（ETH）对美元的价格。
3. 使用 **UUPS / 透明代理模式** 实现合约升级。

## 项目要求

1. **代码质量**：代码清晰、规范，符合 Solidity 最佳实践。
2. **功能完整性**：实现所有要求的功能，包括 NFT 拍卖、价格计算和合约升级。
3. **测试覆盖率**：编写全面的测试，覆盖所有功能。
4. **文档**：提供详细的文档，包括项目结构、功能说明和部署步骤。

## 功能需求

1. **NFT 合约**：基于 ERC721，支持铸造（mint）和转移。
2. **拍卖合约**：
   - 创建拍卖：卖家指定 NFT、起拍价、拍卖时长。
   - 出价：支持使用 ETH 或 ERC20 代币出价。
   - 结束拍卖：拍卖结束后，NFT 转给最高出价者，资金转给卖家。
3. **价格计算**：通过 Chainlink 预言机把 ETH / ERC20 出价换算为美元，用于统一比较出价高低。
4. **合约升级**：拍卖相关合约可升级，并通过 V2 版本演示升级后数据保留。
5. **测试**：单元测试覆盖所有功能，并输出覆盖率报告。
6. **部署**：部署到 Sepolia 测试网，并在 Etherscan 验证合约源码。

## 待确认 / 后续补充事项

### 阶段 0：需求范围

- [x] 拍卖管理方式：**不采用工厂模式**，由单个 `NFTAuction` 合约通过 `auctionId` 管理所有拍卖
- [x] 进阶项：**实现动态手续费**（按成交额阶梯收取不同比例）；**不做**跨链拍卖（CCIP）
- [x] 升级方案：**UUPS**（`NFTAuction` 为可升级合约，并提供 V2 演示升级）
- [x] 部署网络：**Sepolia 测试网**（使用真实 Chainlink 预言机，并在 Etherscan 验证源码）

### 阶段 1：技术栈

- [x] 开发框架：**Hardhat 3**
- [x] 测试语言：**TypeScript**
- [x] 客户端库：**ethers v6**
- [x] 部署工具：**普通 scripts**（`scripts/deploy.ts`、`scripts/upgrade.ts`）
- [x] 代理部署方式：**不使用 OpenZeppelin upgrades 插件**，参考项目做法，手动部署 `ERC1967Proxy`（实现合约 + `initialize` 编码数据），升级时调用 `upgradeToAndCall`
- [x] Chainlink 依赖：**不安装 `@chainlink/contracts`**（npm 12 禁止其 git 依赖，且会引入多个旧版 OpenZeppelin），`AggregatorV3Interface` 原样放入 `contracts/interfaces/`，链上仍对接 Chainlink 官方喂价合约
- [x] 测试形式：只写 TypeScript 测试，不写 Solidity 测试（`forge-std` 同样是 git 依赖）
- [x] 敏感配置管理：**Hardhat 3 keystore**（`SEPOLIA_RPC_URL`、`SEPOLIA_PRIVATE_KEY`、`ETHERSCAN_API_KEY`），通过 `configVariable` 读取，不写入任何文件

## 参考项目

- <https://github.com/lc3091/hardhatV3Nft>（Hardhat 3 + UUPS / 透明代理 + Chainlink 的 NFT 拍卖示例）
- 借鉴：Hardhat 3 配置写法、`tokenToOracle` 映射（`address(0)` 代表 ETH）、`MockOracle` / `MockERC20` 测试合约、手动部署代理、`_toUsd` 换算思路
- 需改进（在本项目中修正）：
  1. 拍卖由管理员代卖家创建（`start` 为 `onlyOwner` 且传入 `seller`）→ 改为卖家自己创建
  2. 同一出价者再次加价时，旧出价不退款，资金被锁死
  3. 用 `transfer` 直接推送退款（push），恶意合约出价者可让后续出价全部失败（DoS）→ 改为 pull 提现
  4. 无人出价时 `end` 直接 revert，NFT 永远卡在合约里 → 流拍需退回卖家
  5. 预言机未检查 `answer > 0` 与数据过期，精度硬编码 `10**8` → 读取 `decimals()` 并做校验
  6. 缺少重入保护、结束状态标记、存储间隙（`__gap` / ERC-7201），事件参数不完整（ERC20 出价时 `Bid` 事件金额为 0）
  7. 没有手续费 → 本项目实现动态手续费

### 阶段 2：架构设计

- [x] 创建拍卖：**卖家自己调用 `createAuction`**（`seller = msg.sender`），需先 approve，创建时 NFT 托管进拍卖合约
- [x] 出价币种：**ETH + 创建时指定的一种 ERC20**，出价可混用，统一换算为美元比较高低
- [x] 退款方式：**pull 模式**，被超越的出价记入 `pendingReturns`，出价者自行调用 `withdraw` 取回（按币种分别记账）
- [x] 结算规则：到期后**任何人**可调用 `endAuction`；有出价 → NFT 给最高出价者、扣手续费后货款给卖家；无出价（流拍）→ NFT 退回卖家
- [x] 取消拍卖：卖家可在**无人出价**时取消，NFT 退回
- [x] 动态手续费：按成交价美元金额分三档，**< $1,000 收 5%，$1,000–$10,000 收 3%，≥ $10,000 收 1%**；阶梯存在合约里，管理员可修改；手续费转给管理员设置的 `feeRecipient`
- [x] 加价规则：首次出价 **≥ 起拍价**（美元），之后的出价换算成美元后 **> 当前最高价** 即可
- [x] 拍卖时长：**60 秒 ~ 30 天**（下限较短，方便测试网演示）
- [x] 预言机校验：`answer > 0`，数据更新时间不超过过期阈值（默认 **1 天**，管理员可调；Sepolia 上的喂价更新频率较低）；精度读取 `decimals()`，不硬编码
- [x] NFT 合约：**ERC721URIStorage + Ownable**，仅 owner 可调用 `mint(to, uri)`
- [x] V2 升级演示：版本号返回 2，并新增**防狙击延时**（拍卖最后 5 分钟内有人出价，自动延长 5 分钟），同时验证升级后原有拍卖数据保留
