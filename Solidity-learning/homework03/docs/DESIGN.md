# NFT 拍卖市场 · 合约设计文档

> 本文档是编写合约代码的依据，需求来源见 [`TASK_REQUIREMENTS.md`](../TASK_REQUIREMENTS.md)。设计变更时请先更新本文档。

## 1. 合约总览

| 合约 | 类型 | 部署方式 | 说明 |
| --- | --- | --- | --- |
| `AuctionNFT` | ERC721URIStorage + Ownable | 普通部署 | 拍品 NFT，仅 owner 可铸造 |
| `NFTAuction` | UUPS 可升级（V1） | `ERC1967Proxy` 代理 | 拍卖核心逻辑：创建、出价、结算、取消、提现、价格换算、动态手续费 |
| `NFTAuctionV2` | 继承 `NFTAuction` | 通过 `upgradeToAndCall` 升级 | 版本号变为 2，新增防狙击延时 |
| `MockV3Aggregator` | 测试合约 | 仅本地测试 | 模拟 Chainlink 喂价，可设置价格和更新时间 |
| `MockERC20` | 测试合约 | 本地 + 测试网 | 可自由铸造、可自定义精度的 ERC20（测试网上当作 USDC 使用） |
| `MaliciousBidder` | 测试合约 | 仅本地测试 | 模拟恶意出价者（重入、拒收 ETH），验证安全防护 |
| `AggregatorV3Interface` | 接口 | — | Chainlink 官方接口 |

### 1.1 合约关系

```
                        ┌──────────────────────┐
  用户 ───调用───▶      │   ERC1967Proxy       │  （存储所有状态，地址不变）
                        │   delegatecall ▼     │
                        └──────────┬───────────┘
                                   │ 实现合约（可替换）
                        ┌──────────▼───────────┐   升级    ┌──────────────────┐
                        │   NFTAuction (V1)    │ ───────▶ │  NFTAuctionV2    │
                        └──┬───────┬───────┬───┘          └──────────────────┘
          transferFrom     │       │       │ latestRoundData()
       ┌───────────────────┘       │       └──────────────────────┐
       ▼                           ▼                              ▼
┌──────────────┐        ┌────────────────────┐        ┌──────────────────────────┐
│  AuctionNFT  │        │ ERC20（如 USDC）   │        │ Chainlink 喂价合约        │
│  （ERC721）   │        │ SafeERC20 转账     │        │ ETH/USD、USDC/USD ...     │
└──────────────┘        └────────────────────┘        └──────────────────────────┘
```

### 1.2 NFTAuction 继承关系

```
NFTAuction
 ├── Initializable            （@openzeppelin/contracts-upgradeable）
 ├── OwnableUpgradeable       （管理员权限；ERC-7201 命名空间存储）
 ├── UUPSUpgradeable          （升级入口 upgradeToAndCall；5.6 版无需调用初始化函数）
 └── ReentrancyGuard          （@openzeppelin/contracts；ERC-7201 存储，代理下初始值 0 也能正常工作）
```

## 2. 核心业务流程

### 2.1 拍卖生命周期

```
            createAuction
  (无) ─────────────────────▶ Active ──── cancelAuction（仅卖家，且无人出价）───▶ Cancelled
                                │                                                  NFT 退回卖家
                                │ bidWithETH / bidWithToken（到期前）
                                │ 被超越的出价 → pendingReturns
                                ▼
                     到期后任何人 endAuction
                                │
                ┌───────────────┴────────────────┐
           有人出价                            无人出价（流拍）
   NFT → 最高出价者                        NFT 退回卖家
   (成交额 - 手续费) → 卖家 pendingReturns
   手续费 → feeRecipient pendingReturns
                └───────────────┬────────────────┘
                                ▼
                              Ended
```

### 2.2 资金流向（统一 pull 模式）

合约**从不在业务流程中主动推送资金**，所有应收款都记入 `pendingReturns[用户][币种]`，由用户调用 `withdraw(币种)` 自行取回：

