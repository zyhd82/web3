/**
 * 测试网演示一场完整拍卖，并演示"升级后数据保留"与 V2 防狙击延时
 *
 * 同一个命令运行两次，脚本根据演示记录与链上状态自动判断阶段：
 *   第一阶段（V1）：卖家 A 铸造 NFT 并创建拍卖，买家 B 用 ETH 出价
 *   → 执行 upgrade 升级到 V2 →
 *   第二阶段（V2）：核对升级前的拍卖数据 → 等到最后 5 分钟，B 用 mUSDC 加价触发延时
 *                  → 到期后结算 → A、B 分别提现
 *
 * 账户：A 为部署账户（NFT owner、卖家、手续费接收地址）；
 *      B 在本地网络为第二个账户，在测试网从 keystore 的 SEPOLIA_BIDDER_PRIVATE_KEY 读取
 * 记录：每笔交易写入 deployments/<网络名>-demo.json
 *
 * 用法：npm run demo:local / npm run demo:sepolia
 */
import hre, { network } from "hardhat";
import { configVariable } from "hardhat/config";
import { existsSync, readFileSync, writeFileSync } from "node:fs";
import path from "node:path";
import { Wallet, ZeroAddress, type ContractTransactionResponse, type Signer } from "ethers";
import { ANTI_SNIPING, DEMO, EXPLORERS } from "./lib/config.js";
import { deploymentPath, loadDeployment } from "./lib/deployments.js";

const { ethers, networkName } = await network.create();
const isLocal = networkName === "localhost" || networkName === "hardhat";
const explorer = EXPLORERS[networkName];

/** 演示过程中的一笔交易 */
interface DemoStep {
  step: string;
  from: string;
  txHash: string;
  url?: string;
}

/** 演示记录，保存在 deployments/<网络名>-demo.json */
interface DemoRecord {
  network: string;
  auctionId: string;
  tokenId: string;
  seller: string;
  bidder: string;
  startedAt: string;
  finished: boolean;
  steps: DemoStep[];
}

const recordFile = deploymentPath(networkName).replace(/\.json$/, "-demo.json");

function loadRecord(): DemoRecord | undefined {
  return existsSync(recordFile) ? (JSON.parse(readFileSync(recordFile, "utf8")) as DemoRecord) : undefined;
}

function saveRecord(record: DemoRecord) {
  writeFileSync(recordFile, JSON.stringify(record, null, 2) + "\n");
}

/** 发送交易、等待上链，并追加到演示记录（每笔都立即保存，中途失败也能保留已完成的痕迹） */
async function send(record: DemoRecord, step: string, from: Signer, txPromise: Promise<ContractTransactionResponse>) {
  const tx = await txPromise;
  await tx.wait();
  const url = explorer && `${explorer}/tx/${tx.hash}`;
  record.steps.push({ step, from: await from.getAddress(), txHash: tx.hash, ...(url && { url }) });
  saveRecord(record);
  console.log(`  ✓ ${step}\n    ${url ?? tx.hash}`);
  return tx;
}

/** 获取买家账户：本地使用第二个内置账户，测试网从 keystore 读取私钥（与部署账户分开读取，不影响其他脚本） */
async function getBidder(): Promise<Signer> {
  if (isLocal) return (await ethers.getSigners())[1];
  const key = await hre.hooks.runHandlerChain(
    "configurationVariables",
    "fetchValue",
    [configVariable(DEMO.bidderKeyName)],
    async (_context, variable) => {
      const value = process.env[variable.name];
      if (value === undefined) {
        throw new Error(`未找到买家私钥，请先执行 npx hardhat keystore set ${variable.name}`);
      }
      return value;
    },
  );
  return new Wallet(key, ethers.provider);
}

/** 最新区块时间 */
async function now() {
  return BigInt((await ethers.provider.getBlock("latest"))!.timestamp);
}

/** 等到链上时间不早于 target：本地直接推进时间，测试网轮询等待 */
async function waitUntil(target: bigint, reason: string) {
  let current = await now();
  if (current >= target) return;
  console.log(`\n⏳ ${reason}：还需约 ${target - current} 秒`);
  if (isLocal) {
    await ethers.provider.send("evm_increaseTime", [Number(target - current)]);
    await ethers.provider.send("evm_mine", []);
    return;
  }
  while (current < target) {
    await new Promise((resolve) => setTimeout(resolve, 15_000));
    current = await now();
    if (current < target) console.log(`   还需约 ${target - current} 秒`);
  }
}

