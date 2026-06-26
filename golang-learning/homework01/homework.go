package homework01

import "sort"

// 1. 只出现一次的数字
// 给定一个非空整数数组，除了某个元素只出现一次以外，其余每个元素均出现两次。找出那个只出现了一次的元素。
func SingleNumber(nums []int) int {
	if len(nums) == 0 {
		return 0
	}

	var result int = 0
	var numMap map[int]int
	numMap = make(map[int]int)
	// 1、循环遍历数组
	for i := 0; i < len(nums); i++ {
		// 2、将每个item值和次数记录到map中
		numMap[nums[i]] = numMap[nums[i]] + 1
	}

	// 3、从map中取出次数为2的item，返回它的数值
	for num := range numMap {
		if numMap[num] == 1 {
			return num
		}
	}

	return result
}

// 2. 回文数
// 判断一个整数是否是回文数
func IsPalindrome(x int) bool {
	// 1、排除肯定不是回文数的数值
	if x < 0 || (x%10 == 0 && x != 0) {
		return false
	}

	// 2、利用取模%算法反转数值
	revertedNumber := 0
	for x > revertedNumber {
		revertedNumber = revertedNumber*10 + x%10
		x /= 10 // 去掉最后一次多加的0
	}

	// 3、当x这个数值的长度是奇数时，通过上面的for循环反转结果中revertedNumber会比x多一位，
	// 比如12312，反转后x=12，revertedNumber=123循环就结束了，此时x是不等于revertedNumber的
	return x == revertedNumber || x == revertedNumber/10
}

// 3. 有效的括号
// 给定一个只包括 '(', ')', '{', '}', '[', ']' 的字符串，判断字符串是否有效
func IsValid(s string) bool {
	var stack string // 直接用空字符串作为栈

	pairs := map[byte]byte{
		')': '(',
		']': '[',
		'}': '{',
	}

	for i := 0; i < len(s); i++ {
		char := s[i]

		// 如果是右括号
		if left, ok := pairs[char]; ok {
			// 栈为空，或者栈顶元素不匹配，直接返回 false
			if len(stack) == 0 || stack[len(stack)-1] != left {
				return false
			}
			// 匹配成功，通过切片“弹出”栈顶元素
			stack = stack[:len(stack)-1]
		} else {
			// 是左括号，拼接到字符串末尾（入栈）
			stack += string(char)
		}
	}

	// 遍历结束后，栈必须为空才说明完全匹配
	return len(stack) == 0
}

// 4. 最长公共前缀
// 查找字符串数组中的最长公共前缀
func LongestCommonPrefix(strs []string) string {
	if strs == nil || len(strs) == 0 {
		return ""
	}

	// 1、取第一个字符串作为外层比较对象，逐个字符与剩下所有的字符串字符比较
	for i := 0; i < len(strs[0]); i++ {
		char := strs[0][i]

		for j := 1; j < len(strs); j++ {
			// 如果当前字符串长度不足，或者字符不匹配,则返回从开头到当前位置(不包含)的子串
			if i >= len(strs[j]) || strs[j][i] != char {
				return strs[0][:i]
			}
		}
	}

	// 2、如果循环全部结束，则说明当前字符串就是最短公共前缀，直接返回当前字符串
	return strs[0]
}

// 5. 加一
// 给定一个由整数组成的非空数组所表示的非负整数，在该数的基础上加一
func PlusOne(digits []int) []int {
	// 从数组的末尾开始循环，向前遍历，注意处理全是9的特殊情况
	for i := len(digits) - 1; i >= 0; i-- {
		// 如果取出的数组项数值小于9，则直接加一就可以退出了
		// 因为小于9的数值都会加一，所以即便前一轮是9，正好也处理了进位问题，此处不需要再额外考虑进位
		if digits[i] < 9 {
			digits[i]++
			return digits
		}

		// 没有退出，就说明当前是9，则将当前位记为0即可,然后继续循环
		digits[i] = 0
	}

	//如果全部循环完都没有退出，说明遇到了全是9的特殊情况，此时需要将数组长度加一位并且第一位数值赋值为1，其余保持默认0
	result := make([]int, len(digits)+1)
	result[0] = 1
	return result
}

// 6. 删除有序数组中的重复项
// 给你一个有序数组 nums ，请你原地删除重复出现的元素，使每个元素只出现一次，返回删除后数组的新长度。
// 不要使用额外的数组空间，你必须在原地修改输入数组并在使用 O(1) 额外空间的条件下完成。
func RemoveDuplicates(nums []int) int {
	if len(nums) == 0 {
		return 0
	}

	currentIndex := 0 // currentIndex下标表示最后一个不重复元素的位置

	// 此处循环从1开始，因为第一个元素必定是还没重复的元素
	for i := 1; i < len(nums); i++ {
		// 当第i个元素不等于最后一个不重复的元素时，把这个元素放到currentIndex元素的后面，成为新的最后一个不重复元素位置
		if nums[i] != nums[currentIndex] {
			currentIndex++
			nums[currentIndex] = nums[i]
		}
	}

	// 新的不重复数组的长度为currentIndex + 1，因为数组下标从0开始
	return currentIndex + 1
}

// 7. 合并区间
// 以数组 intervals 表示若干个区间的集合，其中单个区间为 intervals[i] = [starti, endi] 。
// 请你合并所有重叠的区间，并返回一个不重叠的区间数组，该数组需恰好覆盖输入中的所有区间。
func Merge(intervals [][]int) [][]int {
	// 如果区间数组为0或者只有一个，则直接返回
	if len(intervals) <= 1 {
		return intervals
	}

	// 1、按照区间的起始位置进行升序排序
	sort.Slice(intervals, func(i, j int) bool {
		return intervals[i][0] < intervals[j][0]
	})

	// 2、使用一个切片来存储合并后的区间，初始放入第一个区间
	merged := [][]int{intervals[0]}

	// 3、遍历排序后的区间数组（从第二个开始）
	for i := 1; i < len(intervals); i++ {
		current := intervals[i]
		last := merged[len(merged)-1] // 获取切片中最后一个区间

		// 判断是否有重叠：当前区间的起始位置 <= 最后一个区间的结束位置
		if current[0] <= last[1] {
			// 有重叠，合并区间：更新最后一个区间的结束位置为两者的最大值
			if current[1] > last[1] {
				last[1] = current[1]
			}
		} else {
			// 没有重叠，将当前区间添加到切片中
			merged = append(merged, current)
		}
	}

	return merged
}

// 8. 两数之和
// 给定一个整数数组 nums 和一个目标值 target，请你在该数组中找出和为目标值的那两个整数
func TwoSum(nums []int, target int) []int {
	// 创建一个 map，key 存储数组的值，value 存储对应的索引
	numMap := make(map[int]int)

	// 通过循环遍历查找和为目标值的两个整数项，返回它们的索引位置
	for i, num := range nums {
		// 计算当前数字需要的“互补数”
		complement := target - num

		// 在 map 中查找互补数是否已经存在
		if idx, ok := numMap[complement]; ok {
			// 如果存在，直接返回这两个数的索引
			return []int{idx, i}
		}

		// 如果不存在，将当前数字及其索引存入 map，供后续数字查找
		numMap[num] = i
	}

	return nil
}
