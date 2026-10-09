import { existsSync, mkdirSync, readFileSync, writeFileSync } from "node:fs";
import path from "node:path";

/** 单个已部署合约的记录 */
export interface ContractRecord {
  address: string;
  /** 构造函数参数（用于 Etherscan 验证），bigint 以字符串保存 */
  constructorArgs: unknown[];
  /** 部署交易哈希 */
  txHash?: string;
}

/** 升级记录 */
export interface UpgradeRecord {
  version: number;
  implementation: string;
  txHash: string;
  timestamp: string;
}

/** 一个网络上的完整部署记录，保存在 deployments/<网络名>.json */
export interface DeploymentRecord {
  network: string;
  chainId: number;
  deployer: string;
  feeRecipient: string;
  deployedAt: string;
  priceFeeds: { ETH: string; USDC: string };
  contracts: {
    AuctionNFT: ContractRecord;
    MockUSDC: ContractRecord;
    /** 拍卖合约代理地址——用户交互的唯一入口，升级后不变 */
    NFTAuctionProxy: ContractRecord;
    /** 初始实现合约（V1） */
    NFTAuctionV1: ContractRecord;
    /** 升级后的实现合约（V2） */
    NFTAuctionV2?: ContractRecord;
    /** 本地网络部署的 Mock 喂价 */
    MockEthUsdFeed?: ContractRecord;
    MockUsdcUsdFeed?: ContractRecord;
  };
  upgrades: UpgradeRecord[];
}

const DEPLOYMENTS_DIR = path.resolve(import.meta.dirname, "../../deployments");

/** 部署记录文件路径 */
export function deploymentPath(networkName: string) {
  return path.join(DEPLOYMENTS_DIR, `${networkName}.json`);
}

/** 读取部署记录，不存在时报错并提示先运行 deploy.ts */
export function loadDeployment(networkName: string): DeploymentRecord {
  const file = deploymentPath(networkName);
  if (!existsSync(file)) {
    throw new Error(`未找到 ${file}，请先运行 scripts/deploy.ts 部署到 ${networkName}`);
  }
  return JSON.parse(readFileSync(file, "utf8")) as DeploymentRecord;
}

/** 保存部署记录（bigint 转为字符串） */
export function saveDeployment(record: DeploymentRecord) {
  mkdirSync(DEPLOYMENTS_DIR, { recursive: true });
  const json = JSON.stringify(record, (_, value) => (typeof value === "bigint" ? value.toString() : value), 2);
  writeFileSync(deploymentPath(record.network), json + "\n");
  console.log(`\n部署记录已保存：${path.relative(process.cwd(), deploymentPath(record.network))}`);
}

/** 是否存在部署记录 */
export function hasDeployment(networkName: string) {
  return existsSync(deploymentPath(networkName));
}
