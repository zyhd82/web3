// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

/**
 * @title AggregatorV3Interface
 * @notice Chainlink 价格喂价（Data Feeds）官方接口
 * @dev 原样摘自 @chainlink/contracts v1.5.0：src/v0.8/shared/interfaces/AggregatorV3Interface.sol
 *      未直接安装该 npm 包，原因是它会连带引入多个旧版 OpenZeppelin 及 git 依赖，而本项目只需这一个接口。
 */
interface AggregatorV3Interface {
  function decimals() external view returns (uint8);

  function description() external view returns (string memory);

  function version() external view returns (uint256);

  function getRoundData(
    uint80 _roundId
  ) external view returns (uint80 roundId, int256 answer, uint256 startedAt, uint256 updatedAt, uint80 answeredInRound);

  function latestRoundData()
    external
    view
    returns (uint80 roundId, int256 answer, uint256 startedAt, uint256 updatedAt, uint80 answeredInRound);
}
