/**
 * 部署 NFT 拍卖市场（V1）
 *
 * 部署顺序：AuctionNFT → MockUSDC → 喂价（本地部署 Mock / 测试网使用 Chainlink）
 *          → NFTAuction 实现合约 → ERC1967Proxy（附带 initialize 调用）→ 配置喂价
 *
 * 用法：
 *   本地：npx hardhat node（另开终端）→ npm run deploy:local
 *   测试网：npm run deploy:sepolia
 *   已有部署记录时默认拒绝覆盖，确需重新部署请设置环境变量 REDEPLOY=true
 */
import { network } from "hardhat";
import { ZeroAddress, type BaseContract, type ContractTransactionResponse } from "ethers";
import { LOCAL_ETH_PRICE, LOCAL_USDC_PRICE, MOCK_USDC, NETWORK_CONFIGS, NFT, type FeedConfig } from "./lib/config.js";
import { hasDeployment, saveDeployment, type ContractRecord, type DeploymentRecord } from "./lib/deployments.js";

const { ethers, networkName } = await network.create();

/** 等待合约部署完成并生成部署记录 */
async function record(contract: BaseContract, constructorArgs: unknown[]): Promise<ContractRecord> {
  await contract.waitForDeployment();
  const address = await contract.getAddress();
  return { address, constructorArgs, txHash: contract.deploymentTransaction()?.hash };
}

/** 发送交易并等待上链 */
async function send(description: string, txPromise: Promise<ContractTransactionResponse>) {
  const tx = await txPromise;
  await tx.wait();
  console.log(`  ✓ ${description}（${tx.hash}）`);
}

/** 校验链上喂价合约的描述与配置一致，防止地址填错 */
async function checkFeed(feed: FeedConfig) {
  const aggregator = await ethers.getContractAt("AggregatorV3Interface", feed.address);
  const description = await aggregator.description();
  if (description !== feed.description) {
    throw new Error(`喂价 ${feed.address} 的描述为 "${description}"，期望 "${feed.description}"`);
  }
  const [, answer] = await aggregator.latestRoundData();
  const decimals = await aggregator.decimals();
  console.log(`  ✓ ${description}：${ethers.formatUnits(answer, decimals)}（${feed.address}）`);
}

async function main() {
  if (hasDeployment(networkName) && process.env.REDEPLOY !== "true") {
    throw new Error(`${networkName} 已有部署记录，如需重新部署请设置 REDEPLOY=true`);
  }

  const [deployer] = await ethers.getSigners();
  const chainId = Number((await ethers.provider.getNetwork()).chainId);
  const feeRecipient = deployer.address;
  console.log(`网络：${networkName}（chainId ${chainId}）`);
  console.log(`部署账户：${deployer.address}`);
  console.log(`账户余额：${ethers.formatEther(await ethers.provider.getBalance(deployer.address))} ETH\n`);

  // 1. NFT 与测试代币
  console.log("1. 部署 NFT 与测试代币");
  const nftArgs = [NFT.name, NFT.symbol, deployer.address];
  const nft = await record(await ethers.deployContract("AuctionNFT", nftArgs), nftArgs);
  console.log(`  ✓ AuctionNFT：${nft.address}`);

  const usdcArgs = [MOCK_USDC.name, MOCK_USDC.symbol, MOCK_USDC.decimals];
  const usdc = await record(await ethers.deployContract("MockERC20", usdcArgs), usdcArgs);
  console.log(`  ✓ MockUSDC：${usdc.address}`);

  // 2. 喂价：测试网使用 Chainlink 官方喂价，本地部署 Mock
  console.log("\n2. 准备 Chainlink 喂价");
  const networkConfig = NETWORK_CONFIGS[networkName];
  let ethFeed: string;
  let usdcFeed: string;
  let mockEthFeed: ContractRecord | undefined;
  let mockUsdcFeed: ContractRecord | undefined;
  if (networkConfig) {
    await checkFeed(networkConfig.ethUsdFeed);
    await checkFeed(networkConfig.usdcUsdFeed);
    ethFeed = networkConfig.ethUsdFeed.address;
    usdcFeed = networkConfig.usdcUsdFeed.address;
  } else {
    const ethArgs = [8, LOCAL_ETH_PRICE];
    mockEthFeed = await record(await ethers.deployContract("MockV3Aggregator", ethArgs), ethArgs);
    const usdcFeedArgs = [8, LOCAL_USDC_PRICE];
    mockUsdcFeed = await record(await ethers.deployContract("MockV3Aggregator", usdcFeedArgs), usdcFeedArgs);
    ethFeed = mockEthFeed.address;
    usdcFeed = mockUsdcFeed.address;
    console.log(`  ✓ Mock ETH/USD：${ethFeed}（$3000）`);
    console.log(`  ✓ Mock USDC/USD：${usdcFeed}（$1）`);
  }

  // 3. 拍卖合约：实现合约 + 代理（部署代理时在同一交易内调用 initialize，防止被抢先初始化）
  console.log("\n3. 部署拍卖合约（UUPS 代理）");
  const impl = await record(await ethers.deployContract("NFTAuction"), []);
  console.log(`  ✓ NFTAuction V1 实现合约：${impl.address}`);

  const auctionInterface = (await ethers.getContractFactory("NFTAuction")).interface;
  const initData = auctionInterface.encodeFunctionData("initialize", [deployer.address, feeRecipient]);
  const proxyArgs = [impl.address, initData];
  const proxy = await record(await ethers.deployContract("ERC1967Proxy", proxyArgs), proxyArgs);
  console.log(`  ✓ ERC1967Proxy（拍卖合约入口）：${proxy.address}`);

  // 4. 配置喂价
  console.log("\n4. 配置喂价");
  const auction = await ethers.getContractAt("NFTAuction", proxy.address);
  await send("ETH → ETH/USD", auction.setPriceFeed(ZeroAddress, ethFeed));
  await send("MockUSDC → USDC/USD", auction.setPriceFeed(usdc.address, usdcFeed));

  // 5. 自检
  console.log("\n5. 部署自检");
  console.log(`  version：${await auction.version()}`);
  console.log(`  owner：${await auction.owner()}`);
  console.log(`  feeRecipient：${await auction.feeRecipient()}`);
  console.log(`  1 ETH = $${ethers.formatUnits(await auction.toUsd(ZeroAddress, ethers.parseEther("1")), 18)}`);
  console.log(`  1 mUSDC = $${ethers.formatUnits(await auction.toUsd(usdc.address, 10n ** 6n), 18)}`);

  const deployment: DeploymentRecord = {
    network: networkName,
    chainId,
    deployer: deployer.address,
    feeRecipient,
    deployedAt: new Date().toISOString(),
    priceFeeds: { ETH: ethFeed, USDC: usdcFeed },
    contracts: {
      AuctionNFT: nft,
      MockUSDC: usdc,
      NFTAuctionProxy: proxy,
      NFTAuctionV1: impl,
      ...(mockEthFeed && { MockEthUsdFeed: mockEthFeed }),
      ...(mockUsdcFeed && { MockUsdcUsdFeed: mockUsdcFeed }),
    },
    upgrades: [],
  };
  saveDeployment(deployment);
}

await main();
