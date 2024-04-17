package common

import (
	"hash/fnv"
	"math/rand"
)

func Init() {
	fileInit()
}

// 删除子元素
func Remove[T comparable](slice []T, t T) ([]T, int) {
	return RemoveFor(slice, func(item T) bool {
		return item == t
	})
}

// 删除子元素, condition：自定义判断规则
func RemoveFor[T comparable](slice []T, condition func(item T) bool) ([]T, int) {
	if len(slice) == 0 {
		return slice, -1
	}

	for idx := 0; idx < len(slice); idx++ {
		if condition(slice[idx]) {
			slice = append(slice[:idx], slice[idx+1:]...)
			return slice, idx
		}
	}

	return slice, -1
}

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

func Hash(str string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(str))
	return h.Sum32()
}