| 场景 | 记账对象 | 金额 |
| --- | --- | --- |
| 出价被超越（含同一人再次加价） | 原最高出价者 | 原出价金额（原币种） |
| 拍卖成交 | 卖家 | 成交金额 − 手续费 |
| 拍卖成交 | `feeRecipient` | 手续费 |

这样可以避免：① 恶意合约拒收 ETH 导致后续出价或结算全部失败（DoS）；② 转账时发生重入。

## 3. 价格换算（Chainlink）

### 3.1 喂价配置

- `priceFeeds[token]` 保存代币对应的喂价合约地址，**`address(0)` 代表 ETH**。
- 管理员通过 `setPriceFeed(token, feed)` 配置。

### 3.2 读取价格 `getLatestPrice(token)`

```solidity
(, int256 answer, , uint256 updatedAt, ) = feed.latestRoundData();
```

校验规则：

| 检查 | 失败时的错误 |
| --- | --- |
| 喂价合约已配置 | `PriceFeedNotSet(token)` |
| `answer > 0` | `InvalidPrice(token, answer)` |
| `updatedAt != 0` 且 `block.timestamp - updatedAt <= priceStaleThreshold` | `StalePrice(token, updatedAt)` |

返回 `(uint256 price, uint8 feedDecimals)`，精度读取 `feed.decimals()`，不硬编码。

### 3.3 换算公式 `toUsd(token, amount)`

**合约内所有美元金额统一使用 18 位精度**（`1e18` = $1），例如起拍价 $100 传入 `100e18`。

```
usd = amount × price × 10^18 / 10^(tokenDecimals + feedDecimals)
```

- `tokenDecimals`：ETH 为 18，ERC20 读取 `IERC20Metadata.decimals()`
- 使用 OpenZeppelin `Math.mulDiv(amount, price * 1e18, 10 ** (tokenDecimals + feedDecimals))`，避免中间结果溢出

**示例**：ETH 价格 $3000（喂价 8 位精度，`answer = 3000e8`），出价 0.5 ETH

```
usd = 0.5e18 × 3000e8 × 1e18 / 1e(18+8) = 1500e18   → $1500
```

**示例**：USDC 价格 $1（`answer = 1e8`），出价 2000 USDC（6 位精度，`amount = 2000e6`）

```
usd = 2000e6 × 1e8 × 1e18 / 1e(6+8) = 2000e18       → $2000
```

### 3.4 已知限制

- 每笔出价的美元价值**在出价时刻锁定**（存入 `highestBidUsd`），之后币价波动不会重新计算。新出价以"当前价格换算的美元"与"已锁定的最高出价美元"比较。
- 不支持转账收费型（fee-on-transfer）或余额会自动变化（rebasing）的 ERC20。

## 4. 动态手续费

### 4.1 阶梯规则

```solidity
struct FeeTier {
    uint256 minUsd;  // 该档起点（含），18 位精度
    uint16  feeBps;  // 费率，基点（1 bps = 0.01%）
}
```

默认阶梯（`initialize` 时写入）：

| 档位 | 成交额（美元） | `minUsd` | 费率 | `feeBps` |
| --- | --- | --- | --- | --- |
| 1 | < $1,000 | `0` | 5% | 500 |
| 2 | $1,000 ~ $10,000 | `1000e18` | 3% | 300 |
| 3 | ≥ $10,000 | `10000e18` | 1% | 100 |

### 4.2 计算方式

1. 用成交价的美元值 `highestBidUsd`，从高到低找到第一个 `minUsd <= highestBidUsd` 的档位，得到 `feeBps`
2. `fee = highestBidAmount × feeBps / 10000`（**按成交币种原始金额计算**，手续费与货款同币种）
3. 卖家所得 `= highestBidAmount − fee`

### 4.3 修改阶梯 `setFeeTiers(FeeTier[] tiers)` 的校验

| 规则 | 失败时的错误 |
| --- | --- |
| 档位数 1 ~ `MAX_FEE_TIERS`（10） | `InvalidFeeTiers()` |
| 第一档 `minUsd == 0` | `InvalidFeeTiers()` |
| `minUsd` 严格递增 | `InvalidFeeTiers()` |
| 每档 `feeBps <= MAX_FEE_BPS`（1000，即 10%） | `InvalidFeeTiers()` |

