// SPDX-License-Identifier: MIT
pragma solidity 0.8.28;

import {ERC20} from "@openzeppelin/contracts/token/ERC20/ERC20.sol";

/**
 * @title MockERC20
 * @notice 可自定义精度、任何人可铸造的测试代币（测试网上当作 USDC 使用）
 * @dev 仅用于测试，不可用于生产环境
 */
contract MockERC20 is ERC20 {
    uint8 private immutable _decimals;

    /**
     * @param name_ 代币名称
     * @param symbol_ 代币符号
     * @param decimals_ 代币精度（如 USDC 为 6）
     */
    constructor(string memory name_, string memory symbol_, uint8 decimals_) ERC20(name_, symbol_) {
        _decimals = decimals_;
    }

    /**
     * @notice 铸造代币，任何人都可以调用，便于测试和演示时领取
     * @param to 接收地址
     * @param amount 铸造数量（按代币精度）
     */
    function mint(address to, uint256 amount) external {
        _mint(to, amount);
    }

    /// @notice 返回代币精度
    function decimals() public view override returns (uint8) {
        return _decimals;
    }
}