const deployment = loadDeployment(networkName);
const proxy = deployment.contracts.NFTAuctionProxy.address;
const [seller] = await ethers.getSigners();
const nft = await ethers.getContractAt("AuctionNFT", deployment.contracts.AuctionNFT.address);
const usdc = await ethers.getContractAt("MockERC20", deployment.contracts.MockUSDC.address);
// 两个阶段都只用到 V1 已有的函数时按 V1 ABI 调用；V2 专有的函数在第二阶段单独挂载
const auction = await ethers.getContractAt("NFTAuction", proxy);
/** 美元金额（18 位精度）保留两位小数显示 */
const fmtUsd = (value: bigint) => `$${Number(ethers.formatUnits(value, 18)).toFixed(2)}`;

/** 打印拍卖当前状态 */
async function showAuction(auctionId: bigint) {
  const a = await auction.getAuction(auctionId);
  const token = a.highestBidToken === ZeroAddress ? "ETH" : "mUSDC";
  const amount = a.highestBidToken === ZeroAddress ? ethers.formatEther(a.highestBidAmount) : ethers.formatUnits(a.highestBidAmount, 6);
  console.log(`  拍卖 #${auctionId}：卖家 ${a.seller}，tokenId ${a.tokenId}，起拍价 ${fmtUsd(a.startPriceUsd)}`);
  console.log(`    最高出价 ${amount} ${token}（${fmtUsd(a.highestBidUsd)}），出价者 ${a.highestBidder}`);
  console.log(`    结束时间 ${new Date(Number(a.endTime) * 1000).toISOString()}，状态 ${["进行中", "已结束", "已取消"][Number(a.status)]}`);
  return a;
}

/** 第一阶段：在 V1 上创建拍卖并用 ETH 出价 */
async function startDemo(bidder: Signer) {
  const bidderAddress = await bidder.getAddress();
  const version = await auction.version();
  if (version !== 1n) {
    console.log(`⚠ 当前合约版本为 ${version}，第一阶段应在 V1 上执行才能演示"升级后数据保留"，演示仍会继续。`);
  }

  // ETH 出价金额按当前喂价换算，向上取整到 0.0001 ETH，确保换算后不低于目标美元价值且金额易读
  const usdPerEth = await auction.toUsd(ZeroAddress, ethers.parseEther("1"));
  const step = 10n ** 14n;
  const ethBid = ((DEMO.ethBidUsd * 10n ** 18n) / usdPerEth / step + 1n) * step;
  const bidderBalance = await ethers.provider.getBalance(bidderAddress);
  if (bidderBalance < ethBid) {
    throw new Error(`买家 ${bidderAddress} 余额 ${ethers.formatEther(bidderBalance)} ETH，不足以出价 ${ethers.formatEther(ethBid)} ETH（另需少量 gas）`);
  }

  const tokenId = await nft.nextTokenId();
  const auctionId = await auction.nextAuctionId();
  const record: DemoRecord = {
    network: networkName,
    auctionId: auctionId.toString(),
    tokenId: tokenId.toString(),
    seller: seller.address,
    bidder: bidderAddress,
    startedAt: new Date().toISOString(),
    finished: false,
    steps: [],
  };

  console.log(`\n第一阶段（合约版本 ${version}）：创建拍卖并出价`);
  await send(record, `A 铸造 NFT #${tokenId}`, seller, nft.mint(seller.address, `ipfs://auction-demo/${tokenId}`));
  await send(record, `A 授权拍卖合约转移 NFT #${tokenId}`, seller, nft.approve(proxy, tokenId));
  await send(
    record,
    `A 创建拍卖 #${auctionId}（起拍价 ${fmtUsd(DEMO.startPriceUsd)}，时长 ${DEMO.duration} 秒，接受 ETH 与 mUSDC）`,
    seller,
    auction.createAuction(nft.target, tokenId, usdc.target, DEMO.startPriceUsd, DEMO.duration),
  );
  await send(
    record,
    `B 用 ${ethers.formatEther(ethBid)} ETH 出价（约 ${fmtUsd(await auction.toUsd(ZeroAddress, ethBid))}）`,
    bidder,
    auction.connect(bidder).bidWithETH(auctionId, { value: ethBid }),
  );

  console.log("\n当前状态：");
  const a = await showAuction(auctionId);
  const remaining = a.endTime - (await now());
  console.log(`\n下一步：请在 ${remaining} 秒内执行升级（npm run upgrade:${isLocal ? "local" : networkName}），然后再次运行本脚本进入第二阶段。`);
}

