/**
 * 将拍卖合约从 V1 升级到 V2（新增防狙击延时）
 *
 * 流程：读取部署记录 → 记录升级前状态 → 部署 V2 实现合约
 *      → 代理调用 upgradeToAndCall（同一交易内执行 initializeV2）→ 对比升级前后状态
 *
 * 用法：npm run upgrade:local / npm run upgrade:sepolia
 */
import { network } from "hardhat";
import { ANTI_SNIPING } from "./lib/config.js";
import { loadDeployment, saveDeployment } from "./lib/deployments.js";

const { ethers, networkName } = await network.create();

/** ERC-1967 实现合约地址存储槽 */
const IMPLEMENTATION_SLOT = "0x360894a13ba1a3210667c828492db98dca3e2076cc3735a920a3ca505d382bbc";

async function getImplementation(proxy: string) {
  const raw = await ethers.provider.getStorage(proxy, IMPLEMENTATION_SLOT);
  return ethers.getAddress("0x" + raw.slice(-40));
}

async function main() {
  const deployment = loadDeployment(networkName);
  const proxyAddress = deployment.contracts.NFTAuctionProxy.address;
  const [signer] = await ethers.getSigners();
  const auction = await ethers.getContractAt("NFTAuction", proxyAddress);

  console.log(`网络：${networkName}`);
  console.log(`代理地址：${proxyAddress}`);

  const owner = await auction.owner();
  if (owner.toLowerCase() !== signer.address.toLowerCase()) {
    throw new Error(`当前账户 ${signer.address} 不是合约 owner（${owner}），无权升级`);
  }
  const currentVersion = await auction.version();
  if (currentVersion !== 1n) {
    throw new Error(`当前版本为 ${currentVersion}，本脚本仅支持从 V1 升级到 V2`);
  }

  // 升级前快照，用于升级后对比
  const before = {
    implementation: await getImplementation(proxyAddress),
    nextAuctionId: await auction.nextAuctionId(),
    feeRecipient: await auction.feeRecipient(),
    priceStaleThreshold: await auction.priceStaleThreshold(),
  };
  console.log("\n升级前：");
  console.log(`  version：${currentVersion}`);
  console.log(`  实现合约：${before.implementation}`);
  console.log(`  nextAuctionId：${before.nextAuctionId}`);

  // 部署 V2 实现合约
  const v2 = await ethers.deployContract("NFTAuctionV2");
  await v2.waitForDeployment();
  const v2Address = await v2.getAddress();
  console.log(`\n✓ NFTAuctionV2 实现合约：${v2Address}`);

  // 升级并在同一交易内初始化 V2
  const initData = v2.interface.encodeFunctionData("initializeV2", [ANTI_SNIPING.window, ANTI_SNIPING.extension]);
  const tx = await auction.upgradeToAndCall(v2Address, initData);
  await tx.wait();
  console.log(`✓ 升级完成（${tx.hash}）`);

  // 升级后校验
  const auctionV2 = await ethers.getContractAt("NFTAuctionV2", proxyAddress);
  const after = {
    implementation: await getImplementation(proxyAddress),
    nextAuctionId: await auctionV2.nextAuctionId(),
    feeRecipient: await auctionV2.feeRecipient(),
    priceStaleThreshold: await auctionV2.priceStaleThreshold(),
  };
  console.log("\n升级后：");
  console.log(`  version：${await auctionV2.version()}`);
  console.log(`  实现合约：${after.implementation}`);
  console.log(`  nextAuctionId：${after.nextAuctionId}（升级前 ${before.nextAuctionId}）`);
  console.log(`  防狙击：最后 ${await auctionV2.antiSnipingWindow()} 秒内出价，延长 ${await auctionV2.antiSnipingExtension()} 秒`);

  if (after.implementation !== v2Address) throw new Error("实现合约地址未切换到 V2");
  if (
    after.nextAuctionId !== before.nextAuctionId ||
    after.feeRecipient !== before.feeRecipient ||
    after.priceStaleThreshold !== before.priceStaleThreshold
  ) {
    throw new Error("升级后状态与升级前不一致！");
  }
  console.log("\n✓ 状态校验通过：代理地址不变，原有数据完整保留");

  deployment.contracts.NFTAuctionV2 = { address: v2Address, constructorArgs: [], txHash: v2.deploymentTransaction()?.hash };
  deployment.upgrades.push({ version: 2, implementation: v2Address, txHash: tx.hash, timestamp: new Date().toISOString() });
  saveDeployment(deployment);
}

await main();
