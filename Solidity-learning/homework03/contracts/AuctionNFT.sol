// SPDX-License-Identifier: MIT
pragma solidity 0.8.28;

import {ERC721} from "@openzeppelin/contracts/token/ERC721/ERC721.sol";
import {ERC721URIStorage} from "@openzeppelin/contracts/token/ERC721/extensions/ERC721URIStorage.sol";
import {Ownable} from "@openzeppelin/contracts/access/Ownable.sol";

/**
 * @title AuctionNFT
 * @notice 拍卖市场使用的 NFT，每个 token 带独立的元数据 URI，仅 owner 可铸造
 */
contract AuctionNFT is ERC721URIStorage, Ownable {
    /// @notice 下一个待铸造的 tokenId，从 1 开始
    uint256 public nextTokenId = 1;

    /**
     * @param name_ NFT 名称
     * @param symbol_ NFT 符号
     * @param initialOwner 合约 owner（拥有铸造权限）
     */
    constructor(
        string memory name_,
        string memory symbol_,
        address initialOwner
    ) ERC721(name_, symbol_) Ownable(initialOwner) {}

    /**
     * @notice 铸造一个 NFT 并设置元数据 URI
     * @param to 接收地址
     * @param uri 元数据 URI（如 ipfs://...）
     * @return tokenId 新铸造的 tokenId
     */
    function mint(address to, string calldata uri) external onlyOwner returns (uint256 tokenId) {
        tokenId = nextTokenId++;
        _safeMint(to, tokenId);
        _setTokenURI(tokenId, uri);
    }
}
