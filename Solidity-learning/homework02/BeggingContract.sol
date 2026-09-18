// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

/// @title BeggingContract - 捐赠合约
/// @notice 允许用户在指定时间段内向合约捐赠以太币，所有者可部分提取资金，并支持查询捐赠排行榜
contract BeggingContract {
    // 合约所有者（部署后不可变更，声明为 immutable 以节省每次读取的 gas）
    address public immutable owner;

    // 每个捐赠者的累计捐赠金额
    mapping(address => uint256) public donations;

    // 捐赠者地址列表，用于遍历计算排行榜（mapping 本身不可遍历）
    address[] private donors;

    // 每天允许捐赠的时间段（以 UTC 秒数表示，对应北京时间 9:00-17:00）
    // 北京时间 = UTC+8，故北京 9:00 = UTC 1:00，北京 17:00 = UTC 9:00
    uint256 private constant DONATE_START_UTC_SECONDS = 1 hours; // 1:00 UTC
    uint256 private constant DONATE_END_UTC_SECONDS = 9 hours; // 9:00 UTC

    // 捐赠事件：记录捐赠者地址和捐赠金额
    event Donation(address indexed donor, uint256 amount);

    // 提取事件：记录所有者提取的金额
    event Withdrawal(address indexed owner, uint256 amount);

    // 自定义 error：比 require 字符串更省 gas（部署字节码更小，revert 成本更低）
    error NotOwner();
    error OutsideDonationTime();
    error ZeroAmount();
    error InsufficientBalance();

    modifier onlyOwner() {
        if (msg.sender != owner) revert NotOwner();
        _;
    }

    // 限制只能在每天北京时间 9:00-17:00 内捐赠
    modifier withinDonationTime() {
        uint256 secondsOfDay = block.timestamp % 1 days;
        if (secondsOfDay < DONATE_START_UTC_SECONDS || secondsOfDay >= DONATE_END_UTC_SECONDS) {
            revert OutsideDonationTime();
        }
        _;
    }

    constructor() {
        owner = msg.sender;
    }

    /// @notice 捐赠函数，用户发送以太币即可完成捐赠
    function donate() external payable withinDonationTime {
        if (msg.value == 0) revert ZeroAmount();

        // 缓存已捐赠金额，避免重复读取 storage（原写法会因 `==0` 判断和 `+=` 各读一次）
        uint256 existingAmount = donations[msg.sender];

        // 首次捐赠的地址记录到 donors 数组，避免重复添加
        if (existingAmount == 0) {
            donors.push(msg.sender);
        }

        donations[msg.sender] = existingAmount + msg.value;

        emit Donation(msg.sender, msg.value);
    }

    /// @notice 合约所有者提取资金，支持指定金额部分提取
    /// @param amount 提取金额（单位：wei）
    function withdraw(uint256 amount) external onlyOwner {
        if (amount == 0) revert ZeroAmount();
        if (amount > address(this).balance) revert InsufficientBalance();

        payable(owner).transfer(amount);

        emit Withdrawal(owner, amount);
    }

    /// @notice 查询某个地址的捐赠金额
    function getDonation(address donor) external view returns (uint256) {
        return donations[donor];
    }

    /// @notice 查询捐赠金额最多的前 3 个地址（遍历计算，不在链上维护榜单）
    /// @return topDonors 前 3 名捐赠者地址（不足 3 人时后面的地址为零地址）
    /// @return topAmounts 对应的捐赠金额
    function getTopDonors() external view returns (address[3] memory topDonors, uint256[3] memory topAmounts) {
        for (uint256 i = 0; i < donors.length; i++) {
            address donor = donors[i];
            uint256 amount = donations[donor];

            for (uint256 j = 0; j < 3; j++) {
                if (amount > topAmounts[j]) {
                    // 将当前位置及之后的名次依次后移，插入新的记录
                    for (uint256 k = 2; k > j; k--) {
                        topAmounts[k] = topAmounts[k - 1];
                        topDonors[k] = topDonors[k - 1];
                    }
                    topAmounts[j] = amount;
                    topDonors[j] = donor;
                    break;
                }
            }
        }
    }
}
