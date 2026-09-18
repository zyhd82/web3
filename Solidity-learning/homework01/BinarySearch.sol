// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

// 二分查找合约：在升序数组中查找目标值，返回其下标
contract BinarySearch {
    // 在升序数组 nums 中查找 target，找到返回下标，未找到返回 -1
    function search(uint256[] memory nums, uint256 target) public pure returns (int256) {
        if (nums.length == 0) {
            return -1;
        }

        int256 left = 0;
        int256 right = int256(nums.length) - 1;

        while (left <= right) {
            int256 mid = left + (right - left) / 2;
            uint256 midValue = nums[uint256(mid)];

            if (midValue == target) {
                return mid;
            } else if (midValue < target) {
                left = mid + 1;
            } else {
                right = mid - 1;
            }
        }

        return -1;
    }
}
