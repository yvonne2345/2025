package main

import "fmt"

// 两数之和,nums = [2,7,11,15], target = 9,因为 nums[0] + nums[1] == 9 ，返回 [0, 1]
func twoSum(nums []int, target int) []int {
	ans := make(map[int]int)
	for index, value := range nums {
		valueIndex, ok := ans[value]
		if ok {
			return []int{valueIndex, index}
		} else {
			ans[target-value] = index
		}
	}
	return []int{}
}

func main() {
	nums := []int{2, 7, 11, 15}
	target := 9
	fmt.Println(twoSum(nums, target))
}
