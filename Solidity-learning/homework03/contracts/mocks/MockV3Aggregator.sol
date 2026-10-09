// SPDX-License-Identifier: MIT
pragma solidity 0.8.28;

import {AggregatorV3Interface} from "../interfaces/AggregatorV3Interface.sol";

/**
 * @title MockV3Aggregator
 * @notice 模拟 Chainlink 价格喂价，仅用于本地测试
 * @dev 参照 Chainlink 官方 MockV3Aggregator 实现，可手动设置价格与更新时间（用于测试喂价过期）
 */
contract MockV3Aggregator is AggregatorV3Interface {
    /// @notice 当前 Mock 的接口版本号
    uint256 public constant override version = 4;

    /// @notice 价格精度（Chainlink 美元喂价通常为 8 位）
    uint8 public override decimals;

    /// @notice 最新价格
    int256 public latestAnswer;
    /// @notice 最新价格的更新时间
    uint256 public latestTimestamp;
    /// @notice 最新轮次 ID
    uint256 public latestRound;

    mapping(uint256 => int256) private _answers;
    mapping(uint256 => uint256) private _timestamps;
    mapping(uint256 => uint256) private _startedAts;

    /**
     * @param decimals_ 价格精度
     * @param initialAnswer 初始价格
     */
    constructor(uint8 decimals_, int256 initialAnswer) {
        decimals = decimals_;
        updateAnswer(initialAnswer);
    }

    /**
     * @notice 更新价格，更新时间取当前区块时间
     * @param answer 新价格
     */
    function updateAnswer(int256 answer) public {
        latestAnswer = answer;
        latestTimestamp = block.timestamp;
        latestRound++;
        _answers[latestRound] = answer;
        _timestamps[latestRound] = block.timestamp;
        _startedAts[latestRound] = block.timestamp;
    }

    /**
     * @notice 以自定义轮次与时间更新价格，用于构造过期喂价等场景
     * @param roundId 轮次 ID
     * @param answer 价格
     * @param updatedAt 更新时间
     * @param startedAt 轮次开始时间
     */
    function updateRoundData(uint80 roundId, int256 answer, uint256 updatedAt, uint256 startedAt) external {
        latestRound = roundId;
        latestAnswer = answer;
        latestTimestamp = updatedAt;
        _answers[roundId] = answer;
        _timestamps[roundId] = updatedAt;
        _startedAts[roundId] = startedAt;
    }

    /// @inheritdoc AggregatorV3Interface
    function getRoundData(
        uint80 roundId
    ) external view override returns (uint80, int256, uint256, uint256, uint80) {
        return (roundId, _answers[roundId], _startedAts[roundId], _timestamps[roundId], roundId);
    }

    /// @inheritdoc AggregatorV3Interface
    function latestRoundData() external view override returns (uint80, int256, uint256, uint256, uint80) {
        uint80 round = uint80(latestRound);
        return (round, _answers[latestRound], _startedAts[latestRound], _timestamps[latestRound], round);
    }

    /// @inheritdoc AggregatorV3Interface
    function description() external pure override returns (string memory) {
        return "MockV3Aggregator";
    }
}
