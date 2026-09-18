// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

// 投票合约：记录候选人得票数，支持投票、查询和重置
contract Voting {
    // 候选人姓名 => 得票数
    mapping(string => uint256) private votes;

    // 已投票的候选人列表，用于 resetVotes 时遍历清零
    string[] private candidates;

    // 记录候选人是否已存在于 candidates 数组中，避免重复添加
    mapping(string => bool) private candidateExists;

    // 给指定候选人投一票
    function vote(string memory candidate) public {
        if (!candidateExists[candidate]) {
            candidateExists[candidate] = true;
            candidates.push(candidate);
        }
        votes[candidate] += 1;
    }

    // 查询指定候选人的得票数
    function getVotes(string memory candidate) public view returns (uint256) {
        return votes[candidate];
    }

    // 重置所有候选人的得票数
    function resetVotes() public {
        for (uint256 i = 0; i < candidates.length; i++) {
            votes[candidates[i]] = 0;
        }
    }
}
