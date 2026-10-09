import { expect } from "chai";
import { network } from "hardhat";

const { ethers, networkHelpers } = await network.create();

// Mock 合约的测试：保证后续拍卖合约测试所依赖的工具本身行为正确
describe("Mock 合约", function () {
  async function deployFixture() {
    const [owner, alice] = await ethers.getSigners();
    const feed = await ethers.deployContract("MockV3Aggregator", [8, 3000n * 10n ** 8n]);
    const usdc = await ethers.deployContract("MockERC20", ["USD Coin", "USDC", 6]);
    return { feed, usdc, owner, alice };
  }

  describe("MockV3Aggregator", function () {
    it("部署后返回初始价格、精度与描述", async function () {
      const { feed } = await networkHelpers.loadFixture(deployFixture);
      const [roundId, answer, startedAt, updatedAt, answeredInRound] = await feed.latestRoundData();
      expect(roundId).to.equal(1n);
      expect(answer).to.equal(3000n * 10n ** 8n);
      expect(updatedAt).to.equal(startedAt);
      expect(answeredInRound).to.equal(roundId);
      expect(await feed.decimals()).to.equal(8n);
      expect(await feed.version()).to.equal(4n);
      expect(await feed.description()).to.equal("MockV3Aggregator");
    });

    it("updateAnswer 更新价格并进入新轮次", async function () {
      const { feed } = await networkHelpers.loadFixture(deployFixture);
      await feed.updateAnswer(3500n * 10n ** 8n);
      const [roundId, answer] = await feed.latestRoundData();
      expect(roundId).to.equal(2n);
      expect(answer).to.equal(3500n * 10n ** 8n);
      const [, historical] = await feed.getRoundData(1n);
      expect(historical).to.equal(3000n * 10n ** 8n);
    });

    it("updateRoundData 可自定义更新时间", async function () {
      const { feed } = await networkHelpers.loadFixture(deployFixture);
      await feed.updateRoundData(10n, 2000n * 10n ** 8n, 12345n, 12000n);
      const [roundId, answer, startedAt, updatedAt] = await feed.latestRoundData();
      expect(roundId).to.equal(10n);
      expect(answer).to.equal(2000n * 10n ** 8n);
      expect(startedAt).to.equal(12000n);
      expect(updatedAt).to.equal(12345n);
      expect(await feed.latestTimestamp()).to.equal(12345n);
    });
  });

  describe("MockERC20", function () {
    it("使用自定义精度，任何人都可以铸造", async function () {
      const { usdc, alice } = await networkHelpers.loadFixture(deployFixture);
      expect(await usdc.decimals()).to.equal(6n);
      await usdc.connect(alice).mint(alice.address, 1000n * 10n ** 6n);
      expect(await usdc.balanceOf(alice.address)).to.equal(1000n * 10n ** 6n);
    });
  });
});
