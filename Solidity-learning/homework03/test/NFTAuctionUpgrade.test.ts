import { expect } from "chai";
import { network } from "hardhat";
import { DEFAULT_DURATION, ETH, HOUR, deployAuctionSystem, usd, usdc } from "./helpers/fixtures.js";

const { ethers, networkHelpers } = await network.create();
const { time, loadFixture } = networkHelpers;

/** ERC-1967 实现合约地址所在的存储槽：bytes32(uint256(keccak256("eip1967.proxy.implementation")) - 1) */
const IMPLEMENTATION_SLOT = "0x360894a13ba1a3210667c828492db98dca3e2076cc3735a920a3ca505d382bbc";
/** V2 默认防狙击参数：最后 5 分钟内出价，延长 5 分钟 */
const WINDOW = 5n * 60n;
const EXTENSION = 5n * 60n;

describe("NFTAuction 升级到 V2", function () {
  /** 读取代理当前指向的实现合约地址 */
  async function getImplementation(proxyAddress: string) {
    const raw = await ethers.provider.getStorage(proxyAddress, IMPLEMENTATION_SLOT);
    return ethers.getAddress("0x" + raw.slice(-40));
  }

  /**
   * V1 上已有业务数据：
   * - 0 号拍卖（tokenId 1，接受 USDC）：bidder1 出价 0.1 ETH，bidder2 用 500 USDC 超越 → bidder1 有 0.1 ETH 待提取
   * - 1 号拍卖（tokenId 2，仅 ETH）：无人出价
   * - 过期阈值改为 2 小时，便于验证配置被保留
   */
  async function v1WithDataFixture() {
    const ctx = await deployAuctionSystem(ethers);
    const { auction, nft, usdcToken, seller, bidder1, bidder2 } = ctx;
    const nftAddress = await nft.getAddress();

    await auction.connect(seller).createAuction(nftAddress, 1n, await usdcToken.getAddress(), usd("100"), DEFAULT_DURATION);
    await auction.connect(seller).createAuction(nftAddress, 2n, ETH, usd("50"), DEFAULT_DURATION);
    await auction.connect(bidder1).bidWithETH(0n, { value: ethers.parseEther("0.1") });
    await auction.connect(bidder2).bidWithToken(0n, usdc("500"));
    await auction.setPriceStaleThreshold(2n * HOUR);

    const v2Impl = await ethers.deployContract("NFTAuctionV2");
    return { ...ctx, v2Impl };
  }

  /** 在 v1WithDataFixture 基础上完成升级，并在同一交易中调用 initializeV2 */
  async function upgradedFixture() {
    const ctx = await v1WithDataFixture();
    const initData = ctx.v2Impl.interface.encodeFunctionData("initializeV2", [WINDOW, EXTENSION]);
    await ctx.auction.upgradeToAndCall(await ctx.v2Impl.getAddress(), initData);
    const auctionV2 = await ethers.getContractAt("NFTAuctionV2", await ctx.auction.getAddress());
    return { ...ctx, auctionV2 };
  }

  // ============================================================
  //                           升级过程
  // ============================================================

  describe("升级过程", function () {
    it("owner 升级：代理地址不变，实现合约切换为 V2，版本号从 1 变为 2", async function () {
      const { auction, impl, v2Impl } = await loadFixture(v1WithDataFixture);
      const proxyAddress = await auction.getAddress();
      const v2Address = await v2Impl.getAddress();

      expect(await auction.version()).to.equal(1n);
      expect(await getImplementation(proxyAddress)).to.equal(await impl.getAddress());

      const initData = v2Impl.interface.encodeFunctionData("initializeV2", [WINDOW, EXTENSION]);
      const tx = await auction.upgradeToAndCall(v2Address, initData);
      await expect(tx).to.emit(auction, "Upgraded").withArgs(v2Address);

      const auctionV2 = await ethers.getContractAt("NFTAuctionV2", proxyAddress);
      await expect(tx).to.emit(auctionV2, "AntiSnipingUpdated").withArgs(WINDOW, EXTENSION);

      expect(await getImplementation(proxyAddress)).to.equal(v2Address);
      expect(await auctionV2.version()).to.equal(2n);
      expect(await auctionV2.antiSnipingWindow()).to.equal(WINDOW);
      expect(await auctionV2.antiSnipingExtension()).to.equal(EXTENSION);
    });

    it("非 owner 不能升级", async function () {
      const { auction, v2Impl, other } = await loadFixture(v1WithDataFixture);
      await expect(auction.connect(other).upgradeToAndCall(await v2Impl.getAddress(), "0x"))
        .to.be.revertedWithCustomError(auction, "OwnableUnauthorizedAccount")
        .withArgs(other.address);
    });

    it("不能绕过代理直接在实现合约上升级", async function () {
      const { impl, v2Impl } = await loadFixture(v1WithDataFixture);
      await expect(impl.upgradeToAndCall(await v2Impl.getAddress(), "0x")).to.be.revertedWithCustomError(
        impl,
        "UUPSUnauthorizedCallContext",
      );
    });

    it("不能升级到非 UUPS 合约（防止代理永久失去升级能力）", async function () {
      const { auction, nft } = await loadFixture(v1WithDataFixture);
      const nftAddress = await nft.getAddress();
      await expect(auction.upgradeToAndCall(nftAddress, "0x"))
        .to.be.revertedWithCustomError(auction, "ERC1967InvalidImplementation")
        .withArgs(nftAddress);
    });

    it("initializeV2 只能调用一次", async function () {
      const { auctionV2 } = await loadFixture(upgradedFixture);
      await expect(auctionV2.initializeV2(WINDOW, EXTENSION)).to.be.revertedWithCustomError(
        auctionV2,
        "InvalidInitialization",
      );
    });

    it("先升级、后初始化时，只有 owner 能调用 initializeV2", async function () {
      const { auction, v2Impl, other } = await loadFixture(v1WithDataFixture);
      await auction.upgradeToAndCall(await v2Impl.getAddress(), "0x");
      const auctionV2 = await ethers.getContractAt("NFTAuctionV2", await auction.getAddress());

      await expect(auctionV2.connect(other).initializeV2(WINDOW, EXTENSION))
        .to.be.revertedWithCustomError(auctionV2, "OwnableUnauthorizedAccount")
        .withArgs(other.address);
      await auctionV2.initializeV2(WINDOW, EXTENSION);
      expect(await auctionV2.antiSnipingWindow()).to.equal(WINDOW);
    });

    it("V2 实现合约本身不能被初始化", async function () {
      const { v2Impl, owner } = await loadFixture(v1WithDataFixture);
      await expect(v2Impl.initialize(owner.address, owner.address)).to.be.revertedWithCustomError(
        v2Impl,
        "InvalidInitialization",
      );
      await expect(v2Impl.initializeV2(WINDOW, EXTENSION)).to.be.revertedWithCustomError(
        v2Impl,
        "OwnableUnauthorizedAccount",
      );
    });
  });

  // ============================================================
  //                           数据保留
  // ============================================================

  describe("升级后数据保留", function () {
    it("管理员配置全部保留：owner、手续费接收地址、过期阈值、手续费阶梯、喂价", async function () {
      const { auctionV2, owner, feeRecipient, usdcToken, ethFeed, usdcFeed } = await loadFixture(upgradedFixture);
      expect(await auctionV2.owner()).to.equal(owner.address);
      expect(await auctionV2.feeRecipient()).to.equal(feeRecipient.address);
      expect(await auctionV2.priceStaleThreshold()).to.equal(2n * HOUR);
      expect((await auctionV2.getFeeTiers()).map((t) => t.feeBps)).to.deep.equal([500n, 300n, 100n]);
      expect(await auctionV2.priceFeeds(ETH)).to.equal(await ethFeed.getAddress());
      expect(await auctionV2.priceFeeds(await usdcToken.getAddress())).to.equal(await usdcFeed.getAddress());
    });

    it("拍卖数据、待提取余额、托管的 NFT 与资金全部保留", async function () {
      const { auctionV2, nft, usdcToken, seller, bidder1, bidder2 } = await loadFixture(upgradedFixture);
      const proxyAddress = await auctionV2.getAddress();

      expect(await auctionV2.nextAuctionId()).to.equal(2n);

      const a0 = await auctionV2.getAuction(0n);
      expect(a0.seller).to.equal(seller.address);
      expect(a0.tokenId).to.equal(1n);
      expect(a0.highestBidder).to.equal(bidder2.address);
      expect(a0.highestBidToken).to.equal(await usdcToken.getAddress());
      expect(a0.highestBidAmount).to.equal(usdc("500"));
      expect(a0.highestBidUsd).to.equal(usd("500"));
      expect(a0.status).to.equal(0n);

      const a1 = await auctionV2.getAuction(1n);
      expect(a1.tokenId).to.equal(2n);
      expect(a1.paymentToken).to.equal(ETH);
      expect(a1.startPriceUsd).to.equal(usd("50"));

      expect(await auctionV2.pendingReturns(bidder1.address, ETH)).to.equal(ethers.parseEther("0.1"));
      expect(await nft.ownerOf(1n)).to.equal(proxyAddress);
      expect(await nft.ownerOf(2n)).to.equal(proxyAddress);
      expect(await ethers.provider.getBalance(proxyAddress)).to.equal(ethers.parseEther("0.1"));
      expect(await usdcToken.balanceOf(proxyAddress)).to.equal(usdc("500"));
    });

    it("升级前的待提取余额在升级后可以正常提现", async function () {
      const { auctionV2, bidder1 } = await loadFixture(upgradedFixture);
      const amount = ethers.parseEther("0.1");
      await expect(auctionV2.connect(bidder1).withdraw(ETH)).to.changeEtherBalance(ethers, bidder1, amount);
    });

    it("升级前创建的拍卖在升级后可以正常结算，手续费按原阶梯计算", async function () {
      const { auctionV2, nft, usdcToken, seller, feeRecipient, bidder2 } = await loadFixture(upgradedFixture);
      const usdcAddress = await usdcToken.getAddress();
      await time.increaseTo((await auctionV2.getAuction(0n)).endTime);

      await auctionV2.endAuction(0n);
      expect(await nft.ownerOf(1n)).to.equal(bidder2.address);
      expect(await auctionV2.pendingReturns(seller.address, usdcAddress)).to.equal(usdc("475"));
      expect(await auctionV2.pendingReturns(feeRecipient.address, usdcAddress)).to.equal(usdc("25"));
    });

    it("升级后仍可以创建新拍卖，ID 接着递增", async function () {
      const { auctionV2, nft, seller } = await loadFixture(upgradedFixture);
      await auctionV2.connect(seller).createAuction(await nft.getAddress(), 3n, ETH, usd("10"), DEFAULT_DURATION);
      expect(await auctionV2.nextAuctionId()).to.equal(3n);
      expect((await auctionV2.getAuction(2n)).tokenId).to.equal(3n);
    });
  });

  // ============================================================
  //                        V2 新功能：防狙击
  // ============================================================

  describe("防狙击延时", function () {
    it("距结束超过窗口时出价，不延长", async function () {
      const { auctionV2, bidder1 } = await loadFixture(upgradedFixture);
      const endTime = (await auctionV2.getAuction(1n)).endTime;
      await time.setNextBlockTimestamp(endTime - WINDOW);

      await expect(auctionV2.connect(bidder1).bidWithETH(1n, { value: ethers.parseEther("0.1") })).not.to.emit(
        auctionV2,
        "AuctionExtended",
      );
      expect((await auctionV2.getAuction(1n)).endTime).to.equal(endTime);
    });

    it("升级前创建的拍卖，最后 5 分钟内出价会延长 5 分钟（新逻辑作用于旧数据）", async function () {
      const { auctionV2, bidder1 } = await loadFixture(upgradedFixture);
      const endTime = (await auctionV2.getAuction(1n)).endTime;
      await time.setNextBlockTimestamp(endTime - WINDOW + 1n);

      await expect(auctionV2.connect(bidder1).bidWithETH(1n, { value: ethers.parseEther("0.1") }))
        .to.emit(auctionV2, "AuctionExtended")
        .withArgs(1n, endTime + EXTENSION);
      expect((await auctionV2.getAuction(1n)).endTime).to.equal(endTime + EXTENSION);
    });

    it("ERC20 出价同样触发延时", async function () {
      const { auctionV2, bidder1 } = await loadFixture(upgradedFixture);
      const endTime = (await auctionV2.getAuction(0n)).endTime;
      await time.setNextBlockTimestamp(endTime - 10n);

      await expect(auctionV2.connect(bidder1).bidWithToken(0n, usdc("600")))
        .to.emit(auctionV2, "AuctionExtended")
        .withArgs(0n, endTime + EXTENSION);
    });

    it("延长后：原结束时间之后仍可出价，且可以多次延长；到新的结束时间前不能结算", async function () {
      const { auctionV2, bidder1, bidder2 } = await loadFixture(upgradedFixture);
      const endTime = (await auctionV2.getAuction(1n)).endTime;

      await time.setNextBlockTimestamp(endTime - 1n);
      await auctionV2.connect(bidder1).bidWithETH(1n, { value: ethers.parseEther("0.1") });
      const firstExtended = endTime + EXTENSION;

      // 原结束时间已过，但在延长后的窗口内再次出价，再次延长
      await time.setNextBlockTimestamp(endTime + 60n);
      await expect(auctionV2.connect(bidder2).bidWithETH(1n, { value: ethers.parseEther("0.2") }))
        .to.emit(auctionV2, "AuctionExtended")
        .withArgs(1n, firstExtended + EXTENSION);

      const finalEnd = firstExtended + EXTENSION;
      await time.setNextBlockTimestamp(finalEnd - 1n);
      await expect(auctionV2.endAuction(1n))
        .to.be.revertedWithCustomError(auctionV2, "AuctionNotYetEnded")
        .withArgs(1n, finalEnd);

      await time.setNextBlockTimestamp(finalEnd);
      await expect(auctionV2.endAuction(1n))
        .to.emit(auctionV2, "AuctionEnded")
        .withArgs(1n, bidder2.address, ETH, ethers.parseEther("0.2"), ethers.parseEther("0.01"));
    });

    it("窗口设为 0 时关闭防狙击", async function () {
      const { auctionV2, bidder1 } = await loadFixture(upgradedFixture);
      await auctionV2.setAntiSniping(0n, 0n);
      const endTime = (await auctionV2.getAuction(1n)).endTime;
      await time.setNextBlockTimestamp(endTime - 1n);

      await expect(auctionV2.connect(bidder1).bidWithETH(1n, { value: ethers.parseEther("0.1") })).not.to.emit(
        auctionV2,
        "AuctionExtended",
      );
    });

    it("升级时未调用 initializeV2，则防狙击默认关闭", async function () {
      const { auction, v2Impl, bidder1 } = await loadFixture(v1WithDataFixture);
      await auction.upgradeToAndCall(await v2Impl.getAddress(), "0x");
      const auctionV2 = await ethers.getContractAt("NFTAuctionV2", await auction.getAddress());
      const endTime = (await auctionV2.getAuction(1n)).endTime;
      await time.setNextBlockTimestamp(endTime - 1n);

      await expect(auctionV2.connect(bidder1).bidWithETH(1n, { value: ethers.parseEther("0.1") })).not.to.emit(
        auctionV2,
        "AuctionExtended",
      );
    });

    describe("setAntiSniping", function () {
      it("owner 可以调整参数并触发事件", async function () {
        const { auctionV2 } = await loadFixture(upgradedFixture);
        await expect(auctionV2.setAntiSniping(10n * 60n, 2n * 60n))
          .to.emit(auctionV2, "AntiSnipingUpdated")
          .withArgs(10n * 60n, 2n * 60n);
        expect(await auctionV2.antiSnipingWindow()).to.equal(10n * 60n);
        expect(await auctionV2.antiSnipingExtension()).to.equal(2n * 60n);
      });

      it("参数恰好等于上限 1 小时时可以设置", async function () {
        const { auctionV2 } = await loadFixture(upgradedFixture);
        await auctionV2.setAntiSniping(HOUR, HOUR);
        expect(await auctionV2.antiSnipingWindow()).to.equal(HOUR);
      });

      const invalidCases: Array<[string, bigint, bigint]> = [
        ["窗口超过 1 小时", HOUR + 1n, 60n],
        ["延长时长超过 1 小时", 60n, HOUR + 1n],
        ["开启防狙击但延长时长为 0", 60n, 0n],
      ];
      for (const [name, window, extension] of invalidCases) {
        it(`${name}时失败`, async function () {
          const { auctionV2 } = await loadFixture(upgradedFixture);
          await expect(auctionV2.setAntiSniping(window, extension)).to.be.revertedWithCustomError(
            auctionV2,
            "InvalidAntiSnipingConfig",
          );
        });
      }

      it("非 owner 调用失败", async function () {
        const { auctionV2, other } = await loadFixture(upgradedFixture);
        await expect(auctionV2.connect(other).setAntiSniping(60n, 60n))
          .to.be.revertedWithCustomError(auctionV2, "OwnableUnauthorizedAccount")
          .withArgs(other.address);
      });
    });
  });

  // ============================================================
  //                          继续升级
  // ============================================================

  describe("继续升级", function () {
    it("V2 仍保留升级能力，可以再次升级（如回滚到 V1 实现）", async function () {
      const { auctionV2, impl } = await loadFixture(upgradedFixture);
      await auctionV2.upgradeToAndCall(await impl.getAddress(), "0x");
      expect(await auctionV2.version()).to.equal(1n);
      expect(await getImplementation(await auctionV2.getAddress())).to.equal(await impl.getAddress());
    });
  });
});
