import hardhatToolboxMochaEthersPlugin from "@nomicfoundation/hardhat-toolbox-mocha-ethers";
import { configVariable, defineConfig } from "hardhat/config";

export default defineConfig({
  plugins: [hardhatToolboxMochaEthersPlugin],
  solidity: {
    profiles: {
      // 本地开发与测试：不开优化器，便于调试和覆盖率统计
      default: {
        version: "0.8.28",
      },
      // 测试网部署：开启优化器，降低 gas 与合约体积
      production: {
        version: "0.8.28",
        settings: {
          optimizer: {
            enabled: true,
            runs: 200,
          },
        },
      },
    },
    // 代理合约来自 npm 包，需要显式声明才会被编译，供测试与部署脚本使用
    npmFilesToBuild: ["@openzeppelin/contracts/proxy/ERC1967/ERC1967Proxy.sol"],
  },
  networks: {
    // 本地模拟网络
    hardhatMainnet: {
      type: "edr-simulated",
      chainType: "l1",
    },
    // 本地节点（先运行 npx hardhat node），用于完整演练部署与升级脚本
    localhost: {
      type: "http",
      chainType: "l1",
      url: "http://127.0.0.1:8545",
    },
    // Sepolia 测试网：敏感信息通过 keystore 读取（npx hardhat keystore set <KEY>）
    sepolia: {
      type: "http",
      chainType: "l1",
      url: configVariable("SEPOLIA_RPC_URL"),
      accounts: [configVariable("SEPOLIA_PRIVATE_KEY")],
    },
  },
  verify: {
    etherscan: {
      apiKey: configVariable("ETHERSCAN_API_KEY"),
    },
  },
});
