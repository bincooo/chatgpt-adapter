package common

import "math/rand"

// 判断切片是否包含子元素
func Contains[T comparable](slice []T, t T) bool {
	return ContainFor(slice, func(item T) bool {
		return item == t
	})
}

// 判断切片是否包含子元素， condition：自定义判断规则
func ContainFor[T comparable](slice []T, condition func(item T) bool) bool {
	if len(slice) == 0 {
		return false
	}

	for idx := 0; idx < len(slice); idx++ {
		if condition(slice[idx]) {
			return true
		}
	}
	return false
}

func RandStr(n int) string {
	var runes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890")
	bytes := make([]rune, n)
	for i := range bytes {
		bytes[i] = runes[rand.Intn(len(runes))]
	}
	return string(bytes)
}

// int 取绝对值
func Abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
