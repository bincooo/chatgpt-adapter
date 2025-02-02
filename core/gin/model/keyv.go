package model

import (
	"encoding/json"
	"maps"
	"reflect"
	"strings"
)

type Keyv[V any] map[string]V

func (kv Keyv[V]) Set(key string, value V)  { kv[key] = value }
func (kv Keyv[V]) Get(key string) (V, bool) { value, ok := kv[key]; return value, ok }
func (kv Keyv[V]) Has(key string) bool      { _, ok := kv.Get(key); return ok }
func (kv Keyv[V]) String() string           { bytes, _ := json.Marshal(kv); return string(bytes) }
func (kv Keyv[V]) Clone() Keyv[V]           { return maps.Clone(kv) }

func (kv Keyv[V]) GetKeyv(key string) (value Keyv[interface{}]) {
	if val, ok := kv[key]; ok {
		var v interface{} = val
		if n, o := v.(map[string]interface{}); o {
			value = n
		}
	}
	return
}

func (kv Keyv[V]) GetSlice(key string) (values []interface{}) {
	if value, ok := kv[key]; ok {
		var v interface{} = value
		values, ok = v.([]interface{})
	}
	return
}

func (kv Keyv[V]) GetString(key string) (value string) {
	if val, ok := kv[key]; ok {
		var v interface{} = val
		value, ok = v.(string)
	}
	return
}

func (kv Keyv[V]) GetInt(key string) (value int) {
	if val, ok := kv[key]; ok {
		var v interface{} = val
		value, ok = v.(int)
	}
	return
}

func (kv Keyv[V]) Is(key string, value V) (out bool) {
	if !kv.Has(key) {
		return
	}

	v, _ := kv.Get(key)
	return reflect.DeepEqual(v, value)
}

func (kv Keyv[V]) In(key string, values ...V) (out bool) {
	if !kv.Has(key) {
		return
	}

	v, _ := kv.Get(key)
	for _, value := range values {
		if reflect.DeepEqual(v, value) {
			return true
		}
	}
	return
}

func (kv Keyv[V]) IsString(key string) bool {
	if value, ok := kv[key]; ok {
		var v interface{} = value
		if _, ok = v.(string); ok {
			return true
		}
	}
	return false
}

func (kv Keyv[V]) IsSlice(key string) bool {
	if value, ok := kv[key]; ok {
		var v interface{} = value
		if _, ok = v.([]interface{}); ok {
			return true
		}
	}
	return false
}

func (kv Keyv[V]) IsE(key string) bool {
	value, ok := kv.Get(key)
	if ok {
		var v interface{} = value
		if str, o := v.(string); o {
			return strings.TrimSpace(str) == ""
		}
	}
	return true
}
