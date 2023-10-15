package store

import (
	"sync"
)

var (
	mu           sync.RWMutex
	messageStore = make(map[string][]Kv)
)

type Kv = map[string]string

// 缓存消息
func CacheMessages(uid string, messages []Kv) {
	mu.Lock()
	defer mu.Unlock()
	messageStore[uid] = messages
}

// 删除消息
func DeleteMessages(uid string) {
	mu.Lock()
	defer mu.Unlock()
	delete(messageStore, uid)
}

// 删除指定消息Id的内容
func DeleteMessageFor(uid, messageId string) {
	mu.Lock()
	defer mu.Unlock()
	messages := GetMessages(uid)
	for i, message := range messages {
		if id, ok := message["id"]; ok && id == messageId {
			messages = append(messages[:i], messages[i+1:]...)
		}
	}
}

// 获取消息
func GetMessages(uid string) []Kv {
	if result, ok := messageStore[uid]; ok {
		return result
	}
	return make([]Kv, 0)
}

// 添加消息内容
func AddMessage(uid string, messages Kv) {
	CacheMessages(uid, append(GetMessages(uid), messages))
}

// 添加多个消息内容
func AddMessages(uid string, messages []Kv) {
	CacheMessages(uid, append(GetMessages(uid), messages...))
}
