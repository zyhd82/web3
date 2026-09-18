// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

// 罗马数字转整数合约：将罗马数字字符串转换为对应的整数
contract RomanToInt {
    // 将罗马数字字符串转换为整数，例如 "IX" -> 9
    function romanToInt(string memory s) public pure returns (uint256) {
        bytes memory b = bytes(s);
        uint256 total = 0;

        for (uint256 i = 0; i < b.length; i++) {
            uint256 current = charToValue(b[i]);

            // 当前符号小于下一个符号时，说明是特殊组合（如 IV、IX），需要相减
            if (i + 1 < b.length && current < charToValue(b[i + 1])) {
                total -= current;
            } else {
                total += current;
            }
        }

        return total;
    }

    // 单个罗马字符对应的数值
    function charToValue(bytes1 c) private pure returns (uint256) {
        if (c == "I") return 1;
        if (c == "V") return 5;
        if (c == "X") return 10;
        if (c == "L") return 50;
        if (c == "C") return 100;
        if (c == "D") return 500;
        if (c == "M") return 1000;
        revert("invalid roman character");
    }
}