## 5. NFTAuction（V1）详细设计

### 5.1 常量

| 常量 | 值 | 说明 |
| --- | --- | --- |
| `MIN_DURATION` | `60`（秒） | 最短拍卖时长，便于测试网演示 |
| `MAX_DURATION` | `30 days` | 最长拍卖时长 |
| `BPS_DENOMINATOR` | `10_000` | 基点分母 |
| `MAX_FEE_BPS` | `1_000` | 单档费率上限 10% |
| `MAX_FEE_TIERS` | `10` | 阶梯数量上限 |
| `DEFAULT_STALE_THRESHOLD` | `1 days` | 默认喂价过期阈值 |

### 5.2 数据结构

```solidity
enum AuctionStatus { Active, Ended, Cancelled }

struct Auction {
    address seller;            // 卖家（创建者）
    address nft;               // NFT 合约地址
    uint256 tokenId;           // NFT tokenId
    address paymentToken;      // 允许的 ERC20；address(0) 表示仅接受 ETH
    uint256 startPriceUsd;     // 起拍价（美元，18 位精度）
    uint64  startTime;         // 开始时间
    uint64  endTime;           // 结束时间（V2 防狙击会延长它）
    address highestBidder;     // 当前最高出价者；address(0) 表示无人出价
    address highestBidToken;   // 最高出价币种；address(0) 表示 ETH
    uint256 highestBidAmount;  // 最高出价金额（原币种精度）
    uint256 highestBidUsd;     // 最高出价的美元价值（出价时锁定，18 位精度）
    AuctionStatus status;      // 状态
}
```

> 存储结束时间 `endTime` 而不是时长，V2 延时只需修改这一个字段。

### 5.3 状态变量（存储布局，顺序即槽位顺序）

| # | 变量 | 类型 | 可见性 | 说明 |
| --- | --- | --- | --- | --- |
| 1 | `nextAuctionId` | `uint256` | public | 下一个拍卖 ID，从 0 开始自增 |
| 2 | `_auctions` | `mapping(uint256 => Auction)` | private | 拍卖数据，通过 `getAuction` 读取 |
| 3 | `priceFeeds` | `mapping(address => address)` | public | 代币 → 喂价合约 |
| 4 | `priceStaleThreshold` | `uint256` | public | 喂价过期阈值（秒） |
| 5 | `pendingReturns` | `mapping(address => mapping(address => uint256))` | public | 用户 → 币种 → 待提取金额 |
| 6 | `feeRecipient` | `address` | public | 手续费接收地址 |
| 7 | `_feeTiers` | `FeeTier[]` | private | 手续费阶梯，通过 `getFeeTiers` 读取 |
| 8 | `__gap` | `uint256[50]` | private | 预留槽位 |

> 父合约 `OwnableUpgradeable`、`ReentrancyGuard` 使用 ERC-7201 命名空间存储，不占用上表的顺序槽位。

### 5.4 函数清单

#### 初始化与升级

| 函数 | 权限 | 说明 |
| --- | --- | --- |
| `constructor()` | — | 仅调用 `_disableInitializers()`，防止实现合约被直接初始化 |
| `initialize(address initialOwner, address feeRecipient_)` | `initializer` | 设置 owner、手续费接收地址、默认过期阈值、默认手续费阶梯 |
| `_authorizeUpgrade(address)` | `onlyOwner` | UUPS 升级授权 |
| `version() → uint256` | pure, virtual | 返回 `1`（V2 重写为 `2`） |

#### 管理员配置

| 函数 | 校验 | 事件 |
| --- | --- | --- |
| `setPriceFeed(address token, address feed)` | `feed != 0` | `PriceFeedUpdated` |
| `setPriceStaleThreshold(uint256 threshold)` | `threshold > 0` | `PriceStaleThresholdUpdated` |
| `setFeeRecipient(address recipient)` | `recipient != 0` | `FeeRecipientUpdated` |
| `setFeeTiers(FeeTier[] calldata tiers)` | 见 4.3 | `FeeTiersUpdated` |

