/**
 * 在 Etherscan 上验证已部署合约的源码
 *
 * 依次验证：AuctionNFT、MockUSDC、NFTAuction V1 实现、ERC1967Proxy、NFTAuction V2 实现（若已升级）
 * 注意：必须使用与部署时相同的编译配置（npm 脚本统一使用 --build-profile production）
 *
 * 用法：npm run verify:sepolia
 */
import hre, { network } from "hardhat";
import { verifyContract } from "@nomicfoundation/hardhat-verify/verify";
import { loadDeployment, type ContractRecord } from "./lib/deployments.js";

const { networkName } = await network.create();

async function verify(name: string, record: ContractRecord | undefined) {
  if (!record) return;
  console.log(`\n验证 ${name}（${record.address}）...`);
  try {
    await verifyContract({ address: record.address, constructorArgs: record.constructorArgs }, hre);
    console.log(`  ✓ ${name} 验证成功`);
  } catch (error) {
    // 单个合约失败不影响其余合约，最后统一查看输出
    console.error(`  ✗ ${name} 验证失败：${(error as Error).message}`);
  }
}

async function main() {
  if (networkName === "hardhat" || networkName === "localhost") {
    throw new Error("本地网络无需验证，请使用 --network sepolia");
  }
  const { contracts } = loadDeployment(networkName);
  await verify("AuctionNFT", contracts.AuctionNFT);
  await verify("MockUSDC", contracts.MockUSDC);
  await verify("NFTAuction V1 实现合约", contracts.NFTAuctionV1);
  await verify("ERC1967Proxy", contracts.NFTAuctionProxy);
  await verify("NFTAuction V2 实现合约", contracts.NFTAuctionV2);
}

await main();
