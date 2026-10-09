// SPDX-License-Identifier: MIT
pragma solidity 0.8.28;

import {NFTAuction} from "../NFTAuction.sol";

/**
 * @title MaliciousBidder
 * @notice 模拟恶意出价者，仅用于测试拍卖合约的安全性
 * @dev 两种攻击模式：
 *      - Reenter：收到 ETH 时尝试重入 withdraw，记录重入失败的原因
 *      - Reject：拒收 ETH，验证 pull 模式下它无法阻塞其他人出价
 */
contract MaliciousBidder {
    /// @notice 攻击模式
    enum Mode {
        None,
        Reenter,
        Reject
    }

    /// @notice 被攻击的拍卖合约
    NFTAuction public immutable auction;
    /// @notice 当前攻击模式
    Mode public mode;
    /// @notice 重入调用失败时返回的错误数据
    bytes public reentryError;
    /// @notice receive 被触发的次数
    uint256 public receiveCount;

    /// @param auction_ 拍卖合约（代理）地址
    constructor(NFTAuction auction_) {
        auction = auction_;
    }

    /// @notice 设置攻击模式
    function setMode(Mode mode_) external {
        mode = mode_;
    }

    /// @notice 以本合约身份用 ETH 出价
    function bid(uint256 auctionId) external payable {
        auction.bidWithETH{value: msg.value}(auctionId);
    }

    /// @notice 以本合约身份提取 ETH
    function withdrawETH() external {
        auction.withdraw(address(0));
    }

    /// @notice 收到 ETH 时按模式执行攻击
    receive() external payable {
        receiveCount++;
        if (mode == Mode.Reject) {
            revert("MaliciousBidder: reject ETH");
        }
        if (mode == Mode.Reenter) {
            // 尝试重入：捕获错误，让外层提现本身成功，便于断言重入被拦截
            try auction.withdraw(address(0)) {} catch (bytes memory reason) {
                reentryError = reason;
            }
        }
    }
}
