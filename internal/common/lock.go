package common

import (
	"context"
	"sync"
	"time"
)

type ExpireLock struct {
	// 计数
	count int
	// 核心锁
	mutex sync.Mutex
}

func NewExpireLock() *ExpireLock {
	return &ExpireLock{
		count: 0,
	}
}

// Lock 加锁
func (e *ExpireLock) Lock(ctx context.Context) bool {
	timeout, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	e.count++
	for {
		select {
		case <-timeout.Done():
			// 如果上下文超时，返回false
			e.count--
			return false
		default:
			// 尝试获取锁
			if e.mutex.TryLock() {
				// 如果成功获取到锁，返回true
				return true
			}
			// 如果没有获取到锁，等待一段时间后重试
			time.Sleep(100 * time.Millisecond)
		}
	}
}

func (e *ExpireLock) Unlock() {
	e.count--
	e.mutex.Unlock()
}

func (e *ExpireLock) IsIdle() bool {
	return e.count < 1
}
