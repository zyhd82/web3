// SPDX-License-Identifier: MIT
pragma solidity ^0.8.28;

/// @title 计数器合约
/// @notice 用于演示 abigen 生成 Go 绑定代码并与合约交互
contract Counter {
    /// 当前计数值
    uint256 private count;

    /// 合约部署者，只有其可以重置计数
    address public owner;

    /// 计数变化事件：记录调用者和变化后的值
    event CountChanged(address indexed caller, uint256 newCount);

    error NotOwner();

    /// @param initialCount 初始计数值
    constructor(uint256 initialCount) {
        count = initialCount;
        owner = msg.sender;
    }

    /// 计数加 1
    function increment() external {
        count += 1;
        emit CountChanged(msg.sender, count);
    }

    /// 计数增加指定值
    function add(uint256 delta) external {
        count += delta;
        emit CountChanged(msg.sender, count);
    }

    /// 重置计数为 0，仅部署者可调用
    function reset() external {
        if (msg.sender != owner) revert NotOwner();
        count = 0;
        emit CountChanged(msg.sender, count);
    }

    /// 读取当前计数
    function getCount() external view returns (uint256) {
        return count;
    }
}
