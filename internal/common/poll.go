package common

import (
	"context"
	"errors"
	"fmt"
	"time"
)

type RollContainer[T interface{}] struct {
	slice   []T
	markers map[interface{}]byte
	mu      ExpireLock

	Condition func(T) bool
}

func NewRollContainer[T interface{}](slice []T) RollContainer[T] {
	return RollContainer[T]{
		slice:   slice,
		markers: make(map[interface{}]byte),
	}
}

func (container *RollContainer[T]) Roll() (T, error) {
	var zero T
	if container.Condition == nil {
		return zero, errors.New("condition is nil")
	}

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

func (container *RollContainer[T]) SetMarker(key interface{}, value byte) error {
	timeout, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if container.mu.Lock(timeout) {
		defer container.mu.Unlock()
		container.markers[key] = value
	} else {
		return context.DeadlineExceeded
	}
	return nil
}

func (container *RollContainer[T]) GetMarker(key interface{}) (byte, error) {
	timeout, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if container.mu.Lock(timeout) {
		defer container.mu.Unlock()
		return container.markers[key], nil
	} else {
		return 0, context.DeadlineExceeded
	}
}

func (container *RollContainer[T]) Len() int {
	return len(container.slice)
}
