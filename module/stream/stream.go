package stream

import "reflect"

const buffer = 10

type Stream[T any] struct {
	ch   chan T
	size int
}

type Pair[A, B any] struct {
	Val1 A
	Val2 B
}

func (s Stream[T]) close() {
	close(s.ch)
}

func (s Stream[T]) ToSlice() []T {
	var out []T
	for t := range s.ch {
		out = append(out, t)
	}
	return out
}

func (s Stream[T]) Filter(f func(T) bool) Stream[T] {
	out := of[T](s.size)
	go func() {
		defer out.close()
		for t := range s.ch {
			if f(t) {
				out.ch <- t
			}
		}
	}()
	return out
}

func (s Stream[T]) Range(f func(T)) {
	for t := range s.ch {
		f(t)
	}
}

func (s Stream[T]) RangeErr(f func(T) error) error {
	var err error
	for t := range s.ch {
		if err == nil {
			err = f(t)
		}
	}
	return err
}

func (s Stream[T]) One() T {
	var out T
	for t := range s.ch {
		if !NotNil[T]()(out) {
			out = t
		}
	}
	return out
}

func Map[T, E any](s Stream[T], f func(T) E) Stream[E] {
	return processing(s, func(t T, s Stream[E]) {
		s.ch <- f(t)
	})
}

func MapPair[T, E any](s Stream[T], f func(T) E) Stream[Pair[T, E]] {
	return processing(s, func(t T, s Stream[Pair[T, E]]) {
		s.ch <- Pair[T, E]{Val1: t, Val2: f(t)}
	})
}

func FlatMap[T, E any](s Stream[T], f func(T) []E) Stream[E] {
	return processing(s, func(t T, s Stream[E]) {
		for _, v := range f(t) {
			s.ch <- v
		}
	})
}

func processing[T, R any](in Stream[T], fn func(T, Stream[R])) Stream[R] {
	out := of[R](in.size)
	go func() {
		defer out.close()
		for t := range in.ch {
			fn(t, out)
		}
	}()
	return out
}

func OfSlice[T any](v []T) Stream[T] {
	s := of[T](len(v))
	if len(v) != 0 {
		go func() {
			defer s.close()
			for _, t := range v {
				s.ch <- t
			}
		}()
	} else {
		s.close()
	}
	return s
}

func OfMap[K comparable, V any](m map[K]V) Stream[Pair[K, V]] {
	s := of[Pair[K, V]](len(m))
	if len(m) != 0 {
		go func() {
			defer s.close()
			for k, v := range m {
				s.ch <- Pair[K, V]{Val1: k, Val2: v}
			}
		}()
	} else {
		s.close()
	}
	return s
}

func of[T any](l int) Stream[T] {
	return Stream[T]{
		ch:   make(chan T, l),
		size: l,
	}
}

func (s Stream[T]) FlatMap(f func(T) []T) Stream[T] {
	return FlatMap(s, f)
}

func (s Stream[T]) Map(f func(T) T) Stream[T] {
	return Map(s, f)
}

func NotNil[T any]() func(v T) bool {
	return func(v T) bool {
		return !reflect.ValueOf(&v).Elem().IsZero()
	}
}
