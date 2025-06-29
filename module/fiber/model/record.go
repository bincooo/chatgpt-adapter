package model

import (
	"encoding/json"
	"iter"
	"maps"
	"reflect"
)

type Record[Key comparable, Value any] map[Key]Value

func (rec Record[Key, Value]) Put(k Key, v Value) {
	rec[k] = v
}

func (rec Record[Key, Value]) Get(k Key) Value {
	return rec[k]
}

func (rec Record[Key, Value]) Del(k Key) {
	delete(rec, k)
}

func (rec Record[Key, Value]) Len() int {
	return len(rec)
}

func (rec Record[Key, Value]) Keys() iter.Seq[Key] {
	return maps.Keys(rec)
}

func (rec Record[Key, Value]) Values() iter.Seq[Value] {
	return maps.Values(rec)
}

func (rec Record[Key, Value]) Each(yield func(Key, Value)) {
	for k, v := range rec {
		yield(k, v)
	}
}
func (rec Record[Key, Value]) Contains(k Key) bool {
	value := rec[k]
	return value != nil
}

func (rec Record[Key, Value]) Clone() Record[Key, Value] {
	return maps.Clone(rec)
}

func (rec Record[Key, Value]) String() string {
	chunk, err := json.MarshalIndent(rec, "", "  ")
	if err != nil {
		panic(err)
	}
	return string(chunk)
}

func (rec Record[Key, Value]) ValueEqual(k Key, v Value) (ok bool) {
	if !rec.Contains(k) {
		return
	}

	return reflect.DeepEqual(v, rec.Get(k))
}

func (rec Record[Key, Value]) ValueContains(k Key, values ...Value) (ok bool) {
	if !rec.Contains(k) {
		return
	}

	for _, value := range values {
		if rec.ValueEqual(k, value) {
			return true
		}
	}
	return
}

func GetValue[Key comparable, Value any](rec Record[Key, any], k Key) (value Value, ok bool) {
	v, ok := get[Key, any](rec, k)
	if !ok {
		return
	}

	value, ok = v.(Value)
	return
}

func get[Key comparable, Value any](rec Record[Key, any], k Key) (value Value, ok bool) {
	if rec == nil || !rec.Contains(k) {
		return
	}

	value = rec.Get(k)
	if value == nil {
		return
	}

	ok = true
	return
}
