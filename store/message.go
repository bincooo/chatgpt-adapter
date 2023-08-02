package store

import (
	"sync"
)

var (
	mu           sync.RWMutex
	messageStore = make(map[string][]Kv)
)

type Kv = map[string]string

func CacheMessages(uid string, messages []Kv) {
	mu.Lock()
	defer mu.Unlock()
	messageStore[uid] = messages
}

func DeleteMessages(uid string) {
	mu.Lock()
	defer mu.Unlock()
	delete(messageStore, uid)
}

func GetMessages(uid string) []Kv {
	if result, ok := messageStore[uid]; ok {
		return result
	}
	return make([]Kv, 0)
}

func AddMessage(uid string, messages Kv) {
	CacheMessages(uid, append(GetMessages(uid), messages))
}

func AddMessages(uid string, messages []Kv) {
	CacheMessages(uid, append(GetMessages(uid), messages...))
}