以上函数均为 `onlyOwner`。

#### 拍卖操作

**`createAuction(address nft, uint256 tokenId, address paymentToken, uint256 startPriceUsd, uint256 duration) → uint256 auctionId`**（`nonReentrant`）

| 检查 | 失败时的错误 |
| --- | --- |
| `nft != 0` | `ZeroAddress()` |
| `MIN_DURATION <= duration <= MAX_DURATION` | `InvalidDuration(duration)` |
| `startPriceUsd > 0` | `InvalidStartPrice()` |
| ETH 喂价已配置；`paymentToken != 0` 时其喂价也已配置 | `PriceFeedNotSet(token)` |
| 调用者是 NFT 持有者且已授权（由 ERC721 `transferFrom` 校验） | ERC721 自带错误 |

生效：写入 `Auction`、`nextAuctionId++` → 交互：`IERC721(nft).transferFrom(msg.sender, address(this), tokenId)` → 事件 `AuctionCreated`

**`bidWithETH(uint256 auctionId)`**（`payable`, `nonReentrant`）
**`bidWithToken(uint256 auctionId, uint256 amount)`**（`nonReentrant`）

两者都调用 `internal virtual _placeBid(auctionId, token, amount)`（V2 通过重写它实现延时）。

| 检查 | 失败时的错误 |
| --- | --- |
| 拍卖存在（`seller != 0`） | `AuctionNotFound(auctionId)` |
| 状态为 `Active` | `AuctionNotActive(auctionId)` |
| `block.timestamp < endTime` | `AuctionAlreadyEnded(auctionId)` |
| 出价者不是卖家 | `SellerCannotBid()` |
| `amount > 0`（ETH 为 `msg.value`） | `ZeroAmount()` |
| `bidWithToken`：该拍卖设置了 `paymentToken` | `TokenNotAccepted(auctionId)` |
| 首次出价：`usd >= startPriceUsd`；否则：`usd > highestBidUsd` | `BidTooLow(usd, 所需最低美元值)` |

生效：若已有最高出价者，把其出价记入 `pendingReturns[原出价者][原币种]`（**同一人再次加价也退回旧出价**），然后更新最高出价 → 交互：`bidWithToken` 调用 `safeTransferFrom(msg.sender, address(this), amount)` → 事件 `BidPlaced`

**`endAuction(uint256 auctionId)`**（`nonReentrant`，任何人可调用）

| 检查 | 失败时的错误 |
| --- | --- |
| 拍卖存在 | `AuctionNotFound(auctionId)` |
| 状态为 `Active` | `AuctionNotActive(auctionId)` |
| `block.timestamp >= endTime` | `AuctionNotYetEnded(auctionId, endTime)` |

生效：`status = Ended`；有出价时按第 4 节计算手续费，把货款和手续费分别记入 `pendingReturns` → 交互：NFT 通过 `transferFrom` 转给最高出价者（流拍时退回卖家）→ 事件 `AuctionEnded(auctionId, winner, token, amount, fee)`（流拍时 `winner = address(0)`）

> 转出 NFT 使用 `transferFrom` 而非 `safeTransferFrom`：避免出价者是未实现 `onERC721Received` 的合约时，结算永远失败导致 NFT 卡死。

**`cancelAuction(uint256 auctionId)`**（`nonReentrant`）

| 检查 | 失败时的错误 |
| --- | --- |
| 拍卖存在 | `AuctionNotFound(auctionId)` |
| 调用者是卖家 | `NotSeller()` |
| 状态为 `Active` | `AuctionNotActive(auctionId)` |
| 无人出价 | `AuctionHasBids(auctionId)` |

生效：`status = Cancelled` → 交互：NFT 退回卖家 → 事件 `AuctionCancelled`

**`withdraw(address token)`**（`nonReentrant`）

- `pendingReturns[msg.sender][token] == 0` → `NothingToWithdraw()`
- 先清零再转账：ETH 用 `call`（失败 → `EthTransferFailed()`），ERC20 用 `safeTransfer`
- 事件 `Withdrawn`

