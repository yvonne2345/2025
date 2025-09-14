package main

import (
	"fmt"
	"strings"
)

// // 字母异位词分组,strs = ["eat", "tea", "tan", "ate", "nat", "bat"],输出[["bat"],["nat","tan"],["ate","eat","tea"]]
//
//	func groupAnagrams(strs []string) [][]string {
//		m := make(map[[26]int][]string)
//		for _, str := range strs {
//			var num [26]int
//			for _, v := range str {
//				num[v-'a']++
//			}
//			m[num] = append(m[num], str)
//		}
//
//		ans := make([][]string, 0, len(strs))
//		for _, v := range m {
//			ans = append(ans, v)
//		}
//		return ans
//	}
func main() {
	str := "{\"success\":true,\"user\":{\"loginName\":\"c_wuweidong-003\",\"map\":{\"p09HandleCode\":\"c\",\"deptName\":\"信息安全部\",\"empType\":\"OUT\",\"deptCode|deptName\":\"TB10053002|信息安全部\",\"companyName\":\"集团\",\"mainAccnoType\":\"MAIN\",\"thirdComCode|thirdComName\":\"null|null\",\"secondComCode|secondComName\":\"null|null\",\"activeDate\":\"2022/06/23 13:48:29\",\"inActiveDate\":\"2023/06/23 13:48:29\",\"p09UnitCode\":\"wuweidong-003\",\"email\":\"c_wuweidong-003@cpic.com.cn\",\"outCompanyName\":\"test\",\"realIp\":\"10.203.34.77\",\"mobileCode\":\"15012345696\",\"companyType\":\"G\",\"deptId\":\"111127679314\",\"employeeId\":\"100000030291\",\"mainType\":\"1\",\"branchComCode|branchComName\":\"null|null\",\"companyId\":\"58398\",\"icCard\":\"320405198301108758\",\"createTime\":1655963309000,\"warrantorMainAccno\":\"xiongyi\",\"deptCode\":\"TB10053002\"},\"operatorId\":\"c_wuweidong-003\",\"realName\":\"吴卫东\",\"securityLevel\":1}}"
	if strings.Contains(string(str), "loginName") {
		//split := strings.Split(string(str), "\"loginName\":")
		split := strings.Split(string(str), "\"loginName\":\"")
		//i := strings.Split(split[1], ",")
		i := strings.Split(split[1], "\"")
		fmt.Println(i[0])
	}
}
