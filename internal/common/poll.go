package common

import (
	"context"
	"errors"
	"fmt"
	"time"
)

type state struct {
	t time.Time
	s byte
}

type PollContainer[T interface{}] struct {
	slice     []T
	markers   map[interface{}]*state
	mu        ExpireLock
	cmu       ExpireLock
	Condition func(T) bool
}

// resetTime 用于复位状态：0 就绪状态，1 使用状态，2 异常状态
func NewPollContainer[T interface{}](slice []T, resetTime time.Duration) *PollContainer[T] {
	container := PollContainer[T]{
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
			return
		}

		timeout, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		if !container.mu.Lock(timeout) {
			cancel()
			time.Sleep(s10)
			continue
		}
		cancel()

		for _, value := range container.slice {
			marker, ok := container.markers[value]
			if !ok {
				continue
			}

			if marker.s == 0 { // 就绪状态
				continue
			}

			// 1 使用中 2 异常冷却中
			if time.Now().Add(-resetTime).After(marker.t) {
				marker.s = 0
			}
		}
		container.mu.Unlock()
		time.Sleep(s10)
	}
}

func (container *PollContainer[T]) Poll() (T, error) {
	var zero T
	if len(container.slice) == 0 {
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

	for _, value := range container.slice {
		if container.Condition(value) {
			err := container.SetMarker(value, 1)
			if err != nil {
				return zero, err
			}
			return value, nil
		}
	}

	return zero, fmt.Errorf("not roll result")
}

// 标记： 0 就绪状态，1 使用状态，2 异常状态
func (container *PollContainer[T]) SetMarker(key interface{}, value byte) error {
	timeout, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if container.mu.Lock(timeout) {
		defer container.mu.Unlock()
		container.markers[key] = &state{
			t: time.Now(),
			s: value,
		}
	} else {
		return context.DeadlineExceeded
	}
	return nil
}

func (container *PollContainer[T]) GetMarker(key interface{}) (byte, error) {
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