#### 查询

| 函数 | 说明 |
| --- | --- |
| `getAuction(uint256 auctionId) → Auction` | 读取拍卖完整信息 |
| `getLatestPrice(address token) → (uint256 price, uint8 decimals)` | 读取并校验喂价，见 3.2 |
| `toUsd(address token, uint256 amount) → uint256` | 换算成美元（18 位精度），见 3.3 |
| `getFeeBps(uint256 amountUsd) → uint16` | 查询某美元金额对应的费率 |
| `getFeeTiers() → FeeTier[]` | 读取全部手续费阶梯 |

合约**不实现 `receive` / `fallback`**，直接向合约转 ETH 会失败，只能通过 `bidWithETH` 转入。

### 5.5 事件

| 事件 | 参数 |
| --- | --- |
| `AuctionCreated` | `uint256 indexed auctionId, address indexed seller, address indexed nft, uint256 tokenId, address paymentToken, uint256 startPriceUsd, uint64 endTime` |
| `BidPlaced` | `uint256 indexed auctionId, address indexed bidder, address token, uint256 amount, uint256 amountUsd` |
| `AuctionEnded` | `uint256 indexed auctionId, address indexed winner, address token, uint256 amount, uint256 fee` |
| `AuctionCancelled` | `uint256 indexed auctionId` |
| `Withdrawn` | `address indexed user, address indexed token, uint256 amount` |
| `PriceFeedUpdated` | `address indexed token, address indexed feed` |
| `PriceStaleThresholdUpdated` | `uint256 threshold` |
| `FeeRecipientUpdated` | `address indexed recipient` |
| `FeeTiersUpdated` | `FeeTier[] tiers` |

### 5.6 自定义错误

| 错误 | 触发场景 |
| --- | --- |
| `ZeroAddress()` | 传入零地址 |
| `InvalidDuration(uint256 duration)` | 时长超出范围 |
| `InvalidStartPrice()` | 起拍价为 0 |
| `InvalidThreshold()` | 过期阈值为 0 |
| `InvalidFeeTiers()` | 手续费阶梯不合法 |
| `PriceFeedNotSet(address token)` | 未配置喂价 |
| `InvalidPrice(address token, int256 answer)` | 喂价 ≤ 0 |
| `StalePrice(address token, uint256 updatedAt)` | 喂价过期 |
| `AuctionNotFound(uint256 auctionId)` | 拍卖不存在 |
| `AuctionNotActive(uint256 auctionId)` | 拍卖已结束或已取消 |
| `AuctionAlreadyEnded(uint256 auctionId)` | 到期后出价 |
| `AuctionNotYetEnded(uint256 auctionId, uint64 endTime)` | 未到期就结算 |
| `AuctionHasBids(uint256 auctionId)` | 有出价时取消 |
| `NotSeller()` | 非卖家取消 |
| `SellerCannotBid()` | 卖家给自己出价 |
| `ZeroAmount()` | 出价金额为 0 |
| `TokenNotAccepted(uint256 auctionId)` | 拍卖仅接受 ETH，不接受 ERC20 出价 |
| `BidTooLow(uint256 bidUsd, uint256 minRequiredUsd)` | 出价不足 |
| `NothingToWithdraw()` | 无可提取金额 |
| `EthTransferFailed()` | ETH 转账失败 |

## 6. NFTAuctionV2（升级演示）

### 6.1 新增内容

`contract NFTAuctionV2 is NFTAuction`，**只在末尾追加状态变量**：

| # | 变量 | 类型 | 说明 |
| --- | --- | --- | --- |
| 9 | `antiSnipingWindow` | `uint256` | 触发窗口（默认 5 分钟） |
| 10 | `antiSnipingExtension` | `uint256` | 每次延长时长（默认 5 分钟） |

