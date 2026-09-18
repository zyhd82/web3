// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

// 整数转罗马数字合约：将 1~3999 之间的整数转换为罗马数字字符串
contract IntToRoman {
    // 按数值从大到小排列，贪心匹配对应罗马符号
    uint16[13] private values = [
        1000, 900, 500, 400, 100, 90, 50, 40, 10, 9, 5, 4, 1
    ];
    string[13] private symbols = [
        "M", "CM", "D", "CD", "C", "XC", "L", "XL", "X", "IX", "V", "IV", "I"
    ];

    // 将整数转换为罗马数字，num 需在 [1, 3999] 范围内
    function intToRoman(uint256 num) public view returns (string memory) {
        require(num >= 1 && num <= 3999, "num must be in [1, 3999]");

        bytes memory result;
        for (uint256 i = 0; i < values.length; i++) {
            while (num >= values[i]) {
                num -= values[i];
                result = abi.encodePacked(result, symbols[i]);
            }
        }

        return string(result);
    }
}
