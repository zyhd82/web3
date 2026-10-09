// SPDX-License-Identifier: MIT
pragma solidity 0.8.28;

import {Initializable} from "@openzeppelin/contracts-upgradeable/proxy/utils/Initializable.sol";
import {OwnableUpgradeable} from "@openzeppelin/contracts-upgradeable/access/OwnableUpgradeable.sol";
import {UUPSUpgradeable} from "@openzeppelin/contracts-upgradeable/proxy/utils/UUPSUpgradeable.sol";
import {ReentrancyGuard} from "@openzeppelin/contracts/utils/ReentrancyGuard.sol";
import {IERC721} from "@openzeppelin/contracts/token/ERC721/IERC721.sol";
import {IERC20} from "@openzeppelin/contracts/token/ERC20/IERC20.sol";
import {IERC20Metadata} from "@openzeppelin/contracts/token/ERC20/extensions/IERC20Metadata.sol";
import {SafeERC20} from "@openzeppelin/contracts/token/ERC20/utils/SafeERC20.sol";
import {Math} from "@openzeppelin/contracts/utils/math/Math.sol";
import {AggregatorV3Interface} from "./interfaces/AggregatorV3Interface.sol";

/**
 * @title NFTAuction
 * @notice NFT 拍卖市场（V1）：支持 ETH / ERC20 出价，通过 Chainlink 喂价统一换算为美元比较，
 *         按成交额收取动态手续费，采用 UUPS 代理实现可升级
 * @dev 设计说明见 docs/DESIGN.md。所有美元金额统一为 18 位精度（1e18 = $1），
 *      所有应收款采用 pull 模式记入 pendingReturns，由用户调用 withdraw 取回
 */