| 函数 | 权限 | 说明 |
| --- | --- | --- |
| `initializeV2(uint256 window, uint256 extension)` | `onlyOwner` + `reinitializer(2)` | 升级时通过 `upgradeToAndCall` 一并调用，只能调用一次 |
| `setAntiSniping(uint256 window, uint256 extension)` | `onlyOwner` | 调整参数，事件 `AntiSnipingUpdated` |
| `version()` | pure override | 返回 `2` |
| `_placeBid(...)` | internal override | 先执行 `super._placeBid`，若 `endTime - block.timestamp < antiSnipingWindow`，则 `endTime += antiSnipingExtension`，并触发事件 `AuctionExtended(auctionId, newEndTime)` |

> `initializeV2` 加 `onlyOwner`：防止升级与初始化分开执行时，被他人抢先调用并设置参数。

### 6.2 升级需要验证的内容

1. 升级前创建的拍卖、出价、`pendingReturns`、喂价配置、手续费阶梯**全部保留**
2. 代理地址不变，`version()` 从 1 变为 2
3. 升级前创建的拍卖，在最后 5 分钟内出价会被延时（新逻辑作用于旧数据）
4. 非 owner 调用 `upgradeToAndCall` 失败；`initializeV2` 不能重复调用

### 6.3 存储布局约定

- V1 的 `__gap` 用于**将来修改 V1 自身**时追加变量（届时相应缩减 gap 长度）
- V2 作为子合约，新变量自然排在 V1（含 `__gap`）之后，**不需要改动 V1**
- 禁止删除、重排、修改已有变量类型

## 7. 辅助合约

### 7.1 AuctionNFT

- `constructor(string name, string symbol, address initialOwner)`
- `mint(address to, string calldata uri) → uint256 tokenId`：`onlyOwner`，tokenId 从 1 开始自增
- 其余功能（转移、授权、`tokenURI`）使用 OpenZeppelin 标准实现

### 7.2 MockV3Aggregator

参照 Chainlink 官方 Mock 实现：

- `constructor(uint8 decimals, int256 initialAnswer)`
- `updateAnswer(int256 answer)`：更新价格，`updatedAt = block.timestamp`
- `updateRoundData(uint80 roundId, int256 answer, uint256 updatedAt, uint256 startedAt)`：可自定义更新时间，用于测试喂价过期

### 7.3 MockERC20

- `constructor(string name, string symbol, uint8 decimals)`
- `mint(address to, uint256 amount)`：任何人可铸造（测试代币，便于测试网演示时领取）

## 8. 测试

| 测试文件 | 覆盖内容 |
| --- | --- |
| `test/AuctionNFT.test.ts` | 铸造权限、tokenId 自增、tokenURI、转移 |
| `test/Mocks.test.ts` | Mock 喂价与 Mock 代币 |
| `test/NFTAuction.test.ts` | 初始化、管理员配置、价格换算与喂价校验、创建/出价/结算/取消/提现的正常与失败路径、手续费各档位边界、重入防护 |
| `test/NFTAuctionUpgrade.test.ts` | 升级权限、数据保留、V2 防狙击延时、`initializeV2` 只能调用一次 |

目标：`npx hardhat test --coverage` 的业务合约行覆盖率 **≥ 95%**。实际结果：共 132 个用例，全部合约的行覆盖率与语句覆盖率均为 **100%**。

## 9. 部署

Sepolia 使用的 Chainlink 喂价（**部署前在 [docs.chain.link](https://docs.chain.link/data-feeds/price-feeds/addresses?network=ethereum&page=1#sepolia-testnet) 核实地址**）：

| 喂价 | 用途 |
| --- | --- |
| ETH / USD | ETH 出价换算 |
| USDC / USD | 部署 `MockERC20`（名称 Mock USD Coin，符号 mUSDC，6 位精度），配合真实 USDC/USD 喂价，作为 ERC20 出价币种 |

部署顺序：`AuctionNFT` → `MockERC20` → `NFTAuction` 实现合约 → `ERC1967Proxy`（附带 `initialize` 调用数据）→ `setPriceFeed` × 2 → Etherscan 验证 → 演示升级到 V2。

具体操作步骤见 [部署指南](DEPLOYMENT.md)。
