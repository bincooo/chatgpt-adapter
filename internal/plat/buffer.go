package plat

import (
	"github.com/bincooo/MiaoX/types"
	"time"
)

const (
	cacheMessageL    = 200
	cacheWaitTimeout = 10 * time.Second
)

type CacheBuffer struct {
	timer    time.Time
	cache    string
	complete string

	H func(self *CacheBuffer) error

	Closed bool
}

func (r *CacheBuffer) condition() bool {
	if r.Closed {
		return true
	}
	// n秒内缓存消息
	if r.timer.After(time.Now()) {
		return false
	}

	// n秒后消耗消息
	// 字数太少续n秒
	if len(r.cache) < cacheMessageL {
		r.timer = time.Now().Add(cacheWaitTimeout)
		return false
	}

	return true
}

func (r *CacheBuffer) Read() types.PartialResponse {
	if r.H == nil {
		panic("Please define handle first.")
	}

	var partialResponse types.PartialResponse
	if err := r.H(r); err != nil {
		partialResponse.Error = err
		return partialResponse
	}

	if !r.condition() {
		return partialResponse
	}

	if len(r.cache) > 0 {
		r.complete += r.cache
		partialResponse.Message = r.cache
		r.cache = ""
	}

	partialResponse.Closed = r.Closed
	return partialResponse
}
