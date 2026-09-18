// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

// 字符串反转合约：按字节反转输入字符串（适用于 ASCII，不支持多字节 UTF-8 字符）
contract ReverseString {
    // 反转输入字符串并返回结果，例如 "abcde" -> "edcba"
    function reverse(string memory input) public pure returns (string memory) {
        bytes memory strBytes = bytes(input);
        uint256 length = strBytes.length;
        bytes memory result = new bytes(length);

        for (uint256 i = 0; i < length; i++) {
            result[i] = strBytes[length - 1 - i];
        }

        return string(result);
    }
}
