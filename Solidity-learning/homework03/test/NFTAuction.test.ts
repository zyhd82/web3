import { expect } from "chai";
import { network } from "hardhat";
import {
  DEFAULT_DURATION,
  ETH,
  ETH_PRICE,
  HOUR,
  USDC_PRICE,
  deployAuctionProxy,
  deployAuctionSystem,
  feedPrice,
  usd,
  usdc,
} from "./helpers/fixtures.js";

const { ethers, networkHelpers } = await network.create();
const { time, loadFixture } = networkHelpers;

const DAY = 24n * HOUR;

describe("NFTAuction", function () {
  // ============================================================
  //                           场景
  // ============================================================

  /** 基础环境：拍卖合约 + NFT + USDC + 喂价 */
  async function systemFixture() {
    return deployAuctionSystem(ethers);
  }

  /** 卖家已创建 0 号拍卖：tokenId 1，接受 ETH + USDC，起拍价 $100，时长 1 小时 */
  async function auctionFixture() {
    const ctx = await deployAuctionSystem(ethers);
    await ctx.auction
      .connect(ctx.seller)
      .createAuction(await ctx.nft.getAddress(), 1n, await ctx.usdcToken.getAddress(), usd("100"), DEFAULT_DURATION);
    const endTime = (await ctx.auction.getAuction(0n)).endTime;
    return { ...ctx, auctionId: 0n, endTime };
  }

  /** 卖家已创建 0 号拍卖，且只接受 ETH */
  async function ethOnlyAuctionFixture() {
    const ctx = await deployAuctionSystem(ethers);
    await ctx.auction
      .connect(ctx.seller)
      .createAuction(await ctx.nft.getAddress(), 1n, ETH, usd("100"), DEFAULT_DURATION);
    return { ...ctx, auctionId: 0n };
  }

  // ============================================================
  //                          初始化
  // ============================================================

  describe("初始化", function () {
    it("正确设置 owner、手续费接收地址、过期阈值、默认手续费阶梯与版本号", async function () {
      const { auction, owner, feeRecipient } = await loadFixture(systemFixture);
      expect(await auction.owner()).to.equal(owner.address);
      expect(await auction.feeRecipient()).to.equal(feeRecipient.address);
      expect(await auction.priceStaleThreshold()).to.equal(DAY);
      expect(await auction.nextAuctionId()).to.equal(0n);
      expect(await auction.version()).to.equal(1n);

      const tiers = await auction.getFeeTiers();
      expect(tiers.map((t) => [t.minUsd, t.feeBps])).to.deep.equal([
        [0n, 500n],
        [usd("1000"), 300n],
        [usd("10000"), 100n],
      ]);
    });

    it("代理不能重复初始化", async function () {
      const { auction, owner } = await loadFixture(systemFixture);
      await expect(auction.initialize(owner.address, owner.address)).to.be.revertedWithCustomError(
        auction,
        "InvalidInitialization",
      );
    });

    it("实现合约不能被直接初始化", async function () {
      const { impl, owner } = await loadFixture(systemFixture);
      await expect(impl.initialize(owner.address, owner.address)).to.be.revertedWithCustomError(
        impl,
        "InvalidInitialization",
      );
    });

    it("手续费接收地址为零地址时初始化失败", async function () {
      const [owner] = await ethers.getSigners();
      const impl = await ethers.deployContract("NFTAuction");
      await expect(deployAuctionProxy(ethers, owner.address, ethers.ZeroAddress)).to.be.revertedWithCustomError(
        impl,
        "ZeroAddress",
      );
    });

    it("owner 为零地址时初始化失败", async function () {
      const [owner] = await ethers.getSigners();
      const impl = await ethers.deployContract("NFTAuction");
      await expect(deployAuctionProxy(ethers, ethers.ZeroAddress, owner.address))
        .to.be.revertedWithCustomError(impl, "OwnableInvalidOwner")
        .withArgs(ethers.ZeroAddress);
    });
  });

  // ============================================================
  //                         管理员配置
  // ============================================================

  describe("管理员配置", function () {
    describe("setPriceFeed", function () {
      it("owner 可以设置喂价并触发事件", async function () {
        const { auction, other } = await loadFixture(systemFixture);
        await expect(auction.setPriceFeed(ETH, other.address))
          .to.emit(auction, "PriceFeedUpdated")
          .withArgs(ETH, other.address);
        expect(await auction.priceFeeds(ETH)).to.equal(other.address);
      });

      it("喂价地址为零地址时失败", async function () {
        const { auction } = await loadFixture(systemFixture);
        await expect(auction.setPriceFeed(ETH, ethers.ZeroAddress)).to.be.revertedWithCustomError(
          auction,
          "ZeroAddress",
        );
      });

      it("非 owner 调用失败", async function () {
        const { auction, other } = await loadFixture(systemFixture);
        await expect(auction.connect(other).setPriceFeed(ETH, other.address))
          .to.be.revertedWithCustomError(auction, "OwnableUnauthorizedAccount")
          .withArgs(other.address);
      });
    });

    describe("setPriceStaleThreshold", function () {
      it("owner 可以设置过期阈值并触发事件", async function () {
        const { auction } = await loadFixture(systemFixture);
        await expect(auction.setPriceStaleThreshold(HOUR))
          .to.emit(auction, "PriceStaleThresholdUpdated")
          .withArgs(HOUR);
        expect(await auction.priceStaleThreshold()).to.equal(HOUR);
      });

      it("阈值为 0 时失败", async function () {
        const { auction } = await loadFixture(systemFixture);
        await expect(auction.setPriceStaleThreshold(0n)).to.be.revertedWithCustomError(auction, "InvalidThreshold");
      });

      it("非 owner 调用失败", async function () {
        const { auction, other } = await loadFixture(systemFixture);
        await expect(auction.connect(other).setPriceStaleThreshold(HOUR))
          .to.be.revertedWithCustomError(auction, "OwnableUnauthorizedAccount")
          .withArgs(other.address);
      });
    });

    describe("setFeeRecipient", function () {
      it("owner 可以设置手续费接收地址并触发事件", async function () {
        const { auction, other } = await loadFixture(systemFixture);
        await expect(auction.setFeeRecipient(other.address))
          .to.emit(auction, "FeeRecipientUpdated")
          .withArgs(other.address);
        expect(await auction.feeRecipient()).to.equal(other.address);
      });

      it("零地址时失败", async function () {
        const { auction } = await loadFixture(systemFixture);
        await expect(auction.setFeeRecipient(ethers.ZeroAddress)).to.be.revertedWithCustomError(
          auction,
          "ZeroAddress",
        );
      });

      it("非 owner 调用失败", async function () {
        const { auction, other } = await loadFixture(systemFixture);
        await expect(auction.connect(other).setFeeRecipient(other.address))
          .to.be.revertedWithCustomError(auction, "OwnableUnauthorizedAccount")
          .withArgs(other.address);
      });
    });

    describe("setFeeTiers", function () {
      it("owner 可以整体替换阶梯并触发事件", async function () {
        const { auction } = await loadFixture(systemFixture);
        const tiers = [
          { minUsd: 0n, feeBps: 250 },
          { minUsd: usd("5000"), feeBps: 50 },
        ];
        await expect(auction.setFeeTiers(tiers)).to.emit(auction, "FeeTiersUpdated");
        const stored = await auction.getFeeTiers();
        expect(stored.map((t) => [t.minUsd, t.feeBps])).to.deep.equal([
          [0n, 250n],
          [usd("5000"), 50n],
        ]);
        expect(await auction.getFeeBps(usd("4999"))).to.equal(250n);
        expect(await auction.getFeeBps(usd("5000"))).to.equal(50n);
      });

      it("允许恰好 10 档、费率恰好为上限 10%", async function () {
        const { auction } = await loadFixture(systemFixture);
        const tiers = Array.from({ length: 10 }, (_, i) => ({ minUsd: usd(String(i * 100)), feeBps: 1000 }));
        await auction.setFeeTiers(tiers);
        expect((await auction.getFeeTiers()).length).to.equal(10);
      });

      const invalidCases: Array<[string, Array<{ minUsd: bigint; feeBps: number }>]> = [
        ["空阶梯", []],
        ["超过 10 档", Array.from({ length: 11 }, (_, i) => ({ minUsd: BigInt(i), feeBps: 100 }))],
        ["第一档起点不为 0", [{ minUsd: 1n, feeBps: 100 }]],
        [
          "起点不是严格递增（相等）",
          [
            { minUsd: 0n, feeBps: 100 },
            { minUsd: 0n, feeBps: 50 },
          ],
        ],
        [
          "起点不是严格递增（递减）",
          [
            { minUsd: 0n, feeBps: 100 },
            { minUsd: usd("100"), feeBps: 50 },
            { minUsd: usd("50"), feeBps: 10 },
          ],
        ],
        ["费率超过 10%", [{ minUsd: 0n, feeBps: 1001 }]],
      ];
      for (const [name, tiers] of invalidCases) {
        it(`${name}时失败`, async function () {
          const { auction } = await loadFixture(systemFixture);
          await expect(auction.setFeeTiers(tiers)).to.be.revertedWithCustomError(auction, "InvalidFeeTiers");
        });
      }

      it("非 owner 调用失败", async function () {
        const { auction, other } = await loadFixture(systemFixture);
        await expect(auction.connect(other).setFeeTiers([{ minUsd: 0n, feeBps: 100 }]))
          .to.be.revertedWithCustomError(auction, "OwnableUnauthorizedAccount")
          .withArgs(other.address);
      });
    });
  });

  // ============================================================
  //                       价格换算（Chainlink）
  // ============================================================

  describe("价格换算", function () {
    it("getLatestPrice 返回喂价与精度", async function () {
      const { auction, usdcToken } = await loadFixture(systemFixture);
      expect(await auction.getLatestPrice(ETH)).to.deep.equal([ETH_PRICE, 8n]);
      expect(await auction.getLatestPrice(await usdcToken.getAddress())).to.deep.equal([USDC_PRICE, 8n]);
    });

    it("ETH 换算美元：0.5 ETH × $3000 = $1500", async function () {
      const { auction } = await loadFixture(systemFixture);
      expect(await auction.toUsd(ETH, ethers.parseEther("0.5"))).to.equal(usd("1500"));
    });

    it("6 位精度的 USDC 换算美元：2000 USDC × $1 = $2000", async function () {
      const { auction, usdcToken } = await loadFixture(systemFixture);
      expect(await auction.toUsd(await usdcToken.getAddress(), usdc("2000"))).to.equal(usd("2000"));
    });

    it("喂价精度不是 8 位时也能正确换算", async function () {
      const { auction } = await loadFixture(systemFixture);
      const feed18 = await ethers.deployContract("MockV3Aggregator", [18, ethers.parseUnits("2500", 18)]);
      await auction.setPriceFeed(ETH, await feed18.getAddress());
      expect(await auction.toUsd(ETH, ethers.parseEther("2"))).to.equal(usd("5000"));
    });

    it("价格变动后按最新价格换算", async function () {
      const { auction, ethFeed } = await loadFixture(systemFixture);
      await ethFeed.updateAnswer(feedPrice("3500"));
      expect(await auction.toUsd(ETH, ethers.parseEther("1"))).to.equal(usd("3500"));
    });

    it("未配置喂价时失败", async function () {
      const { auction, other } = await loadFixture(systemFixture);
      await expect(auction.toUsd(other.address, 1n))
        .to.be.revertedWithCustomError(auction, "PriceFeedNotSet")
        .withArgs(other.address);
    });

    it("喂价为 0 或负数时失败", async function () {
      const { auction, ethFeed } = await loadFixture(systemFixture);
      await ethFeed.updateAnswer(0n);
      await expect(auction.getLatestPrice(ETH))
        .to.be.revertedWithCustomError(auction, "InvalidPrice")
        .withArgs(ETH, 0n);
      await ethFeed.updateAnswer(-1n);
      await expect(auction.getLatestPrice(ETH))
        .to.be.revertedWithCustomError(auction, "InvalidPrice")
        .withArgs(ETH, -1n);
    });

    it("喂价超过过期阈值时失败", async function () {
      const { auction, ethFeed } = await loadFixture(systemFixture);
      const now = BigInt(await time.latest());
      const updatedAt = now - DAY - 10n;
      await ethFeed.updateRoundData(2n, ETH_PRICE, updatedAt, updatedAt);
      await expect(auction.getLatestPrice(ETH))
        .to.be.revertedWithCustomError(auction, "StalePrice")
        .withArgs(ETH, updatedAt);
    });

    it("喂价在过期阈值内时可用", async function () {
      const { auction, ethFeed } = await loadFixture(systemFixture);
      const updatedAt = BigInt(await time.latest()) - DAY + 60n;
      await ethFeed.updateRoundData(2n, ETH_PRICE, updatedAt, updatedAt);
      expect(await auction.getLatestPrice(ETH)).to.deep.equal([ETH_PRICE, 8n]);
    });

    it("喂价更新时间为 0 时视为过期", async function () {
      const { auction, ethFeed } = await loadFixture(systemFixture);
      await ethFeed.updateRoundData(2n, ETH_PRICE, 0n, 0n);
      await expect(auction.getLatestPrice(ETH))
        .to.be.revertedWithCustomError(auction, "StalePrice")
        .withArgs(ETH, 0n);
    });
  });

  // ============================================================
  //                          动态手续费
  // ============================================================

  describe("动态手续费 getFeeBps", function () {
    const cases: Array<[string, bigint, bigint]> = [
      ["$0", 0n, 500n],
      ["$999.99", usd("999.99"), 500n],
      ["$1,000（第二档起点）", usd("1000"), 300n],
      ["$9,999.99", usd("9999.99"), 300n],
      ["$10,000（第三档起点）", usd("10000"), 100n],
      ["$1,000,000", usd("1000000"), 100n],
    ];
    for (const [name, amountUsd, expected] of cases) {
      it(`${name} 对应费率 ${expected} bps`, async function () {
        const { auction } = await loadFixture(systemFixture);
        expect(await auction.getFeeBps(amountUsd)).to.equal(expected);
      });
    }
  });

  // ============================================================
  //                          创建拍卖
  // ============================================================

  describe("createAuction", function () {
    it("卖家创建拍卖：NFT 托管到合约、记录拍卖信息并触发事件", async function () {
      const { auction, nft, usdcToken, seller } = await loadFixture(systemFixture);
      const nftAddress = await nft.getAddress();
      const usdcAddress = await usdcToken.getAddress();
      const startTime = BigInt(await time.latest()) + 1n;
      await time.setNextBlockTimestamp(startTime);

      await expect(auction.connect(seller).createAuction(nftAddress, 1n, usdcAddress, usd("100"), DEFAULT_DURATION))
        .to.emit(auction, "AuctionCreated")
        .withArgs(0n, seller.address, nftAddress, 1n, usdcAddress, usd("100"), startTime + DEFAULT_DURATION);

      expect(await nft.ownerOf(1n)).to.equal(await auction.getAddress());
      expect(await auction.nextAuctionId()).to.equal(1n);

      const a = await auction.getAuction(0n);
      expect(a.seller).to.equal(seller.address);
      expect(a.nft).to.equal(nftAddress);
      expect(a.tokenId).to.equal(1n);
      expect(a.paymentToken).to.equal(usdcAddress);
      expect(a.startPriceUsd).to.equal(usd("100"));
      expect(a.startTime).to.equal(startTime);
      expect(a.endTime).to.equal(startTime + DEFAULT_DURATION);
      expect(a.highestBidder).to.equal(ethers.ZeroAddress);
      expect(a.highestBidAmount).to.equal(0n);
      expect(a.status).to.equal(0n); // Active
    });

    it("拍卖 ID 依次递增", async function () {
      const { auction, nft, seller } = await loadFixture(systemFixture);
      const nftAddress = await nft.getAddress();
      expect(
        await auction.connect(seller).createAuction.staticCall(nftAddress, 1n, ETH, usd("1"), DEFAULT_DURATION),
      ).to.equal(0n);
      await auction.connect(seller).createAuction(nftAddress, 1n, ETH, usd("1"), DEFAULT_DURATION);
      await auction.connect(seller).createAuction(nftAddress, 2n, ETH, usd("1"), DEFAULT_DURATION);
      expect(await auction.nextAuctionId()).to.equal(2n);
      expect((await auction.getAuction(1n)).tokenId).to.equal(2n);
    });

    it("paymentToken 为零地址时创建仅接受 ETH 的拍卖", async function () {
      const { auction } = await loadFixture(ethOnlyAuctionFixture);
      expect((await auction.getAuction(0n)).paymentToken).to.equal(ETH);
    });

    it("时长恰好为下限 60 秒和上限 30 天时可以创建", async function () {
      const { auction, nft, seller } = await loadFixture(systemFixture);
      const nftAddress = await nft.getAddress();
      await auction.connect(seller).createAuction(nftAddress, 1n, ETH, usd("1"), 60n);
      await auction.connect(seller).createAuction(nftAddress, 2n, ETH, usd("1"), 30n * DAY);
      expect(await auction.nextAuctionId()).to.equal(2n);
    });

    it("NFT 地址为零地址时失败", async function () {
      const { auction, seller } = await loadFixture(systemFixture);
      await expect(
        auction.connect(seller).createAuction(ethers.ZeroAddress, 1n, ETH, usd("1"), DEFAULT_DURATION),
      ).to.be.revertedWithCustomError(auction, "ZeroAddress");
    });

    for (const duration of [59n, 30n * DAY + 1n]) {
      it(`时长为 ${duration} 秒（超出范围）时失败`, async function () {
        const { auction, nft, seller } = await loadFixture(systemFixture);
        await expect(auction.connect(seller).createAuction(await nft.getAddress(), 1n, ETH, usd("1"), duration))
          .to.be.revertedWithCustomError(auction, "InvalidDuration")
          .withArgs(duration);
      });
    }

    it("起拍价为 0 时失败", async function () {
      const { auction, nft, seller } = await loadFixture(systemFixture);
      await expect(
        auction.connect(seller).createAuction(await nft.getAddress(), 1n, ETH, 0n, DEFAULT_DURATION),
      ).to.be.revertedWithCustomError(auction, "InvalidStartPrice");
    });

    it("ETH 喂价未配置时失败", async function () {
      const [owner, seller] = await ethers.getSigners();
      const { auction } = await deployAuctionProxy(ethers, owner.address, owner.address);
      await expect(auction.connect(seller).createAuction(owner.address, 1n, ETH, usd("1"), DEFAULT_DURATION))
        .to.be.revertedWithCustomError(auction, "PriceFeedNotSet")
        .withArgs(ETH);
    });

    it("ERC20 喂价未配置时失败", async function () {
      const { auction, nft, seller, other } = await loadFixture(systemFixture);
      await expect(
        auction.connect(seller).createAuction(await nft.getAddress(), 1n, other.address, usd("1"), DEFAULT_DURATION),
      )
        .to.be.revertedWithCustomError(auction, "PriceFeedNotSet")
        .withArgs(other.address);
    });

    it("非 NFT 持有者不能拿别人的 NFT 创建拍卖", async function () {
      const { auction, nft, seller, other } = await loadFixture(systemFixture);
      await expect(auction.connect(other).createAuction(await nft.getAddress(), 1n, ETH, usd("1"), DEFAULT_DURATION))
        .to.be.revertedWithCustomError(nft, "ERC721IncorrectOwner")
        .withArgs(other.address, 1n, seller.address);
    });

    it("未授权拍卖合约时失败", async function () {
      const { auction, nft, seller } = await loadFixture(systemFixture);
      await nft.connect(seller).setApprovalForAll(await auction.getAddress(), false);
      await expect(auction.connect(seller).createAuction(await nft.getAddress(), 1n, ETH, usd("1"), DEFAULT_DURATION))
        .to.be.revertedWithCustomError(nft, "ERC721InsufficientApproval")
        .withArgs(await auction.getAddress(), 1n);
    });

    it("查询不存在的拍卖失败", async function () {
      const { auction } = await loadFixture(systemFixture);
      await expect(auction.getAuction(99n)).to.be.revertedWithCustomError(auction, "AuctionNotFound").withArgs(99n);
    });
  });

  // ============================================================
  //                             出价
  // ============================================================

  describe("出价", function () {
    describe("bidWithETH", function () {
      it("首次出价不低于起拍价：记录最高出价、锁定美元价值并触发事件", async function () {
        const { auction, auctionId, bidder1 } = await loadFixture(auctionFixture);
        const amount = ethers.parseEther("0.1"); // $300
        // 同一笔交易做多个断言时先发送交易再分别断言，否则串联的匹配器会把交易发送两次
        const tx = await auction.connect(bidder1).bidWithETH(auctionId, { value: amount });
        await expect(tx).to.emit(auction, "BidPlaced").withArgs(auctionId, bidder1.address, ETH, amount, usd("300"));
        await expect(tx).to.changeEtherBalances(ethers, [bidder1, auction], [-amount, amount]);

        const a = await auction.getAuction(auctionId);
        expect(a.highestBidder).to.equal(bidder1.address);
        expect(a.highestBidToken).to.equal(ETH);
        expect(a.highestBidAmount).to.equal(amount);
        expect(a.highestBidUsd).to.equal(usd("300"));
      });

      it("首次出价恰好等于起拍价时成功", async function () {
        const { auction, nft, seller, bidder1 } = await loadFixture(auctionFixture);
        // 新建 1 号拍卖，起拍价 $300，恰好等于 0.1 ETH × $3000
        await auction.connect(seller).createAuction(await nft.getAddress(), 2n, ETH, usd("300"), DEFAULT_DURATION);
        await expect(auction.connect(bidder1).bidWithETH(1n, { value: ethers.parseEther("0.1") }))
          .to.emit(auction, "BidPlaced")
          .withArgs(1n, bidder1.address, ETH, ethers.parseEther("0.1"), usd("300"));
      });

      it("首次出价低于起拍价时失败", async function () {
        const { auction, auctionId, bidder1 } = await loadFixture(auctionFixture);
        const amount = ethers.parseEther("0.03"); // $90 < $100
        await expect(auction.connect(bidder1).bidWithETH(auctionId, { value: amount }))
          .to.be.revertedWithCustomError(auction, "BidTooLow")
          .withArgs(usd("90"), usd("100"));
      });

      it("出价金额为 0 时失败", async function () {
        const { auction, auctionId, bidder1 } = await loadFixture(auctionFixture);
        await expect(auction.connect(bidder1).bidWithETH(auctionId, { value: 0n })).to.be.revertedWithCustomError(
          auction,
          "ZeroAmount",
        );
      });

      it("拍卖不存在时失败", async function () {
        const { auction, bidder1 } = await loadFixture(auctionFixture);
        await expect(auction.connect(bidder1).bidWithETH(99n, { value: 1n }))
          .to.be.revertedWithCustomError(auction, "AuctionNotFound")
          .withArgs(99n);
      });

      it("卖家不能给自己的拍卖出价", async function () {
        const { auction, auctionId, seller } = await loadFixture(auctionFixture);
        await expect(
          auction.connect(seller).bidWithETH(auctionId, { value: ethers.parseEther("1") }),
        ).to.be.revertedWithCustomError(auction, "SellerCannotBid");
      });

      it("到达结束时间后出价失败", async function () {
        const { auction, auctionId, endTime, bidder1 } = await loadFixture(auctionFixture);
        await time.setNextBlockTimestamp(endTime);
        await expect(auction.connect(bidder1).bidWithETH(auctionId, { value: ethers.parseEther("1") }))
          .to.be.revertedWithCustomError(auction, "AuctionAlreadyEnded")
          .withArgs(auctionId);
      });

      it("结束时间前 1 秒出价仍然成功", async function () {
        const { auction, auctionId, endTime, bidder1 } = await loadFixture(auctionFixture);
        await time.setNextBlockTimestamp(endTime - 1n);
        await expect(auction.connect(bidder1).bidWithETH(auctionId, { value: ethers.parseEther("1") })).to.emit(
          auction,
          "BidPlaced",
        );
      });

      it("已取消的拍卖不能出价", async function () {
        const { auction, auctionId, seller, bidder1 } = await loadFixture(auctionFixture);
        await auction.connect(seller).cancelAuction(auctionId);
        await expect(auction.connect(bidder1).bidWithETH(auctionId, { value: ethers.parseEther("1") }))
          .to.be.revertedWithCustomError(auction, "AuctionNotActive")
          .withArgs(auctionId);
      });

      it("喂价过期时不能出价", async function () {
        const { auction, auctionId, ethFeed, bidder1 } = await loadFixture(auctionFixture);
        await ethFeed.updateRoundData(2n, ETH_PRICE, 1n, 1n);
        await expect(
          auction.connect(bidder1).bidWithETH(auctionId, { value: ethers.parseEther("1") }),
        ).to.be.revertedWithCustomError(auction, "StalePrice");
      });
    });

    describe("bidWithToken", function () {
      it("使用 USDC 出价：代币转入合约并记录最高出价", async function () {
        const { auction, auctionId, usdcToken, bidder1 } = await loadFixture(auctionFixture);
        const amount = usdc("500");
        const tx = await auction.connect(bidder1).bidWithToken(auctionId, amount);
        await expect(tx)
          .to.emit(auction, "BidPlaced")
          .withArgs(auctionId, bidder1.address, await usdcToken.getAddress(), amount, usd("500"));
        await expect(tx).to.changeTokenBalances(ethers, usdcToken, [bidder1, auction], [-amount, amount]);

        const a = await auction.getAuction(auctionId);
        expect(a.highestBidToken).to.equal(await usdcToken.getAddress());
        expect(a.highestBidAmount).to.equal(amount);
        expect(a.highestBidUsd).to.equal(usd("500"));
      });

      it("仅接受 ETH 的拍卖不能用 ERC20 出价", async function () {
        const { auction, auctionId, bidder1 } = await loadFixture(ethOnlyAuctionFixture);
        await expect(auction.connect(bidder1).bidWithToken(auctionId, usdc("500")))
          .to.be.revertedWithCustomError(auction, "TokenNotAccepted")
          .withArgs(auctionId);
      });

      it("拍卖不存在时失败", async function () {
        const { auction, bidder1 } = await loadFixture(auctionFixture);
        await expect(auction.connect(bidder1).bidWithToken(99n, usdc("500")))
          .to.be.revertedWithCustomError(auction, "AuctionNotFound")
          .withArgs(99n);
      });

      it("出价金额为 0 时失败", async function () {
        const { auction, auctionId, bidder1 } = await loadFixture(auctionFixture);
        await expect(auction.connect(bidder1).bidWithToken(auctionId, 0n)).to.be.revertedWithCustomError(
          auction,
          "ZeroAmount",
        );
      });

      it("未授权代币时失败", async function () {
        const { auction, auctionId, usdcToken, other } = await loadFixture(auctionFixture);
        await usdcToken.mint(other.address, usdc("500"));
        await expect(auction.connect(other).bidWithToken(auctionId, usdc("500")))
          .to.be.revertedWithCustomError(usdcToken, "ERC20InsufficientAllowance")
          .withArgs(await auction.getAddress(), 0n, usdc("500"));
      });
    });

    describe("加价与退款（pull 模式）", function () {
      it("新出价必须严格高于当前最高出价的美元价值", async function () {
        const { auction, auctionId, bidder1, bidder2 } = await loadFixture(auctionFixture);
        await auction.connect(bidder1).bidWithToken(auctionId, usdc("300"));
        await expect(auction.connect(bidder2).bidWithETH(auctionId, { value: ethers.parseEther("0.1") })) // $300
          .to.be.revertedWithCustomError(auction, "BidTooLow")
          .withArgs(usd("300"), usd("300") + 1n);
      });

      it("被超越的出价记入原出价者的待提取余额，不直接转账", async function () {
        const { auction, auctionId, bidder1, bidder2 } = await loadFixture(auctionFixture);
        const first = ethers.parseEther("0.1");
        await auction.connect(bidder1).bidWithETH(auctionId, { value: first });

        await expect(
          auction.connect(bidder2).bidWithETH(auctionId, { value: ethers.parseEther("0.2") }),
        ).to.changeEtherBalances(ethers, [bidder1], [0n]);

        expect(await auction.pendingReturns(bidder1.address, ETH)).to.equal(first);
        expect((await auction.getAuction(auctionId)).highestBidder).to.equal(bidder2.address);
      });

      it("ETH 与 USDC 混合竞价：按美元比较，退款按原币种记账", async function () {
        const { auction, auctionId, usdcToken, bidder1, bidder2 } = await loadFixture(auctionFixture);
        const usdcAddress = await usdcToken.getAddress();

        await auction.connect(bidder1).bidWithETH(auctionId, { value: ethers.parseEther("0.5") }); // $1500
        await expect(auction.connect(bidder2).bidWithToken(auctionId, usdc("1500")))
          .to.be.revertedWithCustomError(auction, "BidTooLow")
          .withArgs(usd("1500"), usd("1500") + 1n);

        await auction.connect(bidder2).bidWithToken(auctionId, usdc("1600")); // $1600
        await auction.connect(bidder1).bidWithETH(auctionId, { value: ethers.parseEther("0.6") }); // $1800

        expect(await auction.pendingReturns(bidder1.address, ETH)).to.equal(ethers.parseEther("0.5"));
        expect(await auction.pendingReturns(bidder2.address, usdcAddress)).to.equal(usdc("1600"));
        const a = await auction.getAuction(auctionId);
        expect(a.highestBidder).to.equal(bidder1.address);
        expect(a.highestBidUsd).to.equal(usd("1800"));
      });

      it("同一出价者再次加价时，旧出价同样退回待提取余额（不会被锁死）", async function () {
        const { auction, auctionId, bidder1 } = await loadFixture(auctionFixture);
        await auction.connect(bidder1).bidWithETH(auctionId, { value: ethers.parseEther("0.1") });
        await auction.connect(bidder1).bidWithETH(auctionId, { value: ethers.parseEther("0.2") });
        expect(await auction.pendingReturns(bidder1.address, ETH)).to.equal(ethers.parseEther("0.1"));
        expect((await auction.getAuction(auctionId)).highestBidAmount).to.equal(ethers.parseEther("0.2"));
      });

      it("出价的美元价值在出价时锁定，之后币价变动不影响已有出价", async function () {
        const { auction, auctionId, ethFeed, bidder1, bidder2 } = await loadFixture(auctionFixture);
        await auction.connect(bidder1).bidWithETH(auctionId, { value: ethers.parseEther("1") }); // $3000
        await ethFeed.updateAnswer(feedPrice("2000")); // ETH 跌到 $2000

        expect((await auction.getAuction(auctionId)).highestBidUsd).to.equal(usd("3000"));
        // 1.5 ETH 现在只值 $3000，不高于已锁定的 $3000
        await expect(auction.connect(bidder2).bidWithETH(auctionId, { value: ethers.parseEther("1.5") }))
          .to.be.revertedWithCustomError(auction, "BidTooLow")
          .withArgs(usd("3000"), usd("3000") + 1n);
      });
    });
  });

  // ============================================================
  //                             结算
  // ============================================================

  describe("endAuction", function () {
    it("未到结束时间时失败", async function () {
      const { auction, auctionId, endTime } = await loadFixture(auctionFixture);
      await time.setNextBlockTimestamp(endTime - 1n);
      await expect(auction.endAuction(auctionId))
        .to.be.revertedWithCustomError(auction, "AuctionNotYetEnded")
        .withArgs(auctionId, endTime);
    });

    it("恰好到达结束时间即可结算", async function () {
      const { auction, auctionId, endTime } = await loadFixture(auctionFixture);
      await time.setNextBlockTimestamp(endTime);
      await expect(auction.endAuction(auctionId)).to.emit(auction, "AuctionEnded");
    });

    it("流拍：NFT 退回卖家，不产生手续费", async function () {
      const { auction, auctionId, endTime, nft, seller, other } = await loadFixture(auctionFixture);
      await time.increaseTo(endTime);
      await expect(auction.connect(other).endAuction(auctionId))
        .to.emit(auction, "AuctionEnded")
        .withArgs(auctionId, ethers.ZeroAddress, ETH, 0n, 0n);
      expect(await nft.ownerOf(1n)).to.equal(seller.address);
      expect((await auction.getAuction(auctionId)).status).to.equal(1n); // Ended
    });

    it("ETH 成交（$1500，第二档 3%）：NFT 给买家，货款与手续费分别记账", async function () {
      const { auction, auctionId, endTime, nft, seller, feeRecipient, bidder1, other } =
        await loadFixture(auctionFixture);
      const amount = ethers.parseEther("0.5");
      await auction.connect(bidder1).bidWithETH(auctionId, { value: amount });
      await time.increaseTo(endTime);

      const fee = (amount * 300n) / 10_000n;
      await expect(auction.connect(other).endAuction(auctionId))
        .to.emit(auction, "AuctionEnded")
        .withArgs(auctionId, bidder1.address, ETH, amount, fee);

      expect(await nft.ownerOf(1n)).to.equal(bidder1.address);
      expect(await auction.pendingReturns(seller.address, ETH)).to.equal(amount - fee);
      expect(await auction.pendingReturns(feeRecipient.address, ETH)).to.equal(fee);
    });

    it("USDC 成交（$500，第一档 5%）：手续费以 USDC 收取", async function () {
      const { auction, auctionId, endTime, usdcToken, seller, feeRecipient, bidder1 } =
        await loadFixture(auctionFixture);
      const usdcAddress = await usdcToken.getAddress();
      await auction.connect(bidder1).bidWithToken(auctionId, usdc("500"));
      await time.increaseTo(endTime);
      await auction.endAuction(auctionId);

      expect(await auction.pendingReturns(seller.address, usdcAddress)).to.equal(usdc("475"));
      expect(await auction.pendingReturns(feeRecipient.address, usdcAddress)).to.equal(usdc("25"));
    });

    it("大额成交（$12,000，第三档 1%）", async function () {
      const { auction, auctionId, endTime, seller, feeRecipient, bidder1 } = await loadFixture(auctionFixture);
      const amount = ethers.parseEther("4");
      await auction.connect(bidder1).bidWithETH(auctionId, { value: amount });
      await time.increaseTo(endTime);
      await auction.endAuction(auctionId);

      expect(await auction.pendingReturns(seller.address, ETH)).to.equal(ethers.parseEther("3.96"));
      expect(await auction.pendingReturns(feeRecipient.address, ETH)).to.equal(ethers.parseEther("0.04"));
    });

    it("按成交时的 feeRecipient 记账手续费", async function () {
      const { auction, auctionId, endTime, bidder1, other } = await loadFixture(auctionFixture);
      await auction.connect(bidder1).bidWithETH(auctionId, { value: ethers.parseEther("0.5") });
      await auction.setFeeRecipient(other.address);
      await time.increaseTo(endTime);
      await auction.endAuction(auctionId);
      expect(await auction.pendingReturns(other.address, ETH)).to.equal(ethers.parseEther("0.015"));
    });

    it("不能重复结算", async function () {
      const { auction, auctionId, endTime } = await loadFixture(auctionFixture);
      await time.increaseTo(endTime);
      await auction.endAuction(auctionId);
      await expect(auction.endAuction(auctionId))
        .to.be.revertedWithCustomError(auction, "AuctionNotActive")
        .withArgs(auctionId);
    });

    it("已取消的拍卖不能结算", async function () {
      const { auction, auctionId, seller } = await loadFixture(auctionFixture);
      await auction.connect(seller).cancelAuction(auctionId);
      await expect(auction.endAuction(auctionId))
        .to.be.revertedWithCustomError(auction, "AuctionNotActive")
        .withArgs(auctionId);
    });

    it("拍卖不存在时失败", async function () {
      const { auction } = await loadFixture(auctionFixture);
      await expect(auction.endAuction(99n)).to.be.revertedWithCustomError(auction, "AuctionNotFound").withArgs(99n);
    });
  });

  // ============================================================
  //                             取消
  // ============================================================

  describe("cancelAuction", function () {
    it("卖家在无人出价时取消：NFT 退回并触发事件", async function () {
      const { auction, auctionId, nft, seller } = await loadFixture(auctionFixture);
      await expect(auction.connect(seller).cancelAuction(auctionId))
        .to.emit(auction, "AuctionCancelled")
        .withArgs(auctionId);
      expect(await nft.ownerOf(1n)).to.equal(seller.address);
      expect((await auction.getAuction(auctionId)).status).to.equal(2n); // Cancelled
    });

    it("到期后仍无人出价时，卖家也可以取消", async function () {
      const { auction, auctionId, endTime, seller } = await loadFixture(auctionFixture);
      await time.increaseTo(endTime + 10n);
      await expect(auction.connect(seller).cancelAuction(auctionId)).to.emit(auction, "AuctionCancelled");
    });

    it("非卖家不能取消", async function () {
      const { auction, auctionId, other } = await loadFixture(auctionFixture);
      await expect(auction.connect(other).cancelAuction(auctionId)).to.be.revertedWithCustomError(
        auction,
        "NotSeller",
      );
    });

    it("已有出价时不能取消", async function () {
      const { auction, auctionId, seller, bidder1 } = await loadFixture(auctionFixture);
      await auction.connect(bidder1).bidWithETH(auctionId, { value: ethers.parseEther("0.1") });
      await expect(auction.connect(seller).cancelAuction(auctionId))
        .to.be.revertedWithCustomError(auction, "AuctionHasBids")
        .withArgs(auctionId);
    });

    it("已取消的拍卖不能再次取消", async function () {
      const { auction, auctionId, seller } = await loadFixture(auctionFixture);
      await auction.connect(seller).cancelAuction(auctionId);
      await expect(auction.connect(seller).cancelAuction(auctionId))
        .to.be.revertedWithCustomError(auction, "AuctionNotActive")
        .withArgs(auctionId);
    });

    it("拍卖不存在时失败", async function () {
      const { auction, seller } = await loadFixture(auctionFixture);
      await expect(auction.connect(seller).cancelAuction(99n))
        .to.be.revertedWithCustomError(auction, "AuctionNotFound")
        .withArgs(99n);
    });
  });

  // ============================================================
  //                             提现
  // ============================================================

  describe("withdraw", function () {
    it("被超越的出价者提取 ETH", async function () {
      const { auction, auctionId, bidder1, bidder2 } = await loadFixture(auctionFixture);
      const first = ethers.parseEther("0.1");
      await auction.connect(bidder1).bidWithETH(auctionId, { value: first });
      await auction.connect(bidder2).bidWithETH(auctionId, { value: ethers.parseEther("0.2") });

      const tx = await auction.connect(bidder1).withdraw(ETH);
      await expect(tx).to.emit(auction, "Withdrawn").withArgs(bidder1.address, ETH, first);
      await expect(tx).to.changeEtherBalances(ethers, [bidder1, auction], [first, -first]);
      expect(await auction.pendingReturns(bidder1.address, ETH)).to.equal(0n);
    });

    it("被超越的出价者提取 USDC", async function () {
      const { auction, auctionId, usdcToken, bidder1, bidder2 } = await loadFixture(auctionFixture);
      const usdcAddress = await usdcToken.getAddress();
      await auction.connect(bidder1).bidWithToken(auctionId, usdc("300"));
      await auction.connect(bidder2).bidWithToken(auctionId, usdc("400"));

      const tx = await auction.connect(bidder1).withdraw(usdcAddress);
      await expect(tx).to.emit(auction, "Withdrawn").withArgs(bidder1.address, usdcAddress, usdc("300"));
      await expect(tx).to.changeTokenBalances(ethers, usdcToken, [bidder1, auction], [usdc("300"), -usdc("300")]);
    });

    it("没有待提取金额时失败", async function () {
      const { auction, other } = await loadFixture(auctionFixture);
      await expect(auction.connect(other).withdraw(ETH)).to.be.revertedWithCustomError(auction, "NothingToWithdraw");
    });

    it("完整流程：所有人提现后合约余额归零", async function () {
      const { auction, auctionId, endTime, usdcToken, seller, feeRecipient, bidder1, bidder2 } =
        await loadFixture(auctionFixture);
      const auctionAddress = await auction.getAddress();
      const usdcAddress = await usdcToken.getAddress();

      await auction.connect(bidder1).bidWithToken(auctionId, usdc("1200")); // $1200
      await auction.connect(bidder2).bidWithETH(auctionId, { value: ethers.parseEther("0.5") }); // $1500
      await time.increaseTo(endTime);
      await auction.endAuction(auctionId);

      await auction.connect(bidder1).withdraw(usdcAddress);
      await expect(auction.connect(seller).withdraw(ETH)).to.changeEtherBalance(
        ethers,
        seller,
        ethers.parseEther("0.485"),
      );
      await expect(auction.connect(feeRecipient).withdraw(ETH)).to.changeEtherBalance(
        ethers,
        feeRecipient,
        ethers.parseEther("0.015"),
      );

      expect(await ethers.provider.getBalance(auctionAddress)).to.equal(0n);
      expect(await usdcToken.balanceOf(auctionAddress)).to.equal(0n);
    });
  });

  // ============================================================
  //                             安全性
  // ============================================================

  describe("安全性", function () {
    async function maliciousFixture() {
      const ctx = await auctionFixture();
      const attacker = await ethers.deployContract("MaliciousBidder", [await ctx.auction.getAddress()]);
      return { ...ctx, attacker };
    }

    it("提现时的重入攻击被 nonReentrant 拦截，攻击者只能取回一次", async function () {
      const { auction, auctionId, attacker, bidder2 } = await loadFixture(maliciousFixture);
      const amount = ethers.parseEther("0.1");
      await attacker.bid(auctionId, { value: amount });
      await auction.connect(bidder2).bidWithETH(auctionId, { value: ethers.parseEther("0.2") });
      await attacker.setMode(1); // Reenter

      await expect(attacker.withdrawETH()).to.changeEtherBalance(ethers, attacker, amount);

      expect(await attacker.receiveCount()).to.equal(1n);
      const selector = auction.interface.getError("ReentrancyGuardReentrantCall")!.selector;
      expect(await attacker.reentryError()).to.equal(selector);
      expect(await auction.pendingReturns(await attacker.getAddress(), ETH)).to.equal(0n);
    });

    it("拒收 ETH 的出价者无法阻塞他人出价（pull 模式防 DoS）", async function () {
      const { auction, auctionId, attacker, bidder2 } = await loadFixture(maliciousFixture);
      const amount = ethers.parseEther("0.1");
      await attacker.bid(auctionId, { value: amount });
      await attacker.setMode(2); // Reject

      await expect(auction.connect(bidder2).bidWithETH(auctionId, { value: ethers.parseEther("0.2") })).to.emit(
        auction,
        "BidPlaced",
      );
      // 它自己提现失败，但余额保留，不影响任何人
      await expect(attacker.withdrawETH()).to.be.revertedWithCustomError(auction, "EthTransferFailed");
      expect(await auction.pendingReturns(await attacker.getAddress(), ETH)).to.equal(amount);
    });

    it("未实现 onERC721Received 的合约赢得拍卖时，结算依然成功", async function () {
      const { auction, auctionId, endTime, attacker, nft } = await loadFixture(maliciousFixture);
      await attacker.bid(auctionId, { value: ethers.parseEther("0.1") });
      await time.increaseTo(endTime);
      await auction.endAuction(auctionId);
      expect(await nft.ownerOf(1n)).to.equal(await attacker.getAddress());
    });

    it("直接向合约转 ETH 会失败", async function () {
      const { auction, other } = await loadFixture(auctionFixture);
      await expect(other.sendTransaction({ to: await auction.getAddress(), value: 1n })).to.be.revert(ethers);
    });
  });
});
