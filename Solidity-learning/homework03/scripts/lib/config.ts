/** 链上已有的 Chainlink 喂价配置（地址来源：https://docs.chain.link/data-feeds/price-feeds/addresses） */
export interface FeedConfig {
  /** 喂价合约地址 */
  address: string;
  /** 期望的描述，部署时会与链上 description() 比对，防止地址填错 */
  description: string;
}

export interface NetworkDeployConfig {
  ethUsdFeed: FeedConfig;
  usdcUsdFeed: FeedConfig;
}

/**
 * 各网络的部署配置。未在此列出的网络（hardhat、localhost）会自动部署 MockV3Aggregator
 * 地址已于 2026-09-24 通过链上 description() / latestRoundData() 核实
 */
export const NETWORK_CONFIGS: Record<string, NetworkDeployConfig> = {
  sepolia: {
    ethUsdFeed: { address: "0x694AA1769357215DE4FAC081bf1f309aDC325306", description: "ETH / USD" },
    usdcUsdFeed: { address: "0xA2F78ab2355fe2f984D808B5CeE7FD0A93D5270E", description: "USDC / USD" },
  },
};

/** 本地网络 Mock 喂价的初始价格（8 位精度） */
export const LOCAL_ETH_PRICE = 3000n * 10n ** 8n;
export const LOCAL_USDC_PRICE = 1n * 10n ** 8n;

/** 测试代币参数：刻意使用 Mock 前缀，避免与真实 USDC 混淆 */
export const MOCK_USDC = { name: "Mock USD Coin", symbol: "mUSDC", decimals: 6 };

/** NFT 参数 */
export const NFT = { name: "Auction NFT", symbol: "ANFT" };

/** V2 防狙击默认参数：最后 5 分钟内出价，延长 5 分钟 */
export const ANTI_SNIPING = { window: 5n * 60n, extension: 5n * 60n };

/** 区块浏览器地址，演示脚本用于生成交易链接 */
export const EXPLORERS: Record<string, string> = {
  sepolia: "https://sepolia.etherscan.io",
};

/**
 * 演示脚本参数：金额刻意取小，买家只需少量测试 ETH
 * 时长需留出升级时间，同时让第二阶段能等到防狙击窗口
 */
export const DEMO = {
  /** 拍卖时长（秒） */
  duration: 15n * 60n,
  /** 起拍价（美元，18 位精度） */
  startPriceUsd: 5n * 10n ** 18n,
  /** 第一阶段 ETH 出价的目标美元价值（18 位精度），实际金额按当时喂价换算 */
  ethBidUsd: 6n * 10n ** 18n,
  /** 第二阶段 mUSDC 出价金额（6 位精度） */
  tokenBid: 10n * 10n ** 6n,
  /** 测试网上买家私钥在 keystore 中的名称 */
  bidderKeyName: "SEPOLIA_BIDDER_PRIVATE_KEY",
};
