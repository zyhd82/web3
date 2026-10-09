import { network } from "hardhat";
import { parseUnits, ZeroAddress } from "ethers";

/** 测试文件中 `network.create()` 返回的 ethers 实例类型 */
export type Ethers = Awaited<ReturnType<typeof network.create>>["ethers"];

/** 合约中代表 ETH 的代币地址 */
export const ETH = ZeroAddress;
/** 一小时（秒） */
export const HOUR = 3600n;
/** 默认拍卖时长 */
export const DEFAULT_DURATION = HOUR;

/** 美元金额（18 位精度），如 usd("100") = 100e18 */
export const usd = (value: string) => parseUnits(value, 18);
/** USDC 金额（6 位精度） */
export const usdc = (value: string) => parseUnits(value, 6);
/** 喂价（8 位精度） */
export const feedPrice = (value: string) => parseUnits(value, 8);

/** 默认喂价：ETH = $3000，USDC = $1 */
export const ETH_PRICE = feedPrice("3000");
export const USDC_PRICE = feedPrice("1");

/**
 * 部署 NFTAuction 实现合约 + ERC1967Proxy 代理，并通过代理完成初始化
 * @returns impl 实现合约；auction 按 NFTAuction ABI 挂载到代理地址的合约实例
 */
export async function deployAuctionProxy(ethers: Ethers, owner: string, feeRecipient: string) {
  const impl = await ethers.deployContract("NFTAuction");
  const initData = impl.interface.encodeFunctionData("initialize", [owner, feeRecipient]);
  const proxy = await ethers.deployContract("ERC1967Proxy", [await impl.getAddress(), initData]);
  const auction = await ethers.getContractAt("NFTAuction", await proxy.getAddress());
  return { impl, proxy, auction };
}

/**
 * 部署完整的拍卖环境：
 * - 拍卖合约（代理），已配置 ETH / USDC 喂价
 * - NFT 合约，卖家持有 tokenId 1、2、3，并已授权拍卖合约
 * - USDC（6 位精度），两位出价者各持有 100,000 USDC，并已授权拍卖合约
 */
export async function deployAuctionSystem(ethers: Ethers) {
  const [owner, feeRecipient, seller, bidder1, bidder2, other] = await ethers.getSigners();

  const { impl, proxy, auction } = await deployAuctionProxy(ethers, owner.address, feeRecipient.address);
  const auctionAddress = await auction.getAddress();

  const nft = await ethers.deployContract("AuctionNFT", ["Auction NFT", "ANFT", owner.address]);
  for (let i = 1; i <= 3; i++) {
    await nft.mint(seller.address, `ipfs://token-${i}`);
  }
  await nft.connect(seller).setApprovalForAll(auctionAddress, true);

  const usdcToken = await ethers.deployContract("MockERC20", ["USD Coin", "USDC", 6]);
  for (const bidder of [bidder1, bidder2]) {
    await usdcToken.mint(bidder.address, usdc("100000"));
    await usdcToken.connect(bidder).approve(auctionAddress, ethers.MaxUint256);
  }

  const ethFeed = await ethers.deployContract("MockV3Aggregator", [8, ETH_PRICE]);
  const usdcFeed = await ethers.deployContract("MockV3Aggregator", [8, USDC_PRICE]);
  await auction.setPriceFeed(ETH, await ethFeed.getAddress());
  await auction.setPriceFeed(await usdcToken.getAddress(), await usdcFeed.getAddress());

  return {
    impl,
    proxy,
    auction,
    nft,
    usdcToken,
    ethFeed,
    usdcFeed,
    owner,
    feeRecipient,
    seller,
    bidder1,
    bidder2,
    other,
  };
}
