import { expect } from "chai";
import { network } from "hardhat";

// 整个测试文件共用一个网络连接，loadFixture 依赖同一连接上的快照
const { ethers, networkHelpers } = await network.create();

describe("AuctionNFT", function () {
  // 部署场景：owner 部署 NFT 合约
  async function deployFixture() {
    const [owner, alice, bob] = await ethers.getSigners();
    const nft = await ethers.deployContract("AuctionNFT", ["Auction NFT", "ANFT", owner.address]);
    return { nft, owner, alice, bob };
  }

  describe("部署", function () {
    it("应正确设置名称、符号和 owner", async function () {
      const { nft, owner } = await networkHelpers.loadFixture(deployFixture);
      expect(await nft.name()).to.equal("Auction NFT");
      expect(await nft.symbol()).to.equal("ANFT");
      expect(await nft.owner()).to.equal(owner.address);
      expect(await nft.nextTokenId()).to.equal(1n);
    });

    it("owner 为零地址时部署失败", async function () {
      const factory = await ethers.getContractFactory("AuctionNFT");
      await expect(factory.deploy("Auction NFT", "ANFT", ethers.ZeroAddress))
        .to.be.revertedWithCustomError(factory, "OwnableInvalidOwner")
        .withArgs(ethers.ZeroAddress);
    });
  });

  describe("mint", function () {
    it("owner 可以铸造，tokenId 从 1 开始自增并设置 tokenURI", async function () {
      const { nft, alice, bob } = await networkHelpers.loadFixture(deployFixture);

      await expect(nft.mint(alice.address, "ipfs://token-1"))
        .to.emit(nft, "Transfer")
        .withArgs(ethers.ZeroAddress, alice.address, 1n);
      await nft.mint(bob.address, "ipfs://token-2");

      expect(await nft.ownerOf(1n)).to.equal(alice.address);
      expect(await nft.ownerOf(2n)).to.equal(bob.address);
      expect(await nft.tokenURI(1n)).to.equal("ipfs://token-1");
      expect(await nft.tokenURI(2n)).to.equal("ipfs://token-2");
      expect(await nft.nextTokenId()).to.equal(3n);
    });

    it("mint 的静态调用返回新 tokenId", async function () {
      const { nft, alice } = await networkHelpers.loadFixture(deployFixture);
      expect(await nft.mint.staticCall(alice.address, "ipfs://x")).to.equal(1n);
    });

    it("非 owner 铸造失败", async function () {
      const { nft, alice } = await networkHelpers.loadFixture(deployFixture);
      await expect(nft.connect(alice).mint(alice.address, "ipfs://x"))
        .to.be.revertedWithCustomError(nft, "OwnableUnauthorizedAccount")
        .withArgs(alice.address);
    });

    it("铸造给零地址失败", async function () {
      const { nft } = await networkHelpers.loadFixture(deployFixture);
      await expect(nft.mint(ethers.ZeroAddress, "ipfs://x"))
        .to.be.revertedWithCustomError(nft, "ERC721InvalidReceiver")
        .withArgs(ethers.ZeroAddress);
    });
  });

  describe("转移", function () {
    it("持有者可以转移 NFT", async function () {
      const { nft, alice, bob } = await networkHelpers.loadFixture(deployFixture);
      await nft.mint(alice.address, "ipfs://token-1");
      await nft.connect(alice).transferFrom(alice.address, bob.address, 1n);
      expect(await nft.ownerOf(1n)).to.equal(bob.address);
    });
  });
});
