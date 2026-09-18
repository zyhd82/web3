// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

// 合并两个有序数组合约：将两个升序数组合并为一个新的升序数组
contract MergeSortedArray {
    // 合并两个升序数组 nums1 和 nums2，返回合并后的新升序数组
    function merge(uint256[] memory nums1, uint256[] memory nums2)
        public
        pure
        returns (uint256[] memory)
    {
        uint256 m = nums1.length;
        uint256 n = nums2.length;
        uint256[] memory result = new uint256[](m + n);

        uint256 i = 0;
        uint256 j = 0;
        uint256 k = 0;

        // 双指针依次比较，取较小值放入结果数组
        while (i < m && j < n) {
            if (nums1[i] <= nums2[j]) {
                result[k++] = nums1[i++];
            } else {
                result[k++] = nums2[j++];
            }
        }

        // 处理剩余元素
        while (i < m) {
            result[k++] = nums1[i++];
        }
        while (j < n) {
            result[k++] = nums2[j++];
        }

        return result;
    }
}
