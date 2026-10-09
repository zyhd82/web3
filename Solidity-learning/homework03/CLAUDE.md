# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 项目简介

NFT 拍卖市场：Hardhat 3 + ethers v6 + mocha/chai（TypeScript 测试），Solidity 0.8.28，OpenZeppelin 5.6（含 upgradeable）。拍卖合约采用 **UUPS 可升级**，通过 **Chainlink 价格预言机**把 ETH / ERC20 出价统一换算为美元比较，并按成交额收取**动态手续费**。

这是 `Solidity-learning/` 下的同级作业之一（`homework01`~`homework03`），只有本目录是 Hardhat 工程 —— 所有 `npx hardhat` 命令都要在 `homework03/` 内执行。

需求与已确认的决策记录在 `TASK_REQUIREMENTS.md`，合约详细设计在 `docs/DESIGN.md`，部署步骤在 `docs/DEPLOYMENT.md`，项目总览在 `README.md`。**改需求或设计前先看这两份文档，变更后同步更新。**

## 常用命令

```bash
npx hardhat compile                         # 编译（default profile，不开优化器）
npx hardhat test                            # 运行全部测试
npx hardhat test test/NFTAuction.test.ts    # 运行单个测试文件
npx hardhat test --grep "出价"              # 按用例名过滤
npx hardhat test --coverage                 # 覆盖率报告（输出到 coverage/html）
npx hardhat keystore set SEPOLIA_RPC_URL    # 设置敏感配置（另有 SEPOLIA_PRIVATE_KEY、ETHERSCAN_API_KEY）

npm run node                                # 启动本地节点（另开终端执行下面的 :local 命令）
npm run deploy:local / upgrade:local        # 本地演练部署与升级
npm run deploy:sepolia                      # 部署到 Sepolia（production 编译配置）
npm run upgrade:sepolia                     # 升级到 V2
npm run verify:sepolia                      # Etherscan 源码验证
npm run demo:local / demo:sepolia           # 演示一场拍卖：升级前运行一次，升级后再运行一次
```

部署脚本在 `scripts/`，公共配置（喂价地址、代币参数、防狙击参数、演示参数）在 `scripts/lib/config.ts`。演示脚本 `demo.ts` 的记录写入 `deployments/<网络名>-demo.json`，测试网的买家私钥由脚本单独从 keystore 读取（`SEPOLIA_BIDDER_PRIVATE_KEY`），**不要加进 `hardhat.config.ts` 的 `accounts`**——Hardhat 会在首次请求时解析全部账户，加进去会让部署、升级、验证也必须配置它。部署结果写入 `deployments/<网络名>.json`，升级与验证脚本都从这里读取地址；已有记录时 `deploy.ts` 拒绝覆盖（需 `REDEPLOY=true`）。**部署、升级、验证必须使用同一个 build profile**，否则 Etherscan 验证会因字节码不一致而失败——统一通过 npm 脚本执行即可。

## 环境与依赖注意事项

- 本机 npm 为 12.x，**默认禁止 git 依赖**（`EALLOWGIT`），因此：
  - 不安装 `forge-std`，测试统一用 TypeScript 写，不写 Solidity 测试；
  - 不安装 `@chainlink/contracts`（会连带拉取 git 依赖和多个旧版 OpenZeppelin），`AggregatorV3Interface` 原样放在 `contracts/interfaces/`。
- 不使用 `@openzeppelin/hardhat-upgrades` 插件：代理由 `ERC1967Proxy` 手动部署（已在 `hardhat.config.ts` 的 `npmFilesToBuild` 中声明），升级调用 `upgradeToAndCall`。
- OpenZeppelin 5.6 的 upgradeable 包**没有** `ReentrancyGuardUpgradeable`，直接使用 `@openzeppelin/contracts/utils/ReentrancyGuard.sol`（其存储已采用 ERC-7201 命名空间，可安全用于代理合约）。
- 测试中获取网络连接用 `network.create()`，**不要用已弃用的 `network.connect()`**；同一个用例内的时间推进、交易必须使用同一个连接。

## 目录结构

```text
contracts/
  interfaces/   # 外部协议接口（Chainlink AggregatorV3Interface）
  mocks/        # 仅测试使用的 Mock 合约（MockV3Aggregator、MockERC20），不部署到测试网（Mock ERC20 除外）
  *.sol         # 业务合约：AuctionNFT、NFTAuction、NFTAuctionV2
test/           # TypeScript 测试，一个业务合约对应一个 *.test.ts
scripts/        # 部署、升级、交互脚本
docs/           # 设计文档、部署指南
```

## 编码规范

### 通用

- **所有注释、NatSpec、文档一律使用简体中文**；标识符（合约名、函数名、变量名）使用英文。
- 每个业务合约一个文件，文件名与合约名一致。

### Solidity

- 版本固定 `pragma solidity 0.8.28;`（接口文件可用 `^0.8.0`）。
- 使用具名导入：`import {Foo} from "...";`，不使用 `import "...";` 全量导入。
- 错误处理使用**自定义错误**（`error AuctionNotFound(uint256 auctionId);`），不使用 `require` 字符串。
- 对外函数、事件、自定义错误都写 NatSpec（`@notice`，必要时 `@param` / `@return`）。
- 状态变更遵循 **检查-生效-交互**（Checks-Effects-Interactions）顺序；涉及转账或外部调用的函数加 `nonReentrant`。
- ETH 转账使用 `call` 并检查返回值，不使用 `transfer` / `send`；ERC20 使用 `SafeERC20`。
- 退款采用 **pull 模式**（记账后由用户 `withdraw`），不在出价流程中直接推送资金。
- 命名：私有/内部函数与状态变量以下划线开头（`_toUsd`），常量全大写（`MAX_DURATION`）。

### 可升级合约（UUPS）

- 不写带逻辑的构造函数，构造函数只调用 `_disableInitializers()`；初始化放在 `initialize`（`initializer` 修饰）。
- `_authorizeUpgrade` 必须加 `onlyOwner`。
- **存储布局只能追加**：不得删除、重排或修改已有状态变量类型。V1 末尾预留 `uint256[50] private __gap`，仅在将来修改 V1 自身时缩减；V2 继承 V1，新变量直接声明在 V2 中（自然排在 gap 之后），不改动 V1。
- 不使用 `immutable`（`constant` 可以），不在状态变量声明处赋初值。

### TypeScript 测试

- 使用 `loadFixture` 复用部署场景，按"合约 → 函数 → 场景"组织 `describe` / `it`，用例名用中文描述预期行为。
- 断言 revert 时使用 `revertedWithCustomError` 并校验参数（`.withArgs(...)`）。
- 每个功能至少覆盖：正常路径、权限校验、边界值、失败回滚。