contract NFTAuction is Initializable, OwnableUpgradeable, UUPSUpgradeable, ReentrancyGuard {
    using SafeERC20 for IERC20;

    // ============================================================
    //                          类型定义
    // ============================================================

    /// @notice 拍卖状态
    enum AuctionStatus {
        Active, // 进行中（含已到期但未结算）
        Ended, // 已结算（成交或流拍）
        Cancelled // 卖家已取消
    }

    /// @notice 拍卖信息
    struct Auction {
        address seller; // 卖家（创建者）
        address nft; // NFT 合约地址
        uint256 tokenId; // NFT tokenId
        address paymentToken; // 允许的 ERC20；address(0) 表示仅接受 ETH
        uint256 startPriceUsd; // 起拍价（美元，18 位精度）
        uint64 startTime; // 开始时间
        uint64 endTime; // 结束时间（V2 防狙击会延长它）
        address highestBidder; // 当前最高出价者；address(0) 表示无人出价
        address highestBidToken; // 最高出价币种；address(0) 表示 ETH
        uint256 highestBidAmount; // 最高出价金额（原币种精度）
        uint256 highestBidUsd; // 最高出价的美元价值（出价时锁定，18 位精度）
        AuctionStatus status; // 状态
    }

    /// @notice 手续费阶梯的一档
    struct FeeTier {
        uint256 minUsd; // 该档起点（含），18 位精度
        uint16 feeBps; // 费率，基点（1 bps = 0.01%）
    }

    // ============================================================
    //                            常量
    // ============================================================

    /// @notice 代表 ETH 的代币地址
    address public constant ETH = address(0);
    /// @notice 最短拍卖时长
    uint256 public constant MIN_DURATION = 60;
    /// @notice 最长拍卖时长
    uint256 public constant MAX_DURATION = 30 days;
    /// @notice 基点分母
    uint256 public constant BPS_DENOMINATOR = 10_000;
    /// @notice 单档费率上限（10%）
    uint16 public constant MAX_FEE_BPS = 1_000;
    /// @notice 手续费阶梯数量上限
    uint256 public constant MAX_FEE_TIERS = 10;
    /// @notice 默认喂价过期阈值
    uint256 public constant DEFAULT_STALE_THRESHOLD = 1 days;

    // ============================================================
    //              状态变量（存储布局：只能在末尾追加）
    // ============================================================

    /// @notice 下一个拍卖 ID，从 0 开始自增
    uint256 public nextAuctionId;
    /// @dev 拍卖 ID → 拍卖信息，通过 getAuction 读取
    mapping(uint256 => Auction) private _auctions;
    /// @notice 代币地址 → Chainlink 喂价合约地址（address(0) 代表 ETH）
    mapping(address => address) public priceFeeds;
    /// @notice 喂价过期阈值（秒）
    uint256 public priceStaleThreshold;
    /// @notice 用户 → 币种 → 待提取金额
    mapping(address => mapping(address => uint256)) public pendingReturns;
    /// @notice 手续费接收地址
    address public feeRecipient;
    /// @dev 手续费阶梯（minUsd 严格递增，第一档 minUsd 为 0），通过 getFeeTiers 读取
    FeeTier[] private _feeTiers;

    /// @dev 预留存储槽位，将来修改 V1 自身时追加变量需相应缩减
    uint256[50] private __gap;

    // ============================================================
    //                            事件
    // ============================================================

    /// @notice 创建拍卖
    event AuctionCreated(
        uint256 indexed auctionId,
        address indexed seller,
        address indexed nft,
        uint256 tokenId,
        address paymentToken,
        uint256 startPriceUsd,
        uint64 endTime
    );
    /// @notice 出价成功
    event BidPlaced(
        uint256 indexed auctionId,
        address indexed bidder,
        address token,
        uint256 amount,
        uint256 amountUsd
    );
    /// @notice 拍卖结算（流拍时 winner 为 address(0)）
    event AuctionEnded(uint256 indexed auctionId, address indexed winner, address token, uint256 amount, uint256 fee);
    /// @notice 拍卖被卖家取消
    event AuctionCancelled(uint256 indexed auctionId);
    /// @notice 用户提取资金
    event Withdrawn(address indexed user, address indexed token, uint256 amount);
    /// @notice 喂价配置变更
    event PriceFeedUpdated(address indexed token, address indexed feed);
    /// @notice 喂价过期阈值变更
    event PriceStaleThresholdUpdated(uint256 threshold);
    /// @notice 手续费接收地址变更
    event FeeRecipientUpdated(address indexed recipient);
    /// @notice 手续费阶梯变更
    event FeeTiersUpdated(FeeTier[] tiers);

    // ============================================================
    //                          自定义错误
    // ============================================================

    /// @notice 传入零地址
    error ZeroAddress();
    /// @notice 拍卖时长超出范围
    error InvalidDuration(uint256 duration);
    /// @notice 起拍价为 0
    error InvalidStartPrice();
    /// @notice 喂价过期阈值为 0
    error InvalidThreshold();
    /// @notice 手续费阶梯不合法
    error InvalidFeeTiers();
    /// @notice 代币未配置喂价
    error PriceFeedNotSet(address token);
    /// @notice 喂价小于等于 0
    error InvalidPrice(address token, int256 answer);
    /// @notice 喂价已过期
    error StalePrice(address token, uint256 updatedAt);
    /// @notice 拍卖不存在
    error AuctionNotFound(uint256 auctionId);
    /// @notice 拍卖已结算或已取消
    error AuctionNotActive(uint256 auctionId);
    /// @notice 拍卖已到期，不能再出价
    error AuctionAlreadyEnded(uint256 auctionId);
    /// @notice 拍卖尚未到期，不能结算
    error AuctionNotYetEnded(uint256 auctionId, uint64 endTime);
    /// @notice 已有出价，不能取消
    error AuctionHasBids(uint256 auctionId);
    /// @notice 调用者不是卖家
    error NotSeller();
    /// @notice 卖家不能给自己的拍卖出价
    error SellerCannotBid();
    /// @notice 出价金额为 0
    error ZeroAmount();
    /// @notice 该拍卖不接受 ERC20 出价（仅接受 ETH）
    error TokenNotAccepted(uint256 auctionId);
    /// @notice 出价不足
    error BidTooLow(uint256 bidUsd, uint256 minRequiredUsd);
    /// @notice 没有可提取的金额
    error NothingToWithdraw();
    /// @notice ETH 转账失败
    error EthTransferFailed();

    // ============================================================
    //                        初始化与升级
    // ============================================================

    /// @custom:oz-upgrades-unsafe-allow constructor
    constructor() {
        // 禁止直接初始化实现合约，只能通过代理初始化
        _disableInitializers();
    }

    /**
     * @notice 初始化合约（通过代理调用，只能调用一次）
     * @param initialOwner 管理员地址
     * @param feeRecipient_ 手续费接收地址
     */
    function initialize(address initialOwner, address feeRecipient_) external initializer {
        if (feeRecipient_ == address(0)) revert ZeroAddress();
        __Ownable_init(initialOwner);

        feeRecipient = feeRecipient_;
        priceStaleThreshold = DEFAULT_STALE_THRESHOLD;
        // 默认阶梯：< $1,000 收 5%，$1,000 ~ $10,000 收 3%，≥ $10,000 收 1%
        _feeTiers.push(FeeTier({minUsd: 0, feeBps: 500}));
        _feeTiers.push(FeeTier({minUsd: 1_000e18, feeBps: 300}));
        _feeTiers.push(FeeTier({minUsd: 10_000e18, feeBps: 100}));

        emit FeeRecipientUpdated(feeRecipient_);
        emit PriceStaleThresholdUpdated(DEFAULT_STALE_THRESHOLD);
        emit FeeTiersUpdated(_feeTiers);
    }

    /**
     * @notice 合约版本号
     * @return 版本号（V1 为 1）
     */
    function version() external pure virtual returns (uint256) {
        return 1;
    }

    /// @dev UUPS 升级授权：仅管理员可升级
    function _authorizeUpgrade(address) internal override onlyOwner {}

    // ============================================================
    //                         管理员配置
    // ============================================================

    /**
     * @notice 设置代币的 Chainlink 喂价合约
     * @param token 代币地址，address(0) 代表 ETH
     * @param feed 喂价合约地址（对美元报价）
     */
    function setPriceFeed(address token, address feed) external onlyOwner {
        if (feed == address(0)) revert ZeroAddress();
        priceFeeds[token] = feed;
        emit PriceFeedUpdated(token, feed);
    }

    /**
     * @notice 设置喂价过期阈值
     * @param threshold 阈值（秒），喂价更新时间距今超过该值即视为过期
     */
    function setPriceStaleThreshold(uint256 threshold) external onlyOwner {
        if (threshold == 0) revert InvalidThreshold();
        priceStaleThreshold = threshold;
        emit PriceStaleThresholdUpdated(threshold);
    }

    /**
     * @notice 设置手续费接收地址（只影响之后结算的拍卖）
     * @param recipient 新的接收地址
     */
    function setFeeRecipient(address recipient) external onlyOwner {
        if (recipient == address(0)) revert ZeroAddress();
        feeRecipient = recipient;
        emit FeeRecipientUpdated(recipient);
    }

    /**
     * @notice 整体替换手续费阶梯
     * @param tiers 新阶梯：1 ~ MAX_FEE_TIERS 档，第一档 minUsd 为 0，minUsd 严格递增，每档费率不超过 MAX_FEE_BPS
     */
    function setFeeTiers(FeeTier[] calldata tiers) external onlyOwner {
        uint256 length = tiers.length;
        if (length == 0 || length > MAX_FEE_TIERS || tiers[0].minUsd != 0) revert InvalidFeeTiers();

        delete _feeTiers;
        for (uint256 i = 0; i < length; i++) {
            if (tiers[i].feeBps > MAX_FEE_BPS) revert InvalidFeeTiers();
            if (i > 0 && tiers[i].minUsd <= tiers[i - 1].minUsd) revert InvalidFeeTiers();
            _feeTiers.push(tiers[i]);
        }
        emit FeeTiersUpdated(tiers);
    }

    // ============================================================
    //                          拍卖操作
    // ============================================================

    /**
     * @notice 创建拍卖，NFT 会被托管到本合约（调用前需 approve 或 setApprovalForAll）
     * @param nft NFT 合约地址
     * @param tokenId NFT tokenId（调用者必须是持有者）
     * @param paymentToken 除 ETH 外允许的 ERC20，address(0) 表示仅接受 ETH
     * @param startPriceUsd 起拍价（美元，18 位精度，如 $100 传 100e18）
     * @param duration 拍卖时长（秒），范围 [MIN_DURATION, MAX_DURATION]
     * @return auctionId 新拍卖的 ID
     */
    function createAuction(
        address nft,
        uint256 tokenId,
        address paymentToken,
        uint256 startPriceUsd,
        uint256 duration
    ) external nonReentrant returns (uint256 auctionId) {
        // 检查
        if (nft == address(0)) revert ZeroAddress();
        if (duration < MIN_DURATION || duration > MAX_DURATION) revert InvalidDuration(duration);
        if (startPriceUsd == 0) revert InvalidStartPrice();
        if (priceFeeds[ETH] == address(0)) revert PriceFeedNotSet(ETH);
        if (paymentToken != ETH && priceFeeds[paymentToken] == address(0)) revert PriceFeedNotSet(paymentToken);

        // 生效
        auctionId = nextAuctionId++;
        uint64 endTime = uint64(block.timestamp + duration);
        Auction storage auction = _auctions[auctionId];
        auction.seller = msg.sender;
        auction.nft = nft;
        auction.tokenId = tokenId;
        auction.paymentToken = paymentToken;
        auction.startPriceUsd = startPriceUsd;
        auction.startTime = uint64(block.timestamp);
        auction.endTime = endTime;

        // 交互：托管 NFT（ERC721 会校验调用者是持有者且已授权）
        IERC721(nft).transferFrom(msg.sender, address(this), tokenId);

        emit AuctionCreated(auctionId, msg.sender, nft, tokenId, paymentToken, startPriceUsd, endTime);
    }

    /**
     * @notice 使用 ETH 出价，出价金额即 msg.value
     * @param auctionId 拍卖 ID
     */
    function bidWithETH(uint256 auctionId) external payable nonReentrant {
        _placeBid(auctionId, ETH, msg.value);
    }

    /**
     * @notice 使用拍卖指定的 ERC20 出价（调用前需 approve 本合约）
     * @param auctionId 拍卖 ID
     * @param amount 出价金额（按代币精度）
     */
    function bidWithToken(uint256 auctionId, uint256 amount) external nonReentrant {
        address token = _getAuction(auctionId).paymentToken;
        if (token == ETH) revert TokenNotAccepted(auctionId);

        _placeBid(auctionId, token, amount);

        // 交互：转入出价代币
        IERC20(token).safeTransferFrom(msg.sender, address(this), amount);
    }

    /**
     * @notice 结算已到期的拍卖，任何人都可以调用
     * @dev 有出价：NFT 转给最高出价者，货款（扣除手续费）与手续费分别记入卖家和 feeRecipient 的 pendingReturns；
     *      无出价（流拍）：NFT 退回卖家
     * @param auctionId 拍卖 ID
     */
    function endAuction(uint256 auctionId) external nonReentrant {
        // 检查
        Auction storage auction = _getAuction(auctionId);
        if (auction.status != AuctionStatus.Active) revert AuctionNotActive(auctionId);
        if (block.timestamp < auction.endTime) revert AuctionNotYetEnded(auctionId, auction.endTime);

        // 生效
        auction.status = AuctionStatus.Ended;
        address winner = auction.highestBidder;
        address token = auction.highestBidToken;
        uint256 amount = auction.highestBidAmount;
        uint256 fee;
        if (winner != address(0)) {
            fee = (amount * getFeeBps(auction.highestBidUsd)) / BPS_DENOMINATOR;
            pendingReturns[auction.seller][token] += amount - fee;
            pendingReturns[feeRecipient][token] += fee;
        }

        // 交互：用 transferFrom 而非 safeTransferFrom，避免买家合约拒收导致结算永远失败
        address nftReceiver = winner == address(0) ? auction.seller : winner;
        IERC721(auction.nft).transferFrom(address(this), nftReceiver, auction.tokenId);

        emit AuctionEnded(auctionId, winner, token, amount, fee);
    }

    /**
     * @notice 卖家在无人出价时取消拍卖，NFT 退回卖家
     * @param auctionId 拍卖 ID
     */
    function cancelAuction(uint256 auctionId) external nonReentrant {
        // 检查
        Auction storage auction = _getAuction(auctionId);
        if (msg.sender != auction.seller) revert NotSeller();
        if (auction.status != AuctionStatus.Active) revert AuctionNotActive(auctionId);
        if (auction.highestBidder != address(0)) revert AuctionHasBids(auctionId);

        // 生效
        auction.status = AuctionStatus.Cancelled;

        // 交互
        IERC721(auction.nft).transferFrom(address(this), auction.seller, auction.tokenId);

        emit AuctionCancelled(auctionId);
    }

    /**
     * @notice 提取待领取的资金（被超越的出价、成交货款、手续费）
     * @param token 币种，address(0) 代表 ETH
     */
    function withdraw(address token) external nonReentrant {
        // 检查
        uint256 amount = pendingReturns[msg.sender][token];
        if (amount == 0) revert NothingToWithdraw();

        // 生效：先清零再转账
        pendingReturns[msg.sender][token] = 0;

        // 交互
        if (token == ETH) {
            (bool success, ) = payable(msg.sender).call{value: amount}("");
            if (!success) revert EthTransferFailed();
        } else {
            IERC20(token).safeTransfer(msg.sender, amount);
        }

        emit Withdrawn(msg.sender, token, amount);
    }

    // ============================================================
    //                            查询
    // ============================================================

    /**
     * @notice 读取拍卖完整信息
     * @param auctionId 拍卖 ID
     * @return 拍卖信息
     */
    function getAuction(uint256 auctionId) external view returns (Auction memory) {
        return _getAuction(auctionId);
    }

    /**
     * @notice 读取并校验代币的最新美元价格
     * @param token 代币地址，address(0) 代表 ETH
     * @return price 价格
     * @return feedDecimals 价格精度
     */
    function getLatestPrice(address token) public view returns (uint256 price, uint8 feedDecimals) {
        address feed = priceFeeds[token];
        if (feed == address(0)) revert PriceFeedNotSet(token);

        (, int256 answer, , uint256 updatedAt, ) = AggregatorV3Interface(feed).latestRoundData();
        if (answer <= 0) revert InvalidPrice(token, answer);
        if (updatedAt == 0 || updatedAt + priceStaleThreshold < block.timestamp) {
            revert StalePrice(token, updatedAt);
        }

        price = uint256(answer);
        feedDecimals = AggregatorV3Interface(feed).decimals();
    }

    /**
     * @notice 把代币金额换算成美元
     * @dev usd = amount × price × 1e18 / 10^(tokenDecimals + feedDecimals)
     * @param token 代币地址，address(0) 代表 ETH
     * @param amount 代币金额（按代币精度）
     * @return 美元价值（18 位精度）
     */
    function toUsd(address token, uint256 amount) public view returns (uint256) {
        (uint256 price, uint8 feedDecimals) = getLatestPrice(token);
        uint256 tokenDecimals = token == ETH ? 18 : IERC20Metadata(token).decimals();
        return Math.mulDiv(amount, price * 1e18, 10 ** (tokenDecimals + feedDecimals));
    }

    /**
     * @notice 查询某个成交额对应的手续费率
     * @param amountUsd 成交额（美元，18 位精度）
     * @return 费率（基点）
     */
    function getFeeBps(uint256 amountUsd) public view returns (uint16) {
        // 从最高档往下找第一个 minUsd <= amountUsd 的档位；第一档 minUsd 为 0，保证一定能找到
        uint256 i = _feeTiers.length - 1;
        while (_feeTiers[i].minUsd > amountUsd) {
            i--;
        }
        return _feeTiers[i].feeBps;
    }

    /**
     * @notice 读取全部手续费阶梯
     * @return 手续费阶梯
     */
    function getFeeTiers() external view returns (FeeTier[] memory) {
        return _feeTiers;
    }

    // ============================================================
    //                          内部函数
    // ============================================================

    /**
     * @dev 出价核心逻辑（V2 重写以实现防狙击延时）。只记账，不负责转入 ERC20
     * @param auctionId 拍卖 ID
     * @param token 出价币种
     * @param amount 出价金额
     */
    function _placeBid(uint256 auctionId, address token, uint256 amount) internal virtual {
        // 检查
        Auction storage auction = _getAuction(auctionId);
        if (auction.status != AuctionStatus.Active) revert AuctionNotActive(auctionId);
        if (block.timestamp >= auction.endTime) revert AuctionAlreadyEnded(auctionId);
        if (msg.sender == auction.seller) revert SellerCannotBid();
        if (amount == 0) revert ZeroAmount();

        uint256 bidUsd = toUsd(token, amount);
        address previousBidder = auction.highestBidder;
        if (previousBidder == address(0)) {
            if (bidUsd < auction.startPriceUsd) revert BidTooLow(bidUsd, auction.startPriceUsd);
        } else {
            if (bidUsd <= auction.highestBidUsd) revert BidTooLow(bidUsd, auction.highestBidUsd + 1);
            // 生效：被超越的出价退回待提取余额（同一人再次加价时，旧出价同样退回）
            pendingReturns[previousBidder][auction.highestBidToken] += auction.highestBidAmount;
        }

        // 生效：更新最高出价
        auction.highestBidder = msg.sender;
        auction.highestBidToken = token;
        auction.highestBidAmount = amount;
        auction.highestBidUsd = bidUsd;

        emit BidPlaced(auctionId, msg.sender, token, amount, bidUsd);
    }

    /**
     * @dev 读取拍卖，不存在时回滚
     * @param auctionId 拍卖 ID
     * @return auction 拍卖存储引用
     */
    function _getAuction(uint256 auctionId) internal view returns (Auction storage auction) {
        auction = _auctions[auctionId];
        if (auction.seller == address(0)) revert AuctionNotFound(auctionId);
    }
}
