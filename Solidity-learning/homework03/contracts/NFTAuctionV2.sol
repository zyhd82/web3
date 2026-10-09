// SPDX-License-Identifier: MIT
pragma solidity 0.8.28;

import {NFTAuction} from "./NFTAuction.sol";

/**
 * @title NFTAuctionV2
 * @notice NFT 拍卖市场 V2：在 V1 基础上新增“防狙击延时”——拍卖最后一段时间内有人出价，自动延长结束时间
 * @dev 通过代理的 upgradeToAndCall 升级，并在同一交易中调用 initializeV2。
 *      存储布局：只在 V1 全部状态变量（含 __gap）之后追加新变量，不改动 V1
 */
contract NFTAuctionV2 is NFTAuction {
    /// @notice 防狙击参数上限（窗口和延长时长均不超过 1 小时）
    uint256 public constant MAX_ANTI_SNIPING = 1 hours;

    // ============================================================
    //            V2 新增状态变量（追加在 V1 存储之后）
    // ============================================================

    /// @notice 触发窗口：距结束不足该时长时出价会触发延时（0 表示关闭防狙击）
    uint256 public antiSnipingWindow;
    /// @notice 每次触发延长的时长
    uint256 public antiSnipingExtension;

    // ============================================================
    //                         事件与错误
    // ============================================================

    /// @notice 防狙击参数变更
    event AntiSnipingUpdated(uint256 window, uint256 extension);
    /// @notice 拍卖因防狙击被延长
    event AuctionExtended(uint256 indexed auctionId, uint64 newEndTime);

    /// @notice 防狙击参数不合法
    error InvalidAntiSnipingConfig();

    // ============================================================
    //                          初始化
    // ============================================================

    /**
     * @notice V2 初始化，只能调用一次（升级时通过 upgradeToAndCall 一并调用）
     * @dev 加 onlyOwner：防止升级与初始化分两笔交易执行时被他人抢先调用
     * @param window 触发窗口（秒）
     * @param extension 每次延长时长（秒）
     */
    function initializeV2(uint256 window, uint256 extension) external onlyOwner reinitializer(2) {
        _setAntiSniping(window, extension);
    }

    /**
     * @notice 合约版本号
     * @return 版本号（V2 为 2）
     */
    function version() external pure override returns (uint256) {
        return 2;
    }

    // ============================================================
    //                         管理员配置
    // ============================================================

    /**
     * @notice 调整防狙击参数
     * @param window 触发窗口（秒），0 表示关闭
     * @param extension 每次延长时长（秒）
     */
    function setAntiSniping(uint256 window, uint256 extension) external onlyOwner {
        _setAntiSniping(window, extension);
    }

    // ============================================================
    //                          内部函数
    // ============================================================

    /**
     * @dev 在 V1 出价逻辑之后追加防狙击：剩余时间小于窗口时，结束时间顺延 antiSnipingExtension
     * @param auctionId 拍卖 ID
     * @param token 出价币种
     * @param amount 出价金额
     */
    function _placeBid(uint256 auctionId, address token, uint256 amount) internal override {
        super._placeBid(auctionId, token, amount);

        // 父函数已保证 block.timestamp < endTime，下面的减法不会下溢
        Auction storage auction = _getAuction(auctionId);
        if (auction.endTime - block.timestamp < antiSnipingWindow) {
            uint64 newEndTime = auction.endTime + uint64(antiSnipingExtension);
            auction.endTime = newEndTime;
            emit AuctionExtended(auctionId, newEndTime);
        }
    }

    /**
     * @dev 校验并写入防狙击参数：均不超过 MAX_ANTI_SNIPING；开启时（window > 0）延长时长必须大于 0
     */
    function _setAntiSniping(uint256 window, uint256 extension) private {
        if (window > MAX_ANTI_SNIPING || extension > MAX_ANTI_SNIPING || (window > 0 && extension == 0)) {
            revert InvalidAntiSnipingConfig();
        }
        antiSnipingWindow = window;
        antiSnipingExtension = extension;
        emit AntiSnipingUpdated(window, extension);
    }
}
