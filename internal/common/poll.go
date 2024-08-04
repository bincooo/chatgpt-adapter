package common

import (
	"chatgpt-adapter/logger"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"time"
)

type state struct {
	t time.Time
	s byte
}

type PollContainer[T interface{}] struct {
	name      string
	pos       int
	slice     []T
	markers   map[interface{}]*state
	mu        ExpireLock
	cmu       ExpireLock
	Condition func(T) bool
}

// resetTime 用于复位状态：0 就绪状态，1 使用状态，2 异常状态
func NewPollContainer[T interface{}](name string, slice []T, resetTime time.Duration) *PollContainer[T] {
	container := PollContainer[T]{
		name:    name,
		slice:   slice,
		markers: make(map[interface{}]*state),
	}

	if resetTime > 0 {
		go timer(&container, resetTime)
	}
	return &container
}

// 定时复位状态 0 就绪状态，1 使用状态，2 异常状态
func timer[T interface{}](container *PollContainer[T], resetTime time.Duration) {
	s10 := 10 * time.Second
	for {
		if len(container.slice) == 0 {
			time.Sleep(s10)
			continue
		}

		timeout, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		if !container.mu.Lock(timeout) {
			cancel()
			time.Sleep(s10)
			logger.Errorf("[%s] PollContainer 获取锁失败", container.name)
			continue
		}
		cancel()

		for _, value := range container.slice {
			var obj interface{} = value
			if s, ok := obj.(string); ok {
				obj = s
			} else {
				data, _ := json.Marshal(obj)
				obj = string(data)
			}

			marker, ok := container.markers[obj]
			if !ok {
				continue
			}

			if marker.s == 0 { // 就绪状态
				continue
			}

			// 1 使用中 2 异常冷却中
			if time.Now().Add(-resetTime).After(marker.t) {
				marker.s = 0
				logger.Infof("[%s] PollContainer 冷却完毕: %v", container.name, obj)
			}
		}
		container.mu.Unlock()
		time.Sleep(s10)
	}
}

func (container *PollContainer[T]) Poll() (T, error) {
	var zero T
	if container == nil || len(container.slice) == 0 {
		return zero, errors.New("no elements in slice")
	}

	if container.Condition == nil {
		return zero, errors.New("condition is nil")
	}

	timeout, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if !container.cmu.Lock(timeout) {
		return zero, errors.New("lock timeout")
	}
	defer container.cmu.Unlock()

	pos := container.pos
	sliceL := len(container.slice)
	if pos >= sliceL {
		container.pos = 0
		pos = 0
	}

	for index := 0; index < sliceL; index++ {
		curr := pos + index
		if curr >= sliceL {
			curr = curr - sliceL
		}

		value := container.slice[curr]
		if container.Condition(value) {
			container.pos = curr + 1
			err := container.SetMarker(value, 1)
			if err != nil {
				return zero, err
			}
			return value, nil
		}
	}

	return zero, fmt.Errorf("not roll result")
}

func (container *PollContainer[T]) Del(value T) {
	if container.Len() == 0 {
		return
	}

	for idx := 0; idx < len(container.slice); idx++ {
		if reflect.DeepEqual(container.slice[idx], value) {
			container.slice = append(container.slice[:idx], container.slice[idx+1:]...)
			return
		}
	}
}

func (container *PollContainer[T]) Add(value T) {
	container.slice = append(container.slice, value)
}

// 标记： 0 就绪状态，1 使用状态，2 异常状态
func (container *PollContainer[T]) SetMarker(key interface{}, value byte) error {
	if s, ok := key.(string); ok {
		key = s
	} else {
		data, _ := json.Marshal(key)
		key = string(data)
	}

	timeout, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if container.mu.Lock(timeout) {
		defer container.mu.Unlock()
		container.markers[key] = &state{
			t: time.Now(),
			s: value,
		}
		logger.Infof("[%s] 设置状态值：%d", container.name, value)
	} else {
		return context.DeadlineExceeded
	}
	return nil
}

func (container *PollContainer[T]) GetMarker(key interface{}) (byte, error) {
	if s, ok := key.(string); ok {
		key = s
	} else {
		data, _ := json.Marshal(key)
		key = string(data)
	}

	timeout, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if container.mu.Lock(timeout) {
		defer container.mu.Unlock()
		marker, ok := container.markers[key]
		if !ok {
			return 0, nil
		}
		return marker.s, nil
	} else {
		return 0, context.DeadlineExceeded
	}
}

func (container *PollContainer[T]) Len() int {
	return len(container.slice)
}