/** 第二阶段：在 V2 上核对数据、触发防狙击延时、结算并提现 */
async function finishDemo(record: DemoRecord, bidder: Signer) {
  const auctionId = BigInt(record.auctionId);
  const version = await auction.version();
  if (version !== 2n) {
    throw new Error(`当前合约版本为 ${version}，请先执行 npm run upgrade:${isLocal ? "local" : networkName} 再运行第二阶段`);
  }
  if ((await bidder.getAddress()) !== record.bidder) {
    throw new Error(`买家账户与第一阶段不一致：记录为 ${record.bidder}`);
  }
  const auctionV2 = await ethers.getContractAt("NFTAuctionV2", proxy);

  console.log(`\n第二阶段（合约版本 ${version}）：升级后数据核对`);
  const before = await showAuction(auctionId);
  console.log(`  B 的 ETH 待提现：${ethers.formatEther(await auction.pendingReturns(record.bidder, ZeroAddress))}`);

  if (before.status === 0n && (await now()) < before.endTime) {
    // 等到防狙击窗口内再出价（留 60 秒余量，避免交易上链时已过期）
    const window = await auctionV2.antiSnipingWindow();
    await waitUntil(before.endTime - window + 60n, `等待进入最后 ${window} 秒的防狙击窗口`);

    console.log("\nB 在最后 5 分钟内用 mUSDC 加价：");
    await send(record, `B 领取 ${ethers.formatUnits(DEMO.tokenBid, 6)} mUSDC 测试代币`, bidder, usdc.connect(bidder).mint(record.bidder, DEMO.tokenBid));
    await send(record, "B 授权拍卖合约转移 mUSDC", bidder, usdc.connect(bidder).approve(proxy, DEMO.tokenBid));
    const tx = await send(
      record,
      `B 用 ${ethers.formatUnits(DEMO.tokenBid, 6)} mUSDC 加价（约 ${fmtUsd(await auction.toUsd(usdc.target, DEMO.tokenBid))}）`,
      bidder,
      auction.connect(bidder).bidWithToken(auctionId, DEMO.tokenBid),
    );
    const receipt = await tx.wait();
    const extended = receipt!.logs
      .map((log) => auctionV2.interface.parseLog(log))
      .find((event) => event?.name === "AuctionExtended");

    const after = await showAuction(auctionId);
    console.log(`  结束时间延长 ${after.endTime - before.endTime} 秒（预期 ${ANTI_SNIPING.extension} 秒），AuctionExtended 事件：${extended ? "已触发" : "未触发"}`);
    console.log(`  B 被超越的 ETH 出价已记入待提现：${ethers.formatEther(await auction.pendingReturns(record.bidder, ZeroAddress))} ETH`);
    await waitUntil(after.endTime, "等待拍卖到期");
  } else if (before.status === 0n) {
    console.log("\n⚠ 拍卖已到期，跳过防狙击演示，直接结算。");
  }

  const current = await auction.getAuction(auctionId);
  if (current.status === 0n) {
    console.log("\n结算与提现：");
    await send(record, `结算拍卖 #${auctionId}`, seller, auction.endAuction(auctionId));
  }
  const ended = await showAuction(auctionId);
  console.log(`  NFT #${record.tokenId} 当前持有者：${await nft.ownerOf(record.tokenId)}`);

  // 提现所有待提现余额：A 收到货款与手续费（A 同时是卖家和手续费接收地址），B 取回被超越的出价
  const payouts: [string, Signer, string, string][] = [
    ["A", seller, ended.highestBidToken, ended.highestBidToken === ZeroAddress ? "ETH" : "mUSDC"],
    ["B", bidder, ZeroAddress, "ETH"],
    ["B", bidder, usdc.target as string, "mUSDC"],
  ];
  for (const [label, account, token, symbol] of payouts) {
    const pending = await auction.pendingReturns(await account.getAddress(), token);
    if (pending === 0n) continue;
    const amount = symbol === "ETH" ? ethers.formatEther(pending) : ethers.formatUnits(pending, 6);
    await send(record, `${label} 提现 ${amount} ${symbol}`, account, auction.connect(account).withdraw(token));
  }

  record.finished = true;
  saveRecord(record);
  console.log(`\n✓ 演示完成，全部交易记录：${path.relative(process.cwd(), recordFile)}`);
}

async function main() {
  const bidder = await getBidder();
  const bidderAddress = await bidder.getAddress();
  console.log(`网络：${networkName}`);
  console.log(`拍卖合约（代理）：${proxy}`);
  console.log(`卖家 A：${seller.address}`);
  console.log(`买家 B：${bidderAddress}`);
  if (bidderAddress === seller.address) {
    throw new Error("买家与卖家是同一账户，合约不允许卖家给自己出价，请为买家使用另一个账户");
  }

  // 已完成的演示记录是测试网上的演示凭证，不自动覆盖
  const record = loadRecord();
  if (record?.finished) {
    throw new Error(`演示已完成，记录见 ${path.relative(process.cwd(), recordFile)}；如需重新演示，请先将该文件改名备份`);
  }
  if (record) {
    await finishDemo(record, bidder);
  } else {
    await startDemo(bidder);
  }
}

await main();
